package orchestrator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mpjhorner/superralph/internal/prd"
	"github.com/mpjhorner/superralph/internal/progress"
	"github.com/mpjhorner/superralph/internal/tagging"
)

// OutputType represents the type of output for styled/colored display in TUI
type OutputType string

const (
	OutputText       OutputType = "text"        // Claude's explanatory text (white)
	OutputToolUse    OutputType = "tool_use"    // Tool being invoked (cyan)
	OutputToolInput  OutputType = "tool_input"  // Tool input/command (cyan, indented)
	OutputToolResult OutputType = "tool_result" // Tool output (muted gray)
	OutputPhase      OutputType = "phase"       // Phase changes (purple)
	OutputSuccess    OutputType = "success"     // Success messages (green)
	OutputError      OutputType = "error"       // Errors (red)
	OutputInfo       OutputType = "info"        // Info/status (muted)
)

// Orchestrator manages the agent loop
type Orchestrator struct {
	workDir        string
	claudePath     string
	session        *Session
	debug          bool
	tagger         *tagging.Tagger
	parallel       *ParallelExecutor
	snapshotConfig SnapshotConfig
	toolConfig     ToolConfig

	// Progress tracking
	progressWriter *progress.Writer
	currentEntry   *ProgressEntryBuilder // Builder for the current progress entry

	// Initial tags for planning context
	initialTags []string

	// Callbacks for UI integration
	onMessage     func(role, content string)
	onAction      func(action Action, params ActionParams)
	onState       func(state any)
	onThinking    func(thinking string)
	onDebug       func(msg string)
	onOutput      func(line string)
	onTypedOutput func(outputType OutputType, content string)
	onActivity    func(activity string) // Current activity summary (e.g., "Reading src/main.go")
	onStep        func(step Step)       // Current step in the iteration
	promptUser    func(question string) (string, error)
}

// New creates a new Orchestrator
func New(workDir string) *Orchestrator {
	return &Orchestrator{
		workDir:        workDir,
		claudePath:     findClaudeBinary(),
		tagger:         tagging.New(workDir),
		parallel:       NewParallelExecutor(workDir),
		snapshotConfig: DefaultSnapshotConfig(),
		toolConfig:     DefaultToolConfig(),
		progressWriter: progress.NewWriter(workDir),
		session: &Session{
			ID:       uuid.New().String(),
			WorkDir:  workDir,
			Messages: []Message{},
		},
	}
}

