package types

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// FleetFileManager manages files for a fleet of parallel AFL fuzzers.
// It manages a few top-level directories, but routes most of its
// responsibilities for individual fuzzers to the relevant AflFileManager.
//
// It can be run on either a roving client or a roving server, since both
// have the same directory structure. We follow the AFL directory convention,
// and add some conventions of our own:
//
// ├── output/
// │   ├── FUZZER_ID_123
// │   │    ├── crashes/
// │   │    │    ├── crash1
// │   │    │    ├── crash2
// │   │    │    └── crash3
// │   │    ├── hangs/
// │   │    │    ├── hang1
// │   │    │    ├── hang2
// │   │    │    └── hang3
// │   │    ├── queue/
// │   │    │    ├── queue1
// │   │    │    ├── queue2
// │   │    │    └── queue3
// │   │    └── fuzzer_stats
// │   └── FUZZER_ID_456
// │        ├── crashes/
// │        ├── hangs/
// │        ├── queue/
// │        └── fuzzer_stats
// ├── input/
// │   ├── input1
// │   ├── input2
// │   └── input999
// └── dict.txt
type FleetFileManager struct {
	Basedir string
}

// WriteOutput writes the given AflOutput for the given fuzzerId to the
// appropriate location
func (m FleetFileManager) WriteOutput(fuzzerId string, output *AflOutput) error {
	return m.aflFileManager(fuzzerId).WriteOutput(output)
}

// ReadOutputs reads all outputs for all fuzzers in the fleet that have
// output saved to disk.
//
// It returns this data as a map from fuzzerId => *AflOutput.
func (m FleetFileManager) ReadOutputs() (map[string]*AflOutput, error) {
	fuzzerIds, err := m.FuzzerIds()
	if err != nil {
		return nil, err
	}

	outputs := make(map[string]*AflOutput)
	for _, fuzzerId := range fuzzerIds {
		output, err := m.aflFileManager(fuzzerId).ReadOutput()
		// Something bad has happened so let's bail immediately
		if err != nil {
			return nil, err
		}
		// TODO(rob): this should already be a pointer
		outputs[fuzzerId] = &output
	}

	return outputs, nil
}

// FuzzerIds returns the IDs of all of the fuzzers in the fleet
// with output saved to disk.
func (m FleetFileManager) FuzzerIds() ([]string, error) {
	fileInfos, err := ioutil.ReadDir(m.TopLevelOutputDir())
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0)
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			fuzzerId := filepath.Base(fileInfo.Name())
			ids = append(ids, fuzzerId)
		}
	}

	return ids, nil
}

// ReadInputs reads the inputs for the fleet from disk.
//
// AFL's "inputs" are example test cases to give fuzzers somewhere to
// start from. We therefore only need to save one, top-level copy of them.
func (m FleetFileManager) ReadInputs() (*InputCorpus, error) {
	inputDir := filepath.Join(m.Basedir, "input")
	return ReadInputCorpus(inputDir)
}

// TopLevelOutputDir returns the path of the top-level output dir for the fleet
func (m FleetFileManager) TopLevelOutputDir() string {
	return filepath.Join(m.Basedir, "output")
}

// MkTopLevelOutputDir creates the top-level output dir for the fleet. If
// this dir already exists, it is a no-op
func (m FleetFileManager) MkTopLevelOutputDir() error {
	return os.MkdirAll(m.TopLevelOutputDir(), 0755)
}

// TopLevelInputDir returns the path of the top-level input dir for the fleet
func (m FleetFileManager) TopLevelInputDir() string {
	return filepath.Join(m.Basedir, "input")
}

// MkTopLevelInputDir creates the top-level input dir for the fleet. If
// this dir already exists, it is a no-op
func (m FleetFileManager) MkTopLevelInputDir() error {
	return os.MkdirAll(m.TopLevelInputDir(), 0755)
}

