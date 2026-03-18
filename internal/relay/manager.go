package relay

import (
	"io"
	"sync"
)

// HubManager manages the lifecycle of Hub instances keyed by session ID.
// It is safe for concurrent use.
type HubManager struct {
	mu   sync.Mutex
	hubs map[string]*Hub
}

// NewHubManager creates an empty HubManager.
func NewHubManager() *HubManager {
	return &HubManager{
		hubs: make(map[string]*Hub),
	}
}

// Create instantiates a Hub for the given session ID, stores it, and starts
// its Run goroutine. If a hub already exists for sessionID it is returned as-is
// (callers should call Get first to avoid unintentional re-creation).
func (m *HubManager) Create(sessionID string, reader io.Reader, writer io.Writer) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.hubs[sessionID]; ok {
		return existing
	}

	hub := NewHub(sessionID, reader, writer, DefaultScrollbackBytes)
	m.hubs[sessionID] = hub
	go hub.Run()
	return hub
}

// Get looks up a Hub by session ID. Returns the hub and true if found.
func (m *HubManager) Get(sessionID string) (*Hub, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	hub, ok := m.hubs[sessionID]
	return hub, ok
}

// Remove shuts down and removes the Hub for the given session ID.
// No-op if the session does not exist.
func (m *HubManager) Remove(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[sessionID]; ok {
		hub.Shutdown()
		delete(m.hubs, sessionID)
	}
}

// Shutdown shuts down all hubs and clears the map.
func (m *HubManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, hub := range m.hubs {
		hub.Shutdown()
		delete(m.hubs, id)
	}
}

// SessionIDs returns a snapshot of all current session IDs.
func (m *HubManager) SessionIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]string, 0, len(m.hubs))
	for id := range m.hubs {
		ids = append(ids, id)
	}
	return ids
}
