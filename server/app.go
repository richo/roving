package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/zenazn/goji/web"
)

var indexTemplate *template.Template

func init() {
	var err error
	indexTemplate, err = template.New("index.html").ParseFiles("server/templates/index.html")
	if err != nil {
		log.Fatalf("Couldn't parse template")
	}
}

func index(c web.C, w http.ResponseWriter, r *http.Request) {
	nodesLock.RLock()
	defer nodesLock.RUnlock()
	err := indexTemplate.Execute(w, nodes)
	if err != nil {
		log.Fatalf("Couldn't execute template", err)
	}
}
