package docker_test

import (
	"context"
	"errors"
	"testing"

	"devmux/internal/docker"
	"devmux/internal/docker/mocks"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func NewClientWithMock(t *testing.T) (*mocks.MockUnderlyingDockerClient, docker.DockerAPI) {
	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockUnderlyingDockerClient(ctrl)

	client, err := docker.NewClient(mockClient)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return mockClient, client
}

func TestCheckHealth_Success(t *testing.T) {
	mockClient, dockerAPI := NewClientWithMock(t)

	mockClient.EXPECT().Ping(gomock.Any()).Return(types.Ping{}, nil)

	containers := []container.Summary{
		{
			ID:    "container123456789012",
			Names: []string{"/container1"},
			Ports: []container.Port{
				{IP: "0.0.0.0", PrivatePort: 8080, PublicPort: 80, Type: "tcp"},
			},
		},
	}
	mockClient.EXPECT().ContainerList(gomock.Any(), gomock.Any()).Return(containers, nil)

	containerInfo := container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			State: &container.State{
				Running: true,
				Health: &container.Health{
					Status: "healthy",
				},
			},
		},
	}
	mockClient.EXPECT().ContainerInspect(gomock.Any(), "container123456789012").Return(containerInfo, nil)

	result := dockerAPI.CheckHealth(context.Background())

	assert.True(t, result.Connected)
	assert.Equal(t, "Successfully connected to Docker daemon", result.Message)
	assert.Nil(t, result.Error)
	assert.Len(t, result.Containers, 1)

	container := result.Containers[0]
	assert.Equal(t, "container123", container.ID)
	assert.Equal(t, "/container1", container.Names)
	assert.Equal(t, "healthy", container.Health)
	assert.True(t, container.Running)
	assert.Len(t, container.Ports, 1)
	assert.Equal(t, "0.0.0.0", container.Ports[0].IP)
	assert.Equal(t, uint16(8080), container.Ports[0].PrivatePort)
	assert.Equal(t, uint16(80), container.Ports[0].PublicPort)
	assert.Equal(t, "tcp", container.Ports[0].Type)
}

func TestCheckHealth_PingFailed(t *testing.T) {
	mockClient, dockerAPI := NewClientWithMock(t)

	mockClient.EXPECT().Ping(gomock.Any()).Return(types.Ping{}, errors.New("connection refused"))

	result := dockerAPI.CheckHealth(context.Background())

	assert.False(t, result.Connected)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "failed to ping Docker daemon")
	assert.Empty(t, result.Containers)
}
