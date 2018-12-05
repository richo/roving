package types

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Check that a round-trip from memory => disk => memory works
func TestReadAndWriteOutputs(t *testing.T) {
	basedir, err := ioutil.TempDir("", "roving-fleet-file-manager-test")
	if err != nil {
		t.Fatal(err)
	}

	fm := FleetFileManager{
		Basedir: basedir,
	}

	aflOutput1 := AflOutput{
		Queue: &InputCorpus{
			Inputs: []Input{
				Input{
					Name: "queue1-1",
					Body: []byte("queue1-1-body"),
				},
			},
		},
		Hangs: &InputCorpus{
			Inputs: []Input{
				Input{
					Name: "hang1-1",
					Body: []byte("hang1-1-body"),
				},
			},
		},
		Crashes: &InputCorpus{
			Inputs: []Input{
				Input{
					Name: "crash1-1",
					Body: []byte("crash1-1-body"),
				},
			},
		},
	}
	aflOutput2 := AflOutput{
		Queue: &InputCorpus{
			Inputs: []Input{
				Input{
					Name: "queue2-1",
					Body: []byte("queue2-1-body"),
				},
			},
		},
		Hangs: &InputCorpus{
			Inputs: []Input{
				Input{
					Name: "hang2-1",
					Body: []byte("hang2-1-body"),
				},
			},
		},
		Crashes: &InputCorpus{
			Inputs: []Input{
				Input{
					Name: "crash2-1",
					Body: []byte("crash2-1-body"),
				},
			},
		},
	}

	fm.MkAllOutputDirs("fuzzer1")
	fm.MkAllOutputDirs("fuzzer2")
	fm.WriteOutput("fuzzer1", &aflOutput1)
	fm.WriteOutput("fuzzer2", &aflOutput2)

	expectedOutputs := map[string]*AflOutput{
		"fuzzer1": &aflOutput1,
		"fuzzer2": &aflOutput2,
	}
	actualOutputs, err := fm.ReadOutputs()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expectedOutputs, actualOutputs)
}

// Check that a round-trip from memory => disk => memory works
func TestReadAndWriteQueues(t *testing.T) {
	basedir, err := ioutil.TempDir("", "roving-fleet-file-manager-test")
	if err != nil {
		t.Fatal(err)
	}

	fm := FleetFileManager{
		Basedir: basedir,
	}

	queue1 := InputCorpus{
		Inputs: []Input{
			Input{
				Name: "queue1-1",
				Body: []byte("queue1-1-body"),
			},
			Input{
				Name: "queue1-2",
				Body: []byte("queue1-2-body"),
			},
		},
	}
	queue2 := InputCorpus{
		Inputs: []Input{
			Input{
				Name: "queue2-1",
				Body: []byte("queue2-1-body"),
			},
			Input{
				Name: "queue2-2",
				Body: []byte("queue2-2-body"),
			},
		},
	}
	queues := map[string]*InputCorpus{
		"fuzzer1": &queue1,
		"fuzzer2": &queue2,
	}

	fm.MkQueueDir("fuzzer1")
	fm.MkQueueDir("fuzzer2")
	fm.WriteQueues(&queues)

	actualQueues, err := fm.ReadQueues()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, queues, actualQueues)
}
