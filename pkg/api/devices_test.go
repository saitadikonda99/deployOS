package api

import (
	"encoding/json"
	"testing"
	"time"
)

func TestListDevicesResponseRoundTripsThroughJSON(t *testing.T) {
	resp := ListDevicesResponse{
		Devices: []Device{
			{
				ID:              "11111111-1111-1111-1111-111111111111",
				Hostname:        "dev-box",
				OperatingSystem: "linux",
				Architecture:    "amd64",
				Status:          "registered",
				CreatedAt:       time.Now().Truncate(time.Second).UTC(),
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded ListDevicesResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(decoded.Devices) != 1 || decoded.Devices[0].ID != resp.Devices[0].ID {
		t.Fatalf("decoded response = %+v, want %+v", decoded, resp)
	}
}