// findClaudeBinary searches for the Claude CLI binary
func findClaudeBinary() string {
	if envPath := os.Getenv("CLAUDE_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	homeDir, _ := os.UserHomeDir()
	locations := []string{
		"claude",
		filepath.Join(homeDir, ".claude", "local", "claude"),
		"/usr/local/bin/claude",
		"/usr/bin/claude",
		filepath.Join(homeDir, ".local", "bin", "claude"),
	}

	for _, loc := range locations {
		if loc == "claude" {
			if path, err := exec.LookPath("claude"); err == nil {
				return path
			}
			continue
		}
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return "claude"
}

// SetDebug enables debug mode
func (o *Orchestrator) SetDebug(debug bool) *Orchestrator {
	o.debug = debug
	return o
}

// OnMessage sets the callback for messages
func (o *Orchestrator) OnMessage(fn func(role, content string)) *Orchestrator {
	o.onMessage = fn
	return o
}

// OnAction sets the callback for actions
func (o *Orchestrator) OnAction(fn func(action Action, params ActionParams)) *Orchestrator {
	o.onAction = fn
	return o
}

// OnState sets the callback for state updates
func (o *Orchestrator) OnState(fn func(state any)) *Orchestrator {
	o.onState = fn
	return o
}

// OnThinking sets the callback for thinking (debug)
func (o *Orchestrator) OnThinking(fn func(thinking string)) *Orchestrator {
	o.onThinking = fn
	return o
}

// OnDebug sets the callback for debug messages
func (o *Orchestrator) OnDebug(fn func(msg string)) *Orchestrator {
	o.onDebug = fn
	return o
}

// OnOutput sets the callback for Claude's streaming output
func (o *Orchestrator) OnOutput(fn func(line string)) *Orchestrator {
	o.onOutput = fn
	return o
}

// OnTypedOutput sets the callback for typed/colored output messages
func (o *Orchestrator) OnTypedOutput(fn func(outputType OutputType, content string)) *Orchestrator {
	o.onTypedOutput = fn
	return o
}

// OnActivity sets the callback for current activity updates
func (o *Orchestrator) OnActivity(fn func(activity string)) *Orchestrator {
	o.onActivity = fn
	return o
}

// OnStep sets the callback for step changes
func (o *Orchestrator) OnStep(fn func(step Step)) *Orchestrator {
	o.onStep = fn
	return o
}

// SetPromptUser sets the function to prompt the user
func (o *Orchestrator) SetPromptUser(fn func(question string) (string, error)) *Orchestrator {
	o.promptUser = fn
	return o
}

// SetInitialTags sets the initial file tags for planning context
func (o *Orchestrator) SetInitialTags(tags []string) *Orchestrator {
	o.initialTags = tags
	return o
}

// GetInitialTags returns the initial file tags
func (o *Orchestrator) GetInitialTags() []string {
	return o.initialTags
}

// SetSnapshotConfig sets the snapshot configuration
func (o *Orchestrator) SetSnapshotConfig(config SnapshotConfig) *Orchestrator {
	o.snapshotConfig = config
	return o
}

// GetSnapshotConfig returns the current snapshot configuration
func (o *Orchestrator) GetSnapshotConfig() SnapshotConfig {
	return o.snapshotConfig
}

// SetMaxTreeDepth sets the maximum directory tree depth
func (o *Orchestrator) SetMaxTreeDepth(depth int) *Orchestrator {
	o.snapshotConfig.MaxTreeDepth = depth
	return o
}

// SetMaxFileSizeBytes sets the maximum file size for key files
func (o *Orchestrator) SetMaxFileSizeBytes(size int64) *Orchestrator {
	o.snapshotConfig.MaxFileSizeBytes = size
	return o
}

// SetIncludeKeyFiles enables or disables automatic key file inclusion
func (o *Orchestrator) SetIncludeKeyFiles(include bool) *Orchestrator {
	o.snapshotConfig.IncludeKeyFiles = include
	return o
}

// SetToolConfig sets the tool configuration
func (o *Orchestrator) SetToolConfig(config ToolConfig) *Orchestrator {
	o.toolConfig = config
	return o
}

// GetToolConfig returns the current tool configuration
func (o *Orchestrator) GetToolConfig() ToolConfig {
	return o.toolConfig
}

// SetAllowedBashCommands sets the list of allowed bash command prefixes
func (o *Orchestrator) SetAllowedBashCommands(commands []string) *Orchestrator {
	o.toolConfig.AllowedBashCommands = commands
	return o
}

// AddAllowedBashCommand adds a bash command to the allowed list
func (o *Orchestrator) AddAllowedBashCommand(command string) *Orchestrator {
	o.toolConfig.AllowedBashCommands = append(o.toolConfig.AllowedBashCommands, command)
	return o
}

// SetAllowRead enables or disables the Read tool
func (o *Orchestrator) SetAllowRead(allow bool) *Orchestrator {
	o.toolConfig.AllowRead = allow
	return o
}

// SetAllowWrite enables or disables the Write tool
func (o *Orchestrator) SetAllowWrite(allow bool) *Orchestrator {
	o.toolConfig.AllowWrite = allow
	return o
}

// SetAllowEdit enables or disables the Edit tool
func (o *Orchestrator) SetAllowEdit(allow bool) *Orchestrator {
	o.toolConfig.AllowEdit = allow
	return o
}

// GetProgressWriter returns the progress writer for external use
func (o *Orchestrator) GetProgressWriter() *progress.Writer {
	return o.progressWriter
}

// StartProgressEntry begins a new progress entry for the current iteration.
// Call this at the start of each iteration to track work done.
func (o *Orchestrator) StartProgressEntry(iteration int, currentPRD *prd.PRD) {
	stats := currentPRD.Stats()
	nextFeature := currentPRD.NextFeature()

	var featureRef *progress.FeatureRef
	if nextFeature != nil {
		featureRef = &progress.FeatureRef{
			ID:          nextFeature.ID,
			Description: nextFeature.Description,
		}
	}

	o.currentEntry = NewProgressEntryBuilder(iteration)
	o.currentEntry.SetStartingState(stats.TotalFeatures, stats.PassingFeatures, featureRef)
}

// AddProgressWork adds a work item to the current progress entry
func (o *Orchestrator) AddProgressWork(work string) {
	if o.currentEntry != nil {
		o.currentEntry.AddWorkDone(work)
	}
}

// SetProgressTestResult sets the test result for the current progress entry
func (o *Orchestrator) SetProgressTestResult(command string, passed bool, details string) {
	if o.currentEntry != nil {
		o.currentEntry.SetTestResult(command, passed, details)
	}
}

// AddProgressCommit adds a git commit to the current progress entry
func (o *Orchestrator) AddProgressCommit(hash, message string) {
	if o.currentEntry != nil {
		o.currentEntry.AddCommit(hash, message)
	}
}

// AddProgressNote adds a note for the next session
func (o *Orchestrator) AddProgressNote(note string) {
	if o.currentEntry != nil {
		o.currentEntry.AddNote(note)
	}
}

// FinishProgressEntry completes and writes the current progress entry.
// Call this at the end of each iteration or significant checkpoint.
func (o *Orchestrator) FinishProgressEntry(currentPRD *prd.PRD, allTestsPassing bool) error {
	if o.currentEntry == nil {
		return nil // No entry to finish
	}

	stats := currentPRD.Stats()
	entry := o.currentEntry.Build(stats.TotalFeatures, stats.PassingFeatures, allTestsPassing)

	// Write the entry to the progress file
	if err := o.progressWriter.Append(entry); err != nil {
		return fmt.Errorf("failed to write progress entry: %w", err)
	}

	// Clear the current entry
	o.currentEntry = nil
	return nil
}

// HasProgressEntry returns true if there's an active progress entry being built
func (o *Orchestrator) HasProgressEntry() bool {
	return o.currentEntry != nil
}

// GetCurrentProgressEntry returns the current progress entry builder (for testing)
func (o *Orchestrator) GetCurrentProgressEntry() *ProgressEntryBuilder {
	return o.currentEntry
}

// LoadSession loads a session from disk
func (o *Orchestrator) LoadSession(id string) error {
	path := filepath.Join(o.workDir, ".superralph", "sessions", id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &o.session)
}

// SaveResumeState saves the current build state for later resumption.
// This is called during graceful shutdown to preserve progress.
func (o *Orchestrator) SaveResumeState(state *ResumeState) error {
	dir := filepath.Join(o.workDir, ".superralph")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create .superralph directory: %w", err)
	}

	state.Timestamp = time.Now().UTC()
	state.WorkDir = o.workDir
	if state.PRDPath == "" {
		state.PRDPath = "prd.json"
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal resume state: %w", err)
	}

	path := filepath.Join(o.workDir, ResumeStateFile)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write resume state: %w", err)
	}

	return nil
}

// LoadResumeState loads the resume state from disk.
// Returns nil if no resume state exists.
func (o *Orchestrator) LoadResumeState() (*ResumeState, error) {
	path := filepath.Join(o.workDir, ResumeStateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No resume state - this is normal
		}
		return nil, fmt.Errorf("failed to read resume state: %w", err)
	}

	var state ResumeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse resume state: %w", err)
	}

	return &state, nil
}

// ClearResumeState removes the resume state file.
// This is called on successful completion.
func (o *Orchestrator) ClearResumeState() error {
	path := filepath.Join(o.workDir, ResumeStateFile)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove resume state: %w", err)
	}
	return nil
}

