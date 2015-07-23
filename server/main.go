package main

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"net/http"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"github.com/richo/roving/types"
)

// For now, the server stores a great deal of state in memory, although it will
// write as much as it can out to directories that will look reasonable to afl.

var binary []byte

var nodes = make(map[string]types.State)

func post(c web.C, w http.ResponseWriter, r *http.Request) {
	state := types.State{}

	encoder := json.NewDecoder(r.Body)
	encoder.Decode(&state)

	nodes[state.Id] = state
}

func get(c web.C, w http.ResponseWriter, r *http.Request) {
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

	corpus := types.ReadCorpus("input")

	encoder := json.NewEncoder(w)
	encoder.Encode(corpus)
}

func setupAndServe() {
	goji.Post("/state", post)
	goji.Get("/state/:id", get)
	goji.Get("/target", target)
	goji.Get("/inputs", inputs)
	goji.Serve()
}

func main() {
	var err error
	binary, err = ioutil.ReadFile("target")
	if err != nil {
		log.Panicf("Couldn't load target")
	}

	setupAndServe()
}
