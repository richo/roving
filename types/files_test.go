package types

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateBadInputName(t *testing.T) {
	err := validateInputName("../../../../../etc/shadow")

	assert.NotEmpty(t, err, "Did not error when passed invalid inputName")
}

func TestValidateGoodInputName(t *testing.T) {
	err := validateInputName("id:000000,orig:1")

	assert.Empty(t, err, "Errored when passed good input")
}

func TestPickUpExistingFiles(t *testing.T) {
	dstDir, err := ioutil.TempDir("", "roving-server-file-manager-test")
	if err != nil {
		t.Fatal(err)
	}

	fm1 := NewAflFileManager(dstDir)
	fm1.MkAllOutputDirs()
	fm1.MkInputDir()

	queue := InputCorpus{
		Inputs: []Input{
			Input{
				Name: "queue1",
				Body: []byte("queue1-body"),
			},
			Input{
				Name: "queue2",
				Body: []byte("queue2-body"),
			},
		},
	}
	crashes := InputCorpus{
		Inputs: []Input{
			Input{
				Name: "crash1",
				Body: []byte("crash1-body"),
			},
			Input{
				Name: "crash2",
				Body: []byte("crash2-body"),
			},
		},
	}
	hangs := InputCorpus{
		Inputs: []Input{
			Input{
				Name: "hang1",
				Body: []byte("hang1-body"),
			},
			Input{
				Name: "hang2",
				Body: []byte("hang2-body"),
			},
		},
	}
	err = fm1.WriteQueue(&queue)
	if err != nil {
		t.Fatal(err)
	}
	err = fm1.WriteCrashes(&crashes)
	if err != nil {
		t.Fatal(err)
	}
	err = fm1.WriteHangs(&hangs)
	if err != nil {
		t.Fatal(err)
	}

	fm2 := NewAflFileManager(dstDir)
	// Make sure that re-running the dir-creation commands
	// doesn't break anything.
	fm2.MkAllOutputDirs()
	fm2.MkInputDir()

	readQueue, err := fm2.ReadQueue()
	if err != nil {
		t.Fatal(err)
	}
	readCrashes, err := fm2.ReadCrashes()
	if err != nil {
		t.Fatal(err)
	}
	readHangs, err := fm2.ReadHangs()
	if err != nil {
		t.Fatal(err)
	}
	// Make sure that the second fileManager reads the same
	// data that was originally wirtten there by the first one.
	assert.Equal(t, queue, *readQueue)
	assert.Equal(t, crashes, *readCrashes)
	assert.Equal(t, hangs, *readHangs)
}
