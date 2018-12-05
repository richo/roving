package client

import (
	"log"
	"time"

	"github.com/richo/roving/types"
)

// QueueDownloader periodically downloads the queues of all the fuzzers in
// the roving cluster from the server. It writes them to disk using the
// FleetFileManager. Because all clients on a client machine share the same
// FleetFileManager, we only need to run 1 QueueDownloader per client machin.
type QueueDownloader struct {
	Interval    time.Duration
	Server      *RovingServerClient
	fileManager *types.FleetFileManager
}

// run runs the QueueDownloader periodically forever. It should never return.
func (q *QueueDownloader) run() {
	ticker := time.NewTicker(q.Interval)

	for {
		select {
		case <-ticker.C:
			q.downloadQueues()
		}
	}
}

// downloadQueues downloads the queues from the roving server and saves them
// to disk.
func (q *QueueDownloader) downloadQueues() {
	log.Printf("Downloading the queues")

	// We don't need any metric tags for now
	metricTags := make(map[string]string)

	queues, err := q.Server.FetchQueues()
	if err != nil {
		// Fail without panicking so that we can retry in the
		// next QueueDownloader cycle.
		log.Printf("Error downloading queue err=%v", err)

		types.SubmitMetricCount("queue_downloader.download_queue.fail", 1, metricTags)
		return
	}

	log.Printf("Writing queues to disk")

	if err = q.fileManager.WriteQueues(queues); err != nil {
		// Fail without panicking so that we can retry in the
		// next QueueDownloader cycle.
		log.Printf("Error writing queue to disk err=%v", err)

		types.SubmitMetricCount("queue_downloader.download_queue.fail", 1, metricTags)
		return
	}

	types.SubmitMetricCount("queue_downloader.download_queue.success", 1, metricTags)
}
