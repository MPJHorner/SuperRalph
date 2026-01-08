package orchestrator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mpjhorner/superralph/internal/progress"
)

// Action represents what Claude wants to do next
type Action string

const (
	ActionAskUser    Action = "ask_user"    // Ask the user a question
	ActionReadFiles  Action = "read_files"  // Read files from the codebase
	ActionWriteFile  Action = "write_file"  // Write a file
	ActionRunCommand Action = "run_command" // Run a shell command
	ActionDone       Action = "done"        // Task is complete
	ActionParallel   Action = "parallel"    // Execute multiple actions in parallel
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
	CurrentStep    Step   `json:"current_step,omitempty"` // Granular step tracking
}

// Phase represents the current phase of the three-phase loop
type Phase string

const (
	PhasePlanning   Phase = "planning"
	PhaseValidating Phase = "validating"
	PhaseExecuting  Phase = "executing"
)

// Step represents the granular step within an iteration
type Step string

const (
	StepReading    Step = "reading"    // Reading PRD, progress, codebase
	StepPlanning   Step = "planning"   // Creating implementation plan
	StepCoding     Step = "coding"     // Writing/editing code
	StepTesting    Step = "testing"    // Running tests
	StepCommitting Step = "committing" // Making git commit
	StepUpdating   Step = "updating"   // Updating prd.json, progress.txt
	StepComplete   Step = "complete"   // Iteration complete
	StepIdle       Step = "idle"       // Not doing anything
)

// StepInfo provides details about a step
type StepInfo struct {
	Step        Step   // Current step
	Description string // Human-readable description
}

// AllSteps returns all steps in order
func AllSteps() []Step {
	return []Step{
		StepReading,
		StepPlanning,
		StepCoding,
		StepTesting,
		StepCommitting,
		StepUpdating,
		StepComplete,
	}
}

// String returns the step as a human-readable string
func (s Step) String() string {
	switch s {
	case StepReading:
		return "Reading"
	case StepPlanning:
		return "Planning"
	case StepCoding:
		return "Coding"
	case StepTesting:
		return "Testing"
	case StepCommitting:
		return "Committing"
	case StepUpdating:
		return "Updating"
	case StepComplete:
		return "Complete"
	case StepIdle:
		return "Idle"
	default:
		return string(s)
	}
}

// ValidationResult represents the result of the validation phase
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Issues   []string `json:"issues,omitempty"`
	Feedback string   `json:"feedback,omitempty"`
}

// PlanOutput represents the output from the planning phase
type PlanOutput struct {
	Plan  string   `json:"plan"`
	Steps []string `json:"steps"`
}

// PhaseConfig holds configuration for the three-phase loop
type PhaseConfig struct {
	MaxValidationAttempts int // Maximum times to loop back from validation to planning (default: 3)
}

// IterationContext holds fresh, self-contained context for each Claude iteration.
// This ensures no conversation history accumulates - each call gets exactly what it needs.
type IterationContext struct {
	// PRDContent is the raw prd.json content
	PRDContent string `json:"prd_content"`

	// ProgressContent is the raw progress.txt content
	ProgressContent string `json:"progress_content"`

	// TaggedFiles maps file paths to their contents
	TaggedFiles map[string]string `json:"tagged_files,omitempty"`

	// TagPatterns stores the original tag patterns used to populate TaggedFiles
	// e.g., ["@src/**/*.go", "@!vendor", "@main.go"]
	TagPatterns []string `json:"tag_patterns,omitempty"`

	// DirectoryTree is the codebase structure
	DirectoryTree string `json:"directory_tree,omitempty"`

	// KeyFiles maps file paths to their contents for automatically detected key files
	// (e.g., go.mod, package.json, Cargo.toml, README.md, main entry points)
	KeyFiles map[string]string `json:"key_files,omitempty"`

	// CurrentFeature is the feature being worked on (if any)
	CurrentFeature *FeatureContext `json:"current_feature,omitempty"`

	// Phase is the current phase (planning, validating, executing)
	Phase Phase `json:"phase,omitempty"`

	// Iteration is the iteration number
	Iteration int `json:"iteration"`

	// PreviousPlan holds the plan from the planning phase (used in validation/execution)
	PreviousPlan string `json:"previous_plan,omitempty"`

	// ValidationFeedback holds feedback from a failed validation (used when re-planning)
	ValidationFeedback string `json:"validation_feedback,omitempty"`

	// ValidationAttempt tracks which validation attempt this is (1-3)
	ValidationAttempt int `json:"validation_attempt,omitempty"`
}

