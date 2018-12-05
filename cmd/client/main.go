package main

import (
	"flag"
	"log"
	"os"

	"github.com/richo/roving/client"
	"github.com/richo/roving/types"
)

// Returns a bool of whether new dir created
func setupWorkDir(workdir string) bool {
	var err error

	err = os.Mkdir(workdir, 0755)
	newDirCreated := err == nil

	log.Printf("Attempting to create new working directory at %s", workdir)
	if newDirCreated {
		log.Printf("Created new working directory")
	} else {
		log.Printf("Workdir already exists, assuming we're joining an existing run")
	}

	return newDirCreated
}

func main() {
	var conf types.ClientConfig
	var serverArg string

	var parallelismArg int
	flag.IntVar(
		&parallelismArg,
		"parallelism",
		1,
		"The number of fuzzers to run in parallel")

	flag.StringVar(
		&serverArg,
		"server-address",
		"",
		"The host:port address of the roving server")
	flag.Parse()

	conf.ServerAddress = serverArg
	conf.Parallelism = parallelismArg

	log.Printf("Server has address " + conf.ServerAddress)

	workdir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	isNewRun := setupWorkDir(workdir)
	client.SetupAndRun(conf, workdir, isNewRun)
}
