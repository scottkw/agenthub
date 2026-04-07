package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/scottkw/agenthub/internal/tailnet"
)

const cacheTTL = 30 * time.Second

// tailnetCache caches peer discovery results with a 30-second TTL.
// Uses a full Mutex (not RWMutex) for getOrRefresh to prevent thundering herd.
type tailnetCache struct {
	mu       sync.Mutex
	result   []tailnet.Peer
	cachedAt time.Time
}

func (c *tailnetCache) get() ([]tailnet.Peer, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.cachedAt) < cacheTTL {
		return c.result, true
	}
	return nil, false
}

func (c *tailnetCache) set(peers []tailnet.Peer) {
	c.mu.Lock()
	c.result = peers
	c.cachedAt = time.Now()
	c.mu.Unlock()
}

// discoverFunc is the injectable discovery function for testability.
type discoverFunc func(ctx context.Context) ([]tailnet.Peer, error)

// getOrRefresh returns cached results if fresh, otherwise runs discovery.
// Uses full Mutex to prevent thundering herd on cache expiry.
func (c *tailnetCache) getOrRefresh(ctx context.Context, fn discoverFunc) []tailnet.Peer {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.cachedAt) < cacheTTL {
		return c.result
	}
	peers, err := fn(ctx)
	if err != nil {
		peers = []tailnet.Peer{}
	}
	c.result = peers
	c.cachedAt = time.Now()
	return c.result
}
