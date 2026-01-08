package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/mpjhorner/superralph/internal/tui/components"
)

// RunState represents the current state of the build
type RunState int

const (
	StateIdle RunState = iota
	StateRunning
	StatePaused
	StateComplete
	StateError
)

func (s RunState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StatePaused:
		return "paused"
	case StateComplete:
		return "complete"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Model is the main TUI model
type Model struct {
	// PRD data
	PRD      *prd.PRD
	PRDPath  string
	PRDStats prd.PRDStats

	// Run state
	State            RunState
	CurrentIteration int
	MaxIterations    int
	CurrentFeature   *prd.Feature
	StartTime        time.Time
	ErrorMsg         string
	RetryCount       int
	MaxRetries       int

	// Phase tracking
	CurrentPhase components.Phase

	// Current activity (what Claude is doing right now)
	CurrentActivity string

	// UI components
	Spinner        spinner.Model
	LogView        *components.LogView
	PhaseIndicator *components.PhaseIndicator
	ActionPanel    *components.ActionPanel
	Width          int
	Height         int

	// Debug mode
	DebugMode bool

	// Callbacks
	OnQuit   func()
	OnPause  func()
	OnResume func()
	OnDebug  func(enabled bool)
}

// NewModel creates a new TUI model
func NewModel(p *prd.PRD, prdPath string, maxIterations int) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	return Model{
		PRD:            p,
		PRDPath:        prdPath,
		PRDStats:       p.Stats(),
		State:          StateIdle,
		MaxIterations:  maxIterations,
		MaxRetries:     3,
		CurrentPhase:   components.PhaseNone,
		Spinner:        s,
		LogView:        components.NewLogView(80, 10),
		PhaseIndicator: components.NewPhaseIndicator(),
		ActionPanel:    components.NewActionPanel(80, 8),
		Width:          80,
		Height:         24,
		DebugMode:      false,
	}
}

