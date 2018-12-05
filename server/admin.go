package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	raven "github.com/getsentry/raven-go"
	"goji.io/pat"
)

// admin.go contains the routes and logic for the roving
// server admin interface. The routes' are bound to paths in
// server.go.

var archiveTemplate *template.Template
var indexTemplate *template.Template
var inputTemplate *template.Template
var outputTemplate *template.Template

func init() {
	archiveTemplate = parseTemplate("archive")
	indexTemplate = parseTemplate("index")
	inputTemplate = parseTemplate("input")
	outputTemplate = parseTemplate("output")
}

func buildTemplatePath(name string) string {
	return fmt.Sprintf("server/templates/%s.html", name)
}

// parseTemplate loads a template from disk and parses it
// into a golang Template.
func parseTemplate(name string) *template.Template {
	templatePath := buildTemplatePath(name)
	templateContents := webfaceTemplates[templatePath]
	if templateContents == "" {
		log.Fatalf("Couldn't read template: %s", templatePath)
	}

	funcMap := template.FuncMap{
		"fmtTimestamp": func(ts uint64) string {
			return time.Unix(int64(ts), 0).Format(time.RFC3339)
		},
		"contractString": func(s string, maxLength int) string {
			if len(s) > (maxLength - 3) {
				return s[0:maxLength-3] + "..."
			}
			return s
		},
		"contractByteArray": func(bs []byte, maxLength int) string {
			toStrs := func(bytes []byte) []string {
				asStrs := make([]string, maxLength-3)
				for i, b := range bytes {
					asStrs[i] = fmt.Sprintf("%d", int(b))
				}
				return asStrs
			}

			var useStrs []string
			if len(bs) > (maxLength - 3) {
				useStrs = append(toStrs(bs[0:maxLength-3]), "...")
			} else {
				useStrs = toStrs(bs)
			}

			return strings.Join(useStrs, " ")
		},
		"bytesToStr": func(bs []byte) string {
			return string(bs)
		},
		"joinStringArray": func(strs []string, delimiter string) string {
			return strings.Join(strs, delimiter)
		},
	}

	headerPath := buildTemplatePath("_header")
	headerTmpl, err := template.New(name).
		Parse(webfaceTemplates[headerPath])
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
	tmpl, err := headerTmpl.
		Funcs(funcMap).
		Parse(webfaceTemplates[templatePath])
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}

	return tmpl
}

func adminIndex(w http.ResponseWriter, r *http.Request) {
	nodes.statsLock.RLock()
	defer nodes.statsLock.RUnlock()

	templateData := map[string]interface{}{
		"Nodes":         &nodes,
		"FuzzerConfig":  &fuzzerConf,
		"ArchiveConfig": &archiveConf,
	}

	err := indexTemplate.Execute(w, templateData)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatalf("Couldn't execute template: %s", err)
	}
}

func adminInput(w http.ResponseWriter, r *http.Request) {
	var err error

	fuzzerId := pat.Param(r, "fuzzerId")
	inputType := pat.Param(r, "type")
	inputName := pat.Param(r, "name")

	input, err := fileManager.ReadInput(fuzzerId, inputType, inputName)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatalf("Couldn't execute template: %s", err)
	}

	err = inputTemplate.Execute(w, input)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatalf("Couldn't execute template: %s", err)
	}
}

func adminOutput(w http.ResponseWriter, r *http.Request) {
	var err error

	outputs, err := fileManager.ReadOutputs()
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatalf("Couldn't load outputs: %s", err)
	}

	err = outputTemplate.Execute(w, outputs)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatalf("Couldn't execute template: %s", err)
	}
}

func adminArchive(w http.ResponseWriter, r *http.Request) {
	var err error

	realtimeCrashNames, err := archiver.LsDstFiles("realtime-crashes")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}

	templateData := map[string]interface{}{
		"RealtimeCrashArchive": realtimeCrashNames,
		"ArchiveConfig":        archiveConf,
	}
	err = archiveTemplate.Execute(w, templateData)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatalf("Couldn't execute template: %s", err)
	}
}
