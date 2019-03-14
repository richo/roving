package types

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var readmeFilename = "README.txt"
var Crashes string = "crashes"
var Hangs string = "hangs"
var Queue string = "queue"

// AflFileManager is your one-stop shop for interacting with files
// written to the filesystem by AFL. Once you give it the base
// input and output directories, it knows where to find all of AFL's
// outputs and inputs, including the Queue, Crashes, and Hangs.
//
// It also knows how to read this data into golang structs, and how
// to write these structs back to the filesystem.
//
// You should never find yourself interacting with AFL's files
// directly yourself. If you do, please update AflFileManager to
// support your usecase!
//
// For an ASCII sketch of the directory structure used by roving,
// see fleet_file_manager.go.
type AflFileManager struct {
	basedir  string
	fuzzerId string
}

func (m AflFileManager) InputDir() string {
	return filepath.Join(m.basedir, "input")
}

// AFL accepts a -o option that tels it the directory to which
// it should write its output. If it has been passed a fuzzer ID
// then it writes the output to `./$OUTPUT_DIR/$FUZZER_ID/...`.
// If it has not then it writes it to `./$OUTPUT_DIR/...`. We
// should therefore take care to *read* data from `OutputDir()`, but
// pass `OutputDirToPassIntoAfl()` into AFL's -o option.
func (m AflFileManager) OutputDir() string {
	if m.fuzzerId == "" {
		return m.OutputDirToPassIntoAfl()
	} else {
		return filepath.Join(m.OutputDirToPassIntoAfl(), m.fuzzerId)
	}
}

func (m AflFileManager) OutputDirToPassIntoAfl() string {
	return filepath.Join(m.basedir, "output")
}

// WriteOutput writes all of the corpuses in AflOutput to the correct
// location on disk.
func (m AflFileManager) WriteOutput(output *AflOutput) error {
	var err error
	err = m.WriteQueue(output.Queue)
	if err != nil {
		return err
	}
	err = m.WriteCrashes(output.Crashes)
	if err != nil {
		return err
	}
	err = m.WriteHangs(output.Hangs)
	if err != nil {
		return err
	}
	return nil
}

func (m AflFileManager) WriteQueue(queue *InputCorpus) error {
	return WriteInputCorpusToFile(queue, m.QueueDir())
}

func (m AflFileManager) WriteCrashes(crashes *InputCorpus) error {
	return WriteInputCorpusToFile(crashes, m.CrashesDir())
}

func (m AflFileManager) WriteHangs(hangs *InputCorpus) error {
	return WriteInputCorpusToFile(hangs, m.HangsDir())
}

func (m AflFileManager) WriteInputs(inputs *InputCorpus) error {
	return WriteInputCorpusToFile(inputs, m.InputDir())
}

func (m AflFileManager) corpusDir(corpusType string) (string, error) {
	var dir string
	var err error

	switch corpusType {
	case Queue:
		dir = m.QueueDir()
	case Crashes:
		dir = m.CrashesDir()
	case Hangs:
		dir = m.HangsDir()
	default:
		dir = ""
		err = errors.New(fmt.Sprintf("Unknown corpusType: %s", corpusType))
	}
	return dir, err
}

// ReadOutput reads all of AFL's corpuses and returns them in an
// AflOutput.
func (m AflFileManager) ReadOutput() (AflOutput, error) {
	var err error

	queue, err := m.ReadQueue()
	if err != nil {
		return AflOutput{}, err
	}
	crashes, err := m.ReadCrashes()
	if err != nil {
		return AflOutput{}, err
	}
	hangs, err := m.ReadHangs()
	if err != nil {
		return AflOutput{}, err
	}

	return AflOutput{
		Queue:   queue,
		Crashes: crashes,
		Hangs:   hangs,
	}, nil
}

func (m AflFileManager) ReadQueue() (*InputCorpus, error) {
	return ReadInputCorpus(m.QueueDir())
}

func (m AflFileManager) ReadCrashes() (*InputCorpus, error) {
	return ReadInputCorpus(m.CrashesDir())
}

