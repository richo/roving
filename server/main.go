package main

import (
	"fmt"
	"net/http"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

func put(c web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Put a thing")
}

func get(c web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Got a thing")
}

func main() {
	goji.Put("/state", put)
	goji.Get("/state", get)
	goji.Serve()
}
