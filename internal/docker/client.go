package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerAPI interface {
	CheckHealth(ctx context.Context) HealthCheckResult
	Close() error
}

type dockerClient struct {
	client *client.Client
}

type ContainerHealth struct {
	ID      string
	Names   string
	Health  string
	Running bool
	Ports   []Port
}

type Port struct {
	IP          string
	PrivatePort uint16
	PublicPort  uint16
	Type        string
}

type HealthCheckResult struct {
	Connected  bool
	Error      error
	Message    string
	Containers []ContainerHealth
}

func NewClient() (DockerAPI, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &dockerClient{client: cli}, nil
}

func (c *dockerClient) CheckHealth(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{
		Connected:  false,
		Containers: []ContainerHealth{},
	}

	_, err := c.client.Ping(ctx)
	if err != nil {
		result.Error = fmt.Errorf("failed to ping Docker daemon: %w", err)
		return result
	}

	result.Connected = true
	result.Message = "Successfully connected to Docker daemon"

	containers, err := c.client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		result.Error = fmt.Errorf("failed to list containers: %w", err)
		return result
	}

	for _, container := range containers {
		containerInfo, err := c.client.ContainerInspect(ctx, container.ID)

		health := ContainerHealth{
			ID:      container.ID[:12],
			Names:   strings.Join(container.Names, ", "),
			Health:  "N/A",
			Running: containerInfo.State.Running,
			Ports:   []Port{},
		}

		for _, port := range container.Ports {
			health.Ports = append(health.Ports, Port{
				IP:          port.IP,
				PrivatePort: port.PrivatePort,
				PublicPort:  port.PublicPort,
				Type:        port.Type,
			})
		}

		if err == nil && containerInfo.State != nil && containerInfo.State.Health != nil {
			health.Health = containerInfo.State.Health.Status
		}

		result.Containers = append(result.Containers, health)
	}

	return result
}

func (c *dockerClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
