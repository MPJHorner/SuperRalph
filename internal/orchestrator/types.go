package orchestrator

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

// Phase represents the current phase of the three-phase loop
type Phase string

const (
	PhasePlanning   Phase = "planning"
	PhaseValidating Phase = "validating"
	PhaseExecuting  Phase = "executing"
)

// IterationContext holds fresh, self-contained context for each Claude iteration.
// This ensures no conversation history accumulates - each call gets exactly what it needs.
type IterationContext struct {
	// PRDContent is the raw prd.json content
	PRDContent string `json:"prd_content"`

	// ProgressContent is the raw progress.txt content
	ProgressContent string `json:"progress_content"`

	// TaggedFiles maps file paths to their contents
	TaggedFiles map[string]string `json:"tagged_files,omitempty"`

	// DirectoryTree is the codebase structure
	DirectoryTree string `json:"directory_tree,omitempty"`

	// CurrentFeature is the feature being worked on (if any)
	CurrentFeature *FeatureContext `json:"current_feature,omitempty"`

	// Phase is the current phase (planning, validating, executing)
	Phase Phase `json:"phase,omitempty"`

	// Iteration is the iteration number
	Iteration int `json:"iteration"`
}

// FeatureContext holds information about the feature being worked on
type FeatureContext struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
	Priority    string   `json:"priority"`
	Category    string   `json:"category"`
}

// BuildPrompt generates the complete prompt from the iteration context
func (ic *IterationContext) BuildPrompt() string {
	var sb strings.Builder

	sb.WriteString("You are implementing features from a PRD. Here is the current state:\n\n")

	// PRD content
	sb.WriteString("## prd.json\n")
	sb.WriteString(ic.PRDContent)
	sb.WriteString("\n\n")

	// Progress content
	sb.WriteString("## progress.txt\n")
	if ic.ProgressContent != "" {
		sb.WriteString(ic.ProgressContent)
	} else {
		sb.WriteString("(empty)")
	}
	sb.WriteString("\n\n")

	// Directory tree if available
	if ic.DirectoryTree != "" {
		sb.WriteString("## Directory Structure\n```\n")
		sb.WriteString(ic.DirectoryTree)
		sb.WriteString("\n```\n\n")
	}

	// Tagged files if any
	if len(ic.TaggedFiles) > 0 {
		sb.WriteString("## Tagged Files\n")
		for path, content := range ic.TaggedFiles {
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", path, content))
		}
	}

	// Current feature context
	if ic.CurrentFeature != nil {
		sb.WriteString("## Current Feature\n")
		sb.WriteString(fmt.Sprintf("Working on: %s - %s\n", ic.CurrentFeature.ID, ic.CurrentFeature.Description))
		sb.WriteString(fmt.Sprintf("Priority: %s, Category: %s\n", ic.CurrentFeature.Priority, ic.CurrentFeature.Category))
		sb.WriteString("Steps:\n")
		for i, step := range ic.CurrentFeature.Steps {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
		}
		sb.WriteString("\n")
	}

	// Phase-specific instructions
	if ic.Phase != "" {
		sb.WriteString(fmt.Sprintf("## Current Phase: %s\n", string(ic.Phase)))
		switch ic.Phase {
		case PhasePlanning:
			sb.WriteString("Create a detailed implementation plan for the current feature.\n")
		case PhaseValidating:
			sb.WriteString("Review your implementation plan for gaps, edge cases, and issues.\n")
		case PhaseExecuting:
			sb.WriteString("Execute your validated plan step by step.\n")
		}
		sb.WriteString("\n")
	}

	// Task instructions
	sb.WriteString(`## Your Task

1. Look at the PRD and find the highest priority feature where "passes" is false
2. Implement that feature
3. Run the tests using the testCommand from the PRD
4. If tests pass:
   - Update prd.json to set "passes": true for the completed feature
   - Make a git commit with a descriptive message
   - Append a summary to progress.txt
5. If tests fail, fix the issues and try again
6. Continue until all features pass or you need to stop

IMPORTANT RULES:
- Tests MUST pass before any commit
- Work on ONE feature at a time
- Make small, incremental changes
- Always run tests after changes

Start by reading the codebase to understand the current implementation, then implement the next feature.`)

	return sb.String()
}
