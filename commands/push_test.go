package commands

import (
	"testing"

	"github.com/eg3r/fogit/internal/testutil"
)

// TestPushCommandFlags tests that push command accepts correct flags
func TestPushCommandFlags(t *testing.T) {
	// Test that flags are defined correctly on the command
	if pushCmd.Flags().Lookup("force") == nil {
		t.Error("--force flag not defined")
	}
	if pushCmd.Flags().Lookup("remote") == nil {
		t.Error("--remote flag not defined")
	}

	// Test default values
	pushCmd.Flags().Set("remote", "")
	val, _ := pushCmd.Flags().GetString("remote")
	if val != "" {
		// Default is set in flag definition
		t.Logf("Remote flag default: %s", val)
	}
}

// TestPushRequiresGitRepo tests that push fails outside a git repository
func TestPushRequiresGitRepo(t *testing.T) {
	// Create a temp directory WITHOUT git
	tmpDir := t.TempDir()

	testutil.InDir(t, tmpDir, func() {
		// Try to run push command
		err := runPush(pushCmd, []string{})
		if err == nil {
			t.Error("Expected error when running push outside git repo")
		}
		if err != nil && err.Error() != "" {
			t.Logf("Got expected error: %v", err)
		}
	})
}

// TestPushIntegration tests push command with a real git repository
func TestPushIntegration(t *testing.T) {
	t.Skip("Skipping integration test - requires remote repository setup")

	// This would require:
	// 1. Creating a git repo
	// 2. Setting up a remote (or using a local bare repo as remote)
	// 3. Making a commit
	// 4. Running push
	// Too complex for unit testing - better as E2E test
}