// SnapshotConfig holds configuration for codebase snapshots
type SnapshotConfig struct {
	// MaxTreeDepth is the maximum depth for directory tree (default: 4)
	MaxTreeDepth int `json:"max_tree_depth,omitempty"`

	// MaxFileSizeBytes is the maximum file size to include in key files (default: 50KB)
	MaxFileSizeBytes int64 `json:"max_file_size_bytes,omitempty"`

	// IncludeKeyFiles enables automatic inclusion of key files (default: true)
	IncludeKeyFiles bool `json:"include_key_files,omitempty"`
}

// ToolConfig holds configuration for Claude's native tool permissions
type ToolConfig struct {
	// AllowRead permits the Read tool (default: true)
	AllowRead bool `json:"allow_read,omitempty"`

	// AllowWrite permits the Write tool (default: true, restricted to project dir)
	AllowWrite bool `json:"allow_write,omitempty"`

	// AllowEdit permits the Edit tool (default: true, restricted to project dir)
	AllowEdit bool `json:"allow_edit,omitempty"`

	// AllowedBashCommands is the list of allowed bash command prefixes
	// Each entry allows commands starting with that prefix (e.g., "go" allows "go test", "go build")
	// Default: ["go", "npm", "yarn", "pnpm", "cargo", "python", "pytest", "git", "make"]
	AllowedBashCommands []string `json:"allowed_bash_commands,omitempty"`
}

// DefaultAllowedBashCommands is the default list of safe bash command prefixes
var DefaultAllowedBashCommands = []string{
	"go",     // Go toolchain (go build, go test, go run, etc.)
	"npm",    // Node.js package manager
	"yarn",   // Alternative Node.js package manager
	"pnpm",   // Alternative Node.js package manager
	"cargo",  // Rust package manager and build tool
	"python", // Python interpreter
	"pytest", // Python test framework
	"git",    // Git version control
	"make",   // Make build tool
}

// DefaultToolConfig returns the default tool configuration
// This provides safe defaults that allow Claude to read, write, edit files
// and run common development commands
func DefaultToolConfig() ToolConfig {
	return ToolConfig{
		AllowRead:           true,
		AllowWrite:          true,
		AllowEdit:           true,
		AllowedBashCommands: DefaultAllowedBashCommands,
	}
}

// BuildAllowedToolsFlag builds the --allowedTools flag value from the config
// Format: "Read,Write,Edit,Bash(go:*),Bash(npm:*),.."
func (tc *ToolConfig) BuildAllowedToolsFlag() string {
	var tools []string

	if tc.AllowRead {
		tools = append(tools, "Read")
	}
	if tc.AllowWrite {
		tools = append(tools, "Write")
	}
	if tc.AllowEdit {
		tools = append(tools, "Edit")
	}

	for _, cmd := range tc.AllowedBashCommands {
		tools = append(tools, "Bash("+cmd+":*)")
	}

	return strings.Join(tools, ",")
}

// DefaultSnapshotConfig returns the default snapshot configuration
func DefaultSnapshotConfig() SnapshotConfig {
	return SnapshotConfig{
		MaxTreeDepth:     4,
		MaxFileSizeBytes: 50 * 1024, // 50KB
		IncludeKeyFiles:  true,
	}
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

	// Key files if any (automatically detected project files)
	if len(ic.KeyFiles) > 0 {
		sb.WriteString("## Key Files\n")
		sb.WriteString("These are automatically detected important project files:\n\n")
		for path, content := range ic.KeyFiles {
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", path, content))
		}
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
		sb.WriteString(fmt.Sprintf("## Current Phase: %s\n\n", string(ic.Phase)))
		sb.WriteString(ic.buildPhaseInstructions())
		sb.WriteString("\n")
	} else {
		// Default task instructions when no phase is set (legacy mode)
		sb.WriteString(ic.buildDefaultTaskInstructions())
	}

	return sb.String()
}

// buildPhaseInstructions returns phase-specific instructions
func (ic *IterationContext) buildPhaseInstructions() string {
	switch ic.Phase {
	case PhasePlanning:
		return ic.buildPlanningInstructions()
	case PhaseValidating:
		return ic.buildValidatingInstructions()
	case PhaseExecuting:
		return ic.buildExecutingInstructions()
	default:
		return ic.buildDefaultTaskInstructions()
	}
}

