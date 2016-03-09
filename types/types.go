package types

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
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

// TODO(richo) flesh this out at some point
type FuzzerStats struct {
}

type State struct {
	Id      string
	Stats   FuzzerStats
	Queue   []byte
	Crashes InputCorpus
	Hangs   InputCorpus
}

// Read the contents of a directory out
func ReadDir(path string) InputCorpus {
	corpus := InputCorpus{}
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

func RandInt() int64 {
	to := big.NewInt(1 << 62)
	i, err := rand.Int(rand.Reader, to)
	if err != nil {
		log.Fatal("Couldn't get a random number", err)
	}
	return i.Int64()
}
