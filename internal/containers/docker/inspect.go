package docker

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/saitadikonda99/deployOS/internal/containers"
)

// containerInspect is the shape (trimmed to the fields DeployOS uses) of
// the Docker API's GET /containers/{id}/json response.
type containerInspect struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	Image string `json:"Image"`
	State struct {
		Status string `json:"Status"`
	} `json:"State"`
	Created string `json:"Created"`
	Config  struct {
		Image string   `json:"Image"`
		Cmd   []string `json:"Cmd"`
		Env   []string `json:"Env"`
	} `json:"Config"`
	RestartCount    int `json:"RestartCount"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
		Networks map[string]struct{} `json:"Networks"`
	} `json:"NetworkSettings"`
	Mounts []struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		Mode        string `json:"Mode"`
	} `json:"Mounts"`
}

// InspectContainer implements containers.Runtime.
func (r *Runtime) InspectContainer(ctx context.Context, id string) (containers.ContainerDetails, error) {
	var inspect containerInspect
	err := r.get(ctx, "/containers/"+url.PathEscape(id)+"/json", &inspect)
	if errors.Is(err, errNotFound) {
		return containers.ContainerDetails{}, fmt.Errorf("%w: %s", containers.ErrContainerNotFound, id)
	}
	if err != nil {
		return containers.ContainerDetails{}, err
	}

	// Best-effort: a malformed Created timestamp shouldn't fail the
	// whole inspect, just leave Created zero.
	created, _ := time.Parse(time.RFC3339Nano, inspect.Created)

	return containers.ContainerDetails{
		Container: containers.Container{
			ID:      inspect.ID,
			Name:    strings.TrimPrefix(inspect.Name, "/"),
			Image:   inspect.Config.Image,
			Status:  inspect.State.Status,
			State:   inspect.State.Status,
			Created: created,
		},
		Command:      inspect.Config.Cmd,
		Env:          inspect.Config.Env,
		Mounts:       inspectMounts(inspect),
		Ports:        inspectPorts(inspect),
		Networks:     inspectNetworks(inspect),
		RestartCount: inspect.RestartCount,
	}, nil
}

func inspectMounts(inspect containerInspect) []containers.Mount {
	mounts := make([]containers.Mount, 0, len(inspect.Mounts))
	for _, m := range inspect.Mounts {
		mounts = append(mounts, containers.Mount{
			Source:      m.Source,
			Destination: m.Destination,
			Mode:        m.Mode,
		})
	}
	return mounts
}

func inspectPorts(inspect containerInspect) []containers.Port {
	ports := make([]containers.Port, 0, len(inspect.NetworkSettings.Ports))
	for spec, bindings := range inspect.NetworkSettings.Ports {
		containerPort, protocol := splitPortSpec(spec)
		for _, b := range bindings {
			hostPort, _ := strconv.Atoi(b.HostPort)
			ports = append(ports, containers.Port{
				ContainerPort: containerPort,
				HostPort:      hostPort,
				Protocol:      protocol,
			})
		}
	}
	return ports
}

func inspectNetworks(inspect containerInspect) []string {
	networks := make([]string, 0, len(inspect.NetworkSettings.Networks))
	for name := range inspect.NetworkSettings.Networks {
		networks = append(networks, name)
	}
	return networks
}

// splitPortSpec splits a Docker port spec like "80/tcp" into its
// container port and protocol.
func splitPortSpec(spec string) (int, string) {
	port, protocol, found := strings.Cut(spec, "/")
	if !found {
		protocol = "tcp"
	}
	containerPort, _ := strconv.Atoi(port)
	return containerPort, protocol
}
