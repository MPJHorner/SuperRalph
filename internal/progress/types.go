package progress

import "time"

// Entry represents a single progress entry/session
type Entry struct {
	Timestamp           time.Time
	Iteration           int
	StartingState       State
	WorkDone            []string
	Testing             TestResult
	Commits             []Commit
	EndingState         State
	NotesForNextSession []string
}

// State represents the state of the project at a point in time
type State struct {
	FeaturesTotal   int
	FeaturesPassing int
	WorkingOn       *FeatureRef
	AllTestsPassing bool
}

// FeatureRef is a reference to a feature
type FeatureRef struct {
	ID          string
	Description string
}

// TestResult represents the result of running tests
type TestResult struct {
	Command string
	Passed  bool
	Details string
}

// Commit represents a git commit
type Commit struct {
	Hash    string
	Message string
}

// Progress represents the full progress file
type Progress struct {
	Entries []Entry
}

// LatestIteration returns the latest iteration number
func (p *Progress) LatestIteration() int {
	if len(p.Entries) == 0 {
		return 0
	}
	return p.Entries[len(p.Entries)-1].Iteration
}

// LatestEntry returns the most recent entry
func (p *Progress) LatestEntry() *Entry {
	if len(p.Entries) == 0 {
		return nil
	}
	return &p.Entries[len(p.Entries)-1]
}
