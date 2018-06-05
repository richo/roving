package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"net/http"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"github.com/richo/roving/types"
)

// For now, the server stores a great deal of state in memory, although it will
// write as much as it can out to directories that will look reasonable to afl.

var targetBinary []byte

// `nodes` is a map from hostname => the State of that host
var nodes = make(map[string]types.State)
var nodesLock sync.RWMutex

// Clients use this route to periodically report their states. The server uses
// this information to update its `nodes` map. It also writes hangs and crashes
// to the ./hangs and ./crashes directories.
func postState(c web.C, w http.ResponseWriter, r *http.Request) {
	state := types.State{}

	encoder := json.NewDecoder(r.Body)
	encoder.Decode(&state)

	for _, hang := range state.Hangs.Inputs {
		hang.WriteToPath("hangs")
	}

	for _, crash := range state.Crashes.Inputs {
		crash.WriteToPath("crashes")
	}

	nodesLock.Lock()
	defer nodesLock.Unlock()
	nodes[state.Id] = state

	updatesLock.Lock()
	defer updatesLock.Unlock()
	updates[state.Id] = time.Now()
}

// Returns the server's `nodes` map as JSON.
func getState(c web.C, w http.ResponseWriter, r *http.Request) {
	nodesLock.RLock()
	defer nodesLock.RUnlock()
	var values []types.State
	for _, v := range nodes {
		values = append(values, v)
	}
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.Encode(values)
}

// Returns the target binary for clients to download before they begin
// fuzzing.
func getTarget(c web.C, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(targetBinary)
}

// Returns the initialization corpus to bootstrap the fuzzing process.
func getInputs(c web.C, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	corpus := types.ReadDir("input")

	encoder := json.NewEncoder(w)
	encoder.Encode(corpus)
}

func setupAndServe() {
	// Browser endpoints
	goji.Get("/", index)
	// Client endpoints
	goji.Post("/state", postState)
	goji.Get("/state/:id", getState)
	goji.Get("/target", getTarget)
	goji.Get("/inputs", getInputs)
	goji.Serve()
}

func main() {
	var err error
	args := os.Args
	if len(args) != 2 {
		log.Printf("Usage: ./server <workdir>")
	}

	err = os.Chdir(args[1])
	if err != nil {
		log.Panicf("Couldn't move into work directory", err)
	}

	targetBinary, err = ioutil.ReadFile("target")
	if err != nil {
		log.Panicf("Couldn't load target")
	}

	err = os.Mkdir("hangs", 0755)
	if err != nil {
		// fatal()
	}

	err = os.Mkdir("crashes", 0755)
	if err != nil {
		// fatal()
	}

	reaper := newReaper(1 * time.Hour)
	go reaper.run()
	setupAndServe()
}
