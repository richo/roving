package main

import (
	"log"
	"time"
)

// A Reaper removes old nodes from `nodes` and `updates` maps after
// a prolonged period of inactivity.

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

// Removes nodes that have been inactive for longer than `Interval`
// from the `nodes` map.
func (r *Reaper) cleanUpOldNodes() {
	now := time.Now()

	for id, _ := range nodes.states {
		if r.Nodes.updates[id].Add(r.Interval).Before(now) {
			r.Nodes.deleteNode(id)
		}
	}
}

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
