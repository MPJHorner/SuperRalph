package progress

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultFilename = "progress.txt"

// Writer handles writing progress entries to the progress file
type Writer struct {
	path string
}

// NewWriter creates a new progress writer for the given directory
func NewWriter(dir string) *Writer {
	return &Writer{
		path: filepath.Join(dir, DefaultFilename),
	}
}

// NewWriterForCurrentDir creates a new progress writer for the current directory
func NewWriterForCurrentDir() (*Writer, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	return NewWriter(cwd), nil
}

// Append appends a new entry to the progress file
func (w *Writer) Append(entry Entry) error {
	content := formatEntry(entry)

	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open progress file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("failed to write progress entry: %w", err)
	}

	return nil
}

// Path returns the path to the progress file
func (w *Writer) Path() string {
	return w.path
}

// Exists checks if the progress file exists
func (w *Writer) Exists() bool {
	_, err := os.Stat(w.path)
	return err == nil
}

func formatEntry(e Entry) string {
	var sb strings.Builder

	// Header
	sb.WriteString("================================================================================\n")
	sb.WriteString(fmt.Sprintf("Session: %s\n", e.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Iteration: %d\n", e.Iteration))
	sb.WriteString("================================================================================\n\n")

	// Starting State
	sb.WriteString("## Starting State\n")
	sb.WriteString(fmt.Sprintf("- Features passing: %d/%d\n", e.StartingState.FeaturesPassing, e.StartingState.FeaturesTotal))
	if e.StartingState.WorkingOn != nil {
		sb.WriteString(fmt.Sprintf("- Working on: %s \"%s\"\n", e.StartingState.WorkingOn.ID, e.StartingState.WorkingOn.Description))
	}
	sb.WriteString("\n")

	// Work Done
	sb.WriteString("## Work Done\n")
	for _, work := range e.WorkDone {
		sb.WriteString(fmt.Sprintf("- %s\n", work))
	}
	sb.WriteString("\n")

	// Testing
	sb.WriteString("## Testing\n")
	sb.WriteString(fmt.Sprintf("- Test command: %s\n", e.Testing.Command))
	if e.Testing.Passed {
		sb.WriteString("- Result: PASSED\n")
	} else {
		sb.WriteString("- Result: FAILED\n")
	}
	if e.Testing.Details != "" {
		sb.WriteString(fmt.Sprintf("- Details: %s\n", e.Testing.Details))
	}
	sb.WriteString("\n")

	// Commits
	sb.WriteString("## Commits\n")
	for _, commit := range e.Commits {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", commit.Hash, commit.Message))
	}
	sb.WriteString("\n")

	// Ending State
	sb.WriteString("## Ending State\n")
	sb.WriteString(fmt.Sprintf("- Features passing: %d/%d\n", e.EndingState.FeaturesPassing, e.EndingState.FeaturesTotal))
	if e.EndingState.WorkingOn != nil {
		sb.WriteString(fmt.Sprintf("- Feature %s marked as passes: true\n", e.EndingState.WorkingOn.ID))
	}
	if e.EndingState.AllTestsPassing {
		sb.WriteString("- All tests passing: YES\n")
	} else {
		sb.WriteString("- All tests passing: NO\n")
	}
	sb.WriteString("\n")

	// Notes for Next Session
	sb.WriteString("## Notes for Next Session\n")
	for _, note := range e.NotesForNextSession {
		sb.WriteString(fmt.Sprintf("- %s\n", note))
	}
	sb.WriteString("\n")

	return sb.String()
}

// GetPath returns the path to the progress file in the given directory
func GetPath(dir string) string {
	return filepath.Join(dir, DefaultFilename)
}

// GetPathForCurrentDir returns the path to the progress file in the current directory
func GetPathForCurrentDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return GetPath(cwd), nil
}

// ExistsInDir checks if progress file exists in directory
func ExistsInDir(dir string) bool {
	_, err := os.Stat(GetPath(dir))
	return err == nil
}

// Read reads the progress file content
func Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No progress yet is fine
		}
		return "", err
	}
	return string(data), nil
}

// ReadFromCurrentDir reads progress from current directory
func ReadFromCurrentDir() (string, error) {
	path, err := GetPathForCurrentDir()
	if err != nil {
		return "", err
	}
	return Read(path)
}
