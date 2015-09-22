package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/richo/roving/types"
)

type Server struct {
	hostport string
}

type Fuzzer struct {
	cmd *exec.Cmd
	Id  string
}

func newFuzzer() Fuzzer {
	name, err := os.Hostname()
	if err != nil {
		log.Fatal("Couldn't get hostname", err)
	}

	number := types.RandInt() & 0xffff

	return Fuzzer{
		Id: fmt.Sprintf("%s-%x", name, number),
	}
}

type WatchDog struct {
	Interval time.Duration
	Fuzzer   *Fuzzer
	Server   *Server
}

func (f *Fuzzer) run() {
	f.cmd = exec.Command(f.path(),
		"-o", "output",
		"-i", "input",
		"-S", f.Id,
		"./target",
	)
	stdout, err := f.cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Couldn't get stdout handle", err)
	}

	stderr, err := f.cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Couldn't get stderr handle", err)
	}

	if err := f.cmd.Start(); err != nil {
		log.Fatalf("Couldn't start fuzzer", err)
	}

	go func() {
		io.Copy(os.Stdout, stdout)
	}()

	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	log.Printf("Waiting for fuzzer to exit")

	err = f.cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

}

func (f *Fuzzer) stop() {
	log.Printf("Stopping the fuzzer")
	f.cmd.Process.Signal(syscall.SIGSTOP)
}

func (f *Fuzzer) start() {
	log.Printf("Starting the fuzzer")
	f.cmd.Process.Signal(syscall.SIGCONT)
}

func (f *Fuzzer) State() types.State {
	state := types.State{
		Id:      f.Id,
		Queue:   types.ReadQueue(fmt.Sprintf("output/%s/queue", f.Id)),
		Crashes: types.ReadDir(fmt.Sprintf("output/%s/crashes", f.Id)),
		Hangs:   types.ReadDir(fmt.Sprintf("output/%s/hangs", f.Id)),
	}

	return state
}

func (f *Fuzzer) path() string {
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
		select {
		case <-ticker.C:
			w.Fuzzer.stop()
			defer w.Fuzzer.start()

			log.Printf("Uploading our corpus")
			state := w.Fuzzer.State()
			w.Server.UploadState(state)

			log.Printf("Downloading their corpus")
			other := w.Server.FetchState(w.Fuzzer.Id)

			log.Printf("Unpacking their state")
			w.Fuzzer.UnpackStates(other)
		}
	}
}

func (s *Server) fetchToFile(resource, file string) {
	target := s.getPath(resource)
	defer target.Body.Close()

	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE, 0755)

	if err != nil {
		log.Panicf("Couldn't open %s for writing", file, err)
	}

	io.Copy(f, target.Body)
}

func (s *Server) getPath(path string) *http.Response {
	resource := fmt.Sprintf("http://%s/%s", s.hostport, path)
	log.Printf("Fetching %s", resource)

	resp, err := http.Get(resource)
	if err != nil {
		log.Panicf("Couldn't fetch target", err)
	}

	return resp
}

func (s *Server) FetchTarget() {
	s.fetchToFile("target", "target")
}

func base64ToPath(content, path string) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0755)

	if err != nil {
		log.Panicf("Couldn't open %s for writing", path, err)
	}

	body, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		log.Fatalf("Couldn't decode body for %s", path, err)
	}
	f.Write([]byte(body))
}

func (s *Server) FetchInputs() {
	os.Mkdir("input", 0755)

	inputs := s.getPath("inputs")
	defer inputs.Body.Close()

	inps := &types.InputCorpus{}

	encoder := json.NewDecoder(inputs.Body)
	encoder.Decode(&inps)

	for _, inp := range inps.Inputs {
		path := fmt.Sprintf("input/%s", inp.Name)
		base64ToPath(inp.Body, path)
	}
}

func (s *Fuzzer) UnpackStates(other []types.State) {
	for _, state := range other {
		cmd := exec.Command("tar", "-kxjf", "-")

		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Fatalf("Couldn't get stdin handle", err)
		}

		body, err := base64.StdEncoding.DecodeString(state.Queue)
		if err != nil {
			log.Fatal("Couldn't decode queue from %s", state.Id, err)
		}

		go func() {
			_, err := stdin.Write(body)
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

func (s *Server) FetchState(id string) []types.State {
	var state []types.State
	resp := s.getPath(fmt.Sprintf("state/%s", id))

	encoder := json.NewDecoder(resp.Body)
	encoder.Decode(&state)

	return state
}

func (s *Server) UploadState(state types.State) {
	data, err := json.Marshal(state)
	if err != nil {
		log.Fatal("Couldn't marshall state", err)
	}

	buffer := bytes.NewReader(data)

	resource := fmt.Sprintf("http://%s/%s", s.hostport, "state")
	_, err = http.Post(resource, "application/json", buffer)

	if err != nil {
		log.Panicf("Couldn't upload state", err)
	}
}

func setupWorkDir() {
	var err error
	// TODO(richo) Ephemeral workdirs for concurrency
	if err = os.Mkdir("work", 0755); err != nil {
		log.Println("Workdir already exists, assuming we're joining an existing run")
	}
	if err = os.Chdir("work"); err != nil {
		log.Panicf("Couldn't change to workdir", err)
	}
}

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Printf("Usage: ./client server:port")
		return
	}

	setupWorkDir()

	server := Server{args[1]}
	server.FetchTarget()
	server.FetchInputs()

	fuzzer := newFuzzer()

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
	go watchdog.run()

	fuzzer.run()

	// TODO(richo) Fetch the input corpus
}