// Messages
type (
	// TickMsg is sent periodically to update the UI
	TickMsg time.Time

	// LogMsg adds a plain text line to the log
	LogMsg string

	// TypedLogMsg adds a colored/typed line to the log
	TypedLogMsg struct {
		Type    components.LogEntryType
		Content string
	}

	// ActivityMsg updates the current activity display
	ActivityMsg string

	// StateChangeMsg changes the run state
	StateChangeMsg RunState

	// IterationStartMsg signals a new iteration started
	IterationStartMsg struct {
		Iteration int
		Feature   *prd.Feature
	}

	// IterationCompleteMsg signals an iteration completed
	IterationCompleteMsg struct {
		Iteration int
		Success   bool
	}

	// PRDUpdateMsg signals the PRD was updated
	PRDUpdateMsg struct {
		PRD   *prd.PRD
		Stats prd.PRDStats
	}

	// ErrorMsg signals an error occurred
	ErrorMsgType struct {
		Error string
	}

	// PhaseChangeMsg signals a phase change
	PhaseChangeMsg struct {
		Phase components.Phase
	}

	// ActionAddMsg adds an action to the panel
	ActionAddMsg struct {
		Action components.ActionItem
	}

	// ActionUpdateMsg updates an action status
	ActionUpdateMsg struct {
		ID     string
		Status components.ActionStatus
		Output string
	}

	// ActionClearMsg clears all actions
	ActionClearMsg struct{}

	// DebugToggleMsg toggles debug mode
	DebugToggleMsg struct{}

	// BuildCompleteMsg signals that the build has finished
	BuildCompleteMsg struct {
		Success bool
		Error   error
	}
)

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.Spinner.Tick,
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.OnQuit != nil {
				m.OnQuit()
			}
			return m, tea.Quit
		case "p":
			if m.State == StateRunning {
				m.State = StatePaused
				if m.OnPause != nil {
					m.OnPause()
				}
			}
		case "r":
			if m.State == StatePaused {
				m.State = StateRunning
				if m.OnResume != nil {
					m.OnResume()
				}
			}
		case "d":
			m.DebugMode = !m.DebugMode
			if m.OnDebug != nil {
				m.OnDebug(m.DebugMode)
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.LogView.Width = msg.Width - 4
		m.LogView.Height = m.Height / 4
		m.ActionPanel.Width = msg.Width - 4
		m.ActionPanel.Height = 8
		m.PhaseIndicator.Width = msg.Width - 4

	case TickMsg:
		return m, tickCmd()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	case LogMsg:
		m.LogView.AddLine(string(msg))

	case TypedLogMsg:
		m.LogView.AddEntry(msg.Type, msg.Content)

	case ActivityMsg:
		m.CurrentActivity = string(msg)

	case StateChangeMsg:
		m.State = RunState(msg)
		if m.State == StateRunning && m.StartTime.IsZero() {
			m.StartTime = time.Now()
		}

	case IterationStartMsg:
		m.CurrentIteration = msg.Iteration
		m.CurrentFeature = msg.Feature
		m.RetryCount = 0
		m.ActionPanel.Clear()

	case IterationCompleteMsg:
		if msg.Success {
			m.RetryCount = 0
		} else {
			m.RetryCount++
		}

	case PRDUpdateMsg:
		m.PRD = msg.PRD
		m.PRDStats = msg.Stats

	case ErrorMsgType:
		m.ErrorMsg = msg.Error
		m.State = StateError

	case PhaseChangeMsg:
		m.CurrentPhase = msg.Phase
		m.PhaseIndicator.SetPhase(msg.Phase)

	case ActionAddMsg:
		m.ActionPanel.AddAction(msg.Action)

	case ActionUpdateMsg:
		m.ActionPanel.UpdateAction(msg.ID, msg.Status, msg.Output)

	case ActionClearMsg:
		m.ActionPanel.Clear()

	case DebugToggleMsg:
		m.DebugMode = !m.DebugMode
		if m.OnDebug != nil {
			m.OnDebug(m.DebugMode)
		}

	case BuildCompleteMsg:
		if msg.Success {
			m.State = StateComplete
		} else {
			m.State = StateError
			if msg.Error != nil {
				m.ErrorMsg = msg.Error.Error()
			}
		}
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Progress section
	b.WriteString(m.renderProgress())
	b.WriteString("\n")

	// Phase indicator (if we're in a phase)
	if m.CurrentPhase != components.PhaseNone {
		b.WriteString(m.renderPhase())
		b.WriteString("\n")
	}

	// Status section
	b.WriteString(m.renderStatus())
	b.WriteString("\n")

	// Action panel (if there are actions)
	if len(m.ActionPanel.Actions) > 0 {
		b.WriteString(m.renderActions())
		b.WriteString("\n")
	}

	// Log section
	b.WriteString(m.renderLog())
	b.WriteString("\n")

	// Help
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m Model) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render("SuperRalph")

	projectInfo := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render(fmt.Sprintf("  %s • %s", m.PRD.Name, m.PRDPath))

	return BoxStyle.Render(title + projectInfo)
}

