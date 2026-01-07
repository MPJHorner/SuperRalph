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

	systemPrompt := o.buildBuildSystemPrompt()
	o.session.Messages = append(o.session.Messages, Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// Start with initial prompt
	initialPrompt := "Read the prd.json and progress.txt to understand the current state, then begin implementing."
	return o.runLoop(ctx, initialPrompt)
}

// runLoop is the main agent loop
func (o *Orchestrator) runLoop(ctx context.Context, initialPrompt string) error {
	// Add initial user message to kick things off
	o.session.Messages = append(o.session.Messages, Message{
		Role:    "user",
		Content: initialPrompt,
	})

	maxIterations := 100 // Safety limit
	for i := 0; i < maxIterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Call Claude
		response, err := o.callClaude(ctx)
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

		// Add assistant message to history
		o.session.Messages = append(o.session.Messages, Message{
			Role:    "assistant",
			Content: response.Message,
		})

		// Handle state update
		if response.State != nil && o.onState != nil {
			o.onState(response.State)
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

		if done {
			// Save final session
			_ = o.SaveSession()
			return nil
		}

		// Add result as user message for next iteration
		if result != "" {
			o.session.Messages = append(o.session.Messages, Message{
				Role:    "user",
				Content: result,
			})
		}

		// Save session after each iteration
		_ = o.SaveSession()
	}

	return fmt.Errorf("max iterations reached")
}

// responseSchema is the JSON schema for structured responses
const responseSchema = `{
	"type": "object",
	"properties": {
		"thinking": {"type": "string"},
		"action": {"type": "string", "enum": ["ask_user", "read_files", "write_file", "run_command", "done"]},
		"action_params": {
			"type": "object",
			"properties": {
				"question": {"type": "string"},
				"paths": {"type": "array", "items": {"type": "string"}},
				"path": {"type": "string"},
				"content": {"type": "string"},
				"command": {"type": "string"}
			}
		},
		"message": {"type": "string"},
		"state": {"type": "object"}
	},
	"required": ["action"]
}`

