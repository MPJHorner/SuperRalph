package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mpjhorner/superralph/internal/orchestrator"
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

	// Step tracking (granular step within iteration)
	CurrentStep orchestrator.Step

	// Current activity (what Claude is doing right now)
	CurrentActivity string

	// Tab navigation
	TabBar    *components.TabBar
	ActiveTab components.Tab

	// UI components
	Spinner                spinner.Model
	LogView                *components.LogView
	LogTab                 *components.LogTab
	Dashboard              *components.Dashboard
	PhaseIndicator         *components.PhaseIndicator
	ActionPanel            *components.ActionPanel
	FeatureList            *components.FeatureList
	InteractiveFeatureList *components.InteractiveFeatureList
	StepIndicator          *components.StepIndicator
	Width                  int
	Height                 int

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

	// Initialize feature list (compact view for dashboard)
	featureList := components.NewFeatureList(30, 15)
	featureList.UpdateFromPRD(p, "")

	// Initialize interactive feature list (full view for features tab)
	interactiveFeatureList := components.NewInteractiveFeatureList(80, 20)
	interactiveFeatureList.SetPRD(p, "")

	// Initialize dashboard
	dashboard := components.NewDashboard(80, 20)
	dashboard.SetPRD(p, prdPath)
	dashboard.MaxIterations = maxIterations
	dashboard.MaxRetries = 3

	// Initialize tab bar
	tabBar := components.NewTabBar()

	// Initialize log tab
	logTab := components.NewLogTab(80, 15)

	return Model{
		PRD:                    p,
		PRDPath:                prdPath,
		PRDStats:               p.Stats(),
		State:                  StateIdle,
		MaxIterations:          maxIterations,
		MaxRetries:             3,
		CurrentPhase:           components.PhaseNone,
		CurrentStep:            orchestrator.StepIdle,
		TabBar:                 tabBar,
		ActiveTab:              components.TabDashboard,
		Spinner:                s,
		LogView:                components.NewLogView(80, 10),
		LogTab:                 logTab,
		Dashboard:              dashboard,
		PhaseIndicator:         components.NewPhaseIndicator(),
		ActionPanel:            components.NewActionPanel(80, 8),
		FeatureList:            featureList,
		InteractiveFeatureList: interactiveFeatureList,
		StepIndicator:          components.NewStepIndicator(),
		Width:                  80,
		Height:                 24,
		DebugMode:              false,
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

	// StepChangeMsg signals a step change
	StepChangeMsg struct {
		Step orchestrator.Step
	}

	// TabChangeMsg signals a tab change
	TabChangeMsg struct {
		Tab components.Tab
	}

	// FileDiffMsg signals a file diff to display
	FileDiffMsg struct {
		Diff *orchestrator.FileDiff
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
		// When on Features tab and not in filter/detail mode, pass messages to InteractiveFeatureList
		// But always handle quit and tab switching globally
		if m.ActiveTab == components.TabFeatures {
			// Handle feature list key events when filtering or showing detail
			if m.InteractiveFeatureList.IsFiltering() || m.InteractiveFeatureList.IsShowingDetail() {
				// Let feature list handle these modes, but still allow quit
				if msg.String() == "ctrl+c" {
					if m.OnQuit != nil {
						m.OnQuit()
					}
					return m, tea.Quit
				}
				var cmd tea.Cmd
				m.InteractiveFeatureList, cmd = m.InteractiveFeatureList.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			// Don't quit if in filter mode on Features tab - q should close it
			if m.ActiveTab == components.TabFeatures && m.InteractiveFeatureList.IsShowingDetail() {
				var cmd tea.Cmd
				m.InteractiveFeatureList, cmd = m.InteractiveFeatureList.Update(msg)
				return m, cmd
			}
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
		case "1", "2", "3":
			// Tab switching via number keys
			tab := components.TabFromKey(msg.String())
			if tab >= 0 {
				m.ActiveTab = tab
				m.TabBar.SetActiveTab(tab)
			}
		case "tab":
			// Tab key switches to next tab
			m.TabBar.NextTab()
			m.ActiveTab = m.TabBar.GetActiveTab()
		case "shift+tab":
			// Shift+Tab switches to previous tab
			m.TabBar.PrevTab()
			m.ActiveTab = m.TabBar.GetActiveTab()
		case "a":
			// Toggle auto-scroll in log tab
			if m.ActiveTab == components.TabLogs {
				m.LogTab.ToggleAutoScroll()
			}
		case "j", "down":
			// Scroll down in log tab
			if m.ActiveTab == components.TabLogs {
				m.LogTab.ScrollDown(1)
			} else if m.ActiveTab == components.TabFeatures {
				var cmd tea.Cmd
				m.InteractiveFeatureList, cmd = m.InteractiveFeatureList.Update(msg)
				return m, cmd
			}
		case "k", "up":
			// Scroll up in log tab
			if m.ActiveTab == components.TabLogs {
				m.LogTab.ScrollUp(1)
			} else if m.ActiveTab == components.TabFeatures {
				var cmd tea.Cmd
				m.InteractiveFeatureList, cmd = m.InteractiveFeatureList.Update(msg)
				return m, cmd
			}
		case "g":
			// Go to top in log tab
			if m.ActiveTab == components.TabLogs {
				m.LogTab.GotoTop()
			} else if m.ActiveTab == components.TabFeatures {
				var cmd tea.Cmd
				m.InteractiveFeatureList, cmd = m.InteractiveFeatureList.Update(msg)
				return m, cmd
			}
		case "G":
			// Go to bottom in log tab
			if m.ActiveTab == components.TabLogs {
				m.LogTab.GotoBottom()
			} else if m.ActiveTab == components.TabFeatures {
				var cmd tea.Cmd
				m.InteractiveFeatureList, cmd = m.InteractiveFeatureList.Update(msg)
				return m, cmd
			}
		case "pgup", "ctrl+u":
			// Page up in log tab
			if m.ActiveTab == components.TabLogs {
				m.LogTab.ScrollUp(10)
			}
		case "pgdown", "ctrl+d":
			// Page down in log tab
			if m.ActiveTab == components.TabLogs {
				m.LogTab.ScrollDown(10)
			}
		default:
			// Pass other key messages to InteractiveFeatureList when on Features tab
			if m.ActiveTab == components.TabFeatures {
				var cmd tea.Cmd
				m.InteractiveFeatureList, cmd = m.InteractiveFeatureList.Update(msg)
				return m, cmd
			}
			// Pass to LogTab for viewport handling when on Logs tab
			if m.ActiveTab == components.TabLogs {
				var cmd tea.Cmd
				m.LogTab, cmd = m.LogTab.Update(msg)
				return m, cmd
			}
		}

	case tea.MouseMsg:
		// Handle mouse events for log tab scrolling
		if m.ActiveTab == components.TabLogs {
			var cmd tea.Cmd
			m.LogTab, cmd = m.LogTab.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Calculate feature list width
		featureListWidth := 35
		if m.Width < 100 {
			featureListWidth = 28
		}
		mainColWidth := m.Width - featureListWidth - 7 // Account for gap and borders
		m.LogView.Width = msg.Width - 4
		m.LogView.Height = m.Height / 4
		m.ActionPanel.Width = mainColWidth
		m.ActionPanel.Height = 8
		m.PhaseIndicator.Width = mainColWidth
		m.StepIndicator.Width = mainColWidth
		m.FeatureList.Width = featureListWidth
		m.FeatureList.Height = 12
		m.TabBar.Width = msg.Width
		m.LogTab.Resize(msg.Width-4, m.Height-8)
		m.Dashboard.Width = mainColWidth
		m.Dashboard.Height = m.Height - 10
		m.InteractiveFeatureList.Resize(msg.Width-4, m.Height-10)

	case TickMsg:
		return m, tickCmd()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	case LogMsg:
		m.LogView.AddLine(string(msg))
		m.LogTab.AddLine(string(msg))

	case TypedLogMsg:
		m.LogView.AddEntry(msg.Type, msg.Content)
		m.LogTab.AddEntry(msg.Type, msg.Content)

	case ActivityMsg:
		m.CurrentActivity = string(msg)
		m.Dashboard.SetActivity(string(msg))

	case StateChangeMsg:
		m.State = RunState(msg)
		if m.State == StateRunning && m.StartTime.IsZero() {
			m.StartTime = time.Now()
		}
		// Sync with dashboard
		m.Dashboard.SetState(runStateToDashboardState(m.State))

	case IterationStartMsg:
		m.CurrentIteration = msg.Iteration
		m.CurrentFeature = msg.Feature
		m.RetryCount = 0
		m.ActionPanel.Clear()
		m.CurrentStep = orchestrator.StepReading
		m.StepIndicator.SetStep(orchestrator.StepReading)
		// Update feature list with current feature
		if m.CurrentFeature != nil {
			m.FeatureList.UpdateFromPRD(m.PRD, m.CurrentFeature.ID)
			m.InteractiveFeatureList.SetPRD(m.PRD, m.CurrentFeature.ID)
		}
		// Sync with dashboard
		m.Dashboard.SetIteration(msg.Iteration, m.MaxIterations)
		m.Dashboard.SetFeature(msg.Feature)
		m.Dashboard.SetRetry(0, m.MaxRetries)
		m.Dashboard.ClearActions()
		m.Dashboard.SetStep(orchestrator.StepReading)

	case IterationCompleteMsg:
		if msg.Success {
			m.RetryCount = 0
		} else {
			m.RetryCount++
		}
		m.Dashboard.SetRetry(m.RetryCount, m.MaxRetries)

	case PRDUpdateMsg:
		m.PRD = msg.PRD
		m.PRDStats = msg.Stats
		// Update feature lists
		currentFeatureID := ""
		if m.CurrentFeature != nil {
			currentFeatureID = m.CurrentFeature.ID
		}
		m.FeatureList.UpdateFromPRD(m.PRD, currentFeatureID)
		m.InteractiveFeatureList.SetPRD(m.PRD, currentFeatureID)
		// Sync with dashboard
		m.Dashboard.UpdateStats(msg.Stats)

	case ErrorMsgType:
		m.ErrorMsg = msg.Error
		m.State = StateError
		m.Dashboard.SetError(msg.Error)
		m.Dashboard.SetState(components.DashboardStateError)

	case PhaseChangeMsg:
		m.CurrentPhase = msg.Phase
		m.PhaseIndicator.SetPhase(msg.Phase)
		m.Dashboard.SetPhase(msg.Phase)

	case StepChangeMsg:
		m.CurrentStep = msg.Step
		m.StepIndicator.SetStep(msg.Step)
		m.Dashboard.SetStep(msg.Step)

	case ActionAddMsg:
		m.ActionPanel.AddAction(msg.Action)
		m.Dashboard.AddAction(msg.Action)

	case ActionUpdateMsg:
		m.ActionPanel.UpdateAction(msg.ID, msg.Status, msg.Output)
		m.Dashboard.UpdateAction(msg.ID, msg.Status, msg.Output)

	case ActionClearMsg:
		m.ActionPanel.Clear()
		m.Dashboard.ClearActions()

	case DebugToggleMsg:
		m.DebugMode = !m.DebugMode
		if m.OnDebug != nil {
			m.OnDebug(m.DebugMode)
		}

	case BuildCompleteMsg:
		if msg.Success {
			m.State = StateComplete
			m.Dashboard.SetState(components.DashboardStateComplete)
		} else {
			m.State = StateError
			m.Dashboard.SetState(components.DashboardStateError)
			if msg.Error != nil {
				m.ErrorMsg = msg.Error.Error()
				m.Dashboard.SetError(msg.Error.Error())
			}
		}

	case TabChangeMsg:
		m.ActiveTab = msg.Tab
		m.TabBar.SetActiveTab(msg.Tab)

	case FileDiffMsg:
		// Display the file diff in the log
		if msg.Diff != nil {
			m.addDiffToLog(msg.Diff)
		}
	}

	return m, nil
}

// runStateToDashboardState converts RunState to DashboardState
func runStateToDashboardState(state RunState) components.DashboardState {
	switch state {
	case StateRunning:
		return components.DashboardStateRunning
	case StatePaused:
		return components.DashboardStatePaused
	case StateComplete:
		return components.DashboardStateComplete
	case StateError:
		return components.DashboardStateError
	default:
		return components.DashboardStateIdle
	}
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	// Header (full width)
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Tab bar
	m.TabBar.Width = m.Width
	b.WriteString(m.TabBar.Render())
	b.WriteString("\n\n")

	// Render content based on active tab
	switch m.ActiveTab {
	case components.TabDashboard:
		b.WriteString(m.renderDashboardTab())
	case components.TabFeatures:
		b.WriteString(m.renderFeaturesTab())
	case components.TabLogs:
		b.WriteString(m.renderLogsTab())
	}

	// Help (always visible)
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

// renderDashboardTab renders the dashboard view (default tab)
func (m Model) renderDashboardTab() string {
	var b strings.Builder

	// Calculate column widths
	featureListWidth := 35
	if m.Width < 100 {
		featureListWidth = 28 // Narrower on small terminals
	}
	mainColWidth := m.Width - featureListWidth - 3 // Account for gap

	// Build left column (main content)
	var leftCol strings.Builder

	// Progress section
	leftCol.WriteString(m.renderProgress())
	leftCol.WriteString("\n")

	// Phase indicator (if we're in a phase)
	if m.CurrentPhase != components.PhaseNone {
		leftCol.WriteString(m.renderPhase())
		leftCol.WriteString("\n")
	}

	// Step indicator (shows current step in iteration)
	if m.State == StateRunning {
		leftCol.WriteString(m.renderStep())
		leftCol.WriteString("\n")
	}

	// Status section
	leftCol.WriteString(m.renderStatus())
	leftCol.WriteString("\n")

	// Action panel (if there are actions)
	if len(m.ActionPanel.Actions) > 0 {
		leftCol.WriteString(m.renderActions())
		leftCol.WriteString("\n")
	}

	// Build right column (feature list - compact view)
	m.FeatureList.Width = featureListWidth
	m.FeatureList.Height = 12
	rightCol := m.FeatureList.Render()

	// Join columns side by side
	leftContent := leftCol.String()
	leftLines := strings.Split(leftContent, "\n")
	rightLines := strings.Split(rightCol, "\n")

	// Pad to same height
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	// Render side by side
	for i := 0; i < maxLines; i++ {
		left := leftLines[i]
		right := rightLines[i]

		// Pad left column to width
		leftPadded := lipgloss.NewStyle().Width(mainColWidth).Render(left)
		b.WriteString(leftPadded)
		b.WriteString("  ") // Gap between columns
		b.WriteString(right)
		b.WriteString("\n")
	}

	// Log section (compact, full width)
	b.WriteString(m.renderLog())

	return b.String()
}

// renderFeaturesTab renders the features list view
func (m Model) renderFeaturesTab() string {
	// Use the interactive feature list for full-featured navigation
	m.InteractiveFeatureList.Resize(m.Width-4, m.Height-10)
	return m.InteractiveFeatureList.View()
}

// renderLogsTab renders the dedicated logs view
func (m Model) renderLogsTab() string {
	m.LogTab.Resize(m.Width-2, m.Height-10)
	return m.LogTab.Render()
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

func (m Model) renderStep() string {
	var b strings.Builder
	b.WriteString(BoldStyle.Render("Step: "))
	b.WriteString(m.StepIndicator.Render())
	return b.String()
}

func (m Model) renderActions() string {
	return m.ActionPanel.Render()
}

func (m Model) renderHelp() string {
	var keys []string

	// Tab navigation
	keys = append(keys, "[1-3/Tab] Switch tabs")

	// Tab-specific help
	switch m.ActiveTab {
	case components.TabLogs:
		keys = append(keys, "[j/k] Scroll", "[g/G] Top/Bottom")
		if m.LogTab.IsAutoScrollEnabled() {
			keys = append(keys, "[a] Auto-scroll ON")
		} else {
			keys = append(keys, "[a] Auto-scroll OFF")
		}
	case components.TabFeatures:
		if m.InteractiveFeatureList.IsFiltering() {
			keys = append(keys, "[Enter] Apply filter", "[Esc] Cancel")
		} else if m.InteractiveFeatureList.IsShowingDetail() {
			keys = append(keys, "[Enter/Esc] Close detail")
		} else {
			keys = append(keys, "[j/k] Navigate", "[Enter] Details", "[/] Search")
		}
	}

	// Global controls
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
	m.LogTab.AddLine(line)
}

// addDiffToLog formats and adds a file diff to the log views
func (m *Model) addDiffToLog(diff *orchestrator.FileDiff) {
	// Create a compact diff header
	var statsStr string
	if diff.AddedLines > 0 || diff.RemovedLines > 0 {
		parts := []string{}
		if diff.AddedLines > 0 {
			parts = append(parts, fmt.Sprintf("+%d", diff.AddedLines))
		}
		if diff.RemovedLines > 0 {
			parts = append(parts, fmt.Sprintf("-%d", diff.RemovedLines))
		}
		statsStr = " (" + strings.Join(parts, ", ") + " lines)"
	}

	prefix := "Modified"
	if diff.IsNewFile {
		prefix = "Created"
	}

	// Add a diff header entry
	headerLine := fmt.Sprintf("%s: %s%s", prefix, diff.FilePath, statsStr)
	m.LogView.AddEntry(components.LogTypeDiff, headerLine)
	m.LogTab.AddEntry(components.LogTypeDiff, headerLine)

	// For small diffs (less than 20 lines changed), show inline diff preview
	totalChanges := diff.AddedLines + diff.RemovedLines
	if totalChanges > 0 && totalChanges <= 20 {
		// Generate a simple inline diff preview
		diffLines := m.generateInlineDiff(diff.OldContent, diff.NewContent, 5)
		for _, line := range diffLines {
			m.LogView.AddEntry(components.LogTypeDiff, line)
			m.LogTab.AddEntry(components.LogTypeDiff, line)
		}
	} else if totalChanges > 20 {
		m.LogView.AddEntry(components.LogTypeInfo, "  (diff too large to display inline)")
		m.LogTab.AddEntry(components.LogTypeInfo, "  (diff too large to display inline)")
	}
}

// generateInlineDiff creates a simple inline diff preview
func (m *Model) generateInlineDiff(oldContent, newContent string, maxLines int) []string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	if oldContent == "" {
		oldLines = []string{}
	}
	if newContent == "" {
		newLines = []string{}
	}

	var result []string
	shown := 0

	// Simple diff: show lines that differ
	i, j := 0, 0
	for (i < len(oldLines) || j < len(newLines)) && shown < maxLines*2 {
		if i >= len(oldLines) {
			// Remaining new lines are additions
			line := newLines[j]
			if len(line) > 60 {
				line = line[:57] + "..."
			}
			result = append(result, "  + "+line)
			j++
			shown++
		} else if j >= len(newLines) {
			// Remaining old lines are removals
			line := oldLines[i]
			if len(line) > 60 {
				line = line[:57] + "..."
			}
			result = append(result, "  - "+line)
			i++
			shown++
		} else if oldLines[i] == newLines[j] {
			// Lines match - skip (context)
			i++
			j++
		} else {
			// Lines differ
			oldLine := oldLines[i]
			if len(oldLine) > 60 {
				oldLine = oldLine[:57] + "..."
			}
			result = append(result, "  - "+oldLine)
			shown++

			newLine := newLines[j]
			if len(newLine) > 60 {
				newLine = newLine[:57] + "..."
			}
			result = append(result, "  + "+newLine)
			shown++
			i++
			j++
		}
	}

	// If we truncated, add indicator
	if (i < len(oldLines) || j < len(newLines)) && shown >= maxLines*2 {
		result = append(result, "  ... (more changes)")
	}

	return result
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
