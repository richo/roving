package main

import (
	"flag"
	"log"
	"os/exec"
	"sync"
)

func main() {
	var wg sync.WaitGroup

	clientPathPtr := flag.String(
		"client-path",
		"../../../bazel-bin/roving/client/darwin_amd64_debug/client",
		"The path of the roving client binary")
	nClientsPtr := flag.Int(
		"n-clients",
		2,
		"The number of clients to run.")
	serverPtr := flag.String(
		"server",
		"",
		"The host:port of the roving server eg. localhost:1414")

	flag.Parse()

	clientPath := *clientPathPtr
	nClients := *nClientsPtr
	server := *serverPtr

	log.Printf("Running %d roving clients in parallel", nClients)
	log.Print("Not printing the logs from them, have a look at the server admin page to see what's happening")

	for n := 0; n < nClients; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command(
				clientPath,
				"-server",
				server)
			err := cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	wg.Wait()
}
