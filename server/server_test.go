package server

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/richo/roving/types"
)

func TestArchivingNewCrashes(t *testing.T) {
	var err error
	srcDir, err := ioutil.TempDir("", "roving-server-test-src")
	if err != nil {
		t.Fatal(err)
	}
	fileManager := types.FleetFileManager{Basedir: srcDir}

	dstDir, err := ioutil.TempDir("", "roving-server-test-dst")
	if err != nil {
		t.Fatal(err)
	}
	archiver := DiskArchiver{DstRoot: dstDir}

	crashes := []types.Input{
		types.Input{
			Name: "crash1",
			Body: []byte{1},
		},
		types.Input{
			Name: "crash2",
			Body: []byte{2},
		},
		types.Input{
			Name: "crash3",
			Body: []byte{3},
		},
	}
	output := types.AflOutput{
		Queue:   &types.InputCorpus{},
		Crashes: &types.InputCorpus{Inputs: crashes},
		Hangs:   &types.InputCorpus{},
	}
	if err = fileManager.MkAllOutputDirs("fuzzer-123"); err != nil {
		t.Fatal(err)
	}
	if err = fileManager.WriteOutput("fuzzer-123", &output); err != nil {
		t.Fatal(err)
	}

	names, err := archiver.LsDstFiles("realtime-crashes")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{}, names)

	archiveNewCrashes(&fileManager, archiver)

	archiveFileManager := types.NewAflFileManagerWithFuzzerId(
		filepath.Join(dstDir, "./realtime-crashes"),
		"fuzzer-123",
	)
	archivedCrashNames := make([]string, 0)
	archivedCrashes, err := archiveFileManager.ReadCrashes()
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range archivedCrashes.Inputs {
		archivedCrashNames = append(archivedCrashNames, i.Name)
	}

	assert.Equal(t, []string{
		"crash1",
		"crash2",
		"crash3",
	}, archivedCrashNames)

	crashes2 := []types.Input{
		types.Input{
			Name: "crash4",
			Body: []byte{4},
		},
		types.Input{
			Name: "crash5",
			Body: []byte{5},
		},
		types.Input{
			Name: "crash6",
			Body: []byte{6},
		},
	}
	output2 := types.AflOutput{
		Queue:   &types.InputCorpus{},
		Crashes: &types.InputCorpus{Inputs: crashes2},
		Hangs:   &types.InputCorpus{},
	}
	if err = fileManager.MkAllOutputDirs("fuzzer-456"); err != nil {
		t.Fatal(err)
	}
	if err = fileManager.WriteOutput("fuzzer-456", &output2); err != nil {
		t.Fatal(err)
	}
	fileManager.WriteOutput("fuzzer-456", &output2)

	archiveNewCrashes(&fileManager, archiver)

	archiveFileManager2 := types.NewAflFileManagerWithFuzzerId(
		filepath.Join(dstDir, "./realtime-crashes"),
		"fuzzer-456",
	)
	archivedCrashNames2 := make([]string, 0)
	archivedCrashes2, err := archiveFileManager2.ReadCrashes()
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range archivedCrashes2.Inputs {
		archivedCrashNames2 = append(archivedCrashNames2, i.Name)
	}

	assert.Equal(t, []string{
		"crash4",
		"crash5",
		"crash6",
	}, archivedCrashNames2)
}
