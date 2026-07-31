package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

const (
	deviceIDFileName    = "device_id"
	deviceTokenFileName = "device_token"
)

// loadOrCreateDeviceID returns this agent's persisted device ID, creating
// and persisting a new one if none exists yet. The same ID is returned
// on every subsequent call against the same dataDir, which is what lets
// the agent reuse its identity across restarts.
func loadOrCreateDeviceID(dataDir string) (types.AgentID, error) {
	if id, ok, err := readTrimmedFile(dataDir, deviceIDFileName); err != nil {
		return "", err
	} else if ok {
		return types.AgentID(id), nil
	}

	id := types.AgentID(uuid.NewString())
	if err := writeFile(dataDir, deviceIDFileName, id.String()); err != nil {
		return "", fmt.Errorf("persisting device id: %w", err)
	}
	return id, nil
}

// persistDeviceToken saves the device token returned by a successful
// registration, so it survives an agent restart.
func persistDeviceToken(dataDir, token string) error {
	if err := writeFile(dataDir, deviceTokenFileName, token); err != nil {
		return fmt.Errorf("persisting device token: %w", err)
	}
	return nil
}

// loadDeviceToken reads the currently persisted device token, if any.
// The connection client calls this on every (re)connect attempt, so it
// always authenticates with whatever token registration most recently
// wrote - not one captured once at startup.
func loadDeviceToken(dataDir string) (string, bool, error) {
	return readTrimmedFile(dataDir, deviceTokenFileName)
}

func readTrimmedFile(dataDir, name string) (string, bool, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("reading %s: %w", name, err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

func writeFile(dataDir, name, contents string) error {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, name), []byte(contents), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}
	return nil
}
