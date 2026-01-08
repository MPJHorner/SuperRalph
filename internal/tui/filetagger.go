package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mpjhorner/superralph/internal/tui/components"
)

// FileTagState represents the state of the file tagging model
type FileTagState int

const (
	FileTagStateInput FileTagState = iota
	FileTagStateAutocomplete
	FileTagStateDone
	FileTagStateCanceled
)

// FileTagModel is a TUI model for selecting files to tag
type FileTagModel struct {
	// State
	State FileTagState

	// Components
	TextInput    textinput.Model
	Autocomplete *components.Autocomplete

	// File list from tagger
	Files []string

	// Selected tags (@ prefixed paths)
	SelectedTags []string

	// Dimensions
	Width  int
	Height int

	// Callbacks
	OnComplete func(tags []string)
	OnCancel   func()
}

// NewFileTagModel creates a new file tagging model
func NewFileTagModel(files []string) FileTagModel {
	ti := textinput.New()
	ti.Placeholder = "Type @ to search files, or press Enter to continue..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	ac := components.NewAutocomplete(70, 15)
	ac.MaxVisible = 10
	ac.SetFiles(files)

	return FileTagModel{
		State:        FileTagStateInput,
		TextInput:    ti,
		Autocomplete: ac,
		Files:        files,
		Width:        80,
		Height:       24,
	}
}

// FileTagCompleteMsg signals that file tagging is complete
type FileTagCompleteMsg struct {
	Tags []string
}

// FileTagCancelMsg signals that file tagging was canceled
type FileTagCancelMsg struct{}

// Init initializes the model
func (m FileTagModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m FileTagModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.TextInput.Width = msg.Width - 10
		m.Autocomplete.Width = msg.Width - 10
		return m, nil
	}

	// Update text input
	var cmd tea.Cmd
	m.TextInput, cmd = m.TextInput.Update(msg)

	// Check if we should show autocomplete
	if strings.Contains(m.TextInput.Value(), "@") && !m.Autocomplete.Active {
		m.activateAutocomplete()
	}

	return m, cmd
}

// handleKeyMsg handles key presses
func (m FileTagModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.State {
	case FileTagStateAutocomplete:
		return m.handleAutocompleteKey(msg)
	case FileTagStateInput:
		return m.handleInputKey(msg)
	}
	return m, nil
}

// handleInputKey handles keys in input state
func (m FileTagModel) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.State = FileTagStateCanceled
		if m.OnCancel != nil {
			m.OnCancel()
		}
		return m, tea.Quit

	case "enter":
		// Complete with current selections
		m.State = FileTagStateDone
		m.SelectedTags = m.Autocomplete.GetSelectedTags()
		if m.OnComplete != nil {
			m.OnComplete(m.SelectedTags)
		}
		return m, tea.Quit

	case "@":
		// Activate autocomplete
		m.activateAutocomplete()
		return m, nil

	case "esc":
		// If autocomplete is active, deactivate it
		if m.Autocomplete.Active {
			m.Autocomplete.Deactivate()
			m.State = FileTagStateInput
			m.TextInput.SetValue("")
		}
		return m, nil
	}

	// Default: update text input
	var cmd tea.Cmd
	m.TextInput, cmd = m.TextInput.Update(msg)

	// Check for @ trigger
	val := m.TextInput.Value()
	if strings.HasSuffix(val, "@") && !m.Autocomplete.Active {
		m.activateAutocomplete()
	}

	return m, cmd
}

// handleAutocompleteKey handles keys in autocomplete state
func (m FileTagModel) handleAutocompleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.State = FileTagStateCanceled
		if m.OnCancel != nil {
			m.OnCancel()
		}
		return m, tea.Quit

	case "esc":
		m.Autocomplete.Deactivate()
		m.State = FileTagStateInput
		m.TextInput.SetValue("")
		return m, nil

	case "up", "k":
		m.Autocomplete.MoveUp()
		return m, nil

	case "down", "j":
		m.Autocomplete.MoveDown()
		return m, nil

	case " ":
		// Space toggles selection
		m.Autocomplete.ToggleSelection()
		return m, nil

	case "enter":
		// Enter confirms selection and exits autocomplete
		m.Autocomplete.Deactivate()
		m.State = FileTagStateInput
		m.TextInput.SetValue("")
		return m, nil

	case "tab":
		// Tab selects current item and stays in autocomplete
		m.Autocomplete.ToggleSelection()
		return m, nil

	case "backspace":
		// Backspace removes last character from filter
		if m.Autocomplete.Query != "" {
			m.Autocomplete.Filter(m.Autocomplete.Query[:len(m.Autocomplete.Query)-1])
		} else {
			// Exit autocomplete if no query
			m.Autocomplete.Deactivate()
			m.State = FileTagStateInput
			m.TextInput.SetValue("")
		}
		return m, nil

	default:
		// Other keys filter the list
		if len(msg.String()) == 1 {
			m.Autocomplete.Filter(m.Autocomplete.Query + msg.String())
		}
		return m, nil
	}
}

// activateAutocomplete activates the autocomplete dropdown
func (m *FileTagModel) activateAutocomplete() {
	m.Autocomplete.Activate()
	m.State = FileTagStateAutocomplete

	// Extract query from input (text after last @)
	val := m.TextInput.Value()
	if idx := strings.LastIndex(val, "@"); idx >= 0 {
		query := val[idx+1:]
		m.Autocomplete.Filter(query)
	}
}

// View renders the UI
func (m FileTagModel) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		MarginBottom(1)
	b.WriteString(titleStyle.Render("Tag Files for Context"))
	b.WriteString("\n\n")

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		MarginBottom(1)
	b.WriteString(instructionStyle.Render("Select files to include in Claude's context for planning."))
	b.WriteString("\n")
	b.WriteString(instructionStyle.Render("Type @ to search files. Press Enter when done."))
	b.WriteString("\n\n")

	// Show current selections
	if m.Autocomplete.SelectedCount() > 0 {
		selectedStyle := lipgloss.NewStyle().
			Foreground(ColorSuccess)
		b.WriteString(selectedStyle.Render("Tagged files:"))
		b.WriteString("\n")

		tags := m.Autocomplete.GetSelectedTags()
		for _, tag := range tags {
			b.WriteString("  ")
			b.WriteString(lipgloss.NewStyle().Foreground(ColorHighlight).Render(tag))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Text input (only show when not in autocomplete)
	if m.State == FileTagStateInput {
		b.WriteString(m.TextInput.View())
		b.WriteString("\n\n")
	}

	// Autocomplete dropdown
	if m.Autocomplete.Active {
		b.WriteString(m.Autocomplete.Render())
		b.WriteString("\n")
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		MarginTop(1)

	var helpText string
	if m.State == FileTagStateAutocomplete {
		helpText = "[↑/↓] Navigate  [Space] Toggle  [Enter] Confirm  [Esc] Cancel"
	} else {
		helpText = "[Enter] Continue  [@] Search files  [Ctrl+C] Cancel"
	}
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// GetSelectedTags returns the selected file tags
func (m *FileTagModel) GetSelectedTags() []string {
	return m.Autocomplete.GetSelectedTags()
}

// SetFiles sets the available files
func (m *FileTagModel) SetFiles(files []string) {
	m.Files = files
	m.Autocomplete.SetFiles(files)
}

// RunFileTagger runs the file tagger TUI and returns selected tags
func RunFileTagger(files []string) ([]string, error) {
	model := NewFileTagModel(files)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	m := finalModel.(FileTagModel)
	if m.State == FileTagStateCanceled {
		return nil, nil
	}

	return m.Autocomplete.GetSelectedTags(), nil
}
