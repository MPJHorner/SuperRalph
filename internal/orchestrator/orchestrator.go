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
	onOutput   func(line string)
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

// OnOutput sets the callback for Claude's streaming output
func (o *Orchestrator) OnOutput(fn func(line string)) *Orchestrator {
	o.onOutput = fn
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

	prompt := `Help me create a prd.json file for this project. 

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

Start by asking what I want to build.`

	return o.runClaudeInteractive(ctx, prompt)
}

// RunBuild runs the build loop
func (o *Orchestrator) RunBuild(ctx context.Context) error {
	o.session.Mode = "build"
	o.session.State = &BuildState{Phase: "reading", Iteration: 1}

	// Read current PRD to build the prompt
	prdContent, err := os.ReadFile(filepath.Join(o.workDir, "prd.json"))
	if err != nil {
		return fmt.Errorf("failed to read prd.json: %w", err)
	}

	// Read progress if exists
	var progressContent string
	if data, err := os.ReadFile(filepath.Join(o.workDir, "progress.txt")); err == nil {
		progressContent = string(data)
	}

	prompt := fmt.Sprintf(`You are implementing features from a PRD. Here is the current state:

## prd.json
%s

## progress.txt
%s

## Your Task

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

Start by reading the codebase to understand the current implementation, then implement the next feature.`,
		string(prdContent), progressContent)

	return o.runClaudeInteractive(ctx, prompt)
}

// runClaudeInteractive runs Claude in interactive mode, streaming output
func (o *Orchestrator) runClaudeInteractive(ctx context.Context, prompt string) error {
	o.debugLog("Starting Claude with prompt (%d chars)", len(prompt))

	// Run Claude with the prompt, allowing it to use its tools
	// Note: stream-json requires --verbose flag
	cmd := exec.CommandContext(ctx, o.claudePath,
		"-p", prompt,
		"--permission-mode", "acceptEdits",
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
	fmt.Println("  Claude is working...")
	fmt.Println()

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
			fmt.Println(line)
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
									fmt.Println(text)
								}
							case "tool_use":
								if name, ok := blockMap["name"].(string); ok {
									fmt.Printf("\n  [Using tool: %s]\n", name)
									if input, ok := blockMap["input"].(map[string]any); ok {
										// Show some context about the tool use
										if cmd, ok := input["command"].(string); ok {
											fmt.Printf("  > %s\n", cmd)
										}
										if path, ok := input["file_path"].(string); ok {
											fmt.Printf("  > %s\n", path)
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
								if content, ok := blockMap["content"].(string); ok {
									lines := strings.Split(content, "\n")
									if len(lines) > 10 {
										for _, line := range lines[:5] {
											fmt.Printf("    %s\n", line)
										}
										fmt.Printf("    ... (%d more lines)\n", len(lines)-5)
									} else {
										for _, line := range lines {
											fmt.Printf("    %s\n", line)
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
				fmt.Printf("\n%s\n", result)
			}

			fmt.Printf("\n  [%s: %.1fs", subtype, elapsed)
			if cost, ok := event["total_cost_usd"].(float64); ok {
				fmt.Printf(", $%.4f", cost)
			}
			fmt.Println("]")

		case "error":
			// Error occurred
			if errData, ok := event["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok {
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
	fmt.Printf("\n  [Complete: %.1fs]\n", elapsed)

	return nil
}

// debugLog logs a debug message
func (o *Orchestrator) debugLog(format string, args ...any) {
	if o.debug && o.onDebug != nil {
		o.onDebug(fmt.Sprintf(format, args...))
	}
}

// buildFreshContext creates a clean context for each iteration
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

	return ctx.String()
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
