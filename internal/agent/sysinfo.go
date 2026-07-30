package agent

import (
	"fmt"
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v3/mem"
)

// systemInfo is the machine information reported at device registration.
type systemInfo struct {
	Hostname        string
	OperatingSystem string
	Architecture    string
	CPUCores        int
	MemoryBytes     uint64
}

// collectSystemInfo reads the local machine's hostname, OS, architecture,
// CPU core count, and total physical memory.
func collectSystemInfo() (systemInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return systemInfo{}, fmt.Errorf("reading hostname: %w", err)
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		return systemInfo{}, fmt.Errorf("reading memory info: %w", err)
	}

	return systemInfo{
		Hostname:        hostname,
		OperatingSystem: runtime.GOOS,
		Architecture:    runtime.GOARCH,
		CPUCores:        runtime.NumCPU(),
		MemoryBytes:     vm.Total,
	}, nil
}
