package cli_test

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"devmux/internal/cli"
	"devmux/internal/docker"
)

type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) CheckHealth(ctx context.Context) docker.HealthCheckResult {
	args := m.Called(ctx)
	return args.Get(0).(docker.HealthCheckResult)
}

func (m *MockDockerClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewHealthApp(t *testing.T) {
	app := cli.NewHealthApp()
	assert.NotNil(t, app)
}

func TestInitialModel(t *testing.T) {
	model := cli.NewHealthModel()

	assert.True(t, model.Loading())
	assert.False(t, model.Done())
	assert.Nil(t, model.Error())
	assert.Empty(t, model.Results())
}

func TestModelInit(t *testing.T) {
	model := cli.NewHealthModel()
	assert.NotNil(t, model.Init())
}

func TestModelUpdate_KeyMsg(t *testing.T) {
	model := cli.NewHealthModel()

	testCases := []struct {
		name string
		msg  tea.Msg
	}{
		{"CtrlC", tea.KeyMsg{Type: tea.KeyCtrlC}},
		{"QKey", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newModel, cmd := model.Update(tc.msg)
			assert.Equal(t, model, newModel)
			assert.NotNil(t, cmd)
		})
	}
}

func TestModelUpdate_DockerResultMsg(t *testing.T) {
	model := cli.NewHealthModel()
	results := []string{"Connected to Docker", "No containers running"}

	newModel, cmd := model.Update(cli.NewDockerResultMsg(results))

	healthModel := newModel.(cli.HealthModel)
	assert.False(t, healthModel.Loading())
	assert.True(t, healthModel.Done())
	assert.Equal(t, results, healthModel.Results())
	assert.NotNil(t, cmd)
}

func TestModelUpdate_DockerErrorMsg(t *testing.T) {
	model := cli.NewHealthModel()
	err := errors.New("connection failed")

	newModel, cmd := model.Update(cli.NewDockerErrorMsg(err))

	healthModel := newModel.(cli.HealthModel)
	assert.False(t, healthModel.Loading())
	assert.True(t, healthModel.Done())
	assert.Equal(t, err, healthModel.Error())
	assert.NotNil(t, cmd)
}

func TestModelView(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func() cli.HealthModel
		contains []string
	}{
		{
			name: "Loading View",
			setup: func() cli.HealthModel {
				return cli.NewHealthModel()
			},
			contains: []string{"Checking Docker health"},
		},
		{
			name: "Error View",
			setup: func() cli.HealthModel {
				m := cli.NewHealthModel()
				newModel, _ := m.Update(cli.NewDockerErrorMsg(errors.New("connection error")))
				return newModel.(cli.HealthModel)
			},
			contains: []string{"❌ Error: connection error"},
		},
		{
			name: "Results View",
			setup: func() cli.HealthModel {
				m := cli.NewHealthModel()
				results := []string{"Connected to Docker", "No containers running"}
				newModel, _ := m.Update(cli.NewDockerResultMsg(results))
				return newModel.(cli.HealthModel)
			},
			contains: []string{
				"🐳 Docker Health Check Results",
				"• Connected to Docker",
				"• No containers running",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := tc.setup()
			view := model.View()
			for _, str := range tc.contains {
				assert.Contains(t, view, str)
			}
		})
	}
}
