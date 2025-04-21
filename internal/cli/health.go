package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"devmux/internal/docker"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#008080")).
			Padding(0, 1)

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	baseHeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(true).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
)

type App struct {
	model Model
}

func NewHealthApp(client docker.DockerAPI) *App {
	if client == nil {
		client, _ = docker.NewClient(nil)
	}
	return &App{
		model: initialModel(client),
	}
}

func (a *App) Run() error {
	p := tea.NewProgram(a.model)
	_, err := p.Run()
	return err
}

type Model struct {
	client  docker.DockerAPI
	spinner spinner.Model
	loading bool
	err     error
	results []string
	table   table.Model
	width   int
	done    bool
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

func NewHealthModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	t := table.New(
		table.WithColumns([]table.Column{}),
		table.WithFocused(false),
		table.WithHeight(10),
	)

	return Model{spinner: s, loading: true, table: t}
}

func initialModel(client docker.DockerAPI) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		spinner: s,
		loading: true,
		results: []string{},
		client:  client,
	}
}

type dockerErrorMsg struct {
	err error
}

func NewDockerErrorMsg(err error) tea.Msg {
	return dockerErrorMsg{err: err}
}

func checkDockerHealth(cli docker.DockerAPI) tea.Cmd {
	return func() tea.Msg {
		healthResult := cli.CheckHealth(context.Background())

		if healthResult.Error != nil {
			return dockerErrorMsg{err: healthResult.Error}
		}

		return healthResult
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		checkDockerHealth(m.client),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.table.SetWidth(msg.Width - 4)
		return m, nil

	case docker.HealthCheckResult:
		m.loading = false
		m.done = true
		m.results = formatDockerHealthResults(msg)
		m.table = renderContainerHealthTable(msg)
		return m, nil

	case dockerErrorMsg:
		m.err = msg.err
		m.loading = false
		m.done = true
		return m, tea.Quit
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func renderContainerHealthTable(result docker.HealthCheckResult) table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 12},
		{Title: "Name", Width: 20},
		{Title: "Health", Width: 10},
		{Title: "Status", Width: 10},
		{Title: "Open Ports (#)", Width: 14},
	}

	rows := []table.Row{}

	for _, container := range result.Containers {
		var portsInfo string
		if container.Ports == nil || len(container.Ports) == 0 {
			portsInfo = "❌"
		} else {
			portsInfo = fmt.Sprintf("✅ (%d)", len(container.Ports))
		}

		status := "Stopped"
		if container.Running {
			status = "Running"
		}

		shortID := container.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		rows = append(rows, table.Row{
			shortID,
			container.Names,
			container.Health,
			status,
			portsInfo,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+5),
		table.WithWidth(120),
	)

	s := table.DefaultStyles()
	s.Header = baseHeaderStyle
	s.Selected = selectedStyle
	s.Cell = cellStyle
	t.SetStyles(s)

	return t
}

func formatDockerHealthResults(health docker.HealthCheckResult) []string {
	results := []string{}

	status := "✅ Successfully connected to Docker daemon"
	if !health.Connected {
		status = "❌ Failed to connect to Docker daemon"
		if health.Error != nil {
			status += ": " + health.Error.Error()
		}
	}
	results = append(results, status)

	return results
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

		if len(m.table.Rows()) == 0 {
			output += "No containers running.\n"
		} else {
			output += m.table.View() + "\n"
		}
		output += helpStyle("\n  ↑/↓: Navigate • q: Quit\n")

		return output
	}

	return "Checking Docker health...\n"
}
