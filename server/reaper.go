package main

import (
	"log"
	"sync"
	"time"
)

// A Reaper removes old nodes from `nodes` and `updates` maps after
// a prolonged period of inactivity.

var updates = make(map[string]time.Time)
var updatesLock sync.RWMutex

type Reaper struct {
	Interval time.Duration
}

func newReaper(i time.Duration) *Reaper {
	return &Reaper{
		Interval: i,
	}
}

// Removes nodes that have been inactive for longer than `Interval`
// from the `nodes` map.
func (r *Reaper) cleanUpOldNodes() {
	updatesLock.RLock()
	defer updatesLock.RUnlock()
	nodesLock.Lock()
	defer nodesLock.Unlock()
	now := time.Now()

	for id, _ := range nodes {
		if updates[id].Add(r.Interval).Before(now) {
			delete(nodes, id)
      delete(updates, id)
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