// HasResumeState checks if a resume state file exists.
func (o *Orchestrator) HasResumeState() bool {
	path := filepath.Join(o.workDir, ResumeStateFile)
	_, err := os.Stat(path)
	return err == nil
}

// SaveSession saves the session to disk
func (o *Orchestrator) SaveSession() error {
	dir := filepath.Join(o.workDir, ".superralph", "sessions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, o.session.ID+".json")
	data, err := json.MarshalIndent(o.session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// RunPlan runs the planning loop
func (o *Orchestrator) RunPlan(ctx context.Context) error {
	o.session.Mode = "plan"
	o.session.State = &PlanState{Phase: "gathering"}

	// Build prompt with optional tagged files context
	var promptBuilder strings.Builder

	promptBuilder.WriteString(`Help me create a prd.json file for this project.

Ask me what I want to build, explore the existing codebase if there is one,
and help me define features with clear verification steps.

When done, create the prd.json file with this structure:
{
  "name": "Project Name",
  "description": "Description",
  "testCommand": "command to run tests",
  "features": [
    {
      "id": "feat-001",
      "category": "functional",
      "priority": "high",
      "description": "Feature description",
      "steps": ["Step 1", "Step 2"],
      "passes": false
    }
  ]
}
`)

	// Add tagged files context if any
	if len(o.initialTags) > 0 {
		promptBuilder.WriteString("\n\n## Tagged Files for Context\n\n")
		promptBuilder.WriteString("The user has tagged the following files as important for planning:\n\n")

		// Load tagged file contents
		tags, err := o.tagger.ResolveTags(o.initialTags)
		if err == nil {
			filesMap, err := o.tagger.BuildTaggedFilesMap(tags)
			if err == nil && len(filesMap) > 0 {
				for path, content := range filesMap {
					promptBuilder.WriteString(fmt.Sprintf("### %s\n\n```\n%s\n```\n\n", path, content))
				}
			}
		}
	}

	promptBuilder.WriteString("\nStart by asking what I want to build.")

	return o.runClaudeInteractive(ctx, promptBuilder.String())
}

// BuildConfig holds configuration for the build loop
type BuildConfig struct {
	// MaxIterations is the safety limit for agent loops (default: 50)
	MaxIterations int

	// DelayBetweenIterations is the delay between Claude invocations (default: 3s)
	DelayBetweenIterations time.Duration

	// StartIteration is the iteration to start from (default: 1, used for resume)
	StartIteration int

	// ResumeFeature is the feature ID to resume from (if resuming)
	ResumeFeature string
}

// DefaultBuildConfig returns the default build configuration
func DefaultBuildConfig() BuildConfig {
	return BuildConfig{
		MaxIterations:          50,
		DelayBetweenIterations: 3 * time.Second,
		StartIteration:         1,
	}
}

// RunBuild runs the build loop using fresh context per iteration.
// Each Claude call gets clean, self-contained context with no conversation history accumulation.
// This is the core "Ralph Wiggum" loop: run Claude, check completion, loop with fresh context.
func (o *Orchestrator) RunBuild(ctx context.Context) error {
	return o.RunBuildWithConfig(ctx, DefaultBuildConfig())
}

// RunBuildWithConfig runs the build loop with custom configuration.
// This implements the core Ralph Wiggum concept:
// 1. Read prd.json fresh each iteration
// 2. Check if all features pass - if so, exit
// 3. Build fresh iteration context (clean slate)
// 4. Run Claude once for a single feature
// 5. Wait for Claude to complete
// 6. Loop back to step 1
//
// Graceful shutdown: On context cancellation, the current action is completed before
// saving state and exiting. Use --resume to continue from where you left off.
func (o *Orchestrator) RunBuildWithConfig(ctx context.Context, config BuildConfig) error {
	o.session.Mode = "build"

	// Determine starting iteration
	startIteration := config.StartIteration
	if startIteration < 1 {
		startIteration = 1
	}

	// Track current state for potential resume
	var currentFeatureID string
	var currentPhase Phase

	for iteration := startIteration; iteration <= config.MaxIterations; iteration++ {
		// Check context cancellation at start of each iteration
		if ctx.Err() != nil {
			// Save state for resume
			o.saveInterruptedState(currentFeatureID, currentPhase, iteration, config.MaxIterations)
			return ctx.Err()
		}

		// === Step 1: Read PRD fresh from disk ===
		o.typedOutput(OutputInfo, fmt.Sprintf("=== Iteration %d/%d ===", iteration, config.MaxIterations))
		o.activity("Loading PRD...")

		currentPRD, err := prd.LoadFromDir(o.workDir)
		if err != nil {
			return fmt.Errorf("failed to load prd.json: %w", err)
		}

		// === Step 2: Check if all features complete ===
		if currentPRD.IsComplete() {
			o.typedOutput(OutputSuccess, "All features complete!")
			o.activity("Complete")
			// Clear resume state on successful completion
			_ = o.ClearResumeState()
			return nil
		}

		// Get next feature for display
		nextFeature, reason := currentPRD.NextFeatureWithReason()
		if nextFeature == nil {
			// All features pass or are blocked - should not happen if IsComplete returned false
			o.typedOutput(OutputError, "No available features to work on (all may be blocked)")
			return fmt.Errorf("no available features: %s", reason)
		}

		// Update current feature for potential interrupt save
		currentFeatureID = nextFeature.ID
		currentPhase = "" // Default phase

		stats := currentPRD.Stats()
		o.typedOutput(OutputInfo, fmt.Sprintf("Progress: %d/%d features complete", stats.PassingFeatures, stats.TotalFeatures))
		o.typedOutput(OutputInfo, fmt.Sprintf("Next: %s - %s", nextFeature.ID, nextFeature.Description))

		// === Step 3: Build fresh iteration context (clean slate) ===
		buildState := &BuildState{
			Phase:          "reading",
			Iteration:      iteration,
			CurrentFeature: nextFeature.ID,
		}
		o.session.State = buildState

		// Notify TUI of iteration start
		if o.onState != nil {
			o.onState(buildState)
		}

		iterCtx, err := o.BuildIterationContext(iteration, "", nil)
		if err != nil {
			return fmt.Errorf("failed to build iteration context: %w", err)
		}

		// Generate prompt from fresh context
		prompt := iterCtx.BuildPrompt()

		// Clear any accumulated messages - each iteration is independent
		o.session.Messages = []Message{}

		// === Step 4: Run Claude once ===
		o.activity(fmt.Sprintf("Working on %s...", nextFeature.ID))
		err = o.runClaudeInteractive(ctx, prompt)

		if err != nil {
			// Check if it was a cancellation
			if ctx.Err() != nil {
				// Save state for resume - we're in the middle of this feature
				o.saveInterruptedState(currentFeatureID, currentPhase, iteration, config.MaxIterations)
				return ctx.Err()
			}
			// Log the error but continue to next iteration (Claude may have partially succeeded)
			o.typedOutput(OutputError, fmt.Sprintf("Iteration %d error: %v", iteration, err))
		}

		// === Step 5: Short delay before next iteration ===
		// This allows file system to settle and prevents hammering
		if iteration < config.MaxIterations {
			o.activity("Preparing next iteration...")
			select {
			case <-ctx.Done():
				// Save state - we completed the iteration but were interrupted before next
				o.saveInterruptedState(currentFeatureID, currentPhase, iteration+1, config.MaxIterations)
				return ctx.Err()
			case <-time.After(config.DelayBetweenIterations):
				// Continue to next iteration
			}
		}
	}

	// Reached max iterations
	o.typedOutput(OutputInfo, fmt.Sprintf("Reached maximum iterations (%d)", config.MaxIterations))

	// Final check on completion status
	finalPRD, err := prd.LoadFromDir(o.workDir)
	if err == nil {
		stats := finalPRD.Stats()
		o.typedOutput(OutputInfo, fmt.Sprintf("Final status: %d/%d features complete", stats.PassingFeatures, stats.TotalFeatures))
		if finalPRD.IsComplete() {
			// Clear resume state on successful completion
			_ = o.ClearResumeState()
		}
	}

	return nil
}

// saveInterruptedState saves the current state for later resumption
func (o *Orchestrator) saveInterruptedState(featureID string, phase Phase, iteration, maxIterations int) {
	state := &ResumeState{
		CurrentFeature:  featureID,
		Phase:           phase,
		Iteration:       iteration,
		TotalIterations: maxIterations,
	}
	if err := o.SaveResumeState(state); err != nil {
		o.debugLog("Failed to save resume state: %v", err)
	} else {
		o.typedOutput(OutputInfo, fmt.Sprintf("Build state saved. Use --resume to continue from iteration %d.", iteration))
	}
}

// RunFeatureLoop runs the three-phase loop (PLAN -> VALIDATE -> EXECUTE) for a single feature.
// Returns the Claude output from the execution phase.
func (o *Orchestrator) RunFeatureLoop(ctx context.Context, feature *FeatureContext, config *PhaseConfig) (string, error) {
	if config == nil {
		config = &PhaseConfig{MaxValidationAttempts: 3}
	}
	if config.MaxValidationAttempts <= 0 {
		config.MaxValidationAttempts = 3
	}

	var plan string
	var validationFeedback string
	var validationAttempt int

	// Phase loop: PLAN -> VALIDATE -> (loop back or) EXECUTE
	for validationAttempt < config.MaxValidationAttempts {
		validationAttempt++

		// === PLANNING PHASE ===
		o.debugLog("Starting PLANNING phase (attempt %d/%d)", validationAttempt, config.MaxValidationAttempts)
		o.typedOutput(OutputPhase, fmt.Sprintf("Phase: PLANNING (attempt %d/%d)", validationAttempt, config.MaxValidationAttempts))
		o.activity("Planning...")

		planCtx, err := o.BuildIterationContext(validationAttempt, PhasePlanning, feature)
		if err != nil {
			return "", fmt.Errorf("failed to build planning context: %w", err)
		}
		planCtx.ValidationFeedback = validationFeedback
		planCtx.ValidationAttempt = validationAttempt

		planOutput, err := o.runClaudeWithOutput(ctx, planCtx.BuildPrompt())
		if err != nil {
			return "", fmt.Errorf("planning phase failed: %w", err)
		}

		// Extract the plan from the output
		plan = extractPlan(planOutput)
		if plan == "" {
			// If no explicit plan block, use the whole output
			plan = planOutput
		}

		// === VALIDATION PHASE ===
		o.debugLog("Starting VALIDATION phase")
		o.typedOutput(OutputPhase, "Phase: VALIDATING")
		o.activity("Validating plan...")

		validateCtx, err := o.BuildIterationContext(validationAttempt, PhaseValidating, feature)
		if err != nil {
			return "", fmt.Errorf("failed to build validation context: %w", err)
		}
		validateCtx.PreviousPlan = plan

		validationOutput, err := o.runClaudeWithOutput(ctx, validateCtx.BuildPrompt())
		if err != nil {
			return "", fmt.Errorf("validation phase failed: %w", err)
		}

		// Parse the validation result
		validationResult := parseValidation(validationOutput)

		if validationResult.Valid {
			o.debugLog("Plan validated successfully")
			o.typedOutput(OutputSuccess, "Validation: PASSED")
			break
		}

		// Validation failed - prepare feedback for next planning attempt
		validationFeedback = validationResult.Feedback
		if validationFeedback == "" && len(validationResult.Issues) > 0 {
			validationFeedback = "Issues found:\n"
			for _, issue := range validationResult.Issues {
				validationFeedback += "- " + issue + "\n"
			}
		}

		o.debugLog("Validation failed, feedback: %s", validationFeedback)
		o.typedOutput(OutputError, fmt.Sprintf("Validation: FAILED - %d issues", len(validationResult.Issues)))

		if validationAttempt >= config.MaxValidationAttempts {
			return "", fmt.Errorf("validation failed after %d attempts: %s", config.MaxValidationAttempts, validationFeedback)
		}
	}

	// === EXECUTION PHASE ===
	o.debugLog("Starting EXECUTION phase")
	o.typedOutput(OutputPhase, "Phase: EXECUTING")
	o.activity("Executing plan...")

	executeCtx, err := o.BuildIterationContext(validationAttempt, PhaseExecuting, feature)
	if err != nil {
		return "", fmt.Errorf("failed to build execution context: %w", err)
	}
	executeCtx.PreviousPlan = plan

	executionOutput, err := o.runClaudeWithOutput(ctx, executeCtx.BuildPrompt())
	if err != nil {
		return "", fmt.Errorf("execution phase failed: %w", err)
	}

	o.typedOutput(OutputSuccess, "Phase: COMPLETE")
	o.activity("Complete")
	return executionOutput, nil
}

// runClaudeWithOutput runs Claude and returns the output as a string
func (o *Orchestrator) runClaudeWithOutput(ctx context.Context, prompt string) (string, error) {
	o.debugLog("Running Claude with prompt (%d chars)", len(prompt))

	// Build the allowed tools flag from configuration
	allowedTools := o.toolConfig.BuildAllowedToolsFlag()
	o.debugLog("Using allowed tools: %s", allowedTools)

	cmd := exec.CommandContext(ctx, o.claudePath,
		"-p", prompt,
		"--permission-mode", "acceptEdits",
		"--allowedTools", allowedTools,
		"--output-format", "stream-json",
		"--verbose",
	)
	cmd.Dir = o.workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start claude: %w", err)
	}

	startTime := time.Now()

	// Read stderr in background
	var stderrBuf strings.Builder
	go func() {
		io.Copy(&stderrBuf, stderr)
	}()

	// Collect output
	var outputBuilder strings.Builder
	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			o.output(line)
			continue
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "assistant":
			if msg, ok := event["message"].(map[string]any); ok {
				if content, ok := msg["content"].([]any); ok {
					for _, block := range content {
						if blockMap, ok := block.(map[string]any); ok {
							blockType, _ := blockMap["type"].(string)
							switch blockType {
							case "text":
								if text, ok := blockMap["text"].(string); ok {
									o.typedOutput(OutputText, text)
									outputBuilder.WriteString(text)
									outputBuilder.WriteString("\n")
								}
							case "tool_use":
								if name, ok := blockMap["name"].(string); ok {
									o.typedOutput(OutputToolUse, fmt.Sprintf("Using tool: %s", name))
									if input, ok := blockMap["input"].(map[string]any); ok {
										if cmdStr, ok := input["command"].(string); ok {
											o.typedOutput(OutputToolInput, "> "+cmdStr)
											o.activity(fmt.Sprintf("Running: %s", truncateString(cmdStr, 50)))
										}
										if path, ok := input["file_path"].(string); ok {
											o.typedOutput(OutputToolInput, "> "+path)
											o.activity(fmt.Sprintf("%s: %s", name, path))
										}
										if filePath, ok := input["filePath"].(string); ok {
											o.typedOutput(OutputToolInput, "> "+filePath)
											o.activity(fmt.Sprintf("%s: %s", name, filePath))
										}
									}
								}
							}
						}
					}
				}
			}

		case "user":
			if msg, ok := event["message"].(map[string]any); ok {
				if content, ok := msg["content"].([]any); ok {
					for _, block := range content {
						if blockMap, ok := block.(map[string]any); ok {
							if blockMap["type"] == "tool_result" {
								if contentStr, ok := blockMap["content"].(string); ok {
									lines := strings.Split(contentStr, "\n")
									if len(lines) > 5 {
										for _, l := range lines[:5] {
											o.typedOutput(OutputToolResult, "  "+l)
										}
										o.typedOutput(OutputToolResult, fmt.Sprintf("  ... (%d more lines)", len(lines)-5))
									} else {
										for _, l := range lines {
											o.typedOutput(OutputToolResult, "  "+l)
										}
									}
								}
							}
						}
					}
				}
			}

		case "result":
			elapsed := time.Since(startTime).Seconds()
			subtype, _ := event["subtype"].(string)

			if result, ok := event["result"].(string); ok && result != "" {
				o.typedOutput(OutputText, result)
				outputBuilder.WriteString(result)
			}

			// Build stats message
			statsMsg := fmt.Sprintf("%s: %.1fs", subtype, elapsed)
			if cost, ok := event["total_cost_usd"].(float64); ok {
				statsMsg += fmt.Sprintf(", $%.4f", cost)
			}
			o.typedOutput(OutputInfo, statsMsg)

		case "error":
			if errData, ok := event["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok {
					o.typedOutput(OutputError, "Claude error: "+msg)
					return "", fmt.Errorf("claude error: %s", msg)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading claude output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if stderrBuf.Len() > 0 {
			o.debugLog("Claude stderr: %s", stderrBuf.String())
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 {
				o.debugLog("Claude exited with code %d", exitErr.ExitCode())
			}
		}
	}

	return outputBuilder.String(), nil
}

// extractPlan extracts the plan from Claude's output (between <plan> tags)
func extractPlan(output string) string {
	startTag := "<plan>"
	endTag := "</plan>"

	startIdx := strings.Index(output, startTag)
	if startIdx == -1 {
		return ""
	}

	endIdx := strings.Index(output[startIdx:], endTag)
	if endIdx == -1 {
		return ""
	}

	return strings.TrimSpace(output[startIdx+len(startTag) : startIdx+endIdx])
}

// parseValidation parses the validation result from Claude's output
func parseValidation(output string) ValidationResult {
	result := ValidationResult{Valid: true} // Default to valid if parsing fails

	startTag := "<validation>"
	endTag := "</validation>"

	startIdx := strings.Index(output, startTag)
	if startIdx == -1 {
		// No validation block found - assume valid
		return result
	}

	endIdx := strings.Index(output[startIdx:], endTag)
	if endIdx == -1 {
		return result
	}

	validationText := output[startIdx+len(startTag) : startIdx+endIdx]

	// Parse the validation text
	lines := strings.Split(validationText, "\n")
	var issues []string
	var feedbackLines []string
	inFeedback := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "valid:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "valid:"))
			result.Valid = strings.EqualFold(value, "true")
		} else if strings.HasPrefix(line, "issues:") {
			// Issues section started
			continue
		} else if strings.HasPrefix(line, "- ") && !inFeedback {
			// An issue item
			issues = append(issues, strings.TrimPrefix(line, "- "))
		} else if strings.HasPrefix(line, "feedback:") {
			inFeedback = true
			feedback := strings.TrimSpace(strings.TrimPrefix(line, "feedback:"))
			if feedback != "" {
				feedbackLines = append(feedbackLines, feedback)
			}
		} else if inFeedback {
			feedbackLines = append(feedbackLines, line)
		}
	}

	result.Issues = issues
	result.Feedback = strings.Join(feedbackLines, "\n")

	return result
}

