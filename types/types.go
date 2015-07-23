package types

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
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
	Body string // Base64 encoded body
}

// TODO(richo) flesh this out at some point
type FuzzerStats struct {
}

type State struct {
	Id    string
	Stats FuzzerStats
	Queue InputCorpus
	State string
}

// This doesn't belong here but what the hell

func ReadCorpus(path string) InputCorpus {
	corpus := InputCorpus{}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatalf("Couldn't open %s", path, err)
	}

	for _, f := range files {
		if f.Name() == ".state" {
			// This gets pulled out in a different pass
			continue
		}
		path := fmt.Sprintf("%s/%s", path, f.Name())
		var buf []byte

		buf, err = ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("Couldn't read %s", path, err)
		}

		inp := Input{
			Name: f.Name(),
			Body: base64.StdEncoding.EncodeToString(buf),
		}
		corpus.Add(inp)
	}
	return corpus
}

func ReadState(path string) string {
	fullpath := fmt.Sprintf("%s/.state", path)
	cmd := exec.Command("tar", "-cjf", "-", fullpath)

	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Couldn't tar up %s", fullpath, err)
	}
	return base64.StdEncoding.EncodeToString(output)
}
