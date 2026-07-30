package agent

import "testing"

func TestCollectSystemInfo(t *testing.T) {
	info, err := collectSystemInfo()
	if err != nil {
		t.Fatalf("collectSystemInfo() error = %v", err)
	}

	if info.Hostname == "" {
		t.Error("Hostname is empty")
	}
	if info.OperatingSystem == "" {
		t.Error("OperatingSystem is empty")
	}
	if info.Architecture == "" {
		t.Error("Architecture is empty")
	}
	if info.CPUCores <= 0 {
		t.Errorf("CPUCores = %d, want > 0", info.CPUCores)
	}
	if info.MemoryBytes == 0 {
		t.Error("MemoryBytes = 0, want > 0")
	}
}
