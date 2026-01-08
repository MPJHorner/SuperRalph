package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileTagModel(t *testing.T) {
	files := []string{"main.go", "test.go", "cmd/"}

	m := NewFileTagModel(files)

	assert.Equal(t, FileTagStateInput, m.State)
	assert.NotNil(t, m.TextInput)
	assert.NotNil(t, m.Autocomplete)
	assert.Equal(t, files, m.Files)
	assert.Equal(t, 80, m.Width)
	assert.Equal(t, 24, m.Height)
	assert.False(t, m.Autocomplete.Active)
}

func TestFileTagModelInit(t *testing.T) {
	m := NewFileTagModel([]string{"main.go"})

	cmd := m.Init()

	// Should return textinput.Blink command
	assert.NotNil(t, cmd)
}

func TestFileTagModelWindowResize(t *testing.T) {
	m := NewFileTagModel([]string{"main.go"})

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = newModel.(FileTagModel)

	assert.Equal(t, 100, m.Width)
	assert.Equal(t, 50, m.Height)
	assert.Equal(t, 90, m.TextInput.Width)
	assert.Equal(t, 90, m.Autocomplete.Width)
}

func TestFileTagModelAutocompleteActivation(t *testing.T) {
	m := NewFileTagModel([]string{"main.go", "test.go"})

	// Simulate typing @
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	m = newModel.(FileTagModel)

	assert.True(t, m.Autocomplete.Active)
	assert.Equal(t, FileTagStateAutocomplete, m.State)
}

func TestFileTagModelAutocompleteNavigation(t *testing.T) {
	m := NewFileTagModel([]string{"a.go", "b.go", "c.go"})

	// Activate autocomplete
	m.activateAutocomplete()

	assert.Equal(t, 0, m.Autocomplete.Cursor)

	// Move down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(FileTagModel)
	assert.Equal(t, 1, m.Autocomplete.Cursor)

	// Move down with j
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(FileTagModel)
	assert.Equal(t, 2, m.Autocomplete.Cursor)

	// Move up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(FileTagModel)
	assert.Equal(t, 1, m.Autocomplete.Cursor)

	// Move up with k
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(FileTagModel)
	assert.Equal(t, 0, m.Autocomplete.Cursor)
}

func TestFileTagModelAutocompleteSelection(t *testing.T) {
	m := NewFileTagModel([]string{"main.go", "test.go"})
	m.activateAutocomplete()

	// Toggle selection with space
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newModel.(FileTagModel)

	assert.Equal(t, 1, m.Autocomplete.SelectedCount())
	assert.True(t, m.Autocomplete.Selected["main.go"])

	// Toggle again to deselect
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newModel.(FileTagModel)

	assert.Equal(t, 0, m.Autocomplete.SelectedCount())
}

func TestFileTagModelTabSelection(t *testing.T) {
	m := NewFileTagModel([]string{"main.go", "test.go"})
	m.activateAutocomplete()

	// Tab toggles selection
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(FileTagModel)

	assert.Equal(t, 1, m.Autocomplete.SelectedCount())
	assert.True(t, m.Autocomplete.Selected["main.go"])

	// Still in autocomplete state (tab doesn't exit)
	assert.Equal(t, FileTagStateAutocomplete, m.State)
}

func TestFileTagModelEscapeAutocomplete(t *testing.T) {
	m := NewFileTagModel([]string{"main.go"})
	m.activateAutocomplete()

	assert.True(t, m.Autocomplete.Active)

	// Escape deactivates autocomplete
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(FileTagModel)

	assert.False(t, m.Autocomplete.Active)
	assert.Equal(t, FileTagStateInput, m.State)
}

func TestFileTagModelEnterConfirms(t *testing.T) {
	m := NewFileTagModel([]string{"main.go", "test.go"})
	m.activateAutocomplete()

	// Select first item
	m.Autocomplete.ToggleSelection()

	// Enter confirms
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(FileTagModel)

	// Should exit autocomplete but not quit yet
	assert.False(t, m.Autocomplete.Active)
	assert.Equal(t, FileTagStateInput, m.State)
	assert.Nil(t, cmd) // No quit command yet
}

func TestFileTagModelEnterFromInputQuits(t *testing.T) {
	m := NewFileTagModel([]string{"main.go"})

	// Not in autocomplete state
	assert.Equal(t, FileTagStateInput, m.State)

	// Enter from input state should complete
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(FileTagModel)

	assert.Equal(t, FileTagStateDone, m.State)
	assert.NotNil(t, cmd) // Should have quit command
}

func TestFileTagModelFilter(t *testing.T) {
	m := NewFileTagModel([]string{"main.go", "main_test.go", "other.go"})
	m.activateAutocomplete()

	// Type 'm' to filter
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = newModel.(FileTagModel)

	assert.Equal(t, "m", m.Autocomplete.Query)
	assert.Equal(t, 2, len(m.Autocomplete.Filtered)) // main.go and main_test.go

	// Type 'a' to further filter
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = newModel.(FileTagModel)

	assert.Equal(t, "ma", m.Autocomplete.Query)
}

