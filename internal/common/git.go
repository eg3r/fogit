package common

import (
	"os/exec"
	"strings"
)

// TrunkBranchCandidates is the list of common trunk branch names in priority order.
var TrunkBranchCandidates = []string{"main", "master", "trunk"}

// DetectTrunkBranch detects the trunk/main branch using Git's native methods.
// Priority order:
// 1. Remote HEAD reference (refs/remotes/origin/HEAD) - most reliable for cloned repos
// 2. Current branch if it's main or master - common during fresh init
// 3. Git's configured default branch (init.defaultBranch)
// 4. Check if main or master branch exists locally
// Falls back to "main" if nothing found.
func DetectTrunkBranch(repoPath string) string {
	// Method 1: Check remote HEAD (most reliable for cloned repos)
	// This is what GitHub/GitLab set as the default branch
	cmd := exec.Command("git", "-C", repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	if output, err := cmd.Output(); err == nil {
		// Output is like "refs/remotes/origin/main\n"
		ref := strings.TrimSpace(string(output))
		if branch := strings.TrimPrefix(ref, "refs/remotes/origin/"); branch != ref {
			return branch
		}
	}

	// Method 2: Check current branch - if we're on main/master, use that
	cmd = exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if output, err := cmd.Output(); err == nil {
		currentBranch := strings.TrimSpace(string(output))
		if IsTrunkBranch(currentBranch) {
			return currentBranch
		}
	}

	// Method 3: Check Git's configured default branch
	cmd = exec.Command("git", "-C", repoPath, "config", "--get", "init.defaultBranch")
	if output, err := cmd.Output(); err == nil {
		defaultBranch := strings.TrimSpace(string(output))
		if defaultBranch != "" {
			return defaultBranch
		}
	}

	// Method 4: Check if main or master exists locally
	// Note: branch is from a hard-coded list, not user input, so this is safe
	for _, branch := range TrunkBranchCandidates {
		cmd = exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "refs/heads/"+branch) //nolint:gosec // branch is from hard-coded list
		if err := cmd.Run(); err == nil {
			return branch
		}
	}

	// Default to "main" (modern Git default)
	return "main"
}

// IsTrunkBranch checks if the given branch name is a trunk/main branch.
func IsTrunkBranch(branch string) bool {
	for _, candidate := range TrunkBranchCandidates {
		if branch == candidate {
			return true
		}
	}
	return false
}