func (m Model) renderProgress() string {
	stats := m.PRDStats
	pb := components.NewProgressBar(stats.PassingFeatures, stats.TotalFeatures, 40)

	var b strings.Builder
	b.WriteString(BoldStyle.Render("Progress: "))
	b.WriteString(pb.Render())
	b.WriteString("\n\n")

	// Category breakdown
	b.WriteString(MutedStyle.Render("By Category:") + "                    ")
	b.WriteString(MutedStyle.Render("By Priority:") + "\n")

	categories := prd.ValidCategories()
	priorities := prd.ValidPriorities()
	maxRows := len(categories)
	if len(priorities) > maxRows {
		maxRows = len(priorities)
	}

	for i := 0; i < maxRows; i++ {
		// Category column
		if i < len(categories) {
			cat := categories[i]
			cs := stats.ByCategory[cat]
			mini := components.NewMiniProgressBar(cs.Passing, cs.Total, 10)
			b.WriteString(fmt.Sprintf("  %-12s %s %d/%d", cat, mini.Render(), cs.Passing, cs.Total))
		} else {
			b.WriteString(strings.Repeat(" ", 32))
		}

		b.WriteString("    ")

		// Priority column
		if i < len(priorities) {
			pri := priorities[i]
			ps := stats.ByPriority[pri]
			mini := components.NewMiniProgressBar(ps.Passing, ps.Total, 10)
			b.WriteString(fmt.Sprintf("%-8s %s %d/%d", pri, mini.Render(), ps.Passing, ps.Total))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderStatus() string {
	var b strings.Builder

	// Status badge
	b.WriteString(BoldStyle.Render("Status: "))
	b.WriteString(StatusBadge(m.State.String()))

	// Spinner if running
	if m.State == StateRunning {
		b.WriteString(" ")
		b.WriteString(m.Spinner.View())
	}
	b.WriteString("\n")

	// Iteration info
	if m.MaxIterations > 0 {
		b.WriteString(fmt.Sprintf("Iteration: %d/%d", m.CurrentIteration, m.MaxIterations))
		if m.RetryCount > 0 {
			b.WriteString(WarningStyle.Render(fmt.Sprintf(" (retry %d/%d)", m.RetryCount, m.MaxRetries)))
		}
		b.WriteString("\n")
	}

	// Current feature
	if m.CurrentFeature != nil {
		b.WriteString(fmt.Sprintf("Feature: %s ", HighlightStyle.Render(m.CurrentFeature.ID)))
		b.WriteString(fmt.Sprintf("\"%s\"\n", m.CurrentFeature.Description))
	}

	// Current activity (what Claude is doing right now)
	if m.CurrentActivity != "" && m.State == StateRunning {
		activityStyle := lipgloss.NewStyle().Foreground(ColorSecondary)
		b.WriteString(activityStyle.Render("Activity: " + m.CurrentActivity))
		b.WriteString("\n")
	}

	// Elapsed time
	if !m.StartTime.IsZero() {
		elapsed := time.Since(m.StartTime).Round(time.Second)
		b.WriteString(MutedStyle.Render(fmt.Sprintf("Elapsed: %s\n", elapsed)))
	}

	// Error message
	if m.ErrorMsg != "" {
		b.WriteString(ErrorStyle.Render("Error: " + m.ErrorMsg))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderLog() string {
	m.LogView.Title = "Claude Output"
	return m.LogView.Render()
}

func (m Model) renderPhase() string {
	var b strings.Builder
	b.WriteString(BoldStyle.Render("Phase: "))
	b.WriteString(m.PhaseIndicator.Render())
	return b.String()
}

func (m Model) renderActions() string {
	return m.ActionPanel.Render()
}

func (m Model) renderHelp() string {
	var keys []string
	keys = append(keys, "[q] Quit")
	if m.State == StateRunning {
		keys = append(keys, "[p] Pause")
	}
	if m.State == StatePaused {
		keys = append(keys, "[r] Resume")
	}
	if m.DebugMode {
		keys = append(keys, "[d] Debug ON")
	} else {
		keys = append(keys, "[d] Debug")
	}

	return HelpStyle.Render(strings.Join(keys, "  "))
}

// AddLog adds a log line (can be called from outside)
func (m *Model) AddLog(line string) {
	m.LogView.AddLine(line)
}

// SetState sets the run state
func (m *Model) SetState(state RunState) {
	m.State = state
}

// UpdatePRD updates the PRD and stats
func (m *Model) UpdatePRD(p *prd.PRD) {
	m.PRD = p
	m.PRDStats = p.Stats()
}

// SetPhase sets the current phase
func (m *Model) SetPhase(phase components.Phase) {
	m.CurrentPhase = phase
	m.PhaseIndicator.SetPhase(phase)
}

// AddAction adds an action to the action panel
func (m *Model) AddAction(action components.ActionItem) {
	m.ActionPanel.AddAction(action)
}

// UpdateAction updates an action's status
func (m *Model) UpdateAction(id string, status components.ActionStatus, output string) {
	m.ActionPanel.UpdateAction(id, status, output)
}

// ClearActions clears all actions from the panel
func (m *Model) ClearActions() {
	m.ActionPanel.Clear()
}

// SetDebugMode sets the debug mode
func (m *Model) SetDebugMode(enabled bool) {
	m.DebugMode = enabled
}

// IsDebugMode returns whether debug mode is enabled
func (m *Model) IsDebugMode() bool {
	return m.DebugMode
}
