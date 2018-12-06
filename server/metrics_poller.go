package server

import (
	"log"
	"time"

	"github.com/richo/roving/types"
)

// MetricsPoller periodically logs metrics about a Nodes
// struct.
type MetricsPoller struct {
	Nodes    *Nodes
	Interval time.Duration
}

// run runs the MetricsPoller forever, logging metrics every
// Interval.
func (mp MetricsPoller) run() {
	ticker := time.NewTicker(mp.Interval)
	for {
		select {
		case <-ticker.C:
			mp.logMetrics()
		}
	}
}

// logMetrics logs metrics once.
func (mp MetricsPoller) logMetrics() {
	log.Printf("Logging metrics in MetricsPoller")

	timeNow := uint64(time.Now().Unix())

	mp.Nodes.statsLock.RLock()
	defer mp.Nodes.statsLock.RUnlock()

	for fuzzerID, fuzzerStats := range mp.Nodes.Stats {
		// Log execs_per_sec
		tags := map[string]string{
			"fuzzer_id": fuzzerID,
		}
		types.SubmitMetricGauge(
			"fuzzer.execs_per_sec",
			float32(fuzzerStats.ExecsPerSec),
			tags,
		)

		// Log secs since last update
		secsSinceLastUpdate := timeNow - fuzzerStats.LastUpdate
		types.SubmitMetricGauge(
			"fuzzer.secs_since_last_update",
			float32(secsSinceLastUpdate),
			tags,
		)

		// Log secs since last path
		if fuzzerStats.LastPath > 0 {
			secsSinceLastPath := fuzzerStats.LastUpdate - fuzzerStats.LastPath
			types.SubmitMetricGauge(
				"fuzzer.secs_since_last_path",
				float32(secsSinceLastPath),
				tags,
			)
		}

		// Log paths so we can max(paths) to have an idea on fuzzing progress
		types.SubmitMetricGauge(
			"fuzzer.paths_total",
			float32(fuzzerStats.PathsTotal),
			tags,
		)
	}

	log.Printf("Successfully logged metrics in MetricsPoller n_fuzzers=%d", len(mp.Nodes.Stats))
}
