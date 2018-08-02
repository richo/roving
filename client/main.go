package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/richo/roving/types"
)

type RovingServer struct {
	hostport string
}

type FuzzCommand interface {
	cmd() *exec.Cmd
}

type Fuzzer struct {
	Id          string
	started     bool
	fuzzCommand FuzzCommand
}

type AFLFuzzCommand struct {
	fuzzerId      string
	memLimit      string
	targetCommand []string
}

func (f Fuzzer) run() error {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Couldn't get current directory")
	}
	log.Printf("Starting fuzzer in %s", dir)

	cmd := f.fuzzCommand.cmd()
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

	log.Printf("Waiting for fuzzer to exit")
	f.started = true
	return cmd.Wait()
}

func (f AFLFuzzCommand) cmd() *exec.Cmd {
	fullCommandArgs := append([]string{
		// Memory and timeout limits need to be really high in order to
		// cope with slow slow Ruby targets.
		//
		// TODO(rob): make these configurable
		"-m", "100000",
		"-t", "100000",
		"-o", "output",
		"-i", "input"}, f.targetCommand...)
	c := exec.Command(aflPath(), fullCommandArgs...)

	if f.memLimit != "" {
		c.Args = append(c.Args, "-m")
		c.Args = append(c.Args, f.memLimit)
	}
	return c
}

var preExisting bool = false

var invalidFuzzerNames *regexp.Regexp

func init() {
	invalidFuzzerNames = regexp.MustCompile("[^a-zA-Z0-9_-]")
}

func usableHostName(orig string) (valid string) {
	valid = invalidFuzzerNames.ReplaceAllString(orig, "_")
	// fuzzer name is ${hostname}-xxxx, so this string can be max 29 chars
	if len(valid) > 29 {
		valid = valid[0:29]
	}
	return
}

func newAFLFuzzer(targetCommand []string) Fuzzer {
	name, err := os.Hostname()
	if err != nil {
		log.Fatal("Couldn't get hostname", err)
	}

	name = usableHostName(name)
	number := types.RandInt() & 0xffff
	id := fmt.Sprintf("%s-%x", name, number)

	memLimit := os.Getenv("AFL_MEMORY_LIMIT")

	fuzzCommand := AFLFuzzCommand{
		fuzzerId:      id,
		memLimit:      memLimit,
		targetCommand: targetCommand,
	}

	return Fuzzer{
		Id:          id,
		started:     false,
		fuzzCommand: fuzzCommand,
	}
}

type WatchDog struct {
	Interval time.Duration
	Fuzzer   *Fuzzer
	Server   *RovingServer
}

func (f *Fuzzer) stop() {
	log.Printf("Stopping the fuzzer")
	f.fuzzCommand.cmd().Process.Signal(syscall.SIGSTOP)
}

func (f *Fuzzer) start() {
	log.Printf("Starting the fuzzer")
	f.fuzzCommand.cmd().Process.Signal(syscall.SIGCONT)
}

/// This function is distinct from "running". We're not testing for the process
//liveness, but instead asserting that the fuzzer has made it past the corpus
//test phase and has begun fuzzing.
func (f *Fuzzer) isFuzzing() bool {
	_, err := os.Stat(fmt.Sprintf("output/%s/fuzzer_stats", f.Id))
	return !os.IsNotExist(err)
}

func (f *Fuzzer) State() types.State {
	state := types.State{
		Id:      f.Id,
		Stats:   types.ReadStats(fmt.Sprintf("output/%s/fuzzer_stats", f.Id)),
		Queue:   types.ReadQueue(fmt.Sprintf("output/%s/queue", f.Id)),
		Crashes: types.ReadDir(fmt.Sprintf("output/%s/crashes", f.Id)),
		Hangs:   types.ReadDir(fmt.Sprintf("output/%s/hangs", f.Id)),
	}

	return state
}

func aflPath() string {
	root := os.Getenv("AFL")
	if root == "" { // Not found, hopefully it's in PATH
		return "afl-fuzz"
	}
	return fmt.Sprintf("%s/afl-fuzz", root)
}

func (w *WatchDog) run() {
	log.Printf("Starting watchdog goroutine")
	ticker := time.NewTicker(w.Interval)

	for {
		// Only sleep for a sec before the first sync. We want to upload our first
		// batch of stats really fast. That said, we do need to wait for the fuzzer
		// to actually start before we can read stats.
		time.Sleep(5 * time.Second)
		if w.Fuzzer.isFuzzing() {
			w.syncState()
			break
		}
	}

	for {
		select {
		case <-ticker.C:
			w.syncState()
		}
	}
}

func (w *WatchDog) syncState() {
	if w.Fuzzer.started && w.Fuzzer.isFuzzing() {
		w.Fuzzer.stop()
		defer w.Fuzzer.start()
		w.uploadState()
	}

	w.downloadState()
}

func (w *WatchDog) uploadState() {
	log.Printf("Uploading our corpus")
	state := w.Fuzzer.State()
	w.Server.UploadState(state)
}

func (w *WatchDog) downloadState() {
	log.Printf("Downloading their corpus")
	other := w.Server.FetchState(w.Fuzzer.Id)

	log.Printf("Unpacking their state")
	w.Fuzzer.UnpackStates(other)
}

