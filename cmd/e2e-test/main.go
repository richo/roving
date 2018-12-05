package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/richo/roving/client"
	"github.com/richo/roving/server"
	"github.com/richo/roving/types"
)

type StringSet map[string]bool

func main() {
	var workdir string
	flag.StringVar(
		&workdir,
		"workdir",
		"",
		"The root workdir")

	flag.Parse()

	if workdir == "" {
		log.Fatal("-workdir must be set!")
	}

	ts := time.Now().Unix()
	tsStr := strconv.FormatInt(ts, 10)

	serverWorkdir, _ := filepath.Abs(filepath.Join(workdir, "server", tsStr))
	clientWorkdir, _ := filepath.Abs(filepath.Join(workdir, "client", tsStr))

	go runServer(serverWorkdir)
	log.Printf("Sleeping for 1s to give the server time to start...")
	time.Sleep(time.Millisecond * 1000)

	log.Printf("Starting 2 roving clients, each running 2 AFLs...")
	parallelism := 2
	go runClient(clientWorkdir, parallelism)
	go runClient(clientWorkdir, parallelism)

	log.Printf("Sleeping for 7s to give the clients time to do a batch of work and sync it")
	time.Sleep(time.Millisecond * 7000)

	serverFileManager := types.NewAflFileManager(serverWorkdir)
	clientFileManagers := getClientFileManagers(clientWorkdir)

	printStats(serverFileManager, clientFileManagers)
}

func printStats(serverFileManager *types.AflFileManager, clientFileManagers []*types.AflFileManager) {
	sQueue, _ := serverFileManager.ReadQueue()
	sQueueNames := getCorpusNames(sQueue)

	sCrashes, _ := serverFileManager.ReadCrashes()
	sCrashNames := getCorpusNames(sCrashes)

	for n, cfm := range clientFileManagers {
		cQueue, _ := cfm.ReadQueue()
		cQueueNames := getCorpusNames(cQueue)
		qboth, qSnotC, qCnotS := diffs(sQueueNames, cQueueNames)

		cCrashes, _ := cfm.ReadCrashes()
		cCrashNames := getCorpusNames(cCrashes)
		crboth, crSnotC, crCnotS := diffs(sCrashNames, cCrashNames)

		fmt.Printf("###############\n")
		fmt.Printf("### CLIENT %d ###\n", n)
		fmt.Printf("###############\n")
		fmt.Printf("---QUEUE---\n")
		fmt.Printf("N in both server and client:\t %d\n", qboth)
		fmt.Printf("N in server not client:\t %d\n", qSnotC)
		fmt.Printf("N in client not server:\t %d\n", qCnotS)
		fmt.Printf("")
		fmt.Printf("---CRASHES---\n")
		fmt.Printf("N in both server and client:\t %d\n", crboth)
		fmt.Printf("N in server not client:\t %d\n", crSnotC)
		fmt.Printf("N in client not server:\t %d\n", crCnotS)
		fmt.Printf("\n")
	}
}

// Returns (
//   n in both,
//   n in 1 but not 2,
//   n in 2 but not 1,
// )
func diffs(i1, i2 StringSet) (int, int, int) {
	n1not2 := len(sub(i1, i2))
	n2not1 := len(sub(i2, i1))
	nboth := len(i1) - n1not2

	return nboth, n1not2, n2not1
}

// Returns i1-i2
func sub(i1, i2 StringSet) StringSet {
	ret := make(StringSet)
	for name, _ := range i1 {
		if !i2[name] {
			ret[name] = true
		}
	}
	return ret
}

func getCorpusNames(inputCorpus *types.InputCorpus) StringSet {
	names := make(StringSet)
	for _, i := range inputCorpus.Inputs {
		names[i.Name] = true
	}
	return names
}

func getClientFileManagers(baseWorkdir string) []*types.AflFileManager {
	clientOutputDirs, err := ioutil.ReadDir(filepath.Join(baseWorkdir, "output"))
	if err != nil {
		log.Fatal(err)
	}
	fms := []*types.AflFileManager{}
	for _, fi := range clientOutputDirs {
		fuzzerId := fi.Name()

		fms = append(
			fms,
			types.NewAflFileManagerWithFuzzerId(baseWorkdir, fuzzerId),
		)
	}
	return fms
}

func runServer(workdir string) {
	port := 1414
	target := []byte{1}
	fuzzerConfig := types.FuzzerConfig{
		UseBinary: false,
		Command: []string{
			os.Getenv("HOME") + "/.rbenv/versions/2.4.1/bin/ruby",
			"./cmd/e2e-test/harness.rb",
		},
		SyncInterval: time.Duration(5) * time.Second,
		TimeoutMs:    100000,
		MemLimitMb:   100000,
	}
	archiveConfig := types.ArchiveConfig{}
	metricsReportInterval := 2 * time.Second

	fileManager := types.FleetFileManager{
		Basedir: workdir,
	}

	if err := fileManager.MkTopLevelOutputDir(); err != nil {
		log.Fatal(err)
	}
	if err := fileManager.MkTopLevelInputDir(); err != nil {
		log.Fatal(err)
	}

	input := types.Input{
		Name: "1",
		Body: []byte("hello"),
	}
	fileManager.WriteTopLevelInput(&input)

	server.SetupAndServe(
		port,
		target,
		fuzzerConfig,
		archiveConfig,
		metricsReportInterval,
		workdir,
	)
}

func runClient(workdir string, parallelism int) {
	conf := types.ClientConfig{
		ServerAddress: "http://127.0.0.1:1414",
		Parallelism:   parallelism,
	}
	preExistingRun := false

	fileManager := types.NewAflFileManager(workdir)
	fileManager.MkInputDir()

	client.SetupAndRun(
		conf,
		workdir,
		preExistingRun)
}
