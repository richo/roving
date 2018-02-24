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

	"../types"
)

// For now, the server stores a great deal of state in memory, although it will
// write as much as it can out to directories that will look reasonable to afl.

var binary []byte

var nodes = make(map[string]types.State)
var nodesLock sync.RWMutex

func post(c web.C, w http.ResponseWriter, r *http.Request) {
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

func get(c web.C, w http.ResponseWriter, r *http.Request) {
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

func target(c web.C, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(binary)
}

func inputs(c web.C, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	corpus := types.ReadDir("input")

	encoder := json.NewEncoder(w)
	encoder.Encode(corpus)
}

func setupAndServe() {
	// Browser endpoints
	goji.Get("/", index)
	// Client endpoints
	goji.Post("/state", post)
	goji.Get("/state/:id", get)
	goji.Get("/target", target)
	goji.Get("/inputs", inputs)
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

	binary, err = ioutil.ReadFile("target")
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