// runClaudeInteractive runs Claude in interactive mode, streaming output
func (o *Orchestrator) runClaudeInteractive(ctx context.Context, prompt string) error {
	o.debugLog("Starting Claude with prompt (%d chars)", len(prompt))

	// Build the allowed tools flag from configuration
	// - stream-json requires --verbose flag
	// - allowedTools grants permission for specific tools without prompting
	// - By default we allow: Read, Write, Edit, Bash for common dev commands (go, npm, git, etc.)
	allowedTools := o.toolConfig.BuildAllowedToolsFlag()
	o.debugLog("Using allowed tools: %s", allowedTools)

	cmd := exec.CommandContext(ctx, o.claudePath,
		"-p", prompt,
		"--permission-mode", "acceptEdits",
		"--allowedTools", allowedTools,
		"--output-format", "stream-json",
		"--verbose",
	)
	cmd.Dir = o.workDir

	// Get pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude: %w", err)
	}

	startTime := time.Now()
	o.typedOutput(OutputInfo, "Claude is working...")
	o.activity("Starting Claude...")
	o.step(StepReading) // Start with reading step

	// Read stderr in background
	var stderrBuf strings.Builder
	go func() {
		io.Copy(&stderrBuf, stderr)
	}()

	// Process streaming JSON output
	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Not JSON, might be plain text - show it
			o.output(line)
			continue
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "system":
			// System init message - show in debug
			if subtype, ok := event["subtype"].(string); ok {
				o.debugLog("System: %s", subtype)
			}

		case "assistant":
			// Assistant message with content
			if msg, ok := event["message"].(map[string]any); ok {
				if content, ok := msg["content"].([]any); ok {
					for _, block := range content {
						if blockMap, ok := block.(map[string]any); ok {
							blockType, _ := blockMap["type"].(string)
							switch blockType {
							case "text":
								if text, ok := blockMap["text"].(string); ok {
									o.typedOutput(OutputText, text)
								}
							case "tool_use":
								if name, ok := blockMap["name"].(string); ok {
									o.typedOutput(OutputToolUse, fmt.Sprintf("Using tool: %s", name))
									if input, ok := blockMap["input"].(map[string]any); ok {
										// Show some context about the tool use and update activity
										if cmdStr, ok := input["command"].(string); ok {
											o.typedOutput(OutputToolInput, "> "+cmdStr)
											o.activity(fmt.Sprintf("Running: %s", truncateString(cmdStr, 50)))
											// Detect step from command
											o.detectStepFromCommand(cmdStr)
										}
										if path, ok := input["file_path"].(string); ok {
											o.typedOutput(OutputToolInput, "> "+path)
											o.activity(fmt.Sprintf("%s: %s", name, path))
											// Detect step from file operation
											o.detectStepFromFileOp(name, path)
										}
										if filePath, ok := input["filePath"].(string); ok {
											o.typedOutput(OutputToolInput, "> "+filePath)
											o.activity(fmt.Sprintf("%s: %s", name, filePath))
											// Detect step from file operation
											o.detectStepFromFileOp(name, filePath)
										}
									}
								}
							}
						}
					}
				}
			}

		case "user":
			// Tool results coming back
			if msg, ok := event["message"].(map[string]any); ok {
				if content, ok := msg["content"].([]any); ok {
					for _, block := range content {
						if blockMap, ok := block.(map[string]any); ok {
							if blockMap["type"] == "tool_result" {
								// Show truncated tool result
								if contentStr, ok := blockMap["content"].(string); ok {
									lines := strings.Split(contentStr, "\n")
									if len(lines) > 5 {
										for _, l := range lines[:5] {
											o.typedOutput(OutputToolResult, "  "+l)
										}
										o.typedOutput(OutputToolResult, fmt.Sprintf("  ... (%d more lines)", len(lines)-5))
									} else {
										for _, l := range lines {
											o.typedOutput(OutputToolResult, "  "+l)
										}
									}
								}
							}
						}
					}
				}
			}

		case "result":
			// Final result
			elapsed := time.Since(startTime).Seconds()
			subtype, _ := event["subtype"].(string)

			if result, ok := event["result"].(string); ok && result != "" {
				o.typedOutput(OutputText, result)
			}

			// Build stats message
			statsMsg := fmt.Sprintf("%s: %.1fs", subtype, elapsed)
			if cost, ok := event["total_cost_usd"].(float64); ok {
				statsMsg += fmt.Sprintf(", $%.4f", cost)
			}
			o.typedOutput(OutputInfo, statsMsg)

		case "error":
			// Error occurred
			if errData, ok := event["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok {
					o.typedOutput(OutputError, "Claude error: "+msg)
					return fmt.Errorf("claude error: %s", msg)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading claude output: %w", err)
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		// Check if we have stderr output
		if stderrBuf.Len() > 0 {
			o.debugLog("Claude stderr: %s", stderrBuf.String())
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 0 is success, others might still be ok if we got output
			if exitErr.ExitCode() != 0 {
				o.debugLog("Claude exited with code %d", exitErr.ExitCode())
			}
		}
	}

	elapsed := time.Since(startTime).Seconds()
	o.typedOutput(OutputSuccess, fmt.Sprintf("Complete: %.1fs", elapsed))
	o.activity("Complete")
	o.step(StepComplete)

	return nil
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// debugLog logs a debug message
func (o *Orchestrator) debugLog(format string, args ...any) {
	if o.debug && o.onDebug != nil {
		o.onDebug(fmt.Sprintf(format, args...))
	}
}