// callClaude calls Claude with the current conversation
func (o *Orchestrator) callClaude(ctx context.Context) (*Response, error) {
	// Build the full prompt from conversation history
	var promptBuilder strings.Builder
	for _, msg := range o.session.Messages {
		promptBuilder.WriteString(fmt.Sprintf("[%s]\n%s\n\n", msg.Role, msg.Content))
	}

	prompt := promptBuilder.String()

	// Call Claude with structured output using json-schema
	cmd := exec.CommandContext(ctx, o.claudePath,
		"-p", prompt,
		"--output-format", "json",
		"--json-schema", responseSchema,
	)
	cmd.Dir = o.workDir

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude exited with error: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	// The output should be JSON with a "result" field containing the actual response
	// Format: {"type":"result","subtype":"success","cost_usd":...,"result":"..."}
	var claudeOutput struct {
		Type   string `json:"type"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal(output, &claudeOutput); err != nil {
		// Try to extract JSON directly from output
		return o.parseResponseFromText(string(output))
	}

	// Parse the result field which should be our structured response
	var response Response
	if err := json.Unmarshal([]byte(claudeOutput.Result), &response); err != nil {
		// Claude might return the response directly in result as text
		// Try to extract JSON from it
		return o.parseResponseFromText(claudeOutput.Result)
	}

	return &response, nil
}

// parseResponseFromText tries to extract our Response from text that might contain JSON
func (o *Orchestrator) parseResponseFromText(text string) (*Response, error) {
	extracted := extractJSON(text)
	if extracted != "" {
		var response Response
		if err := json.Unmarshal([]byte(extracted), &response); err == nil {
			return &response, nil
		}
	}
	return nil, fmt.Errorf("failed to parse response from text: %s", text)
}

// extractJSON tries to extract JSON from a string that might contain markdown
func extractJSON(s string) string {
	// Try to find JSON in code blocks
	if start := strings.Index(s, "```json"); start != -1 {
		start += 7
		if end := strings.Index(s[start:], "```"); end != -1 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	if start := strings.Index(s, "```"); start != -1 {
		start += 3
		if end := strings.Index(s[start:], "```"); end != -1 {
			candidate := strings.TrimSpace(s[start : start+end])
			if strings.HasPrefix(candidate, "{") {
				return candidate
			}
		}
	}
	// Try to find raw JSON
	if start := strings.Index(s, "{"); start != -1 {
		// Find matching closing brace
		depth := 0
		for i := start; i < len(s); i++ {
			switch s[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return s[start : i+1]
				}
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
		// Ensure directory exists
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

// buildPlanSystemPrompt builds the system prompt for planning
func (o *Orchestrator) buildPlanSystemPrompt() string {
	return `You are a PRD (Product Requirements Document) planning assistant for SuperRalph.

Your job is to help the user create a prd.json file through a structured conversation.

## Response Format

You MUST respond with valid JSON in this exact format:

{
  "thinking": "Your internal reasoning about what to do next",
  "action": "ask_user|read_files|write_file|done",
  "action_params": {
    "question": "Your question to the user",
    "paths": ["file1.go", "src/main.ts"],
    "path": "prd.json",
    "content": "file contents"
  },
  "message": "Message to display to the user",
  "state": {
    "phase": "gathering|proposing|refining|complete",
    "draft_prd": { ... }
  }
}

## Actions

- ask_user: Ask the user a question. Put the question in action_params.question
- read_files: Read files from the codebase to understand it. Put paths in action_params.paths
- write_file: Write a file. Put path and content in action_params
- done: Planning is complete

## PRD Schema

The prd.json file MUST follow this structure:

{
  "name": "Project Name",
  "description": "Description",
  "testCommand": "npm test",
  "features": [
    {
      "id": "feat-001",
      "category": "functional|ui|integration|performance|security",
      "priority": "high|medium|low",
      "description": "What this feature does",
      "steps": ["Step 1", "Step 2"],
      "passes": false
    }
  ]
}

## Planning Flow

1. GATHERING: Ask about the project, explore existing code
2. PROPOSING: Propose a feature list based on understanding
3. REFINING: Iterate based on user feedback  
4. COMPLETE: Write prd.json and finish

Be thorough. Ask clarifying questions. Look at existing code to understand the project.
When the user is satisfied, write the prd.json file and set action to "done".`
}

// buildBuildSystemPrompt builds the system prompt for building
func (o *Orchestrator) buildBuildSystemPrompt() string {
	return `You are a code implementation agent for SuperRalph.

Your job is to implement features from the PRD, ensuring all tests pass before committing.

## Response Format

You MUST respond with valid JSON in this exact format:

{
  "thinking": "Your internal reasoning",
  "action": "read_files|write_file|run_command|done",
  "action_params": {
    "paths": ["prd.json", "src/main.go"],
    "path": "src/feature.go",
    "content": "file contents",
    "command": "go test ./..."
  },
  "message": "Status message for the user",
  "state": {
    "phase": "reading|implementing|testing|committing|complete",
    "current_feature": "feat-001",
    "tests_passing": false
  }
}

## Actions

- read_files: Read files to understand codebase
- write_file: Write/update a file
- run_command: Run a shell command (tests, git, etc.)
- done: All features complete

## CRITICAL RULES

1. TESTS MUST PASS before any commit
2. Work on ONE feature at a time (highest priority first)
3. Run tests after each change
4. Only commit when tests pass
5. Update prd.json to mark features as passes: true

## Workflow

1. READ: Read prd.json and progress.txt
2. IMPLEMENT: Write code for the highest priority feature with passes: false
3. TEST: Run the test command
4. If tests fail: fix and repeat
5. COMMIT: git add, git commit with descriptive message
6. UPDATE: Set passes: true in prd.json, append to progress.txt
7. Repeat for next feature or set action to "done" if all complete`
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
