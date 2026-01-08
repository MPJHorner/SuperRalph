# SuperRalph - AI Assistant Guide

This document provides context and guidelines for AI assistants working on the SuperRalph codebase.

## Project Overview

**SuperRalph** is a PRD-driven agent harness for long-running Claude development sessions. It orchestrates Claude CLI to implement features incrementally with test-gated commits.

### Core Functionality

1. **PRD Management**: Create, validate, and track Product Requirements Documents (`prd.json`)
2. **Agent Orchestration**: Run Claude CLI in a loop to implement features
3. **Test-Gated Commits**: Ensure all tests pass before any code is committed
4. **Progress Tracking**: Maintain `progress.txt` for session continuity
5. **Self-Update**: Update itself from GitHub releases

### Key Commands

| Command | Description |
|---------|-------------|
| `superralph plan` | Interactive PRD creation with Claude |
| `superralph validate` | Validate prd.json |
| `superralph build` | Run the agent loop to implement features |
| `superralph status` | Show live progress |
| `superralph update` | Self-update to latest version |

---

## Architecture

### Directory Structure

```
superralph/
├── main.go                    # Entry point
├── cmd/                       # CLI commands (Cobra)
│   ├── root.go               # Root command, update checking
│   ├── build.go              # Build command - runs agent loop
│   ├── plan.go               # Plan command - create PRD interactively
│   ├── validate.go           # Validate PRD
│   ├── status.go             # Show live status
│   ├── update.go             # Self-update command
│   └── version.go            # Version info
│
├── internal/                  # Internal packages
│   ├── agent/                # Agent prompt generation
│   ├── git/                  # Git operations
│   ├── notify/               # System notifications
│   ├── orchestrator/         # Main Claude orchestration logic
│   ├── prd/                  # PRD types, loading, validation
│   ├── progress/             # Progress file writing
│   ├── tagging/              # File tagging system
│   ├── tui/                  # Terminal UI (Bubble Tea)
│   │   └── components/       # TUI components
│   └── version/              # Version info and self-update
│
└── build/                    # Build output directory (gitignored)
```

### Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/bubbles` | TUI components |
| `github.com/charmbracelet/lipgloss` | Terminal styling |
| `github.com/charmbracelet/huh` | Interactive forms |
| `github.com/charmbracelet/log` | Structured logging |
| `github.com/stretchr/testify` | Test assertions |
| `github.com/samber/lo` | Generic utilities |
| `github.com/google/uuid` | UUID generation |

---

## Code Style Guide

### Go Conventions

This project follows standard Go conventions with some project-specific patterns:

#### Error Handling

Always wrap errors with context using `fmt.Errorf` with `%w`:

```go
// Good
if err != nil {
    return fmt.Errorf("failed to load PRD: %w", err)
}

// Bad
if err != nil {
    return err
}
```

#### Naming Conventions

- **Packages**: Short, lowercase, no underscores (e.g., `prd`, `orchestrator`)
- **Interfaces**: Use `-er` suffix when representing a single method (e.g., `Reader`, `Writer`)
- **Constructors**: Use `New` prefix (e.g., `New()`, `NewOrchestrator()`)
- **Getters**: No `Get` prefix (e.g., `o.Session()` not `o.GetSession()`)

#### Struct Organization

Order struct fields by:
1. Required fields first
2. Optional/config fields
3. Internal state last
4. Callbacks at the end

```go
type Orchestrator struct {
    // Required
    workDir    string
    claudePath string
    
    // Internal state
    session    *Session
    debug      bool
    
    // Callbacks
    onMessage  func(role, content string)
    onAction   func(action Action, params ActionParams)
}
```

#### Builder Pattern

This project uses the builder pattern for configuration. Return `*Self` for chaining:

```go
func (o *Orchestrator) SetDebug(debug bool) *Orchestrator {
    o.debug = debug
    return o
}

func (o *Orchestrator) OnMessage(fn func(role, content string)) *Orchestrator {
    o.onMessage = fn
    return o
}

// Usage
orch := orchestrator.New(workDir).
    SetDebug(true).
    OnMessage(handleMessage).
    OnAction(handleAction)
```

#### Logging

Use `charmbracelet/log` for structured logging:

```go
import "github.com/charmbracelet/log"

// Package-level logger
var logger = log.NewWithOptions(os.Stderr, log.Options{
    ReportTimestamp: true,
    Prefix:          "orchestrator",
})

// Usage
logger.Info("starting session", "id", session.ID)
logger.Error("failed to execute", "err", err)
logger.Debug("processing event", "type", eventType, "data", data)
```

#### Slice/Map Operations

Use `samber/lo` for common operations:

```go
import "github.com/samber/lo"

// Filter
incomplete := lo.Filter(features, func(f Feature, _ int) bool {
    return !f.Passes
})

// Map
ids := lo.Map(features, func(f Feature, _ int) string {
    return f.ID
})

// Find
feature, found := lo.Find(features, func(f Feature) bool {
    return f.ID == targetID
})
```

---

## Testing Conventions

### Test File Location

Tests live alongside the code they test:

```
internal/prd/
├── loader.go
├── loader_test.go
├── types.go
├── types_test.go
├── validate.go
└── validate_test.go
```

### Test Structure

Use **table-driven tests** with `testify`:

```go
func TestValidateFeature(t *testing.T) {
    tests := []struct {
        name    string
        feature Feature
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid feature",
            feature: Feature{
                ID:          "feat-001",
                Category:    "functional",
                Priority:    "high",
                Description: "Test feature",
                Steps:       []string{"Step 1"},
            },
            wantErr: false,
        },
        {
            name: "missing ID",
            feature: Feature{
                Category: "functional",
            },
            wantErr: true,
            errMsg:  "feature ID is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateFeature(&tt.feature)
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Testify Usage

- Use `require` for fatal assertions (test cannot continue)
- Use `assert` for non-fatal assertions (test can continue)

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
    result, err := DoSomething()
    require.NoError(t, err)           // Fatal if err != nil
    require.NotNil(t, result)          // Fatal if nil
    
    assert.Equal(t, "expected", result.Value)  // Non-fatal
    assert.Len(t, result.Items, 3)             // Non-fatal
}
```

### Test Helpers

For tests needing temp directories or files:

```go
func TestWithTempDir(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up
    
    // Create test files
    prdPath := filepath.Join(tmpDir, "prd.json")
    err := os.WriteFile(prdPath, []byte(`{"name": "test"}`), 0644)
    require.NoError(t, err)
    
    // Run test
    prd, err := prd.Load(prdPath)
    require.NoError(t, err)
    assert.Equal(t, "test", prd.Name)
}
```

---

## Common Commands

### Development

```bash
# Run tests
make test

# Run tests with coverage
make coverage

# Run linter
make lint

# Auto-fix lint issues
make lint-fix

# Format code
make fmt

# Run all checks (fmt, vet, lint, test)
make check

# Build binary
make build

# Build and run
make run

# Install dev tools (golangci-lint)
make setup
```

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Install to /usr/local/bin
make install

# Install to ~/go/bin
make install-user
```

### Dependencies

```bash
# Download dependencies
make deps

# Add a new dependency
go get github.com/example/package

# Update dependencies
go get -u ./...
go mod tidy
```

---

## Common Tasks

### Adding a New CLI Command

1. Create `cmd/mycommand.go`:

```go
package cmd

import (
    "github.com/spf13/cobra"
)

var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Short description",
    Long:  `Longer description of what the command does.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(myCmd)
    
    // Add flags
    myCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
}
```

2. The command is automatically registered via `init()`.

### Adding a New Internal Package

1. Create directory: `internal/mypackage/`
2. Create main file: `internal/mypackage/mypackage.go`
3. Create test file: `internal/mypackage/mypackage_test.go`

```go
// internal/mypackage/mypackage.go
package mypackage

// MyType represents...
type MyType struct {
    // fields
}

// New creates a new MyType
func New() *MyType {
    return &MyType{}
}
```

### Modifying PRD Structure

1. Update types in `internal/prd/types.go`
2. Update validation in `internal/prd/validate.go`
3. Update tests in `internal/prd/types_test.go` and `internal/prd/validate_test.go`
4. Update `example-prd.json` if needed
5. Update README.md documentation

---

## Key Patterns

### Orchestrator Pattern

The `Orchestrator` is the central coordinator for Claude sessions:

```go
orch := orchestrator.New(workDir).
    SetDebug(debug).
    OnMessage(func(role, content string) {
        // Handle messages
    }).
    OnAction(func(action Action, params ActionParams) {
        // Handle actions
    })

// Run build loop
err := orch.RunBuild(ctx)
```

### Iteration Context

Each Claude call gets fresh, self-contained context:

```go
iterCtx, err := o.BuildIterationContext(iteration, phase, feature)
if err != nil {
    return err
}

prompt := iterCtx.BuildPrompt()
```

### TUI with Bubble Tea

The TUI uses the Elm architecture:

```go
type Model struct {
    // State
}

func (m Model) Init() tea.Cmd {
    return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle key presses
    }
    return m, nil
}

func (m Model) View() string {
    // Render the UI
    return lipgloss.JoinVertical(lipgloss.Left,
        m.renderHeader(),
        m.renderContent(),
        m.renderFooter(),
    )
}
```

---

## Files You Should Never Edit Directly

- `go.sum` - Managed by `go mod`
- `build/` - Build artifacts
- Files in `vendor/` (if vendoring is enabled)

## Files to Always Update Together

- `internal/prd/types.go` ↔ `internal/prd/validate.go` ↔ `internal/prd/types_test.go`
- `internal/version/version.go` ↔ `Makefile` (version ldflags)
- `README.md` ↔ `example-prd.json` (keep examples in sync)

---

## Debugging Tips

### Enable Debug Mode

```bash
# Via environment variable
DEBUG=1 superralph build

# Via flag (if implemented)
superralph build --debug
```

### View Claude Communication

The orchestrator has debug callbacks:

```go
orch.OnDebug(func(msg string) {
    fmt.Fprintf(os.Stderr, "[DEBUG] %s\n", msg)
})
```

### Test a Single Package

```bash
go test -v ./internal/prd/...
go test -v ./internal/orchestrator/... -run TestSpecificFunction
```

---

## Release Process

Releases are automated via GitHub Actions:

1. Push to `main` triggers CI
2. Tests run and must pass
3. Version is auto-bumped (patch by default)
4. Cross-platform binaries are built
5. GitHub Release is created with binaries

To trigger a specific version bump, include in commit message:
- `#major` - Bump major version
- `#minor` - Bump minor version
- `#patch` - Bump patch version (default)