// output sends a plain text message through the callback
func (o *Orchestrator) output(content string) {
	if o.onTypedOutput != nil {
		o.onTypedOutput(OutputText, content)
	} else if o.onOutput != nil {
		o.onOutput(content)
	}
}

// typedOutput sends a typed/colored message through the callback
func (o *Orchestrator) typedOutput(outputType OutputType, content string) {
	if o.onTypedOutput != nil {
		o.onTypedOutput(outputType, content)
	} else if o.onOutput != nil {
		o.onOutput(content)
	}
}

// activity updates the current activity display
func (o *Orchestrator) activity(activity string) {
	if o.onActivity != nil {
		o.onActivity(activity)
	}
}

// step updates the current step
func (o *Orchestrator) step(s Step) {
	if o.onStep != nil {
		o.onStep(s)
	}
}

// detectStepFromCommand detects the current step based on a bash command
func (o *Orchestrator) detectStepFromCommand(cmd string) {
	cmdLower := strings.ToLower(cmd)

	// Test commands
	if strings.Contains(cmdLower, "test") ||
		strings.Contains(cmdLower, "pytest") ||
		strings.Contains(cmdLower, "jest") ||
		strings.Contains(cmdLower, "mocha") ||
		strings.Contains(cmdLower, "cargo test") ||
		strings.HasPrefix(cmdLower, "go test") {
		o.step(StepTesting)
		return
	}

	// Git commands
	if strings.HasPrefix(cmdLower, "git commit") ||
		strings.HasPrefix(cmdLower, "git add") && strings.Contains(cmdLower, "-m") {
		o.step(StepCommitting)
		return
	}

	if strings.HasPrefix(cmdLower, "git ") {
		// Other git commands like status, diff, log are usually part of reading
		o.step(StepReading)
		return
	}

	// Build commands might be part of testing
	if strings.HasPrefix(cmdLower, "go build") ||
		strings.HasPrefix(cmdLower, "npm run build") ||
		strings.HasPrefix(cmdLower, "make") {
		o.step(StepTesting)
		return
	}
}

