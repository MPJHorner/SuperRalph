# SuperRalph

PRD-driven agent harness for long-running Claude development sessions.

SuperRalph validates PRD (Product Requirements Document) files and orchestrates Claude to implement features incrementally with test-gated commits.

## Inspiration & Attribution

This project is inspired by and builds upon:

- **[Matt Pocock's Ralph](https://x.com/mattpocockuk/status/2007924876548637089)** - The original concept of using a PRD-driven loop with Claude to implement features incrementally. Matt's approach of running Claude in a loop with `@prd.json` and `@progress.txt` was the foundation for this tool.

- **[Anthropic's "Effective Harnesses for Long-Running Agents"](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)** - Research on how to build effective agent harnesses that work across multiple context windows. Key insights include:
  - Using an initializer agent to set up the environment
  - Working on one feature at a time (incremental progress)
  - Leaving clear artifacts for the next session
  - Test-gated commits to ensure clean state

### Libraries Used

Built with excellent Go libraries from [Charm](https://charm.sh/):

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - TUI framework based on The Elm Architecture
- **[Bubbles](https://github.com/charmbracelet/bubbles)** - Common TUI components (spinners, progress bars)
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** - Style definitions for terminal UIs
- **[Huh](https://github.com/charmbracelet/huh)** - Interactive terminal forms and prompts

Also uses:
- **[Cobra](https://github.com/spf13/cobra)** - CLI framework

## Features

- **PRD Validation** - Ensures your prd.json has the correct structure
- **Interactive Planning** - Claude helps you create a well-structured PRD
- **Test-Gated Commits** - Tests MUST pass before any code is committed (non-negotiable)
- **Live TUI** - Monitor progress with a beautiful terminal interface
- **Automatic Retries** - Failed iterations retry up to 3 times
- **Progress Tracking** - Detailed progress.txt log for session continuity
- **macOS/Linux Notifications** - Get notified when builds complete

## Installation

### From Source

```bash
git clone https://github.com/mpjhorner/SuperRalph
cd SuperRalph
make build
make install  # Installs to /usr/local/bin (may need sudo)
```

Or install to your user directory (no sudo needed):

```bash
make install-user  # Installs to ~/go/bin
```

### Add to your shell

Add one of these to your `~/.zshrc` (or `~/.bashrc`):

```bash
# If installed to /usr/local/bin (already in PATH for most systems)
# Nothing needed!

# If installed to ~/go/bin
export PATH="$HOME/go/bin:$PATH"

# Or create an alias to a custom location
alias superralph="/path/to/superralph"
```

Then reload your shell:

```bash
source ~/.zshrc
```

Verify installation:

```bash
superralph --help
```

## Usage

### Create a PRD

Start an interactive session with Claude to create your PRD:

```bash
superralph plan
```

Claude will:
1. Ask what you're building
2. Help you think through features
3. Ask clarifying questions
4. Create a well-structured prd.json

### Validate your PRD

Check that your prd.json is valid:

```bash
superralph validate
```

### View Status

See live progress of your PRD:

```bash
superralph status
```

### Build Features

Run the Claude agent loop to implement features:

```bash
superralph build
```

You'll be prompted for the number of iterations. The agent will:
1. Pick the highest-priority incomplete feature
2. Implement the feature
3. Run tests (must pass!)
4. Update prd.json and progress.txt
5. Commit changes
6. Repeat

## PRD Format

Create a `prd.json` in your project root:

```json
{
  "name": "My Project",
  "description": "What the project does",
  "testCommand": "go test ./...",
  "features": [
    {
      "id": "feat-001",
      "category": "functional",
      "priority": "high",
      "description": "User can create account",
      "steps": [
        "Navigate to signup page",
        "Fill in email and password",
        "Submit form",
        "Verify account created"
      ],
      "passes": false
    }
  ]
}
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Project name |
| `description` | Yes | What the project does |
| `testCommand` | Yes | Command to run tests (e.g., `go test ./...`, `npm test`) |
| `features` | Yes | Array of features |

### Feature Fields

| Field | Required | Values |
|-------|----------|--------|
| `id` | Yes | Unique identifier (e.g., `feat-001`) |
| `category` | Yes | `functional`, `ui`, `integration`, `performance`, `security` |
| `priority` | Yes | `high`, `medium`, `low` |
| `description` | Yes | What the feature does |
| `steps` | Yes | Array of verification steps |
| `passes` | Yes | `false` initially, `true` when complete |

## Progress File

SuperRalph maintains a `progress.txt` file that Claude appends to after each session:

```
================================================================================
Session: 2026-01-07T12:00:00Z
Iteration: 1
================================================================================

## Starting State
- Features passing: 3/15
- Working on: feat-004 "User can delete messages"

## Work Done
- Implemented message deletion endpoint
- Added DELETE handler with auth checks
- Updated frontend with delete button

## Testing
- Test command: go test ./...
- Result: PASSED
- Details: 47 tests passed

## Commits
- abc1234: feat: add message deletion endpoint

## Ending State
- Features passing: 4/15
- Feature feat-004 marked as passes: true
- All tests passing: YES

## Notes for Next Session
- Consider adding soft-delete for recovery

================================================================================
```

## TUI Controls

| Key | Action |
|-----|--------|
| `q` | Quit |
| `p` | Pause (during build) |
| `r` | Resume (when paused) / Refresh (in status) |

## Requirements

- Go 1.21+ (for building)
- [Claude CLI](https://docs.anthropic.com/en/docs/claude-code/cli-usage) installed and configured
- macOS or Linux

## Philosophy

SuperRalph is built on principles from Anthropic's research on [effective harnesses for long-running agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents):

1. **Incremental Progress** - Work on one feature at a time
2. **Clean State** - Always leave the codebase in a working state
3. **Test-Gated Commits** - Never commit with failing tests
4. **Progress Documentation** - Leave clear notes for the next session
5. **Structured PRD** - Use JSON for reliable feature tracking

## Development

```bash
# Run tests
make test

# Build for development
make build

# Build for all platforms
make build-all

# Clean build artifacts
make clean
```

## License

MIT

## Contributing

Contributions welcome! Please ensure tests pass before submitting PRs.