// buildPlanningInstructions returns instructions for the planning phase
func (ic *IterationContext) buildPlanningInstructions() string {
	var sb strings.Builder

	sb.WriteString(`### Planning Phase Instructions

Create a detailed implementation plan for the current feature. Your plan should:

1. **Analyze the codebase** - Read relevant files to understand the current architecture
2. **Identify changes needed** - List specific files to create, modify, or delete
3. **Define implementation steps** - Break down the work into concrete, sequential steps
4. **Consider edge cases** - Think about error handling, edge cases, and tests
5. **Estimate test coverage** - Identify what tests need to be added or modified

`)

	// If there's validation feedback from a previous attempt, include it
	if ic.ValidationFeedback != "" {
		sb.WriteString(fmt.Sprintf(`### Previous Validation Feedback (Attempt %d/3)

Your previous plan was rejected during validation. Address these issues:

%s

Please revise your plan to address all the issues above.

`, ic.ValidationAttempt, ic.ValidationFeedback))
	}

	sb.WriteString(`### Output Format

At the end of your planning, output your plan in this format:

<plan>
## Implementation Plan for [Feature ID]

### Overview
[Brief description of what will be implemented]

### Files to Modify
- path/to/file1.go: [what changes]
- path/to/file2.go: [what changes]

### Files to Create
- path/to/new_file.go: [purpose]

### Implementation Steps
1. [First step with specific details]
2. [Second step with specific details]
3. [Continue...]

### Tests to Add/Modify
- [Test file and what tests]

### Edge Cases Considered
- [Edge case 1]
- [Edge case 2]
</plan>

IMPORTANT: Only output the plan. Do NOT implement anything yet. The plan will be validated before execution.`)

	return sb.String()
}

// buildValidatingInstructions returns instructions for the validation phase
func (ic *IterationContext) buildValidatingInstructions() string {
	var sb strings.Builder

	sb.WriteString(`### Validation Phase Instructions

Review the implementation plan below for completeness, correctness, and potential issues.

`)

	if ic.PreviousPlan != "" {
		sb.WriteString(fmt.Sprintf(`### Plan to Validate

%s

`, ic.PreviousPlan))
	}

	sb.WriteString(`### Validation Checklist

Check the plan against these criteria:

1. **Completeness** - Does the plan cover all aspects of the feature?
2. **Correctness** - Are the proposed changes technically sound?
3. **Edge Cases** - Are edge cases and error conditions handled?
4. **Test Coverage** - Are appropriate tests included?
5. **Dependencies** - Are all dependencies and imports considered?
6. **Breaking Changes** - Could this break existing functionality?
7. **Code Style** - Does the plan follow the project's patterns?

### Output Format

Output your validation result in this format:

<validation>
valid: [true/false]
issues:
- [Issue 1 if any]
- [Issue 2 if any]
feedback: [Detailed feedback for re-planning if not valid]
</validation>

If the plan is valid, set valid: true and leave issues empty.
If the plan has problems, set valid: false and list all issues with actionable feedback.

IMPORTANT: Be thorough but pragmatic. Minor issues that can be handled during implementation should not block the plan.`)

	return sb.String()
}

// buildExecutingInstructions returns instructions for the execution phase
func (ic *IterationContext) buildExecutingInstructions() string {
	var sb strings.Builder

	sb.WriteString(`### Execution Phase Instructions

Execute the validated implementation plan step by step.

`)

	if ic.PreviousPlan != "" {
		sb.WriteString(fmt.Sprintf(`### Validated Plan to Execute

%s

`, ic.PreviousPlan))
	}

	sb.WriteString(`### Execution Rules

1. **Follow the plan** - Implement each step in order
2. **Run tests frequently** - After each significant change, run tests
3. **Fix issues immediately** - If tests fail, fix before continuing
4. **Commit only when passing** - All tests must pass before committing

### On Completion

When you've finished implementing the feature:

1. Run the test command to verify all tests pass
2. Update prd.json to set "passes": true for this feature
3. Make a git commit with a descriptive message
4. Append a summary to progress.txt

### Output

As you work, explain what you're doing. When complete, output:

<execution_complete>
feature: [Feature ID]
tests_passing: [true/false]
committed: [true/false]
summary: [Brief summary of what was implemented]
</execution_complete>`)

	return sb.String()
}

