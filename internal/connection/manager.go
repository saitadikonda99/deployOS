package connection

import (
	"sync"
	"time"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

// State describes one device's currently active connection.
type State struct {
	DeviceID    types.AgentID
	UserID      string
	SessionID   string
	ConnectedAt time.Time
}

// Manager is a thread-safe, in-memory registry of currently connected
// devices. It holds no persistent state and knows nothing about
// Postgres/Supabase - a device that has never connected simply isn't
// in it, and one that disconnects is removed, not marked offline.
type Manager struct {
	mu     sync.RWMutex
	active map[types.AgentID]State
}

// NewManager returns an empty Manager.
func NewManager() *Manager {
	return &Manager{active: make(map[types.AgentID]State)}
}

// Add records a device as connected, replacing any previous session for
// the same device ID (e.g. if a stale connection hadn't been removed
// yet).
func (m *Manager) Add(state State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active[state.DeviceID] = state
}

// Remove records a device as no longer connected. Removing a device
// that isn't present is a no-op.
func (m *Manager) Remove(deviceID types.AgentID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, deviceID)
}

// IsConnected reports whether deviceID currently has an active
// connection.
func (m *Manager) IsConnected(deviceID types.AgentID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.active[deviceID]
	return ok
}

// Get returns the connection state for deviceID, if connected.
func (m *Manager) Get(deviceID types.AgentID) (State, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.active[deviceID]
	return state, ok
}

// List returns the connection state of every currently connected
// device. The order is unspecified.
func (m *Manager) List() []State {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make([]State, 0, len(m.active))
	for _, state := range m.active {
		states = append(states, state)
	}
	return states
}

// Count returns the number of currently connected devices.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.active)
}