// detectStepFromFileOp detects the current step based on a file operation
func (o *Orchestrator) detectStepFromFileOp(toolName, filePath string) {
	filePathLower := strings.ToLower(filePath)
	toolNameLower := strings.ToLower(toolName)

	// Check if it's a read operation
	if toolNameLower == "read" {
		o.step(StepReading)
		return
	}

	// Write/Edit operations
	if toolNameLower == "write" || toolNameLower == "edit" {
		// Check for PRD/progress file updates
		if strings.Contains(filePathLower, "prd.json") ||
			strings.Contains(filePathLower, "progress.txt") {
			o.step(StepUpdating)
			return
		}

		// Check for test files
		if strings.Contains(filePathLower, "_test.") ||
			strings.Contains(filePathLower, ".test.") ||
			strings.Contains(filePathLower, "/test/") ||
			strings.Contains(filePathLower, "/tests/") ||
			strings.Contains(filePathLower, ".spec.") {
			// Writing tests is usually part of coding
			o.step(StepCoding)
			return
		}

		// Any other file write is coding
		o.step(StepCoding)
		return
	}
}

// BuildIterationContext creates a fresh, self-contained context for a single iteration.
// This ensures each Claude call gets clean context with no conversation history accumulation.
func (o *Orchestrator) BuildIterationContext(iteration int, phase Phase, feature *FeatureContext) (*IterationContext, error) {
	ctx := &IterationContext{
		Iteration:   iteration,
		Phase:       phase,
		TaggedFiles: make(map[string]string),
		KeyFiles:    make(map[string]string),
	}

	// Read prd.json
	prdContent, err := os.ReadFile(filepath.Join(o.workDir, "prd.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read prd.json: %w", err)
	}
	ctx.PRDContent = string(prdContent)

	// Read progress.txt if exists
	progressContent, err := os.ReadFile(filepath.Join(o.workDir, "progress.txt"))
	if err == nil {
		ctx.ProgressContent = string(progressContent)
	}

	// Generate directory tree with configurable depth
	maxDepth := o.snapshotConfig.MaxTreeDepth
	if maxDepth <= 0 {
		maxDepth = 4 // default
	}
	tree, err := o.generateDirectoryTree(maxDepth)
	if err == nil {
		ctx.DirectoryTree = tree
	}

	// Detect and load key files if enabled
	if o.snapshotConfig.IncludeKeyFiles {
		ctx.KeyFiles = o.detectAndLoadKeyFiles()
	}

	// Set current feature context if provided
	if feature != nil {
		ctx.CurrentFeature = feature
	}

	return ctx, nil
}

