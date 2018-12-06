package client

import (
	"log"
	"time"

	"github.com/richo/roving/types"
)

// StateUploader periodically uploads a fuzzer's state to the remote server.
// We need to run a separate StateUploader for each fuzzer.
type StateUploader struct {
	Interval time.Duration
	Fuzzer   *Fuzzer
	Server   *RovingServerClient
}

// run sets the StateUploader running and periodically uploading the fuzzer's
// State. It loops more frequently before the first upload so that it uploads
// State as soon at it is available.
func (s *StateUploader) run() {
	ticker := time.NewTicker(s.Interval)

	for {
		// Only sleep for a sec before the first upload. We want to upload our first
		// State really fast. That said, we do need to wait for the fuzzer
		// to actually start before we can read its fuzzer_stats.
		if s.Fuzzer.started && s.Fuzzer.hasBegunFuzzing() {
			s.uploadState()
			break
		}
		log.Printf("Fuzzer not fuzzing yet - sleeping for 5s before trying to upload state")
		time.Sleep(5 * time.Second)
	}

	for {
		select {
		case <-ticker.C:
			s.uploadState()
		}
	}
}

// uploadState reads the State of the fuzzer and uploads it to the
// roving server. It pauses the fuzzer's process whilst it does so
// in order to read States atomically.
func (s *StateUploader) uploadState() {
	s.Fuzzer.stop()
	defer s.Fuzzer.start()
	defer func(startTime int64) {
		// Don't want to lose precision by converting to float32 too early
		totalTime := float64(time.Now().UnixNano()-startTime) / float64(time.Second)
		log.Printf("event=upload-state fuzzer=%s time_paused_s=%f", s.Fuzzer.Id, totalTime)
		types.SubmitMetricGauge("state_uploader.upload_state.time_paused_s", float32(totalTime), s.metricTags())
	}(time.Now().UnixNano())
	state, err := s.Fuzzer.ReadState()
	if err != nil {
		// Fail without panicking so that we can retry in the
		// next StateUploader cycle.
		log.Printf("Error reading state! %s", err)
		types.SubmitMetricCount("state_uploader.upload_state.fail", 1, s.metricTags())
		return
	}

	err = s.Server.UploadState(state)
	if err != nil {
		log.Printf("Error uploading state! %s", err)
		types.SubmitMetricCount("state_uploader.upload_state.fail", 1, s.metricTags())
		return
	}
	types.SubmitMetricCount("state_uploader.upload_state.success", 1, s.metricTags())
}

func (s *StateUploader) metricTags() map[string]string {
	return map[string]string{
		"fuzzer_id": s.Fuzzer.Id,
	}
}
