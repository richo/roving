package main

import (
	"encoding/json"
	"fmt"
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

func put(c web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Put a thing")
}

func get(c web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Got a thing")
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
	goji.Put("/state", put)
	goji.Get("/state", get)
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
