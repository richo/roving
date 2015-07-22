package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

type Server struct {
	hostport string
}

type Fuzzer struct {
}

func (f *Fuzzer) run() {
	// HACK
	os.Mkdir("input", 0755)
	fh, _ := os.OpenFile("input/hi", os.O_WRONLY|os.O_CREATE, 0755)
	fh.Write([]byte("hi"))
	fh.Close()
	cmd := exec.Command(f.path(),
		"-o", "output",
		"-i", "input",
		"./target",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Couldn't get stdout handle", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Couldn't get stderr handle", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Couldn't start fuzzer", err)
	}

	go func() {
		io.Copy(os.Stdout, stdout)
	}()

	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	log.Printf("Waiting for fuzzer to exit")

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

}

func (f *Fuzzer) path() string {
	root := os.Getenv("AFL")
	if root == "" { // Not found, hopefully it's in PATH
		return "afl-fuzz"
	}
	return fmt.Sprintf("%s/afl-fuzz", root)
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

func setupWorkDir() {
	var err error
	// TODO(richo) Ephemeral workdirs for concurrency
	if err = os.Mkdir("work", 0755); err != nil {
		log.Panicf("Couldn't make workdir", err)
	}
	if err = os.Chdir("work"); err != nil {
		log.Panicf("Couldn't change to workdir", err)
	}
}

func fetchCorpus(server string) {
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

	fuzzer := Fuzzer{}
	fuzzer.run()

	// TODO(richo) Fetch the input corpus
}
