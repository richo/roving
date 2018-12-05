package types

import (
	"crypto/rand"
	"encoding/binary"
	"log"
	"os"
	"path/filepath"
)

// State is a struct representing the current state of a fuzzer
type State struct {
	Id        string
	Stats     FuzzerStats
	AflOutput AflOutput
}

type TargetBinary = []byte

// AflOutput is a struct representing the output dir of a fuzzer
type AflOutput struct {
	Queue   *InputCorpus
	Crashes *InputCorpus
	Hangs   *InputCorpus
}

// InputCorpus is a collection of Inputs. It is used to represent
// the crashes, queue and hangs AFL output directories, as well
// as the input directory.
type InputCorpus struct {
	Inputs []Input
}

func (i *InputCorpus) Add(other Input) {
	i.Inputs = append(i.Inputs, other)
}

// Input is an AFL test case. This can be a test case
// from anywhere - `output/crashes`, `output/queue`, `output/hangs`,
// or `input/`.
type Input struct {
	Name string
	Body []byte
}

// WriteInputCorpusToFile writes each Input in the given
// InputCorpus to a separate file.
func WriteInputCorpusToFile(inputCorpus *InputCorpus, dir string) error {
	for _, input := range inputCorpus.Inputs {
		// Bail immediately if we can't write an input, since we presumably
		// won't be able to write any of the others either.
		if err := WriteInputToFile(&input, dir); err != nil {
			return err
		}
	}
	return nil
}

// WriteInputToFile writes a single Input to a single file.
// The filename is given by the name of the Input.
func WriteInputToFile(i *Input, dir string) error {
	fp := filepath.Join(dir, i.Name)
	f, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	_, err = f.Write(i.Body)
	return err
}

func RandInt() (r uint64) {
	err := binary.Read(rand.Reader, binary.LittleEndian, &r)
	if err != nil {
		log.Fatalf("binary.Read failed: %s", err)
	}
	return
}
