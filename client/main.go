package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

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

func fetchTarget(server string) {
	log.Printf("Fetching target from %s", server)
	target := fmt.Sprintf("http://%s/target", server)
	resp, err := http.Get(target)
	if err != nil {
		log.Panicf("Couldn't fetch target", err)
	}

	defer resp.Body.Close()

	f, err := os.OpenFile("target", os.O_WRONLY|os.O_CREATE, 0755)

	if err != nil {
		log.Panicf("Couldn't open target for writing", err)
	}

	io.Copy(f, resp.Body)
}

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Printf("Usage: ./client server:port")
		return
	}

	setupWorkDir()
	fetchTarget(args[1])
}
