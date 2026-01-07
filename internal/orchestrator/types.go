package orchestrator

import "encoding/json"

// Action represents what Claude wants to do next
type Action string

const (
	ActionAskUser    Action = "ask_user"    // Ask the user a question
	ActionReadFiles  Action = "read_files"  // Read files from the codebase
	ActionWriteFile  Action = "write_file"  // Write a file
	ActionRunCommand Action = "run_command" // Run a shell command
	ActionDone       Action = "done"        // Task is complete
)

// Response is the structured response from Claude
type Response struct {
	// Thinking is Claude's internal reasoning (shown in debug mode)
	Thinking string `json:"thinking,omitempty"`

	// Action is what Claude wants to do next
	Action Action `json:"action"`

	// ActionParams contains parameters for the action
	ActionParams ActionParams `json:"action_params,omitempty"`

	// Message is what to display to the user
	Message string `json:"message,omitempty"`

	// State contains the current state of the task
	State json.RawMessage `json:"state,omitempty"`
}

// ActionParams contains parameters for different actions
type ActionParams struct {
	// For ask_user
	Question string `json:"question,omitempty"`

	// For read_files
	Paths []string `json:"paths,omitempty"`

	// For write_file
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`

	// For run_command
	Command string `json:"command,omitempty"`
}

// Message represents a message in the conversation history
type Message struct {
	Role    string `json:"role"` // "user", "assistant", or "system"
	Content string `json:"content"`
}

// Session holds the conversation state
type Session struct {
	ID       string    `json:"id"`
	Mode     string    `json:"mode"` // "plan" or "build"
	WorkDir  string    `json:"work_dir"`
	Messages []Message `json:"messages"`
	State    any       `json:"state,omitempty"`
}

// PlanState holds state specific to the planning phase
type PlanState struct {
	Phase    string    `json:"phase"` // "gathering", "proposing", "refining", "complete"
	DraftPRD *DraftPRD `json:"draft_prd,omitempty"`
}

// DraftPRD is a work-in-progress PRD
type DraftPRD struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	TestCommand string         `json:"testCommand,omitempty"`
	Features    []DraftFeature `json:"features,omitempty"`
}

// DraftFeature is a work-in-progress feature
type DraftFeature struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Priority    string   `json:"priority"`
	Description string   `json:"description"`
	Steps       []string `json:"steps,omitempty"`
}

// BuildState holds state specific to the build phase
type BuildState struct {
	Phase          string `json:"phase"` // "reading", "implementing", "testing", "committing", "complete"
	CurrentFeature string `json:"current_feature,omitempty"`
	Iteration      int    `json:"iteration"`
	TestsPassing   bool   `json:"tests_passing"`
	LastError      string `json:"last_error,omitempty"`
}
