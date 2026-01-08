package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Runner handles executing Claude commands
type Runner struct {
	workDir    string
	claudePath string
	onOutput   func(line string)
	onError    func(err error)
	onComplete func(output string, success bool)

	mu     sync.Mutex
	cmd    *exec.Cmd
	cancel context.CancelFunc
	paused bool
	output strings.Builder
}

// NewRunner creates a new agent runner
func NewRunner(workDir string) *Runner {
	return &Runner{
		workDir:    workDir,
		claudePath: findClaudeBinary(),
	}
}

// findClaudeBinary searches for the Claude CLI binary in common locations
func findClaudeBinary() string {
	// Check environment variable first
	if envPath := os.Getenv("CLAUDE_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Common locations to check
	homeDir, _ := os.UserHomeDir()
	locations := []string{
		// Standard PATH lookup
		"claude",
		// Claude CLI default install location
		filepath.Join(homeDir, ".claude", "local", "claude"),
		// Other common locations
		"/usr/local/bin/claude",
		"/usr/bin/claude",
		filepath.Join(homeDir, ".local", "bin", "claude"),
		filepath.Join(homeDir, "bin", "claude"),
		// npm global installs
		"/usr/local/lib/node_modules/@anthropic-ai/claude-cli/bin/claude",
		filepath.Join(homeDir, ".npm-global", "bin", "claude"),
	}

	for _, loc := range locations {
		// For "claude" without path, check if it's in PATH
		if loc == "claude" {
			if path, err := exec.LookPath("claude"); err == nil {
				return path
			}
			continue
		}
		// For full paths, check if file exists and is executable
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	// Fallback to "claude" and let it fail with a clear error
	return "claude"
}

// OnOutput sets the callback for output lines
func (r *Runner) OnOutput(fn func(line string)) *Runner {
	r.onOutput = fn
	return r
}

// OnError sets the callback for errors
func (r *Runner) OnError(fn func(err error)) *Runner {
	r.onError = fn
	return r
}

// OnComplete sets the callback for completion
func (r *Runner) OnComplete(fn func(output string, success bool)) *Runner {
	r.onComplete = fn
	return r
}

// Run executes Claude with the given prompt
func (r *Runner) Run(ctx context.Context, prompt string) error {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	defer cancel()

	// Build the command
	// Using claude CLI with permission mode to accept edits automatically
	r.cmd = exec.CommandContext(ctx, r.claudePath,
		"--permission-mode", "acceptEdits",
		"-p", prompt,
	)
	r.cmd.Dir = r.workDir
	r.cmd.Env = os.Environ()

	// Get stdout and stderr pipes
	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := r.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the command
	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude: %w", err)
	}

	// Read output in goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		r.readOutput(stdout)
	}()

	go func() {
		defer wg.Done()
		r.readOutput(stderr)
	}()

	// Wait for output reading to complete
	wg.Wait()

	// Wait for the command to finish
	err = r.cmd.Wait()

	success := err == nil
	output := r.output.String()

	// Check for completion signal
	if ContainsCompletionSignal(output) {
		success = true
	}

	if r.onComplete != nil {
		r.onComplete(output, success)
	}

	return err
}

// RunInteractive runs Claude in interactive mode (for planning)
func (r *Runner) RunInteractive(ctx context.Context, systemPrompt string) error {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	defer cancel()

	// For interactive mode:
	// - Use --system-prompt to give Claude the planning instructions
	// - Pass an initial user message to start the conversation
	// - DO NOT use -p/--print so it stays interactive
	// - Use --allowedTools to let Claude write files
	r.cmd = exec.CommandContext(ctx, r.claudePath,
		"--system-prompt", systemPrompt,
		"--allowedTools", "Write,Edit,Read,Bash",
		"What are you building? Tell me about your project - what's the main purpose, who will use it, and what problem does it solve?",
	)
	r.cmd.Dir = r.workDir
	r.cmd.Env = os.Environ()

	// Connect stdin/stdout/stderr directly to terminal for interactive session
	r.cmd.Stdin = os.Stdin
	r.cmd.Stdout = os.Stdout
	r.cmd.Stderr = os.Stderr

	if err := r.cmd.Run(); err != nil {
		// Check if it was canceled
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}

	return nil
}

func (r *Runner) readOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		r.mu.Lock()
		r.output.WriteString(line)
		r.output.WriteString("\n")
		r.mu.Unlock()

		if r.onOutput != nil {
			r.onOutput(line)
		}
	}
}

// Stop stops the running command
func (r *Runner) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
	}
}

// Pause pauses the runner (sets flag, actual pause handled by caller)
func (r *Runner) Pause() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paused = true
}

// Resume resumes the runner
func (r *Runner) Resume() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paused = false
}

// IsPaused returns whether the runner is paused
func (r *Runner) IsPaused() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.paused
}

// GetOutput returns the accumulated output
func (r *Runner) GetOutput() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.output.String()
}

// ClearOutput clears the accumulated output
func (r *Runner) ClearOutput() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.output.Reset()
}
