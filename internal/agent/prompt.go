package agent

import (
	"fmt"
	"strings"

	"github.com/mpjhorner/superralph/internal/prd"
)

// BuildPrompt builds the prompt for Claude
func BuildPrompt(p *prd.PRD, iteration int) string {
	return fmt.Sprintf(`You are working on a project with a structured PRD. Your job is to make incremental progress while ensuring ALL TESTS PASS before committing.

## Context Files
@prd.json
@progress.txt

## CRITICAL RULES - NON-NEGOTIABLE

1. **TESTS MUST PASS BEFORE ANY COMMIT**
   - Run the test command BEFORE committing: %s
   - If tests fail, FIX THEM before committing
   - NEVER commit with failing tests
   - If your changes break existing tests, fix those too

2. **ONE FEATURE PER SESSION**
   - Find the highest-priority feature with passes: false
   - Implement ONLY that one feature
   - Do not move to another feature until this one passes all tests

3. **CLEAN STATE REQUIREMENT**
   - The codebase must be in a working state when you finish
   - All tests passing
   - Code committed
   - Progress documented

## Workflow

1. Read prd.json and progress.txt to understand current state
2. Run tests first to verify starting state: %s
3. If tests are failing, FIX THEM FIRST before implementing new features
4. Find the highest-priority feature with passes: false
5. Implement the feature
6. Run tests: %s
7. If tests fail:
   - Debug and fix
   - Repeat until ALL tests pass
8. Only after tests pass:
   - Update prd.json: set passes: true for completed feature
   - Make a git commit with descriptive message
   - Append session summary to progress.txt

## Progress File Format

Append a new section to progress.txt with this EXACT format:

================================================================================
Session: [TIMESTAMP]
Iteration: %d
================================================================================

## Starting State
- Features passing: X/Y
- Working on: [feature_id] "[description]"

## Work Done
- [bullet points of what you did]

## Testing
- Test command: %s
- Result: [PASSED/FAILED]
- Details: [test output summary]

## Commits
- [commit hash]: [message]

## Ending State
- Features passing: X/Y
- Feature [feature_id] marked as passes: [true/false]
- All tests passing: [YES/NO]

## Notes for Next Session
- [anything the next agent should know]

================================================================================

## Completion

If ALL features have passes: true and all tests pass, output exactly:
<promise>COMPLETE</promise>

Remember: NEVER COMMIT WITH FAILING TESTS. This is non-negotiable.
`, p.TestCommand, p.TestCommand, p.TestCommand, iteration, p.TestCommand)
}

// BuildPlanPrompt builds the system prompt for the planning phase
func BuildPlanPrompt() string {
	return `You are a PRD (Product Requirements Document) planning assistant for SuperRalph.

Your job is to help the user create a prd.json file for their project through conversation:
1. Understand what they want to build
2. Ask clarifying questions to fully understand the scope
3. Help them break it down into discrete, testable features
4. When ready, create a well-structured prd.json file

## PRD Schema

The prd.json file MUST follow this exact structure:

{
  "name": "Project Name",
  "description": "High-level description of the project",
  "testCommand": "command to run tests (e.g., go test ./..., npm test, pytest)",
  "features": [
    {
      "id": "feat-001",
      "category": "functional|ui|integration|performance|security",
      "priority": "high|medium|low",
      "description": "What this feature does",
      "steps": [
        "Step 1 to verify the feature works",
        "Step 2 to verify",
        "..."
      ],
      "passes": false
    }
  ]
}

## Guidelines

1. **Feature IDs**: Use format "feat-XXX" (e.g., feat-001, feat-002)

2. **Categories**:
   - functional: Core business logic and features
   - ui: User interface components and interactions
   - integration: External service integrations, APIs
   - performance: Speed, efficiency, optimization features
   - security: Authentication, authorization, data protection

3. **Priorities**:
   - high: Must have, core functionality
   - medium: Should have, important but not critical
   - low: Nice to have, can be deferred

4. **Steps**: Each feature should have 2-5 verification steps that describe how to test if the feature works correctly

5. **Test Command**: This is REQUIRED and must be a valid command that can run tests for this project

## Your Approach

1. Ask follow-up questions to understand:
   - The main purpose and users
   - Key features they need
   - Technology stack (to determine test command)
   - Priority of different features
2. Once you understand the project, propose a feature list
3. Iterate with the user until they're satisfied
4. Create the prd.json file with all features having "passes": false

Be thorough but conversational. Help them think through edge cases and important features they might have missed.

IMPORTANT: When the user is satisfied with the plan, you MUST create the prd.json file in the current directory using the Write tool.`
}

// ContainsCompletionSignal checks if the output contains the completion signal
func ContainsCompletionSignal(output string) bool {
	return strings.Contains(output, "<promise>COMPLETE</promise>")
}

// ContainsError checks if the output contains common error patterns
func ContainsError(output string) bool {
	errorPatterns := []string{
		"error:",
		"Error:",
		"ERROR:",
		"fatal:",
		"Fatal:",
		"FATAL:",
		"panic:",
		"failed to",
		"Failed to",
		"FAILED",
	}

	for _, pattern := range errorPatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}
	return false
}