func (m AflFileManager) ReadHangs() (*InputCorpus, error) {
	return ReadInputCorpus(m.HangsDir())
}

func (m AflFileManager) ReadInputs() (*InputCorpus, error) {
	return ReadInputCorpus(m.InputDir())
}

func (m AflFileManager) ReadInput(inputType, inputName string) (*Input, error) {
	inputPath, err := m.InputPath(inputType, inputName)
	if err != nil {
		return nil, err
	}
	input, err := readInput(inputPath)
	if err != nil {
		return nil, err
	}

	return input, nil
}

func (m AflFileManager) InputPath(inputType, inputName string) (string, error) {
	inputDir, err := m.corpusDir(inputType)
	if err != nil {
		return "", err
	}

	err = validateInputName(inputName)
	return filepath.Join(inputDir, inputName), nil
}

func (m AflFileManager) ReadFuzzerStats() (*FuzzerStats, error) {
	buf, err := ioutil.ReadFile(m.FuzzerStatsPath())
	if err != nil {
		return nil, err
	}

	return ParseStats(string(buf))
}

func (m AflFileManager) MkAllOutputDirs() error {
	var err error
	if err = m.MkQueueDir(); err != nil {
		return err
	}
	if err = m.MkHangsDir(); err != nil {
		return err
	}
	if err = m.MkCrashesDir(); err != nil {
		return err
	}
	return nil
}

func (m AflFileManager) MkQueueDir() error {
	return os.MkdirAll(m.QueueDir(), 0755)
}

func (m AflFileManager) MkCrashesDir() error {
	return os.MkdirAll(m.CrashesDir(), 0755)
}

func (m AflFileManager) MkHangsDir() error {
	return os.MkdirAll(m.HangsDir(), 0755)
}

func (m AflFileManager) MkInputDir() error {
	return os.MkdirAll(m.InputDir(), 0755)
}

func (m AflFileManager) QueueDir() string {
	return filepath.Join(m.OutputDir(), Queue)
}

func (m AflFileManager) CrashesDir() string {
	return filepath.Join(m.OutputDir(), Crashes)
}

func (m AflFileManager) HangsDir() string {
	return filepath.Join(m.OutputDir(), Hangs)
}

func (m AflFileManager) FuzzerStatsPath() string {
	return filepath.Join(m.OutputDir(), "fuzzer_stats")
}

// AFL uses a different directory structure for fuzzers
// that have an ID. The input directory stays the same, but
// output is stored in `./output/$ID/[queue]`, instead of
// `./output/[queue]`.
func NewAflFileManagerWithFuzzerId(basedir, fuzzerId string) *AflFileManager {
	return &AflFileManager{
		basedir:  basedir,
		fuzzerId: fuzzerId,
	}
}

func NewAflFileManager(basedir string) *AflFileManager {
	return &AflFileManager{
		basedir: basedir,
	}
}

// ReadInputCorpus reads all files in the given dir and returns
// them as an InputCorpus.
func ReadInputCorpus(dir string) (*InputCorpus, error) {
	var err error

	corpus := InputCorpus{[]Input{}}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return &InputCorpus{}, err
	}

	for _, f := range files {
		if f.Name() == readmeFilename {
			continue
		}

		path := filepath.Join(dir, f.Name())

		fi, err := os.Stat(path)
		if err != nil {
			return &InputCorpus{}, err
		}
		if fi.Mode().IsDir() {
			continue
		}

		inp, err := readInput(path)
		if err != nil {
			return &InputCorpus{}, err
		}
		corpus.Add(*inp)
	}
	return &corpus, nil
}

func readInput(path string) (*Input, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return &Input{}, err
	}

	_, name := filepath.Split(path)
	inp := Input{
		Name: name,
		Body: buf,
	}
	return &inp, nil
}

func validateInputName(inputName string) error {
	// Guard against inputName being `../../../../etc/shadow`
	baseInputName := filepath.Base(inputName)
	if baseInputName != inputName {
		err := errors.New(
			fmt.Sprintf(
				"!! inputName appears to be trying to escalate up the dir structure: %s",
				inputName,
			),
		)
		return err
	}
	return nil
}
