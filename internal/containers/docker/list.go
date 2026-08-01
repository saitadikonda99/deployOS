package docker

import (
	"context"
	"strings"
	"time"

	"github.com/saitadikonda99/deployOS/internal/containers"
)

// containerSummary is the shape of a single entry in the Docker API's
// GET /containers/json response. Only the fields DeployOS uses are
// declared - the response has many more.
type containerSummary struct {
	ID      string   `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	Created int64    `json:"Created"`
	State   string   `json:"State"`
	Status  string   `json:"Status"`
}

// ListContainers implements containers.Runtime.
func (r *Runtime) ListContainers(ctx context.Context) ([]containers.Container, error) {
	var summaries []containerSummary
	if err := r.get(ctx, "/containers/json?all=true", &summaries); err != nil {
		return nil, err
	}

	result := make([]containers.Container, 0, len(summaries))
	for _, s := range summaries {
		result = append(result, containers.Container{
			ID:      s.ID,
			Name:    containerName(s.Names),
			Image:   s.Image,
			Status:  s.Status,
			State:   s.State,
			Created: time.Unix(s.Created, 0).UTC(),
		})
	}
	return result, nil
}

// containerName returns the first Docker container name with its
// leading slash stripped, or "" if names is empty.
func containerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}
