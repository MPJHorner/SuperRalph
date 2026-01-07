package orchestrator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Orchestrator manages the agent loop
type Orchestrator struct {
	workDir    string
	claudePath string
	session    *Session
	debug      bool

	// Callbacks for UI integration
	onMessage  func(role, content string)
	onAction   func(action Action, params ActionParams)
	onState    func(state any)
	onThinking func(thinking string)
	onDebug    func(msg string)
	promptUser func(question string) (string, error)
}

// New creates a new Orchestrator
func New(workDir string) *Orchestrator {
	return &Orchestrator{
		workDir:    workDir,
		claudePath: findClaudeBinary(),
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

// SetPromptUser sets the function to prompt the user
func (o *Orchestrator) SetPromptUser(fn func(question string) (string, error)) *Orchestrator {
	o.promptUser = fn
	return o
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

	systemPrompt := o.buildPlanSystemPrompt()
	o.session.Messages = append(o.session.Messages, Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// Start with initial prompt to Claude
	initialPrompt := "Start the planning session. Ask the user what they want to build."
	return o.runLoop(ctx, initialPrompt)
}

// RunBuild runs the build loop
func (o *Orchestrator) RunBuild(ctx context.Context) error {
	o.session.Mode = "build"
	o.session.State = &BuildState{Phase: "reading", Iteration: 1}

	// Build fresh context for this iteration
	context := o.buildFreshContext()

	// Start with the build prompt
	return o.runBuildLoop(ctx, context)
}

// buildFreshContext creates a clean context for each iteration (no history accumulation)
func (o *Orchestrator) buildFreshContext() string {
	var ctx strings.Builder

	// Read prd.json
	prdContent, err := os.ReadFile(filepath.Join(o.workDir, "prd.json"))
	if err == nil {
		ctx.WriteString("## prd.json\n```json\n")
		ctx.WriteString(string(prdContent))
		ctx.WriteString("\n```\n\n")
	}

	// Read progress.txt if exists
	progressContent, err := os.ReadFile(filepath.Join(o.workDir, "progress.txt"))
	if err == nil {
		ctx.WriteString("## progress.txt\n```\n")
		ctx.WriteString(string(progressContent))
		ctx.WriteString("\n```\n\n")
	}

	// Add directory tree (simple version)
	ctx.WriteString("## Directory Structure\n```\n")
	tree := o.getDirectoryTree(o.workDir, "", 3)
	ctx.WriteString(tree)
	ctx.WriteString("```\n\n")

	return ctx.String()
}

// getDirectoryTree returns a simple directory tree
func (o *Orchestrator) getDirectoryTree(dir string, prefix string, maxDepth int) string {
	if maxDepth <= 0 {
		return ""
	}

	var result strings.Builder
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	// Filter out common ignored directories
	ignored := map[string]bool{
		".git": true, "node_modules": true, "vendor": true,
		".idea": true, ".vscode": true, "__pycache__": true,
		"build": true, "dist": true, ".superralph": true,
	}

	for i, entry := range entries {
		if ignored[entry.Name()] || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		isLast := i == len(entries)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		result.WriteString(prefix + connector + entry.Name())
		if entry.IsDir() {
			result.WriteString("/")
		}
		result.WriteString("\n")

		if entry.IsDir() {
			newPrefix := prefix + "│   "
			if isLast {
				newPrefix = prefix + "    "
			}
			result.WriteString(o.getDirectoryTree(filepath.Join(dir, entry.Name()), newPrefix, maxDepth-1))
		}
	}

	return result.String()
}

// runBuildLoop runs the build loop with fresh context each iteration
func (o *Orchestrator) runBuildLoop(ctx context.Context, freshContext string) error {
	maxIterations := 100

	for i := 0; i < maxIterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		o.debugLog("=== Iteration %d ===", i+1)
		o.debugLog("Calling Claude...")

		// Build the prompt with fresh context each time
		prompt := o.buildBuildPrompt(freshContext, i+1)
		o.debugLog("Prompt length: %d chars", len(prompt))

		// Call Claude
		response, err := o.callClaudeWithRetry(ctx, prompt)
		if err != nil {
			return fmt.Errorf("claude call failed: %w", err)
		}

		// Handle thinking
		if o.debug && response.Thinking != "" && o.onThinking != nil {
			o.onThinking(response.Thinking)
		}

		// Handle message
		if response.Message != "" && o.onMessage != nil {
			o.onMessage("assistant", response.Message)
		}

		// Handle action
		if o.onAction != nil {
			o.onAction(response.Action, response.ActionParams)
		}

		// Execute action
		result, done, err := o.executeAction(ctx, response.Action, response.ActionParams)
		if err != nil {
			return fmt.Errorf("action failed: %w", err)
		}

		o.debugLog("Action: %s, Result length: %d", response.Action, len(result))

		if done {
			return nil
		}

		// Rebuild fresh context for next iteration (includes any file changes)
		freshContext = o.buildFreshContext()

		// Add the result of the last action to context
		if result != "" {
			freshContext += "\n## Last Action Result\n" + result + "\n"
		}
	}

	return fmt.Errorf("max iterations reached")
}

// runLoop is the main agent loop (used for planning)
func (o *Orchestrator) runLoop(ctx context.Context, initialPrompt string) error {
	o.session.Messages = append(o.session.Messages, Message{
		Role:    "user",
		Content: initialPrompt,
	})

	maxIterations := 100
	for i := 0; i < maxIterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Build prompt from session messages
		var promptBuilder strings.Builder
		for _, msg := range o.session.Messages {
			promptBuilder.WriteString(fmt.Sprintf("[%s]\n%s\n\n", msg.Role, msg.Content))
		}

		response, err := o.callClaudeWithRetry(ctx, promptBuilder.String())
		if err != nil {
			return fmt.Errorf("claude call failed: %w", err)
		}

		if o.debug && response.Thinking != "" && o.onThinking != nil {
			o.onThinking(response.Thinking)
		}

		if response.Message != "" && o.onMessage != nil {
			o.onMessage("assistant", response.Message)
		}

		o.session.Messages = append(o.session.Messages, Message{
			Role:    "assistant",
			Content: response.Message,
		})

		if o.onAction != nil {
			o.onAction(response.Action, response.ActionParams)
		}

		result, done, err := o.executeAction(ctx, response.Action, response.ActionParams)
		if err != nil {
			return fmt.Errorf("action failed: %w", err)
		}

		if done {
			_ = o.SaveSession()
			return nil
		}

		if result != "" {
			o.session.Messages = append(o.session.Messages, Message{
				Role:    "user",
				Content: result,
			})
		}

		_ = o.SaveSession()
	}

	return fmt.Errorf("max iterations reached")
}

// callClaudeWithRetry calls Claude and retries with Haiku repair if JSON is broken
func (o *Orchestrator) callClaudeWithRetry(ctx context.Context, prompt string) (*Response, error) {
	// First attempt with main model
	rawResponse, err := o.callClaudeRaw(ctx, prompt)
	if err != nil {
		return nil, err
	}

	o.debugLog("Raw response length: %d", len(rawResponse))

	// Try to parse the response
	response, err := o.parseResponse(rawResponse)
	if err == nil {
		return response, nil
	}

	o.debugLog("JSON parse failed, attempting Haiku repair: %v", err)

	// Try to repair with Haiku
	repaired, repairErr := o.repairJSONWithHaiku(ctx, rawResponse)
	if repairErr != nil {
		// Return original error if repair fails
		return nil, fmt.Errorf("failed to parse response and repair failed: %w (repair error: %v)", err, repairErr)
	}

	// Try parsing the repaired JSON
	response, err = o.parseResponse(repaired)
	if err != nil {
		return nil, fmt.Errorf("repair succeeded but still invalid JSON: %w", err)
	}

	o.debugLog("Haiku repair succeeded")
	return response, nil
}

// callClaudeRaw calls Claude and returns the raw text response
func (o *Orchestrator) callClaudeRaw(ctx context.Context, prompt string) (string, error) {
	o.debugLog("Executing Claude CLI (prompt: %d chars)...", len(prompt))

	// Print waiting message to stdout so user knows we're working
	fmt.Print("  Waiting for Claude")

	cmd := exec.CommandContext(ctx, o.claudePath,
		"-p", prompt,
		"--output-format", "text",
	)
	cmd.Dir = o.workDir

	// Run with a spinner effect
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Print(".")
			}
		}
	}()

	output, err := cmd.Output()
	close(done)
	fmt.Println() // newline after dots

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude exited with error: %s", string(exitErr.Stderr))
		}
		return "", err
	}

	o.debugLog("Claude returned %d bytes", len(output))
	return string(output), nil
}

