package client

import (
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/stripe/veneur/ssf"
	"github.com/stripe/veneur/trace"

	"github.com/richo/roving/types"
)

// RunFuzzerForever kicks off a fuzzer. It constructs all of the necessary output
// and input dirs, as well as the machinery needed to upload the fuzzer's state
// to the server.
func RunFuzzerForever(fuzzer *Fuzzer, serverClient *RovingServerClient, syncInterval time.Duration) {
	var err error
	err = fuzzer.fileManager.MkInputDir()
	if err != nil {
		log.Fatal(err)
	}

	inputs, err := serverClient.FetchInputs()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Downloaded inputs from server n_inputs=%d", len(inputs.Inputs))
	log.Printf("Writing inputs to disk dir=%v", fuzzer.fileManager.InputDir())

	err = fuzzer.fileManager.WriteInputs(inputs)
	if err != nil {
		log.Fatal(err)
	}

	err = fuzzer.fileManager.MkAllOutputDirs()
	if err != nil {
		log.Fatal(err)
	}

	stateUploader := StateUploader{
		Interval: syncInterval,
		Fuzzer:   fuzzer,
		Server:   serverClient,
	}

	log.Printf("We will upload our work to the server every %v", syncInterval)
	go stateUploader.run()

	err = fuzzer.run()

	log.Printf("Priming with upstream state")
	stateUploader.uploadState()

	if err != nil {
		log.Fatal(err)
	}
}

// SetupAndRun runs a fleet of fuzzers. It constructs and starts all of
// the individual fuzzers, as well as the fleet's monitoring and synchronization
// machinery.
func SetupAndRun(conf types.ClientConfig, workdir string, isNewRun bool) {
	trace.Service = "roving-client"
	ssf.NamePrefix = "roving-client."

	parallelism := conf.Parallelism

	serverClient := NewRovingServerClient(conf.ServerAddress)
	fuzzerConfig, err := serverClient.FetchFuzzerConfig()
	if err != nil {
		log.Fatal(err)
	}

	var targetCommand []string
	if fuzzerConfig.UseBinary {
		log.Printf("Downloading binary from server")

		fetchBinaryTo := "./target"
		err := serverClient.FetchTargetBinary(fetchBinaryTo)
		if err != nil {
			if isNewRun {
				log.Fatalf("Couldn't fetch target: %s", err)
			} else {
				log.Printf("Couldn't write target, ignoring since this tree is preexisting")
			}
		}

		targetCommand = []string{fetchBinaryTo}
	} else {
		log.Printf("Not downloading binary from server")

		targetCommand = fuzzerConfig.Command
	}

	fleetFileManager := types.FleetFileManager{
		Basedir: workdir,
	}

	var dictPath string
	if fuzzerConfig.UseDict {
		dictPath = fleetFileManager.DictPath()

		log.Printf("Attempting to downloaded dict from server")
		dict, err := serverClient.FetchDict()
		if err != nil {
			log.Fatal(err)
		}

		if len(dict) > 0 {
			log.Printf("Writing dict to disk path=%v bytes=%d", fleetFileManager.DictPath(), len(dict))

			err = fleetFileManager.WriteDict(dict)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal("Server did not return a dict!")
		}
	} else {
		dictPath = ""
		log.Print("Not downloading dict from server because use_dict is false")
	}

	log.Printf("TargetCommand:\t%s", targetCommand)
	log.Printf("Parallelism:\t%d (num cores: %d)", parallelism, runtime.NumCPU())

	queueDownloader := QueueDownloader{
		Interval:    fuzzerConfig.SyncInterval,
		Server:      serverClient,
		fileManager: &fleetFileManager,
	}

	// Download the queue immediately and syncronously before starting any
	// fuzzers.
	queueDownloader.downloadQueues()
	go queueDownloader.run()

	var wg sync.WaitGroup
	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func(fuzzerN int) {
			fuzzer := newAFLFuzzer(targetCommand, workdir, dictPath, fuzzerConfig.TimeoutMs, fuzzerConfig.MemLimitMb)
			log.Printf("Initialized fuzzer n=%v id=%v", fuzzerN, fuzzer.Id)

			RunFuzzerForever(&fuzzer, serverClient, fuzzerConfig.SyncInterval)
			wg.Done()
		}(i)
	}
	wg.Wait()
}
