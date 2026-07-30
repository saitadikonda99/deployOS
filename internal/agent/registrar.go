package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/saitadikonda99/deployOS/pkg/api"
	"github.com/saitadikonda99/deployOS/pkg/protocol"
)

// registrar registers this agent with the DeployOS control plane. It is
// the agent's only network dependency for registration - it never talks
// to Supabase directly.
type registrar struct {
	baseURL string
	client  *http.Client
}

func newRegistrar(baseURL string) *registrar {
	return &registrar{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// register calls POST /api/v1/devices/register on the control plane.
func (r *registrar) register(
	ctx context.Context,
	userAccessToken string,
	req protocol.DeviceRegisterRequest,
) (protocol.DeviceRegisterResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return protocol.DeviceRegisterResponse{}, fmt.Errorf("encoding registration request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, r.baseURL+"/api/v1/devices/register", bytes.NewReader(body),
	)
	if err != nil {
		return protocol.DeviceRegisterResponse{}, fmt.Errorf("building registration request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+userAccessToken)

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return protocol.DeviceRegisterResponse{}, fmt.Errorf("calling control plane: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		var apiErr api.ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Error != "" {
			return protocol.DeviceRegisterResponse{}, fmt.Errorf(
				"control plane rejected registration (%d): %s", resp.StatusCode, apiErr.Error,
			)
		}
		return protocol.DeviceRegisterResponse{}, fmt.Errorf(
			"control plane rejected registration: status %d", resp.StatusCode,
		)
	}

	var out protocol.DeviceRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return protocol.DeviceRegisterResponse{}, fmt.Errorf("decoding registration response: %w", err)
	}

	return out, nil
}