func (s *RovingServer) fetchToFile(resource, file string) error {
	target := s.getPath(resource)
	defer target.Body.Close()

	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE, 0755)

	if err != nil {
		return err
	}

	io.Copy(f, target.Body)
	f.Close()

	return nil
}

func (s *RovingServer) getPath(path string) *http.Response {
	resource := fmt.Sprintf("%s/%s", s.hostport, path)
	log.Printf("Fetching %s", resource)

	resp, err := http.Get(resource)
	if err != nil {
		log.Panicf("Couldn't fetch target: %s", err)
	}

	return resp
}

// Fetch the target's metadata, including whether we need
// to download the target and what command we should use
// to run it.
func (s *RovingServer) FetchTargetMeta() types.TargetMetadata {
	res := s.getPath("target/meta")
	defer res.Body.Close()

	targetMeta := &types.TargetMetadata{}

	encoder := json.NewDecoder(res.Body)
	encoder.Decode(targetMeta)

	return *targetMeta
}

// Fetch the target binary (if necessary)
func (s *RovingServer) FetchTargetBinary(file string) {
	if err := s.fetchToFile("target/binary", file); err != nil {
		if preExisting {
			log.Printf("Couldn't write target, ignoring since this tree is preexisting")
		} else {
			log.Fatalf("Couldn't fetch target: %s", err)
		}
	}
}

func WriteToPath(content []byte, path string) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0755)

	if err != nil {
		log.Panicf("Couldn't open %s for writing: %s", path, err)
	}

	f.Write(content)
}

func (s *RovingServer) FetchInputs() {
	os.Mkdir("input", 0755)

	inputs := s.getPath("inputs")
	defer inputs.Body.Close()

	inps := &types.InputCorpus{}

	encoder := json.NewDecoder(inputs.Body)
	encoder.Decode(&inps)

	for _, inp := range inps.Inputs {
		path := fmt.Sprintf("input/%s", inp.Name)
		WriteToPath(inp.Body, path)
	}
}

func (s *Fuzzer) UnpackStates(other []types.State) {
	for _, state := range other {
		cmd := exec.Command("tar", "-kxjf", "-")

		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Fatalf("Couldn't get stdin handle: %s", err)
		}

		go func() {
			_, err := stdin.Write(state.Queue)
			if err != nil {
				log.Fatal("Error writing data into tar", err)
			}
			stdin.Close()
		}()

		_ = cmd.Run()
		// YOLO
		// if err != nil {
		// 	log.Fatalf("Couldn't untar state from %s", state.Id, err)
		// }
	}
}

func (s *RovingServer) FetchState(id string) []types.State {
	var state []types.State
	resp := s.getPath(fmt.Sprintf("state/%s", id))

	encoder := json.NewDecoder(resp.Body)
	encoder.Decode(&state)

	return state
}

func (s *RovingServer) UploadState(state types.State) {
	data, err := json.Marshal(state)
	if err != nil {
		log.Fatal("Couldn't marshall state", err)
	}

	buffer := bytes.NewReader(data)

	resource := fmt.Sprintf("%s/%s", s.hostport, "state")
	resp, err := http.Post(resource, "application/json", buffer)
	defer resp.Body.Close()

	if err != nil {
		log.Panicf("Couldn't upload state: %s", err)
	}
}

func setupWorkDir() {
	var err error
	// TODO(richo) Ephemeral workdirs for concurrency
	if err = os.Mkdir("work", 0755); err == nil {
		log.Println("Created new working directory")
	} else {
		log.Println("Workdir already exists, assuming we're joining an existing run")
		preExisting = true
	}
	if err = os.Chdir("work"); err != nil {
		log.Panicf("Couldn't change to workdir: %s", err)
	}
}

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Printf("Usage: ./client server:port")
		return
	}

	serverHostPort := args[1]
	httpPrefix := "http://"
	if !strings.HasPrefix(serverHostPort, httpPrefix) {
		serverHostPort = httpPrefix + serverHostPort
	}
	log.Printf("Server has address " + serverHostPort)

	Start(serverHostPort)
}

func Start(serverHostPort string) {
	setupWorkDir()

	server := RovingServer{serverHostPort}

	var targetCommand []string
	targetMeta := server.FetchTargetMeta()
	if targetMeta.ShouldDownload {
		log.Printf("Downloading binary from server")

		targetBinaryFilename := "../target"
		server.FetchTargetBinary(targetBinaryFilename)

		targetCommand = []string{targetBinaryFilename}
	} else {
		log.Printf("Not downloading binary from server")

		targetCommand = targetMeta.Command
	}
	log.Printf("TargetCommand:\t%s", targetCommand)

	server.FetchInputs()

	fuzzer := newAFLFuzzer(targetCommand)

	log.Printf("Brought up a fuzzer with id %s", fuzzer.Id)

	var interval time.Duration
	if os.Getenv("SHORT_INTERVAL") != "" {
		interval = 3 * time.Second
	} else {
		interval = 10 * time.Minute
	}

	watchdog := WatchDog{
		Interval: interval,
		Fuzzer:   &fuzzer,
		Server:   &server,
	}

	log.Printf("Priming with upstream state")
	watchdog.syncState()

	go watchdog.run()

	err := fuzzer.run()

	watchdog.uploadState()

	if err != nil {
		log.Fatal(err)
	}
}
