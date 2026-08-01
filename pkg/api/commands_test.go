package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSendCommandRequestRoundTripsThroughJSON(t *testing.T) {
	req := SendCommandRequest{Kind: "PING"}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded SendCommandRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(decoded, req) {
		t.Fatalf("decoded request = %+v, want %+v", decoded, req)
	}
}

func TestSendCommandRequestWithArgumentsRoundTripsThroughJSON(t *testing.T) {
	req := SendCommandRequest{Kind: "INSPECT_CONTAINER", Arguments: map[string]string{"id": "c1"}}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded SendCommandRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(decoded, req) {
		t.Fatalf("decoded request = %+v, want %+v", decoded, req)
	}
}

func TestSendCommandResponseRoundTripsThroughJSON(t *testing.T) {
	resp := SendCommandResponse{
		CommandID: "cmd-1",
		Success:   true,
		Message:   "pong",
		Details:   map[string]string{"latency_ms": "3"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded SendCommandResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.CommandID != resp.CommandID || decoded.Success != resp.Success || decoded.Message != resp.Message {
		t.Fatalf("decoded response = %+v, want %+v", decoded, resp)
	}
	if decoded.Details["latency_ms"] != "3" {
		t.Fatalf("decoded Details = %+v, want latency_ms=3", decoded.Details)
	}
}
