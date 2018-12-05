package server

import (
	"log"
	"time"
)

// Reaper removes old nodes from a Nodes struct after
// they have been inactive for > Interval.
type Reaper struct {
	Nodes    Nodes
	Interval time.Duration
}

func newReaper(n Nodes, i time.Duration) *Reaper {
	return &Reaper{
		Nodes:    n,
		Interval: i,
	}
}

// cleanUpOldNodes removes nodes that have been inactive for longer
// than `Interval` from the Nodes.
func (r *Reaper) cleanUpOldNodes() {
	now := time.Now()

	for id := range nodes.Stats {
		if r.Nodes.updates[id].Add(r.Interval).Before(now) {
			r.Nodes.deleteNode(id)
		}
	}
}

// run runs the Reaper forever. It will periodically check for inactive
// nodes, and reap them.
func (r *Reaper) run() {
	log.Printf("Reaper started")
	ticker := time.NewTicker(r.Interval)

	for {
		select {
		case <-ticker.C:
			r.cleanUpOldNodes()
		}
	}
}
