// Package docker implements internal/containers.Runtime against a local
// Docker Engine. The Docker Engine API is plain HTTP+JSON served over a
// unix socket, so this talks to it directly with net/http rather than
// pulling in Docker's own (much larger) client SDK.
package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// defaultRequestTimeout bounds a single Docker API request.
const defaultRequestTimeout = 10 * time.Second

// errNotFound is returned by get when the Docker API responds 404. It
// never escapes this package - callers translate it into a
// containers.Err* sentinel that means something in that package's terms.
var errNotFound = errors.New("docker: not found")

// Runtime implements containers.Runtime by calling the Docker
// Engine API over a unix socket, the same way the `docker` CLI does.
type Runtime struct {
	client  *http.Client
	baseURL string
}

// NewRuntime builds a Runtime that talks to the Docker
// daemon listening on the unix socket at socketPath (typically
// "/var/run/docker.sock").
func NewRuntime(socketPath string) *Runtime {
	return &Runtime{
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", socketPath)
				},
			},
			Timeout: defaultRequestTimeout,
		},
		baseURL: "http://unix",
	}
}

// newRuntimeWithClient builds a Runtime against an arbitrary
// HTTP base URL and client, so tests can point it at an httptest server
// standing in for the Docker daemon instead of a real unix socket.
func newRuntimeWithClient(baseURL string, client *http.Client) *Runtime {
	return &Runtime{client: client, baseURL: baseURL}
}

// get performs a GET request against the Docker API and decodes a JSON
// response body into out. It returns errNotFound for a 404 response.
func (r *Runtime) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building docker api request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("calling docker api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return fmt.Errorf("docker api returned %d: %s", resp.StatusCode, apiErr.Message)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding docker api response: %w", err)
	}
	return nil
}
