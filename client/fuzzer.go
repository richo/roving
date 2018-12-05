package client

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/richo/roving/types"
)

var invalidFuzzerNames *regexp.Regexp

func init() {
	invalidFuzzerNames = regexp.MustCompile("[^a-zA-Z0-9_-]")
}

// Fuzzer runs an AFL fuzzer by shelling out to `afl-fuzz`.
// It keeps track of the fuzzer's process, and of its
// progress using an AflFileManager.
type Fuzzer struct {
	Id          string
	fileManager *types.AflFileManager
	started     bool
	cmd         *exec.Cmd
}

// run starts the fuzzer and sets up its output pipes.
// Once the fuzz command has started, run should never return
// unless something goes wrong with the command.
func (f *Fuzzer) run() error {
	var err error

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Couldn't get cwd: %s", err)
	}
	log.Printf("Starting fuzzer in %s", cwd)

	cmd := f.cmd
	log.Printf("%s %s", cmd.Path, strings.Join(cmd.Args, " "))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Couldn't get stdout handle: %s", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Couldn't get stderr handle: %s", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Couldn't start fuzzer: %s", err)
	}

	go func() {
		io.Copy(os.Stdout, stdout)
	}()

	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	log.Printf("Started fuzzer")
	f.started = true
	return cmd.Wait()
}

// stop pauses the fuzz process by sending it a SIGSTOP.
func (f *Fuzzer) stop() {
	log.Printf("Stopping the fuzzer")
	f.cmd.Process.Signal(syscall.SIGSTOP)
}

// start restarts the fuzz process after it has been stopped
// by sending it a SIGCONT.
func (f *Fuzzer) start() {
	log.Printf("Starting the fuzzer")
	f.cmd.Process.Signal(syscall.SIGCONT)
}

// hasBegunFuzzing returns whether the fuzz command process has
// started fuzzing. This is distinct from "running". hasBegunFuzzing
// does not test for the process liveness, but instead whether the
// fuzz process has made it past the initialization phase and has
// begun the actual task of fuzzing.
func (f *Fuzzer) hasBegunFuzzing() bool {
	_, err := os.Stat(f.fileManager.FuzzerStatsPath())
	return !os.IsNotExist(err)
}

// ReadState returns the State of the Fuzzer.
func (f *Fuzzer) ReadState() (types.State, error) {
	aflOutput, err := f.fileManager.ReadOutput()
	if err != nil {
		return types.State{}, err
	}
	stats, err := f.fileManager.ReadFuzzerStats()
	if err != nil {
		return types.State{}, err
	}
	return types.State{
		Id:        f.Id,
		Stats:     *stats,
		AflOutput: aflOutput,
	}, nil
}

// newAFLFuzzer returns a new fuzzer.
func newAFLFuzzer(targetCommand []string, workdir string, dictPath string, timeoutMs int, memLimitMb int) Fuzzer {
	name, err := os.Hostname()
	if err != nil {
		log.Fatal("Couldn't get hostname", err)
	}

	id := mkFuzzerId(name)
	fileManager := types.NewAflFileManagerWithFuzzerId(workdir, id)

	fuzzCmd := aflFuzzCmd(
		id,
		targetCommand,
		fileManager.OutputDirToPassIntoAfl(),
		fileManager.InputDir(),
		dictPath,
		aflFuzzPath(),
		timeoutMs,
		memLimitMb,
	)

	return Fuzzer{
		Id:          id,
		fileManager: fileManager,
		started:     false,
		cmd:         fuzzCmd,
	}
}

// aflFuzzPath returns the path to afl-fuzz. It first looks for an env var
// called `AFL`, which should be the path to the dir that afl-fuzz is in.
// If it does not find this var then it defaults to `afl-fuzz` and hopes
// that this is in PATH.
func aflFuzzPath() string {
	root := os.Getenv("AFL")
	if root == "" {
		return "afl-fuzz"
	}
	return fmt.Sprintf("%s/afl-fuzz", root)
}

// aflFuzzCmd constucts an afl-fuzz Cmd out of the given options.
func aflFuzzCmd(fuzzerId string, targetCommand []string, outputPath string, inputPath string, dictPath string, aflFuzzPath string, timeoutMs int, memLimitMb int) *exec.Cmd {
	cmdFlags := []string{
		"-S", fuzzerId,
		"-o", outputPath,
		"-i", inputPath,
	}
	if timeoutMs != 0 {
		cmdFlags = append(cmdFlags, "-t", strconv.Itoa(timeoutMs))
	}
	if memLimitMb != 0 {
		cmdFlags = append(cmdFlags, "-m", strconv.Itoa(memLimitMb))
	}

	if dictPath != "" {
		cmdFlags = append(cmdFlags, "-x", dictPath)
	}

	cmdFullArgs := append(cmdFlags, targetCommand...)
	c := exec.Command(aflFuzzPath, cmdFullArgs...)

	return c
}

// mkFuzzerId builds a fuzzerId out of a hostname and a random 4 char hexstring.
// It replaces non-alphanumeric chars in the hostname with underscores, and
// truncates it to 27 chars.
func mkFuzzerId(hostname string) string {
	validHostname := invalidFuzzerNames.ReplaceAllString(hostname, "_")
	// Max AFL fuzzer ID length is 32:
	// https://github.com/mirrorer/afl/blob/2fb5a3482ec27b593c57258baae7089ebdc89043/afl-fuzz.c#L7456
	//
	// Our fuzzer ID is ${hostname}-xxxx, so the hostname portion can
	// be max 32 - 5 = 27 chars.
	maxHostnameLen := 27
	if len(hostname) > maxHostnameLen {
		hostname = hostname[0:maxHostnameLen]
	}

	number := types.RandInt() & 0xffff
	return fmt.Sprintf("%s-%x", validHostname, number)
}
