package connection

import (
	"sync"
	"testing"
	"time"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

func TestManagerAddIsConnectedRemove(t *testing.T) {
	m := NewManager()
	deviceID := types.AgentID("device-1")

	if m.IsConnected(deviceID) {
		t.Fatal("IsConnected() = true before Add()")
	}

	m.Add(State{DeviceID: deviceID, UserID: "user-1", SessionID: "session-1", ConnectedAt: time.Now()})

	if !m.IsConnected(deviceID) {
		t.Fatal("IsConnected() = false after Add()")
	}
	if got := m.Count(); got != 1 {
		t.Fatalf("Count() = %d, want 1", got)
	}

	state, ok := m.Get(deviceID)
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if state.UserID != "user-1" || state.SessionID != "session-1" {
		t.Errorf("Get() = %+v, want UserID=user-1 SessionID=session-1", state)
	}

	m.Remove(deviceID)

	if m.IsConnected(deviceID) {
		t.Fatal("IsConnected() = true after Remove()")
	}
	if got := m.Count(); got != 0 {
		t.Fatalf("Count() = %d after Remove(), want 0", got)
	}
}

func TestManagerRemoveUnknownDeviceIsNoOp(t *testing.T) {
	m := NewManager()
	m.Remove(types.AgentID("never-added")) // must not panic
	if got := m.Count(); got != 0 {
		t.Fatalf("Count() = %d, want 0", got)
	}
}

func TestManagerAddReplacesExistingSession(t *testing.T) {
	m := NewManager()
	deviceID := types.AgentID("device-1")

	m.Add(State{DeviceID: deviceID, SessionID: "session-1"})
	m.Add(State{DeviceID: deviceID, SessionID: "session-2"})

	if got := m.Count(); got != 1 {
		t.Fatalf("Count() = %d, want 1 (replace, not accumulate)", got)
	}
	state, _ := m.Get(deviceID)
	if state.SessionID != "session-2" {
		t.Errorf("SessionID = %q, want %q", state.SessionID, "session-2")
	}
}

func TestManagerList(t *testing.T) {
	m := NewManager()
	m.Add(State{DeviceID: types.AgentID("device-1")})
	m.Add(State{DeviceID: types.AgentID("device-2")})

	states := m.List()
	if len(states) != 2 {
		t.Fatalf("List() returned %d states, want 2", len(states))
	}
}

// TestManagerConcurrentAccess exercises Manager under concurrent
// Add/Remove/IsConnected from many goroutines; run with -race to catch
// data races.
func TestManagerConcurrentAccess(_ *testing.T) {
	m := NewManager()
	const goroutines = 50

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			deviceID := types.AgentID(string(rune('a' + i%26)))
			m.Add(State{DeviceID: deviceID, SessionID: "s"})
			m.IsConnected(deviceID)
			m.List()
			m.Count()
			m.Remove(deviceID)
		}(i)
	}
	wg.Wait()
}
