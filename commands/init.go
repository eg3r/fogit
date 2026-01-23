package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/logger"
	"github.com/eg3r/fogit/internal/setup"
	"github.com/eg3r/fogit/pkg/fogit"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize FoGit repository",
	Long: `Initialize a FoGit repository in the current directory.

Requires a Git repository (run 'git init' first if needed).

Creates the .fogit directory structure:
  .fogit/
    features/    - Feature YAML files
    config.yml   - Repository configuration

Examples:
  # Initialize in current directory
  fogit init

  # Initialize without Git hooks
  fogit init --no-hooks`,
	RunE: runInit,
}

var (
	initNoHooks  bool
	initTemplate string
)

func init() {
	initCmd.Flags().BoolVar(&initNoHooks, "no-hooks", false, "Skip Git hook installation")
	initCmd.Flags().StringVar(&initTemplate, "template", "", "Use configuration template")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if we're in a Git repository (required per spec)
	if _, err := os.Stat(filepath.Join(cwd, ".git")); os.IsNotExist(err) {
		return fogit.NewExitCodeError(
			fmt.Errorf("not a Git repository. Run 'git init' first to create a Git repository"),
			fogit.ExitGitError,
		)
	}

	if err := setup.InitializeRepository(cwd); err != nil {
		return err
	}

	fmt.Printf("Initialized FoGit repository in %s\n", cwd)
	fmt.Println("\nDirectory structure created:")
	fmt.Println("  .fogit/")
	fmt.Println("    features/    - Feature YAML files")
	fmt.Println("    hooks/       - Hook scripts")
	fmt.Println("    metadata/    - Generated indices (gitignored)")
	fmt.Println("    config.yml   - Repository configuration")
	fmt.Println("    .gitignore   - Ignore patterns")

	// Install Git hooks by default (unless --no-hooks)
	if !initNoHooks {
		fmt.Println()
		if err := installGitHooks(cwd); err != nil {
			logger.Warn("failed to install Git hooks", "error", err)
			fmt.Println("You can install them later with 'fogit hooks install'")
		}
	}

	return nil
}

func installGitHooks(cwd string) error {
	// Create hooks directory
	hooksDir := filepath.Join(cwd, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	fmt.Println("Git hooks installed:")

	hooks := map[string]string{
		"pre-commit": `#!/bin/sh
# FOGIT_HOOK_START
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
# FOGIT_HOOK_END
`,
		"post-commit": `#!/bin/sh
# FOGIT_HOOK_START
# FoGit post-commit hook: trigger .fogit/hooks/post-commit
if [ -x ".fogit/hooks/post-commit" ]; then
    .fogit/hooks/post-commit
fi
# FOGIT_HOOK_END
`,
		"pre-push": `#!/bin/sh
# FOGIT_HOOK_START
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
# FOGIT_HOOK_END
`,
		"post-merge": `#!/bin/sh
# FOGIT_HOOK_START
# FoGit post-merge hook: trigger post-feature-update for affected features
changed_features=$(git diff --name-only HEAD@{1} HEAD -- '.fogit/features/*.yml' 2>/dev/null)
if [ -n "$changed_features" ] && [ -x ".fogit/hooks/post-feature-update" ]; then
    echo "$changed_features" | while read feature_file; do
        .fogit/hooks/post-feature-update "$feature_file"
    done
fi
# FOGIT_HOOK_END
`,
	}

	for hookName, content := range hooks {
		hookPath := filepath.Join(hooksDir, hookName)

		// Check for existing hook
		if existingContent, err := os.ReadFile(hookPath); err == nil {
			// Preserve existing hook by prepending
			content = string(existingContent) + "\n" + content
		}

		if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil { //nolint:gosec // hooks need executable permissions
			return fmt.Errorf("failed to write %s hook: %w", hookName, err)
		}
		fmt.Printf("  ✓ .git/hooks/%s\n", hookName)
	}

	fmt.Println()
	fmt.Println("Hook functions:")
	fmt.Println("  pre-commit   → validates features before commit")
	fmt.Println("  post-commit  → triggers .fogit/hooks/post-commit")
	fmt.Println("  pre-push     → runs fogit validate before push")
	fmt.Println("  post-merge   → triggers .fogit/hooks/post-feature-update")
	fmt.Println()
	fmt.Println("Run 'fogit hooks uninstall' to remove Git hook integration.")

	return nil
}
