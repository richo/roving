package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Server struct {
	hostport string
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
	resource := fmt.Sprintf("http://%s/%s", s.hostport)
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
}
