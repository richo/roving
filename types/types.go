package types

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

type InputCorpus struct {
	Inputs []Input
}

func (i *InputCorpus) Add(other Input) {
	i.Inputs = append(i.Inputs, other)
}

type Input struct {
	Name string
	Body []byte
}

func (i *Input) WriteToPath(path string) {
	path = fmt.Sprintf("%s/%s", path, i.Name)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Panicf("Couldn't open %s for writing", path, err)
	}

	f.Write(i.Body)
}

type State struct {
	Id      string
	Stats   FuzzerStats
	Queue   []byte
	Crashes InputCorpus
	Hangs   InputCorpus
}

type Target struct {
	Metadata TargetMetadata
	Binary   []byte
}

type TargetMetadata struct {
	ShouldDownload bool
	Command        []string
}

func ReadStats(path string) FuzzerStats {
	// TODO urgh panic
	var buf []byte

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Couldn't read stats", err)
	}

	stats, err := ParseStats(string(buf))
	if err != nil {
		log.Fatalf("Couldn't read stats", err)
	}

	return *stats
}

// Read the contents of a directory out
func ReadDir(path string) InputCorpus {
	corpus := InputCorpus{[]Input{}}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatalf("Couldn't open %s", path, err)
	}

	for _, f := range files {
		path := fmt.Sprintf("%s/%s", path, f.Name())
		var buf []byte

		buf, err = ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("Couldn't read %s", path, err)
		}

		inp := Input{
			Name: f.Name(),
			Body: buf,
		}
		corpus.Add(inp)
	}
	return corpus
}

func ReadQueue(path string) []byte {
	cmd := exec.Command("tar", "-cjf", "-", path)

	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Couldn't tar up %s", path, err)
	}
	return output
}

func RandInt() (r uint64) {
	err := binary.Read(rand.Reader, binary.LittleEndian, &r)
	if err != nil {
		log.Fatalf("binary.Read failed:", err)
	}
	return
}
