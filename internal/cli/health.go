package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"devmux/internal/docker"
)

type App struct {
	model Model
}

func NewHealthApp() *App {
	return &App{
		model: initialModel(),
	}
}

func (a *App) Run() error {
	p := tea.NewProgram(a.model)
	_, err := p.Run()
	return err
}

type Model struct {
	client  docker.UnderlyingDockerClient
	spinner spinner.Model
	done    bool
	loading bool
	results []string
	err     error
}

type HealthModel interface {
	tea.Model
	Loading() bool
	Done() bool
	Error() error
	Results() []string
	Spinner() spinner.Model
}

func (m Model) Loading() bool          { return m.loading }
func (m Model) Done() bool             { return m.done }
func (m Model) Error() error           { return m.err }
func (m Model) Results() []string      { return m.results }
func (m Model) Spinner() spinner.Model { return m.spinner }

func NewHealthModel() HealthModel {
	return initialModel()
}

func initialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		spinner: s,
		loading: true,
		results: []string{},
	}
}

type dockerResultMsg struct {
	results []string
}

type dockerErrorMsg struct {
	err error
}

func NewDockerResultMsg(results []string) tea.Msg {
	return dockerResultMsg{results: results}
}

func NewDockerErrorMsg(err error) tea.Msg {
	return dockerErrorMsg{err: err}
}

func checkDockerHealth(cli docker.UnderlyingDockerClient) tea.Cmd {
	return func() tea.Msg {
		cli, err := docker.NewClient(cli)
		if err != nil {
			return dockerErrorMsg{err: fmt.Errorf("failed to create Docker client: %w", err)}
		}
		defer cli.Close()

		healthResult := cli.CheckHealth(context.Background())

		if healthResult.Error != nil {
			return dockerErrorMsg{err: healthResult.Error}
		}

		results := []string{healthResult.Message}

		if len(healthResult.Containers) == 0 {
			results = append(results, "No containers running")
		} else {
			results = append(results, fmt.Sprintf("Found %d container(s)", len(healthResult.Containers)))
			for _, container := range healthResult.Containers {
				status := container.Health
				if !container.Running {
					status = "stopped"
				}
				results = append(results, fmt.Sprintf("Container %s (%s): %s",
					container.ID, container.Names, status))

				if len(container.Ports) > 0 && container.Running {
					portInfo := "Ports:"
					for _, port := range container.Ports {
						portInfo += fmt.Sprintf("\n      • %s:%d → %d/%s", port.IP, port.PublicPort, port.PrivatePort, port.Type)
					}
					results = append(results, portInfo)
				}
			}
		}

		return dockerResultMsg{results: results}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		checkDockerHealth(m.client),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	case dockerResultMsg:
		m.loading = false
		m.done = true
		m.results = msg.results
		return m, tea.Quit
	case dockerErrorMsg:
		m.loading = false
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("❌ Error: %s\n", m.err.Error())
	}

	if m.loading {
		return m.spinner.View() + " Checking Docker health...\n"
	}

	if m.done {
		output := "🐳 Docker Health Check Results:\n\n"
		for _, result := range m.results {
			output += fmt.Sprintf("  • %s\n", result)
		}
		return output
	}

	return "Checking Docker health...\n"
}
