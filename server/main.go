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

var nodes Nodes

// Nodes is a struct that gets and sets the states of Roving nodes.
// Reaper to cull inactive nodes.
type Nodes struct {
	states  map[string]types.State
	updates map[string]time.Time

	statesLock  sync.RWMutex
	updatesLock sync.RWMutex
}

// Sets the state for node `nodeId` to `state`, taking out the appropriate
// locks.
func (n Nodes) setState(nodeId string, state types.State) {
	n.statesLock.Lock()
	defer n.statesLock.Unlock()
	n.states[state.Id] = state

	n.updatesLock.Lock()
	defer n.updatesLock.Unlock()
	n.updates[state.Id] = time.Now()
}

// Returns an array of all states that the `Nodes` knows about.
func (n Nodes) getStates() []types.State {
	n.statesLock.RLock()
	defer n.statesLock.RUnlock()

	var values []types.State
	for _, v := range n.states {
		values = append(values, v)
	}
	return values
}

// Deletes `nodeId` from the `Nodes`.
func (n Nodes) deleteNode(nodeId string) {
	n.statesLock.Lock()
	defer n.statesLock.Unlock()
	n.updatesLock.RLock()
	defer n.updatesLock.RUnlock()

	delete(n.states, nodeId)
	delete(n.updates, nodeId)
}

func newNodes() Nodes {
	var states = make(map[string]types.State)
	var updates = make(map[string]time.Time)

	var statesLock sync.RWMutex
	var updatesLock sync.RWMutex

	return Nodes{
		states:      states,
		updates:     updates,
		statesLock:  statesLock,
		updatesLock: updatesLock,
	}
}

var target types.Target

// Clients use this route to periodically report their states. The server uses
// this information to update its `Nodes` information. It also writes hangs and
// crashes to the ./hangs and ./crashes directories.
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

	nodes.setState(state.Id, state)
}

// Returns all node states as JSON.
func getState(c web.C, w http.ResponseWriter, r *http.Request) {
	states := nodes.getStates()
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.Encode(states)
}

// Returns info about the target.
//
// If a target is particularly large and will therefore take a long time
// to download, we want to allow clients to supply their own copy of it.
// In this case this endpoint will return the command that they should
// run in order to fuzz it.
func getTargetMeta(c web.C, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.Encode(target.Metadata)
}

// Returns the target binary for clients to download before they begin
// fuzzing.
func getTargetBinary(c web.C, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(target.Binary)
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
	goji.Get("/target/meta", getTargetMeta)
	goji.Get("/target/binary", getTargetBinary)
	goji.Get("/inputs", getInputs)
	goji.Serve()
}

func main() {
	var err error
	args := os.Args
	if len(args) < 2 {
		log.Printf("Usage: ./server <workdir> [<target-command>]")
		return
	}

	err = os.Chdir(args[1])
	if err != nil {
		log.Panicf("Couldn't move into work directory", err)
	}

	log.Printf("----TARGET DETAILS-----")
	if len(args) > 2 {
		targetCommand := args[2:]

		targetMeta := types.TargetMetadata{
			ShouldDownload: false,
			Command:        targetCommand,
		}
		target = types.Target{
			Metadata: targetMeta,
		}

		log.Printf("ShouldDownload:\tfalse (clients should have their own copy of the target available)")
		log.Printf("Command:\t%s", target.Metadata.Command)
	} else {
		targetBinary, err := ioutil.ReadFile("target")
		if err != nil {
			log.Panicf("Couldn't load target binary")
		}

		targetMeta := types.TargetMetadata{
			ShouldDownload: true,
		}
		target = types.Target{
			Metadata: targetMeta,
			Binary:   targetBinary,
		}

		log.Printf("ShouldDownload:\ttrue (clients should download a copy of the target from the server)")
		log.Printf("binarySize:\t\t%d bytes", len(target.Binary))
	}
	log.Printf("--------")

	err = os.Mkdir("hangs", 0755)
	if err != nil {
		// fatal()
	}

	err = os.Mkdir("crashes", 0755)
	if err != nil {
		// fatal()
	}

	nodes = newNodes()

	reaper := newReaper(nodes, 1*time.Hour)
	go reaper.run()

	setupAndServe()
}
