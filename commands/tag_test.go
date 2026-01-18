package commands

import (
	"testing"
)

func TestTagCommandStructure(t *testing.T) {
	// Verify tag command has expected subcommands
	if tagCmd == nil {
		t.Fatal("tagCmd is nil")
	}

	if tagCmd.Use != "tag" {
		t.Errorf("expected Use 'tag', got %q", tagCmd.Use)
	}

	// Check subcommands exist
	subCmds := tagCmd.Commands()
	if len(subCmds) != 3 {
		t.Errorf("expected 3 subcommands, got %d", len(subCmds))
	}

	// Verify each subcommand
	cmdNames := make(map[string]bool)
	for _, cmd := range subCmds {
		cmdNames[cmd.Use] = true
	}

	expectedCmds := []string{"create <name>", "list", "delete <name>"}
	for _, expected := range expectedCmds {
		if !cmdNames[expected] {
			t.Errorf("missing subcommand: %s", expected)
		}
	}
}

func TestTagCreateCommandArgs(t *testing.T) {
	// Verify create command requires exactly 1 arg
	if tagCreateCmd.Args == nil {
		t.Fatal("tagCreateCmd.Args is nil")
	}

	// Test that command validates args
	err := tagCreateCmd.Args(tagCreateCmd, []string{})
	if err == nil {
		t.Error("expected error for no args")
	}

	err = tagCreateCmd.Args(tagCreateCmd, []string{"v1.0.0"})
	if err != nil {
		t.Errorf("unexpected error for valid args: %v", err)
	}

	err = tagCreateCmd.Args(tagCreateCmd, []string{"v1.0.0", "extra"})
	if err == nil {
		t.Error("expected error for too many args")
	}
}

func TestTagDeleteCommandArgs(t *testing.T) {
	// Verify delete command requires exactly 1 arg
	if tagDeleteCmd.Args == nil {
		t.Fatal("tagDeleteCmd.Args is nil")
	}

	err := tagDeleteCmd.Args(tagDeleteCmd, []string{})
	if err == nil {
		t.Error("expected error for no args")
	}

	err = tagDeleteCmd.Args(tagDeleteCmd, []string{"old-tag"})
	if err != nil {
		t.Errorf("unexpected error for valid args: %v", err)
	}
}

func TestTagListCommandArgs(t *testing.T) {
	// Verify list command takes no args
	if tagListCmd.Args == nil {
		t.Fatal("tagListCmd.Args is nil")
	}

	err := tagListCmd.Args(tagListCmd, []string{})
	if err != nil {
		t.Errorf("unexpected error for no args: %v", err)
	}

	err = tagListCmd.Args(tagListCmd, []string{"extra"})
	if err == nil {
		t.Error("expected error for extra args")
	}
}

func TestTagCreateHasMessageFlag(t *testing.T) {
	flag := tagCreateCmd.Flags().Lookup("message")
	if flag == nil {
		t.Fatal("message flag not found")
	}

	if flag.Shorthand != "m" {
		t.Errorf("expected shorthand 'm', got %q", flag.Shorthand)
	}
}

// Note: Git tag error tests are in internal/git/repository_test.go (TestTagErrors)
