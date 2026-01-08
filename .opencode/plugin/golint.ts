/**
 * Go Linting Plugin for OpenCode
 * 
 * Automatically runs golangci-lint on Go files after they are edited.
 * This ensures code quality issues are caught immediately and reported
 * back to the AI for fixing.
 */

import type { Plugin } from "@opencode-ai/plugin"

export const GoLintPlugin: Plugin = async ({ $, directory }) => {
  return {
    // Hook into file edit events
    "file.edited": async ({ file }) => {
      // Only lint Go files
      if (!file.endsWith(".go")) {
        return
      }

      try {
        // Run golangci-lint on the specific file
        // Use --fix to auto-fix what can be fixed
        const result = await $`golangci-lint run --fix --timeout=60s ${file}`.quiet()
        
        if (result.exitCode !== 0) {
          // Report lint errors back to the AI
          console.error(`[golint] Lint issues in ${file}:`)
          console.error(result.stderr.toString() || result.stdout.toString())
        }
      } catch (error) {
        // golangci-lint not installed or other error
        // This is non-fatal - just log and continue
        if (error instanceof Error && error.message.includes("not found")) {
          console.warn("[golint] golangci-lint not found. Run 'make setup' to install.")
        }
      }
    },

    // Also run lint after tool execution completes (batch operations)
    "tool.execute.after": async (input, output) => {
      // Only care about write/edit operations on Go files
      if (input.tool !== "write" && input.tool !== "edit") {
        return
      }

      const filePath = output.args?.file_path || output.args?.filePath
      if (!filePath || !filePath.endsWith(".go")) {
        return
      }

      try {
        // Run goimports to fix imports (gofmt already runs via formatter)
        await $`goimports -w ${filePath}`.quiet()
        
        // Run golangci-lint with auto-fix
        const result = await $`golangci-lint run --fix --timeout=60s ${filePath}`.quiet()
        
        if (result.exitCode !== 0 && result.stdout.toString().trim()) {
          // Report unfixed lint issues
          console.error(`[golint] Remaining lint issues in ${filePath}:`)
          console.error(result.stdout.toString())
        }
      } catch {
        // Silently ignore - tools may not be installed
      }
    },
  }
}

// Default export for single-function plugins
export default GoLintPlugin
