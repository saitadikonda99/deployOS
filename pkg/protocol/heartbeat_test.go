package protocol

import (
	"encoding/json"
	"testing"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

func TestNewHeartbeatRoundTripsThroughJSON(t *testing.T) {
	hb := NewHeartbeat(types.AgentID("node-1"), "0.1.0")

	data, err := json.Marshal(hb)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded Heartbeat
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded != hb {
		t.Fatalf("decoded heartbeat = %+v, want %+v", decoded, hb)
	}
}
