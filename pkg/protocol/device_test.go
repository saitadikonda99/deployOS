package protocol

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

func TestDeviceRegisterRequestRoundTripsThroughJSON(t *testing.T) {
	req := DeviceRegisterRequest{
		DeviceID:        types.AgentID("11111111-1111-1111-1111-111111111111"),
		Hostname:        "dev-box",
		OperatingSystem: "linux",
		Architecture:    "amd64",
		CPUCores:        8,
		MemoryBytes:     17179869184,
		DeployOSVersion: "0.1.0",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded DeviceRegisterRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded != req {
		t.Fatalf("decoded request = %+v, want %+v", decoded, req)
	}
}

func TestDeviceRegisterResponseRoundTripsThroughJSON(t *testing.T) {
	resp := DeviceRegisterResponse{
		DeviceID:  types.AgentID("11111111-1111-1111-1111-111111111111"),
		Token:     "signed-token",
		ExpiresAt: time.Now().Truncate(time.Second).UTC(),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded DeviceRegisterResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !decoded.ExpiresAt.Equal(resp.ExpiresAt) || decoded.DeviceID != resp.DeviceID || decoded.Token != resp.Token {
		t.Fatalf("decoded response = %+v, want %+v", decoded, resp)
	}
}
