package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/zenazn/goji/web"
)

var indexTemplate *template.Template
var webfaceEnabled bool = true

func init() {
	var err error

	baseDir := filepath.Dir(os.Args[0])

	indexTemplate, err = template.New("index.html").
		ParseFiles(filepath.Join(baseDir, "templates", "index.html"))

	if err != nil {
		log.Printf("WARN: Couldn't parse templates, disabling web interface:", err)
	}
}

func index(c web.C, w http.ResponseWriter, r *http.Request) {
	if !webfaceEnabled {
		w.Write([]byte("Couldn't initialize templates. Web interface disabled"))
		return
	}
	nodes.statesLock.RLock()
	defer nodes.statesLock.RUnlock()
	err := indexTemplate.Execute(w, nodes)
	if err != nil {
		log.Fatalf("Couldn't execute template", err)
	}
}