// generateDirectoryTree creates a textual representation of the directory structure
func (o *Orchestrator) generateDirectoryTree(maxDepth int) (string, error) {
	var sb strings.Builder
	err := o.walkDir(o.workDir, "", 0, maxDepth, &sb)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

// walkDir recursively walks the directory tree
func (o *Orchestrator) walkDir(path, prefix string, depth, maxDepth int, sb *strings.Builder) error {
	if depth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Filter out common ignored directories/files
	var filtered []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files, common build/dependency directories
		if strings.HasPrefix(name, ".") && name != ".gitignore" {
			continue
		}
		if name == "node_modules" || name == "vendor" || name == "__pycache__" ||
			name == "target" || name == "build" || name == "dist" {
			continue
		}
		filtered = append(filtered, entry)
	}

	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		sb.WriteString(prefix + connector + entry.Name())
		if entry.IsDir() {
			sb.WriteString("/")
		}
		sb.WriteString("\n")

		if entry.IsDir() {
			newPrefix := prefix + "│   "
			if isLast {
				newPrefix = prefix + "    "
			}
			_ = o.walkDir(filepath.Join(path, entry.Name()), newPrefix, depth+1, maxDepth, sb)
		}
	}

	return nil
}

// keyFilePatterns defines patterns for automatically detected key files
var keyFilePatterns = []string{
	// Package managers / dependency files
	"go.mod",
	"go.sum",
	"package.json",
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"Cargo.toml",
	"Cargo.lock",
	"requirements.txt",
	"pyproject.toml",
	"Pipfile",
	"Gemfile",
	"composer.json",

	// Documentation
	"README.md",
	"README",
	"README.txt",
	"CONTRIBUTING.md",
	"CHANGELOG.md",

	// Configuration
	"Makefile",
	"Dockerfile",
	"docker-compose.yml",
	"docker-compose.yaml",
	".env.example",
	"tsconfig.json",
	"webpack.config.js",
	"vite.config.js",
	"vite.config.ts",
	".eslintrc.json",
	".prettierrc",

	// CI/CD
	".github/workflows/*.yml",
	".github/workflows/*.yaml",
	".gitlab-ci.yml",
}