// WriteTopLevelInput writes the given input to the fleet's input dir
func (m FleetFileManager) WriteTopLevelInput(input *Input) error {
	return WriteInputToFile(input, m.TopLevelInputDir())
}

// MkAllOutputDirs makes all Afl output directories for the given fuzzer
func (m FleetFileManager) MkAllOutputDirs(fuzzerId string) error {
	return m.aflFileManager(fuzzerId).MkAllOutputDirs()
}

// MkQueueDir makes the queue directory for the given fuzzer
func (m FleetFileManager) MkQueueDir(fuzzerId string) error {
	return m.aflFileManager(fuzzerId).MkQueueDir()
}

// MkCrashesDir makes the crashes directory for the given fuzzer
func (m FleetFileManager) MkCrashesDir(fuzzerId string) error {
	return m.aflFileManager(fuzzerId).MkCrashesDir()
}

// WriteQueues writes all queues in the given `queues` map to disk. `queues` is of
// the form fuzzerId => *InputCorpus. This method is used by the server to persist
// to disk queues that are reported to it by clients.
func (m FleetFileManager) WriteQueues(queues *map[string]*InputCorpus) error {
	for fuzzerId, queue := range *queues {
		if err := m.MkQueueDir(fuzzerId); err != nil {
			return err
		}
		if err := m.aflFileManager(fuzzerId).WriteQueue(queue); err != nil {
			return err
		}
	}
	return nil
}

// ReadQueues reads all queues for all fuzzers with output saved to disk. It
// returns this data as a map from fuzzerId => *InputCorpus.
func (m FleetFileManager) ReadQueues() (map[string]*InputCorpus, error) {
	fuzzerIds, err := m.FuzzerIds()
	if err != nil {
		return nil, err
	}

	queues := make(map[string]*InputCorpus)
	for _, fuzzerId := range fuzzerIds {
		// NOTE: we read all of the output (not just the queue) for simplicity. We
		// can make a separate queue-reading method if we need to speed this up.
		aflFileManager := m.aflFileManager(fuzzerId)
		queue, err := aflFileManager.ReadQueue()
		if err != nil {
			return nil, err
		}
		queues[fuzzerId] = queue
	}

	return queues, nil
}

// ReadInput reads the given input from the given fuzzer
func (m FleetFileManager) ReadInput(fuzzerId, inputType, inputName string) (*Input, error) {
	return m.aflFileManager(fuzzerId).ReadInput(inputType, inputName)
}

// InputPath returns the path for the given input from the given fuzzer
func (m FleetFileManager) InputPath(fuzzerId, inputType, inputName string) (string, error) {
	return m.aflFileManager(fuzzerId).InputPath(inputType, inputName)
}

// CrashPath returns the path for the given crash from the given fuzzer
func (m FleetFileManager) CrashPath(fuzzerId, name string) (string, error) {
	return m.InputPath(fuzzerId, Crashes, name)
}

// ReadDict reads the contents of the AFL seed dict (if any)
func (m FleetFileManager) ReadDict() ([]byte, error) {
	return ioutil.ReadFile(m.DictPath())
}

// WriteDict writes an AFL seed dict to disk. This is used by the clients when
// they receive the dict from the server.
func (m FleetFileManager) WriteDict(dict []byte) error {
	f, err := os.OpenFile(m.DictPath(), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	_, err = f.Write(dict)
	return err
}

// DictPath returns the path of the AFL seed dict. AFL allows dicts to be stored
// anywhere, but we use the convention that it should be stored at the top-level,
// alongside the `input/` and `output/` dirs.
func (m FleetFileManager) DictPath() string {
	return filepath.Join(m.Basedir, "dict.txt")
}

// aflFileManager returns an AflFileManager for the given fuzzer
func (m FleetFileManager) aflFileManager(fuzzerId string) *AflFileManager {
	return NewAflFileManagerWithFuzzerId(
		m.Basedir,
		fuzzerId,
	)
}