// parseResponse tries to parse a Response from Claude's output
func (o *Orchestrator) parseResponse(text string) (*Response, error) {
	// Try to extract JSON from the text
	jsonStr := extractJSON(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var response Response
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate required fields
	if response.Action == "" {
		return nil, fmt.Errorf("missing required field: action")
	}

	return &response, nil
}

// repairJSONWithHaiku uses Haiku to fix broken JSON
func (o *Orchestrator) repairJSONWithHaiku(ctx context.Context, brokenResponse string) (string, error) {
	repairPrompt := fmt.Sprintf(`You are a JSON repair assistant. Fix the following text to be valid JSON matching this schema:

{
  "thinking": "string (optional)",
  "action": "read_files" | "write_file" | "run_command" | "done",
  "action_params": {
    "paths": ["array of file paths"] (for read_files),
    "path": "file path" (for write_file),
    "content": "file content" (for write_file),
    "command": "shell command" (for run_command)
  },
  "message": "string (optional)"
}

Respond with ONLY the fixed JSON, no explanation, no markdown code blocks, just raw JSON.

Text to fix:
%s`, brokenResponse)

	cmd := exec.CommandContext(ctx, o.claudePath,
		"-p", repairPrompt,
		"--output-format", "text",
		"--model", "haiku",
	)
	cmd.Dir = o.workDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("haiku repair call failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// extractJSON tries to extract JSON from a string that might contain markdown
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// If it starts with {, try to parse directly
	if strings.HasPrefix(s, "{") {
		return extractJSONObject(s)
	}

	// Try to find JSON in code blocks
	if start := strings.Index(s, "```json"); start != -1 {
		start += 7
		if end := strings.Index(s[start:], "```"); end != -1 {
			return strings.TrimSpace(s[start : start+end])
		}
	}

	if start := strings.Index(s, "```"); start != -1 {
		start += 3
		// Skip any language identifier on the same line
		if newline := strings.Index(s[start:], "\n"); newline != -1 {
			start += newline + 1
		}
		if end := strings.Index(s[start:], "```"); end != -1 {
			candidate := strings.TrimSpace(s[start : start+end])
			if strings.HasPrefix(candidate, "{") {
				return candidate
			}
		}
	}

	// Try to find raw JSON object
	if start := strings.Index(s, "{"); start != -1 {
		return extractJSONObject(s[start:])
	}

	return ""
}

// extractJSONObject extracts a JSON object by matching braces
func extractJSONObject(s string) string {
	if !strings.HasPrefix(s, "{") {
		return ""
	}

	depth := 0
	inString := false
	escape := false

	for i, c := range s {
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inString {
			escape = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return s[:i+1]
			}
		}
	}

	return ""
}

// executeAction executes the requested action
func (o *Orchestrator) executeAction(ctx context.Context, action Action, params ActionParams) (result string, done bool, err error) {
	switch action {
	case ActionAskUser:
		if o.promptUser == nil {
			return "", false, fmt.Errorf("no prompt user function set")
		}
		answer, err := o.promptUser(params.Question)
		if err != nil {
			return "", false, err
		}
		return fmt.Sprintf("User response: %s", answer), false, nil

	case ActionReadFiles:
		var contents strings.Builder
		for _, path := range params.Paths {
			fullPath := filepath.Join(o.workDir, path)
			data, err := os.ReadFile(fullPath)
			if err != nil {
				contents.WriteString(fmt.Sprintf("File %s: ERROR: %s\n\n", path, err.Error()))
			} else {
				contents.WriteString(fmt.Sprintf("File %s:\n```\n%s\n```\n\n", path, string(data)))
			}
		}
		return contents.String(), false, nil

	case ActionWriteFile:
		fullPath := filepath.Join(o.workDir, params.Path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Sprintf("Failed to create directory: %s", err), false, nil
		}
		if err := os.WriteFile(fullPath, []byte(params.Content), 0644); err != nil {
			return fmt.Sprintf("Failed to write file: %s", err), false, nil
		}
		return fmt.Sprintf("Successfully wrote file: %s", params.Path), false, nil

	case ActionRunCommand:
		cmd := exec.CommandContext(ctx, "sh", "-c", params.Command)
		cmd.Dir = o.workDir
		output, err := cmd.CombinedOutput()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		return fmt.Sprintf("Command: %s\nExit code: %d\nOutput:\n%s", params.Command, exitCode, string(output)), false, nil

	case ActionDone:
		return "", true, nil

	default:
		return "", false, fmt.Errorf("unknown action: %s", action)
	}
}

// debugLog logs a debug message
func (o *Orchestrator) debugLog(format string, args ...any) {
	if o.debug && o.onDebug != nil {
		o.onDebug(fmt.Sprintf(format, args...))
	}
}

// buildBuildPrompt creates the prompt for a build iteration
func (o *Orchestrator) buildBuildPrompt(context string, iteration int) string {
	return fmt.Sprintf(`You are a code implementation agent. Your task is to implement features from the PRD.

## CRITICAL: Response Format

You MUST respond with ONLY a valid JSON object. No explanations, no markdown, just JSON:

{
  "thinking": "your reasoning here",
  "action": "read_files",
  "action_params": {
    "paths": ["file1.go", "file2.go"]
  },
  "message": "Status message for user"
}

Valid actions:
- "read_files" with "paths": ["file1", "file2"] - Read files to understand code
- "write_file" with "path": "file.go" and "content": "..." - Write/update a file
- "run_command" with "command": "go test ./..." - Run a shell command
- "done" - All features are complete

## Current Context

%s

## Iteration: %d

## Your Task

1. Look at prd.json - find the highest priority feature with "passes": false
2. Implement that ONE feature
3. Run tests to verify (test command is in prd.json)
4. If tests pass, commit and update prd.json to mark the feature as passes: true
5. If all features pass, use action "done"

## Rules

- TESTS MUST PASS before committing
- Work on ONE feature at a time
- Make small, incremental changes
- Always run tests after changes

Respond with JSON only:`, context, iteration)
}

// buildPlanSystemPrompt builds the system prompt for planning
func (o *Orchestrator) buildPlanSystemPrompt() string {
	return `You are a PRD planning assistant. Help create a prd.json file.

## CRITICAL: Response Format

You MUST respond with ONLY a valid JSON object:

{
  "thinking": "your reasoning",
  "action": "ask_user",
  "action_params": {
    "question": "What are you building?"
  },
  "message": "Let's plan your project."
}

Valid actions:
- "ask_user" with "question": "..." - Ask the user something
- "read_files" with "paths": [...] - Read existing code
- "write_file" with "path" and "content" - Write prd.json
- "done" - Planning complete

Respond with JSON only.`
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