// buildDefaultTaskInstructions returns the default task instructions (single-feature mode)
func (ic *IterationContext) buildDefaultTaskInstructions() string {
	return `## Your Task - Single Feature Implementation

This is iteration ` + fmt.Sprintf("%d", ic.Iteration) + `. You will implement ONE feature then EXIT.

### Step 1: Select the Next Feature

Look at the PRD and select the next feature using this logic:
- Skip features with passes: true (already done)
- Skip features blocked by unmet dependencies (check depends_on field)
- Pick the highest priority first (high > medium > low)
- Within same priority, pick first by ID order

Report which feature you selected and WHY.

### Step 2: Implement the Feature

1. Read relevant code to understand the current implementation
2. Make the necessary changes to implement the feature
3. Write/update tests as needed

### Step 3: Run Tests

Run the tests using the testCommand from the PRD.

- If tests PASS: Continue to Step 4
- If tests FAIL: Fix the issues and run tests again (max 3 attempts)

### Step 4: Commit and Update (only if tests pass)

1. Update prd.json to set "passes": true for the completed feature
2. Make a git commit with a descriptive message
3. Append a summary to progress.txt with:
   - Feature ID and description
   - What was implemented
   - Any notes for future iterations

### Step 5: EXIT

After completing (or failing) this ONE feature, you are DONE.
Do NOT continue to the next feature.
The orchestrator will restart you with fresh context for the next iteration.

---

## IMPORTANT RULES

- Tests MUST pass before any commit
- Work on ONLY ONE feature per iteration
- EXIT after completing or failing the feature
- Make small, incremental changes
- The orchestrator handles looping - you handle one feature

This "clean slate" approach ensures each iteration starts fresh without accumulated context.`
}

// ProgressEntryBuilder helps construct progress entries incrementally during an iteration.
// It accumulates work done, test results, and commits throughout the iteration,
// then produces a complete progress.Entry when the iteration completes.
type ProgressEntryBuilder struct {
	// Timestamp when the iteration started
	Timestamp time.Time

	// Iteration number
	Iteration int

	// Starting state captured at iteration start
	StartingState progress.State

	// WorkDone accumulates descriptions of work performed
	WorkDone []string

	// Testing holds the most recent test result
	Testing progress.TestResult

	// Commits accumulates git commits made during this iteration
	Commits []progress.Commit

	// NotesForNextSession accumulates notes for future iterations
	NotesForNextSession []string

	// CurrentFeature is the feature being worked on
	CurrentFeature *progress.FeatureRef
}

// NewProgressEntryBuilder creates a new builder initialized with current timestamp
func NewProgressEntryBuilder(iteration int) *ProgressEntryBuilder {
	return &ProgressEntryBuilder{
		Timestamp: time.Now().UTC(),
		Iteration: iteration,
		WorkDone:  []string{},
		Commits:   []progress.Commit{},
	}
}

// SetStartingState sets the starting state from PRD stats
func (b *ProgressEntryBuilder) SetStartingState(total, passing int, feature *progress.FeatureRef) *ProgressEntryBuilder {
	b.StartingState = progress.State{
		FeaturesTotal:   total,
		FeaturesPassing: passing,
		WorkingOn:       feature,
	}
	b.CurrentFeature = feature
	return b
}

// AddWorkDone adds a work item description
func (b *ProgressEntryBuilder) AddWorkDone(work string) *ProgressEntryBuilder {
	b.WorkDone = append(b.WorkDone, work)
	return b
}

// SetTestResult sets the test result
func (b *ProgressEntryBuilder) SetTestResult(command string, passed bool, details string) *ProgressEntryBuilder {
	b.Testing = progress.TestResult{
		Command: command,
		Passed:  passed,
		Details: details,
	}
	return b
}

// AddCommit adds a git commit
func (b *ProgressEntryBuilder) AddCommit(hash, message string) *ProgressEntryBuilder {
	b.Commits = append(b.Commits, progress.Commit{
		Hash:    hash,
		Message: message,
	})
	return b
}

// AddNote adds a note for the next session
func (b *ProgressEntryBuilder) AddNote(note string) *ProgressEntryBuilder {
	b.NotesForNextSession = append(b.NotesForNextSession, note)
	return b
}

// Build creates the final progress.Entry with ending state
func (b *ProgressEntryBuilder) Build(endTotal, endPassing int, allTestsPassing bool) progress.Entry {
	return progress.Entry{
		Timestamp:     b.Timestamp,
		Iteration:     b.Iteration,
		StartingState: b.StartingState,
		WorkDone:      b.WorkDone,
		Testing:       b.Testing,
		Commits:       b.Commits,
		EndingState: progress.State{
			FeaturesTotal:   endTotal,
			FeaturesPassing: endPassing,
			WorkingOn:       b.CurrentFeature,
			AllTestsPassing: allTestsPassing,
		},
		NotesForNextSession: b.NotesForNextSession,
	}
}
