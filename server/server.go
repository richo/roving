package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/stripe/veneur/ssf"
	"github.com/stripe/veneur/trace"

	goji "goji.io"
	"goji.io/pat"

	"github.com/richo/roving/types"
)

var nodes Nodes
var target types.TargetBinary
var fuzzerConf types.FuzzerConfig
var archiver Archiver
var archiveConf types.ArchiveConfig
var fileManager *types.FleetFileManager
var realtimeCrashesPath string = "realtime-crashes"
var dict []byte

// Clients use this route to periodically report their states. The server uses
// this information to update its `Nodes` information. It also writes hangs and
// crashes to the ./hangs and ./crashes directories.
func postState(w http.ResponseWriter, r *http.Request) {
	state := types.State{}

	encoder := json.NewDecoder(r.Body)
	encoder.Decode(&state)

	aflOutput := state.AflOutput
	log.Printf(
		"Received fuzzer state fuzzer_id=%v queue_size=%d crashes_size=%d hangs_size=%d",
		state.Id,
		len(aflOutput.Queue.Inputs),
		len(aflOutput.Crashes.Inputs),
		len(aflOutput.Hangs.Inputs),
	)
	// Log queue size so we can max(queue_size) to have an idea on fuzzing progress
	types.SubmitMetricGauge(
		"fuzzer.queue_size",
		float32(len(aflOutput.Queue.Inputs)),
		map[string]string{"fuzzer_id": state.Id},
	)

	if err := fileManager.MkAllOutputDirs(state.Id); err != nil {
		log.Fatal(err)
	}
	if err := fileManager.WriteOutput(state.Id, &state.AflOutput); err != nil {
		log.Fatal(err)
	}

	archiveNewCrashes(fileManager, archiver)

	nodes.setStats(state.Id, state.Stats)
}

// The getQueues route returns the Queue of each fuzzer that the server
// knows about.
func getQueues(w http.ResponseWriter, r *http.Request) {
	queues, err := fileManager.ReadQueues()
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.Encode(queues)
}

// The getConfig route returns info about the target.
func getConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.Encode(fuzzerConf)
}

// The getTargetBinary route returns the target binary for clients
// to download before they begin fuzzing.
func getTargetBinary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(target)
}

// The getInputs route returns the initial corpus used to
// bootstrap the fuzzing process. Every fuzz system must have
// at least 1 input.
func getInputs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	corpus, err := fileManager.ReadInputs()
	if err != nil {
		log.Fatal(err)
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(corpus)
}

// The getDict route returns the dictionary of common tokens used
// to give hints to the fuzzer. Using a dictionary is recommended
// but not required.
func getDict(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/text")
	w.Write(dict)
}

// archiveNewCrashes reads the crashes from a `FleetFileManager` and compares
// them to the crashes in an `Archiver`'s "./realtime-crashes"
// directory. It copies over any that are missing.
//
// We do this so that we archive crashes as soon as we find them. This
// way we should never lose a crash, even if the server dies before
// the next regularly scheduled run of the archiver.
func archiveNewCrashes(fm *types.FleetFileManager, a Archiver) {
	var err error

	archivedRealtimeCrashPaths, err := a.LsDstFiles(realtimeCrashesPath)
	if err != nil {
		log.Fatal(err)
	}
	// Use a map to approximate a set to prevent quadratic complexity in
	// checking whether a crash has been archived.
	archivedRealtimeCrashPathsMap := make(map[string]bool)
	for _, name := range archivedRealtimeCrashPaths {
		archivedRealtimeCrashPathsMap[name] = true
	}

	localOutputs, err := fm.ReadOutputs()
	if err != nil {
		log.Fatal(err)
	}

	// Construct a manifest of the missing crashes and where they
	// should be copied to.
	manifest := Manifest{srcRoot: fm.Basedir}
	// Iterate through fuzzers in the fleet
	for fuzzerId, output := range localOutputs {
		// Iterate through crashes in the current fuzzer's output
		for _, crash := range output.Crashes.Inputs {
			fullLocalCrashPath, err := fm.CrashPath(fuzzerId, crash.Name)
			if err != nil {
				log.Fatal(err)
			}
			relCrashPath, err := filepath.Rel(fm.Basedir, fullLocalCrashPath)
			if err != nil {
				log.Fatal(err)
			}
			// If the current crash has not yet been archived, add it
			// to the manifest
			if _, present := archivedRealtimeCrashPathsMap[relCrashPath]; !present {
				entry := ManifestEntry{
					src: relCrashPath,
					dst: filepath.Join(realtimeCrashesPath, relCrashPath),
				}
				manifest.entries = append(manifest.entries, entry)
			}
		}
	}
	// Once the manifest has been built, archive everything
	ArchiveManifest(a, manifest)
}

// SetupAndServe is the main entry-point for the roving server.
func SetupAndServe(port int, targetBinary types.TargetBinary, fuzzerConfig types.FuzzerConfig, archiveConfig types.ArchiveConfig, metricsReportInterval time.Duration, workdir string) {
	var err error
	target = targetBinary

	fuzzerConf = fuzzerConfig
	archiveConf = archiveConfig
	fileManager = &types.FleetFileManager{Basedir: workdir}
	nodes = newNodes()

	reaper := newReaper(nodes, 1*time.Hour)
	go reaper.run()

	if metricsReportInterval > 0 {
		metricsPoller := MetricsPoller{
			Nodes:    &nodes,
			Interval: metricsReportInterval,
		}
		go metricsPoller.run()
	}

	if archiveConf.Type != "" {
		switch archiveConf.Type {
		case "disk":
			archiver, err = NewDiskArchiver(archiveConf)
		case "s3":
			archiver, err = NewS3Archiver(archiveConf)
		default:
			log.Fatalf("Unknown archiver type: %s", archiveConf.Type)
		}
		if err != nil {
			log.Fatal(err)
		}
		go ArchiveToTimestampedDirsForever(archiver, fileManager.Basedir, archiveConf.Interval)
	} else {
		archiver = NullArchiver{}
	}

	trace.Service = "roving-srv"
	ssf.NamePrefix = "roving-srv."

	mux := goji.NewMux()

	if fuzzerConfig.UseDict {
		log.Printf("Reading dict...")

		dict, err = fileManager.ReadDict()
		if err != nil {
			log.Fatal(err)
		}

		if len(dict) == 0 {
			log.Fatalf("Failed to read dict - dict was empty! path=%s!", fileManager.DictPath())
		}
		log.Printf("Successfully read dict bytes=%d", len(dict))
	}

	// Admin browser endpoints
	mux.HandleFunc(pat.Get("/"), adminIndex)
	mux.HandleFunc(pat.Get("/admin"), adminIndex)
	mux.HandleFunc(pat.Get("/admin/archive"), adminArchive)
	mux.HandleFunc(pat.Get("/admin/fuzzer/:fuzzerId/input/:type/:name"), adminInput)
	mux.HandleFunc(pat.Get("/admin/output"), adminOutput)
	// Client endpoints
	mux.HandleFunc(pat.Post("/state"), postState)
	mux.HandleFunc(pat.Get("/queue"), getQueues)
	mux.HandleFunc(pat.Get("/config"), getConfig)
	mux.HandleFunc(pat.Get("/target/binary"), getTargetBinary)
	mux.HandleFunc(pat.Get("/inputs"), getInputs)
	mux.HandleFunc(pat.Get("/dict"), getDict)

	log.Printf("Starting Roving server on port %d...", port)

	http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
