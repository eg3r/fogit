package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage Git hooks for FoGit",
	Long: `Install or uninstall Git hooks for FoGit automation.

Git hooks provide automatic validation and trigger FoGit hook scripts.

Installed hooks:
  pre-commit   - Validates features before commit
  post-commit  - Triggers .fogit/hooks/post-commit
  pre-push     - Runs fogit validate before push
  post-merge   - Triggers .fogit/hooks/post-feature-update for affected features`,
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Git hooks",
	Long: `Install Git hooks for FoGit automation.

Hooks are installed to .git/hooks/ and chain to any existing hooks.`,
	RunE: runHooksInstall,
}

var hooksUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Git hooks",
	Long: `Remove FoGit integration from Git hooks.

Existing hooks are preserved; only FoGit integration is removed.
.fogit/hooks/ scripts are not affected (they are shared with the team).`,
	RunE: runHooksUninstall,
}

func init() {
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksUninstallCmd)
	rootCmd.AddCommand(hooksCmd)
}

const fogitHookMarker = "# FOGIT_HOOK_START"
const fogitHookEndMarker = "# FOGIT_HOOK_END"

// Hook templates
var hookTemplates = map[string]string{
	"pre-commit": `#!/bin/sh
%s
# FoGit pre-commit hook: validate features before commit
if command -v fogit >/dev/null 2>&1; then
    fogit validate --quiet 2>/dev/null
    if [ $? -ne 0 ]; then
        echo "FoGit: Feature validation failed. Run 'fogit validate' for details."
        exit 1
    fi
else
    echo "FoGit: Warning - 'fogit' not found in PATH. Skipping validation."
    echo "  Install fogit or run 'fogit hooks uninstall' to remove this hook."
fi
%s
`,
	"post-commit": `#!/bin/sh
%s
# FoGit post-commit hook: trigger .fogit/hooks/post-commit
if [ -x ".fogit/hooks/post-commit" ]; then
    .fogit/hooks/post-commit
fi
%s
`,
	"pre-push": `#!/bin/sh
%s
# FoGit pre-push hook: run validation before push
if command -v fogit >/dev/null 2>&1; then
    fogit validate 2>/dev/null
    if [ $? -ne 0 ]; then
        echo "FoGit: Feature validation failed. Push aborted."
        echo "Run 'fogit validate' for details or 'fogit validate --fix' to attempt auto-repair."
        exit 1
    fi
else
    echo "FoGit: Warning - 'fogit' not found in PATH. Skipping validation."
    echo "  Install fogit or run 'fogit hooks uninstall' to remove this hook."
fi
%s
`,
	"post-merge": `#!/bin/sh
%s
# FoGit post-merge hook: trigger post-feature-update for affected features
# Get list of changed .fogit/features files
changed_features=$(git diff --name-only HEAD@{1} HEAD -- '.fogit/features/*.yml' 2>/dev/null)
if [ -n "$changed_features" ] && [ -x ".fogit/hooks/post-feature-update" ]; then
    echo "$changed_features" | while read feature_file; do
        .fogit/hooks/post-feature-update "$feature_file"
    done
fi
%s
`,
}

