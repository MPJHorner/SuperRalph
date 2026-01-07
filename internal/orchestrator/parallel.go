package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ParallelLimits defines the concurrency limits for parallel execution
type ParallelLimits struct {
	MaxReads    int // Max parallel file reads (default 10)
	MaxCommands int // Max parallel commands (default 3)
	// Writes are always sequential for safety
}

// DefaultParallelLimits returns the default parallel execution limits
func DefaultParallelLimits() ParallelLimits {
	return ParallelLimits{
		MaxReads:    10,
		MaxCommands: 3,
	}
}

// ParallelAction represents a group of actions to execute concurrently
type ParallelAction struct {
	Actions []SubAction `json:"actions"`
}

// SubAction represents a single action within a parallel group
type SubAction struct {
	Type   Action       `json:"type"`
	Params ActionParams `json:"params"`
}

// ActionResult represents the result of executing a single action
type ActionResult struct {
	Action  SubAction `json:"action"`
	Success bool      `json:"success"`
	Output  string    `json:"output"`
	Error   string    `json:"error,omitempty"`
}

// ParallelResult represents the results of executing a parallel action group
type ParallelResult struct {
	Results      []ActionResult `json:"results"`
	AllSucceeded bool           `json:"all_succeeded"`
	FailedCount  int            `json:"failed_count"`
}

// ParallelExecutor executes actions in parallel with configurable limits
type ParallelExecutor struct {
	workDir string
	limits  ParallelLimits
	debug   bool
	onDebug func(msg string)
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(workDir string) *ParallelExecutor {
	return &ParallelExecutor{
		workDir: workDir,
		limits:  DefaultParallelLimits(),
	}
}

// SetLimits sets the concurrency limits
func (pe *ParallelExecutor) SetLimits(limits ParallelLimits) *ParallelExecutor {
	pe.limits = limits
	return pe
}

// SetDebug enables debug mode
func (pe *ParallelExecutor) SetDebug(debug bool, onDebug func(msg string)) *ParallelExecutor {
	pe.debug = debug
	pe.onDebug = onDebug
	return pe
}

func (pe *ParallelExecutor) debugLog(format string, args ...any) {
	if pe.debug && pe.onDebug != nil {
		pe.onDebug(fmt.Sprintf(format, args...))
	}
}

// Execute runs a parallel action group and returns the combined results
func (pe *ParallelExecutor) Execute(ctx context.Context, parallel ParallelAction) ParallelResult {
	if len(parallel.Actions) == 0 {
		return ParallelResult{AllSucceeded: true}
	}

	// Categorize actions by type
	var reads, writes, commands, others []SubAction
	for _, action := range parallel.Actions {
		switch action.Type {
		case ActionReadFiles:
			reads = append(reads, action)
		case ActionWriteFile:
			writes = append(writes, action)
		case ActionRunCommand:
			commands = append(commands, action)
		default:
			others = append(others, action)
		}
	}

	pe.debugLog("Parallel execution: %d reads, %d writes, %d commands, %d others",
		len(reads), len(writes), len(commands), len(others))

	var allResults []ActionResult
	var mu sync.Mutex

	// Execute reads in parallel (up to MaxReads)
	if len(reads) > 0 {
		readResults := pe.executeParallel(ctx, reads, pe.limits.MaxReads, pe.executeRead)
		mu.Lock()
		allResults = append(allResults, readResults...)
		mu.Unlock()
	}

	// Execute writes sequentially (for safety)
	for _, write := range writes {
		result := pe.executeWrite(ctx, write)
		mu.Lock()
		allResults = append(allResults, result)
		mu.Unlock()
	}

	// Execute commands in parallel (up to MaxCommands)
	if len(commands) > 0 {
		cmdResults := pe.executeParallel(ctx, commands, pe.limits.MaxCommands, pe.executeCommand)
		mu.Lock()
		allResults = append(allResults, cmdResults...)
		mu.Unlock()
	}

	// Execute other actions sequentially
	for _, other := range others {
		result := pe.executeOther(ctx, other)
		mu.Lock()
		allResults = append(allResults, result)
		mu.Unlock()
	}

	// Count failures
	failedCount := 0
	for _, r := range allResults {
		if !r.Success {
			failedCount++
		}
	}

	return ParallelResult{
		Results:      allResults,
		AllSucceeded: failedCount == 0,
		FailedCount:  failedCount,
	}
}

// executeParallel runs actions in parallel with a semaphore for limiting concurrency
func (pe *ParallelExecutor) executeParallel(
	ctx context.Context,
	actions []SubAction,
	maxConcurrency int,
	executor func(context.Context, SubAction) ActionResult,
) []ActionResult {
	results := make([]ActionResult, len(actions))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, action := range actions {
		wg.Add(1)
		go func(idx int, act SubAction) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[idx] = ActionResult{
					Action:  act,
					Success: false,
					Error:   "context cancelled",
				}
				return
			}

			// Execute action
			results[idx] = executor(ctx, act)
		}(i, action)
	}

	wg.Wait()
	return results
}

