package server

import (
	"sync"
	"time"

	"github.com/richo/roving/types"
)

// Nodes is a struct that gets and sets the stats of Roving clients.
// Should be used alongside a Reaper in order to cull inactive
// clients.
type Nodes struct {
	Stats   map[string]types.FuzzerStats
	updates map[string]time.Time

	statsLock   *sync.RWMutex
	updatesLock *sync.RWMutex
}

// setStats sets the stats for node `nodeId` to `stats`. It takes
// out the appropriate locks to avoid race conditions.
func (n *Nodes) setStats(nodeId string, stats types.FuzzerStats) {
	n.statsLock.Lock()
	defer n.statsLock.Unlock()
	n.Stats[nodeId] = stats

	n.updatesLock.Lock()
	defer n.updatesLock.Unlock()
	n.updates[nodeId] = time.Now()
}

// deleteNode deletes a node from the Nodes's maps. It takes
// out the appropriate locks to avoid race conditions.
func (n *Nodes) deleteNode(nodeId string) {
	n.statsLock.Lock()
	defer n.statsLock.Unlock()
	n.updatesLock.RLock()
	defer n.updatesLock.RUnlock()

	delete(n.Stats, nodeId)
	delete(n.updates, nodeId)
	types.SubmitMetricCount("reaped", 1, map[string]string{"id": nodeId})
}

func newNodes() Nodes {
	var stats = make(map[string]types.FuzzerStats)
	var updates = make(map[string]time.Time)

	var statsLock sync.RWMutex
	var updatesLock sync.RWMutex

	return Nodes{
		Stats:       stats,
		updates:     updates,
		statsLock:   &statsLock,
		updatesLock: &updatesLock,
	}
}