// mainEntryPatterns defines patterns for main entry point files
var mainEntryPatterns = []string{
	"main.go",
	"cmd/*/main.go",
	"src/main.go",
	"src/index.ts",
	"src/index.js",
	"index.ts",
	"index.js",
	"app.py",
	"main.py",
	"src/main.rs",
	"src/lib.rs",
}

// detectAndLoadKeyFiles finds and loads key project files
func (o *Orchestrator) detectAndLoadKeyFiles() map[string]string {
	keyFiles := make(map[string]string)

	maxSize := o.snapshotConfig.MaxFileSizeBytes
	if maxSize <= 0 {
		maxSize = 50 * 1024 // 50KB default
	}

	// Helper to add a file if it exists and is within size limits
	addFile := func(relPath string) {
		fullPath := filepath.Join(o.workDir, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			return // File doesn't exist
		}

		if info.IsDir() {
			return // Skip directories
		}

		if info.Size() > maxSize {
			// File too large - add truncation note
			keyFiles[relPath] = fmt.Sprintf("[File too large: %d bytes, max %d bytes]", info.Size(), maxSize)
			return
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			return
		}
		keyFiles[relPath] = string(content)
	}

	// Helper to resolve glob patterns
	resolveGlob := func(pattern string) []string {
		matches, err := filepath.Glob(filepath.Join(o.workDir, pattern))
		if err != nil {
			return nil
		}
		var relPaths []string
		for _, match := range matches {
			relPath, err := filepath.Rel(o.workDir, match)
			if err != nil {
				continue
			}
			relPaths = append(relPaths, relPath)
		}
		return relPaths
	}

	// Check standard key file patterns
	for _, pattern := range keyFilePatterns {
		if strings.Contains(pattern, "*") {
			// Glob pattern
			for _, match := range resolveGlob(pattern) {
				addFile(match)
			}
		} else {
			// Exact file
			addFile(pattern)
		}
	}

	// Check main entry point patterns
	for _, pattern := range mainEntryPatterns {
		if strings.Contains(pattern, "*") {
			// Glob pattern
			for _, match := range resolveGlob(pattern) {
				addFile(match)
			}
		} else {
			// Exact file
			addFile(pattern)
		}
	}

	return keyFiles
}

// AddTaggedFile adds a file's contents to the iteration context
func (o *Orchestrator) AddTaggedFile(ctx *IterationContext, filePath string) error {
	fullPath := filePath
	if !filepath.IsAbs(filePath) {
		fullPath = filepath.Join(o.workDir, filePath)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Store with relative path as key
	relPath, err := filepath.Rel(o.workDir, fullPath)
	if err != nil {
		relPath = filePath
	}
	ctx.TaggedFiles[relPath] = string(content)
	return nil
}

// AddTaggedFilesFromTags processes tag strings and adds matching files to the context
// Tags can be:
// - @filepath - exact file path
// - @glob/pattern/**/*.go - glob pattern
// - @!dirname - exclusion pattern
func (o *Orchestrator) AddTaggedFilesFromTags(ctx *IterationContext, tagStrings []string) error {
	tags, err := o.tagger.ResolveTags(tagStrings)
	if err != nil {
		return fmt.Errorf("failed to resolve tags: %w", err)
	}

	filesMap, err := o.tagger.BuildTaggedFilesMap(tags)
	if err != nil {
		return fmt.Errorf("failed to build tagged files map: %w", err)
	}

	// Merge into the context's TaggedFiles
	for relPath, content := range filesMap {
		ctx.TaggedFiles[relPath] = content
	}

	return nil
}

// GetTagger returns the orchestrator's tagger for external use
func (o *Orchestrator) GetTagger() *tagging.Tagger {
	return o.tagger
}

// GetParallelExecutor returns the orchestrator's parallel executor for external use
func (o *Orchestrator) GetParallelExecutor() *ParallelExecutor {
	return o.parallel
}

// SetParallelLimits sets the concurrency limits for parallel execution
func (o *Orchestrator) SetParallelLimits(limits ParallelLimits) *Orchestrator {
	o.parallel.SetLimits(limits)
	return o
}

// ExecuteParallel executes a group of actions in parallel
func (o *Orchestrator) ExecuteParallel(ctx context.Context, actions []SubAction) ParallelResult {
	return o.parallel.Execute(ctx, ParallelAction{Actions: actions})
}

// ListFilesForAutocomplete returns a list of files in the working directory for autocomplete
func (o *Orchestrator) ListFilesForAutocomplete(maxDepth int) ([]string, error) {
	return o.tagger.ListFiles(maxDepth)
}

// DefaultPromptUser provides a simple terminal-based user prompt
func DefaultPromptUser(question string) (string, error) {
	fmt.Println()
	fmt.Println(question)
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(answer), nil
}