// executeRead reads a file or multiple files
func (pe *ParallelExecutor) executeRead(ctx context.Context, action SubAction) ActionResult {
	result := ActionResult{Action: action}

	if len(action.Params.Paths) == 0 {
		result.Error = "no file paths specified"
		return result
	}

	var output strings.Builder
	for _, path := range action.Params.Paths {
		select {
		case <-ctx.Done():
			result.Error = "context cancelled"
			return result
		default:
		}

		fullPath := path
		if !filepath.IsAbs(path) {
			fullPath = filepath.Join(pe.workDir, path)
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			result.Error = fmt.Sprintf("failed to read %s: %v", path, err)
			return result
		}

		output.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", path, string(content)))
	}

	result.Success = true
	result.Output = output.String()
	return result
}

// executeWrite writes content to a file
func (pe *ParallelExecutor) executeWrite(ctx context.Context, action SubAction) ActionResult {
	result := ActionResult{Action: action}

	select {
	case <-ctx.Done():
		result.Error = "context cancelled"
		return result
	default:
	}

	if action.Params.Path == "" {
		result.Error = "no file path specified"
		return result
	}

	fullPath := action.Params.Path
	if !filepath.IsAbs(action.Params.Path) {
		fullPath = filepath.Join(pe.workDir, action.Params.Path)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		result.Error = fmt.Sprintf("failed to create directory %s: %v", dir, err)
		return result
	}

	if err := os.WriteFile(fullPath, []byte(action.Params.Content), 0644); err != nil {
		result.Error = fmt.Sprintf("failed to write %s: %v", action.Params.Path, err)
		return result
	}

	result.Success = true
	result.Output = fmt.Sprintf("Wrote %d bytes to %s", len(action.Params.Content), action.Params.Path)
	return result
}

// executeCommand runs a shell command
func (pe *ParallelExecutor) executeCommand(ctx context.Context, action SubAction) ActionResult {
	result := ActionResult{Action: action}

	if action.Params.Command == "" {
		result.Error = "no command specified"
		return result
	}

	pe.debugLog("Running command: %s", action.Params.Command)

	cmd := exec.CommandContext(ctx, "sh", "-c", action.Params.Command)
	cmd.Dir = pe.workDir

	output, err := cmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		if ctx.Err() != nil {
			result.Error = "context cancelled"
		} else {
			result.Error = err.Error()
		}
		return result
	}

	result.Success = true
	return result
}

// executeOther handles other action types (ask_user, done, etc.)
func (pe *ParallelExecutor) executeOther(ctx context.Context, action SubAction) ActionResult {
	result := ActionResult{Action: action}

	select {
	case <-ctx.Done():
		result.Error = "context cancelled"
		return result
	default:
	}

	switch action.Type {
	case ActionDone:
		result.Success = true
		result.Output = "Action completed"
	case ActionAskUser:
		// AskUser actions can't be parallelized meaningfully
		result.Success = true
		result.Output = fmt.Sprintf("Question: %s", action.Params.Question)
	default:
		result.Error = fmt.Sprintf("unsupported action type: %s", action.Type)
	}

	return result
}

// ExecuteSingleRead is a convenience method for reading a single file
func (pe *ParallelExecutor) ExecuteSingleRead(ctx context.Context, path string) (string, error) {
	result := pe.executeRead(ctx, SubAction{
		Type: ActionReadFiles,
		Params: ActionParams{
			Paths: []string{path},
		},
	})
	if !result.Success {
		return "", fmt.Errorf("%s", result.Error)
	}
	return result.Output, nil
}

// ExecuteSingleWrite is a convenience method for writing a single file
func (pe *ParallelExecutor) ExecuteSingleWrite(ctx context.Context, path, content string) error {
	result := pe.executeWrite(ctx, SubAction{
		Type: ActionWriteFile,
		Params: ActionParams{
			Path:    path,
			Content: content,
		},
	})
	if !result.Success {
		return fmt.Errorf("%s", result.Error)
	}
	return nil
}

// ExecuteSingleCommand is a convenience method for running a single command
func (pe *ParallelExecutor) ExecuteSingleCommand(ctx context.Context, command string) (string, error) {
	result := pe.executeCommand(ctx, SubAction{
		Type: ActionRunCommand,
		Params: ActionParams{
			Command: command,
		},
	})
	if !result.Success {
		return result.Output, fmt.Errorf("%s", result.Error)
	}
	return result.Output, nil
}