func runHooksInstall(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if we're in a Git repository
	gitDir := filepath.Join(cwd, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository (missing .git directory)")
	}

	// Check if FoGit is initialized
	fogitDir := filepath.Join(cwd, ".fogit")
	if _, err := os.Stat(fogitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a FoGit repository (missing .fogit directory)")
	}

	// Create hooks directory if needed
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	fmt.Println("Git hooks installed:")

	existingHooksPreserved := false

	for hookName, template := range hookTemplates {
		hookPath := filepath.Join(hooksDir, hookName)

		var existingContent string
		var existingBefore, existingAfter string

		// Check for existing hook
		if content, err := os.ReadFile(hookPath); err == nil {
			existingContent = string(content)

			// Check if FoGit is already installed
			if strings.Contains(existingContent, fogitHookMarker) {
				// Remove existing FoGit section for reinstall
				existingBefore, existingAfter = extractNonFogitParts(existingContent)
			} else {
				// Preserve existing hook content
				existingBefore = existingContent
				existingHooksPreserved = true
			}
		}

		// Build new hook content
		newContent := fmt.Sprintf(template,
			wrapWithMarkers(existingBefore, "before"),
			wrapExistingHookCall(existingAfter, hookName))

		// Write hook file
		if err := os.WriteFile(hookPath, []byte(newContent), 0755); err != nil { //nolint:gosec // hooks need executable permissions
			return fmt.Errorf("failed to write %s hook: %w", hookName, err)
		}

		fmt.Printf("  ✓ .git/hooks/%s\n", hookName)
	}

	// Print hook descriptions
	fmt.Println()
	fmt.Println("Hook functions:")
	fmt.Println("  pre-commit   → validates features before commit")
	fmt.Println("  post-commit  → triggers .fogit/hooks/post-commit")
	fmt.Println("  pre-push     → runs fogit validate before push")
	fmt.Println("  post-merge   → triggers .fogit/hooks/post-feature-update")

	if existingHooksPreserved {
		fmt.Println()
		fmt.Println("Note: Existing hooks were preserved and will be called after FoGit hooks.")
	}

	fmt.Println()
	fmt.Println("Run 'fogit hooks uninstall' to remove Git hook integration.")

	return nil
}

func runHooksUninstall(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if we're in a Git repository
	gitDir := filepath.Join(cwd, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository (missing .git directory)")
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	fmt.Println("Git hooks removed:")

	for hookName := range hookTemplates {
		hookPath := filepath.Join(hooksDir, hookName)

		content, err := os.ReadFile(hookPath)
		if err != nil {
			continue // Hook doesn't exist
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, fogitHookMarker) {
			continue // Not a FoGit hook
		}

		// Extract non-FoGit parts
		before, after := extractNonFogitParts(contentStr)
		remaining := strings.TrimSpace(before + after)

		if remaining == "" || remaining == "#!/bin/sh" {
			// Hook is empty, remove it
			if err := os.Remove(hookPath); err != nil {
				return fmt.Errorf("failed to remove %s hook: %w", hookName, err)
			}
			fmt.Printf("  ✓ Removed .git/hooks/%s\n", hookName)
		} else {
			// Preserve non-FoGit content
			if err := os.WriteFile(hookPath, []byte(remaining), 0755); err != nil { //nolint:gosec // hooks need executable permissions
				return fmt.Errorf("failed to update %s hook: %w", hookName, err)
			}
			fmt.Printf("  ✓ Removed FoGit integration from .git/hooks/%s\n", hookName)
		}
	}

	fmt.Println()
	fmt.Println(".fogit/hooks/ scripts preserved (shared with team)")

	return nil
}

func extractNonFogitParts(content string) (before, after string) {
	startIdx := strings.Index(content, fogitHookMarker)
	endIdx := strings.Index(content, fogitHookEndMarker)

	if startIdx == -1 {
		return content, ""
	}

	before = content[:startIdx]

	if endIdx != -1 {
		// Find end of line after marker
		endLineIdx := endIdx + len(fogitHookEndMarker)
		if endLineIdx < len(content) && content[endLineIdx] == '\n' {
			endLineIdx++
		}
		after = content[endLineIdx:]
	}

	return strings.TrimSpace(before), strings.TrimSpace(after)
}

func wrapWithMarkers(content, position string) string {
	content = strings.TrimSpace(content)
	if content == "" || content == "#!/bin/sh" {
		return fogitHookMarker
	}

	// Remove shebang from content as we add our own
	if strings.HasPrefix(content, "#!/") {
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) > 1 {
			content = strings.TrimSpace(lines[1])
		} else {
			content = ""
		}
	}

	if content == "" {
		return fogitHookMarker
	}

	return fmt.Sprintf("%s\n# Preserved existing hook (%s):\n%s", fogitHookMarker, position, content)
}

func wrapExistingHookCall(content, _ string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return fogitHookEndMarker
	}

	return fmt.Sprintf("# Continue with preserved hook content:\n%s\n%s", content, fogitHookEndMarker)
}