func TestFileTagModelBackspace(t *testing.T) {
	m := NewFileTagModel([]string{"main.go"})
	m.activateAutocomplete()
	m.Autocomplete.Filter("main")

	assert.Equal(t, "main", m.Autocomplete.Query)

	// Backspace removes character
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(FileTagModel)

	assert.Equal(t, "mai", m.Autocomplete.Query)
}

func TestFileTagModelBackspaceExitsAutocomplete(t *testing.T) {
	m := NewFileTagModel([]string{"main.go"})
	m.activateAutocomplete()

	// No query
	assert.Empty(t, m.Autocomplete.Query)

	// Backspace exits autocomplete when query is empty
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(FileTagModel)

	assert.False(t, m.Autocomplete.Active)
	assert.Equal(t, FileTagStateInput, m.State)
}

func TestFileTagModelCtrlCCancels(t *testing.T) {
	cancelCalled := false
	m := NewFileTagModel([]string{"main.go"})
	m.OnCancel = func() {
		cancelCalled = true
	}

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(FileTagModel)

	assert.Equal(t, FileTagStateCanceled, m.State)
	assert.True(t, cancelCalled)
	assert.NotNil(t, cmd) // Should have quit command
}

func TestFileTagModelCtrlCInAutocomplete(t *testing.T) {
	m := NewFileTagModel([]string{"main.go"})
	m.activateAutocomplete()

	cancelCalled := false
	m.OnCancel = func() {
		cancelCalled = true
	}

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(FileTagModel)

	assert.Equal(t, FileTagStateCanceled, m.State)
	assert.True(t, cancelCalled)
	assert.NotNil(t, cmd)
}

func TestFileTagModelGetSelectedTags(t *testing.T) {
	m := NewFileTagModel([]string{"main.go", "test.go", "cmd/"})
	m.activateAutocomplete()

	// Select first two items
	m.Autocomplete.ToggleSelection()
	m.Autocomplete.MoveDown()
	m.Autocomplete.ToggleSelection()

	tags := m.GetSelectedTags()

	assert.Len(t, tags, 2)
	assert.Contains(t, tags, "@main.go")
	assert.Contains(t, tags, "@test.go")
}

func TestFileTagModelSetFiles(t *testing.T) {
	m := NewFileTagModel([]string{"old.go"})

	newFiles := []string{"new.go", "another.go"}
	m.SetFiles(newFiles)

	assert.Equal(t, newFiles, m.Files)
	assert.Len(t, m.Autocomplete.Files, 2)
}

func TestFileTagModelView(t *testing.T) {
	m := NewFileTagModel([]string{"main.go", "test.go"})

	view := m.View()

	assert.Contains(t, view, "Tag Files for Context")
	assert.Contains(t, view, "Type @ to search files")
	assert.Contains(t, view, "Enter")
}

func TestFileTagModelViewWithAutocomplete(t *testing.T) {
	m := NewFileTagModel([]string{"main.go", "test.go"})
	m.activateAutocomplete()

	view := m.View()

	assert.Contains(t, view, "main.go")
	assert.Contains(t, view, "test.go")
	assert.Contains(t, view, "Navigate")
}

func TestFileTagModelViewWithSelections(t *testing.T) {
	m := NewFileTagModel([]string{"main.go", "test.go"})
	m.activateAutocomplete()
	m.Autocomplete.ToggleSelection()

	// Exit autocomplete to see selections
	m.Autocomplete.Deactivate()
	m.State = FileTagStateInput

	view := m.View()

	assert.Contains(t, view, "Tagged files")
	assert.Contains(t, view, "@main.go")
}

func TestFileTagModelOnCompleteCallback(t *testing.T) {
	var receivedTags []string
	m := NewFileTagModel([]string{"main.go", "test.go"})
	m.OnComplete = func(tags []string) {
		receivedTags = tags
	}

	m.activateAutocomplete()
	m.Autocomplete.ToggleSelection()
	m.Autocomplete.Deactivate()
	m.State = FileTagStateInput

	// Press Enter to complete
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(FileTagModel)

	assert.Equal(t, FileTagStateDone, m.State)
	require.NotNil(t, receivedTags)
	assert.Contains(t, receivedTags, "@main.go")
}

func TestFileTagStateConstants(t *testing.T) {
	// Ensure states have different values
	assert.NotEqual(t, FileTagStateInput, FileTagStateAutocomplete)
	assert.NotEqual(t, FileTagStateInput, FileTagStateDone)
	assert.NotEqual(t, FileTagStateInput, FileTagStateCanceled)
	assert.NotEqual(t, FileTagStateAutocomplete, FileTagStateDone)
	assert.NotEqual(t, FileTagStateAutocomplete, FileTagStateCanceled)
	assert.NotEqual(t, FileTagStateDone, FileTagStateCanceled)
}
