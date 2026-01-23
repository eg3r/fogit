package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var (
	ErrNotGitRepo      = errors.New("not a git repository")
	ErrNoCommits       = errors.New("no commits found")
	ErrEmptyRepository = errors.New("repository has no commits")
	ErrBranchExists    = errors.New("branch already exists")
	ErrNothingToCommit = errors.New("nothing to commit")
	ErrNoRemote        = errors.New("no remote configured")
	ErrTagExists       = errors.New("tag already exists")
	ErrTagNotFound     = errors.New("tag not found")
)

// gitRefPattern validates git ref names (branches, tags)
// Disallows shell metacharacters and control characters
var gitRefPattern = regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)

// isValidGitRef validates a git ref name to prevent command injection
func isValidGitRef(ref string) bool {
	if ref == "" || len(ref) > 255 {
		return false
	}
	// Check for dangerous patterns
	if strings.HasPrefix(ref, "-") || strings.HasPrefix(ref, ".") {
		return false
	}
	if strings.Contains(ref, "..") || strings.Contains(ref, "~") || strings.Contains(ref, "^") {
		return false
	}
	return gitRefPattern.MatchString(ref)
}

// Repository wraps go-git repository operations
type Repository struct {
	repo *git.Repository
	path string
}

// OpenRepository opens a Git repository at the given path
func OpenRepository(path string) (*Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, ErrNotGitRepo
		}
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	return &Repository{
		repo: repo,
		path: path,
	}, nil
}

// GetCurrentBranch returns the name of the current branch
func (r *Repository) GetCurrentBranch() (string, error) {
	head, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !head.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not a branch")
	}

	return head.Name().Short(), nil
}

// GetAuthorsForFile returns all unique authors who modified a file
func (r *Repository) GetAuthorsForFile(filepath string) ([]string, error) {
	commits, err := r.repo.Log(&git.LogOptions{
		FileName: &filepath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	authorsMap := make(map[string]bool)
	var authors []string

	err = commits.ForEach(func(c *object.Commit) error {
		email := c.Author.Email
		if !authorsMap[email] {
			authorsMap[email] = true
			authors = append(authors, email)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	if len(authors) == 0 {
		return nil, ErrNoCommits
	}

	return authors, nil
}

// CreateBranch creates a new branch from the current HEAD
func (r *Repository) CreateBranch(name string) error {
	head, err := r.repo.Head()
	if err != nil {
		// Check if this is an empty repository (no commits yet)
		if err.Error() == "reference not found" {
			return ErrEmptyRepository
		}
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(name)

	// Check if branch already exists
	_, err = r.repo.Reference(refName, true)
	if err == nil {
		return ErrBranchExists
	}

	ref := plumbing.NewHashReference(refName, head.Hash())
	err = r.repo.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	return nil
}

// CheckoutBranch switches to the specified branch.
// Uses native git to avoid CRLF issues with go-git on Windows.
func (r *Repository) CheckoutBranch(name string) error {
	cmd := exec.Command("git", "checkout", name)
	cmd.Dir = r.path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// Commit creates a new commit with the given message
func (r *Repository) Commit(message string, author *object.Signature) (string, error) {
	// Check if there are changes to commit using native git
	// (go-git's Status().IsClean() has CRLF issues on Windows)
	changedFiles, err := r.GetChangedFiles()
	if err != nil {
		return "", fmt.Errorf("failed to check status: %w", err)
	}

	if len(changedFiles) == 0 {
		return "", ErrNothingToCommit
	}

	w, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	err = w.AddGlob(".")
	if err != nil {
		return "", fmt.Errorf("failed to add changes: %w", err)
	}

	// Create commit
	commitOpts := &git.CommitOptions{}
	if author != nil {
		commitOpts.Author = author
	}

	hash, err := w.Commit(message, commitOpts)
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	return hash.String(), nil
}

// GetStatus returns the status of the working tree.
// NOTE: This uses go-git's Status() which has known issues with core.autocrlf
// on Windows. For checking if there are uncommitted changes, use GetChangedFiles()
// instead which uses native git.
func (r *Repository) GetStatus() (map[string]git.StatusCode, error) {
	w, err := r.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	result := make(map[string]git.StatusCode)
	for file, fileStatus := range status {
		result[file] = fileStatus.Staging
	}

	return result, nil
}

// GetChangedFiles returns files that have been modified (uncommitted changes).
// Uses native git to properly handle core.autocrlf and other settings that
// go-git doesn't fully support, especially on Windows.
func (r *Repository) GetChangedFiles() ([]string, error) {
	// Use native git status --porcelain for accurate results
	// go-git's Status() doesn't handle core.autocrlf properly on Windows,
	// causing false positives when comparing CRLF working tree files
	// against LF-normalized index entries.
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = r.path

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var files []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		// Format: XY filename (where X=staging, Y=worktree)
		// Extract filename starting at position 3
		file := strings.TrimSpace(line[3:])
		if file != "" {
			files = append(files, file)
		}
	}

	return files, nil
}

// ErrMergeConflict is returned when a merge has conflicts
var ErrMergeConflict = errors.New("merge conflict")

// MergeBranch merges the specified branch into the current branch using native git.
// This uses the git CLI because go-git doesn't have robust merge support.
// Returns ErrMergeConflict if there are conflicts, nil on success.
func (r *Repository) MergeBranch(branchName string) error {
	cmd := exec.Command("git", "merge", branchName, "--no-edit")
	cmd.Dir = r.path

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a merge conflict
		if strings.Contains(string(output), "CONFLICT") || strings.Contains(string(output), "Automatic merge failed") {
			return ErrMergeConflict
		}
		return fmt.Errorf("merge failed: %s", output)
	}

	return nil
}

// MergeBranchSquash merges the specified branch with squash into the current branch.
// Returns ErrMergeConflict if there are conflicts, nil on success.
func (r *Repository) MergeBranchSquash(branchName string) error {
	cmd := exec.Command("git", "merge", "--squash", branchName)
	cmd.Dir = r.path

	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "CONFLICT") || strings.Contains(string(output), "Automatic merge failed") {
			return ErrMergeConflict
		}
		return fmt.Errorf("squash merge failed: %s", output)
	}

	return nil
}

// IsMerging checks if Git is currently in a merge state
func (r *Repository) IsMerging() bool {
	// Check for .git/MERGE_HEAD file which exists during a merge
	mergeHeadPath := filepath.Join(r.path, ".git", "MERGE_HEAD")
	_, err := os.Stat(mergeHeadPath)
	return err == nil
}

// HasConflicts checks if the working tree has unresolved merge conflicts
func (r *Repository) HasConflicts() (bool, error) {
	cmd := exec.Command("git", "diff", "--check")
	cmd.Dir = r.path

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Exit code 2 means there are conflicts
		if strings.Contains(string(output), "conflict") {
			return true, nil
		}
		// Non-zero exit can also mean conflicts found
		return len(output) > 0, nil
	}
	return false, nil
}

// AbortMerge aborts the current merge operation
func (r *Repository) AbortMerge() error {
	cmd := exec.Command("git", "merge", "--abort")
	cmd.Dir = r.path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to abort merge: %s", output)
	}
	return nil
}

// DeleteBranch deletes the specified branch
func (r *Repository) DeleteBranch(name string) error {
	refName := plumbing.NewBranchReferenceName(name)

	err := r.repo.Storer.RemoveReference(refName)
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	return nil
}

// Push pushes the current branch to the remote
func (r *Repository) Push(remoteName string) error {
	err := r.repo.Push(&git.PushOptions{
		RemoteName: remoteName,
	})
	if err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil // Not an error, just up to date
		}
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// GetRemotes returns the list of configured remotes
func (r *Repository) GetRemotes() ([]string, error) {
	remotes, err := r.repo.Remotes()
	if err != nil {
		return nil, fmt.Errorf("failed to get remotes: %w", err)
	}

	var names []string
	for _, remote := range remotes {
		names = append(names, remote.Config().Name)
	}

	return names, nil
}

// GetRepositoryName returns the repository name from the remote URL
func (r *Repository) GetRepositoryName() string {
	remotes, err := r.repo.Remotes()
	if err != nil || len(remotes) == 0 {
		return ""
	}

	// Try to get name from origin remote first
	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			urls := remote.Config().URLs
			if len(urls) > 0 {
				return extractRepoNameFromURL(urls[0])
			}
		}
	}

	// Fallback to first remote
	urls := remotes[0].Config().URLs
	if len(urls) > 0 {
		return extractRepoNameFromURL(urls[0])
	}

	return ""
}

// extractRepoNameFromURL extracts the repository name from a Git URL
func extractRepoNameFromURL(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle different URL formats
	// SSH: git@github.com:owner/repo
	// HTTPS: https://github.com/owner/repo
	// File: /path/to/repo

	// Extract the last path component
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		// Handle SSH format (git@github.com:owner/repo)
		if colonIndex := strings.LastIndex(name, ":"); colonIndex != -1 {
			name = name[colonIndex+1:]
		}
		return name
	}

	return ""
}

// GetUserConfig returns the Git user name and email from config
func (r *Repository) GetUserConfig() (name, email string, err error) {
	cfg, err := r.repo.Config()
	if err != nil {
		return "", "", fmt.Errorf("failed to get config: %w", err)
	}

	return cfg.User.Name, cfg.User.Email, nil
}

// IsGitRepository checks if the given path is inside a Git repository
func IsGitRepository(path string) bool {
	// Walk up the directory tree looking for .git
	currentPath := path
	for {
		gitPath := filepath.Join(currentPath, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return true
		}

		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			// Reached root
			break
		}
		currentPath = parent
	}
	return false
}

// FindGitRoot finds the root directory of the Git repository
func FindGitRoot(startPath string) (string, error) {
	currentPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	for {
		gitPath := filepath.Join(currentPath, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return currentPath, nil
		}

		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			// Reached root
			return "", ErrNotGitRepo
		}
		currentPath = parent
	}
}

// GetCommitHistory returns commit history for a file
type CommitInfo struct {
	Hash      string
	Author    string
	Email     string
	Message   string
	Timestamp time.Time
}

func (r *Repository) GetCommitHistory(filepath string, limit int) ([]CommitInfo, error) {
	commits, err := r.repo.Log(&git.LogOptions{
		FileName: &filepath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	var history []CommitInfo
	count := 0

	err = commits.ForEach(func(c *object.Commit) error {
		if limit > 0 && count >= limit {
			return fmt.Errorf("limit reached") // Stop iteration
		}

		history = append(history, CommitInfo{
			Hash:      c.Hash.String(),
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			Message:   strings.TrimSpace(c.Message),
			Timestamp: c.Author.When,
		})

		count++
		return nil
	})

	if err != nil && err.Error() != "limit reached" {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return history, nil
}

// CommitLog represents a commit entry for display
type CommitLog struct {
	Hash    string
	Author  string
	Email   string
	Date    time.Time
	Message string
	Files   int
}

// GetLog returns filtered commit log
// path: optional file path to filter commits (relative to repo root, e.g., ".fogit/features/auth.yml")
// author: optional author name/email to filter
// since: optional start date to filter
// limit: maximum number of commits (0 = no limit)
func (r *Repository) GetLog(path string, author string, since *time.Time, limit int) ([]CommitLog, error) {
	// Build log options
	logOpts := &git.LogOptions{}

	if path != "" {
		logOpts.FileName = &path
	}

	commits, err := r.repo.Log(logOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	var logs []CommitLog
	count := 0

	err = commits.ForEach(func(c *object.Commit) error {
		// Check limit
		if limit > 0 && count >= limit {
			return fmt.Errorf("limit reached")
		}

		// Filter by author
		if author != "" {
			if !strings.Contains(c.Author.Name, author) && !strings.Contains(c.Author.Email, author) {
				return nil // Skip this commit
			}
		}

		// Filter by date
		if since != nil && c.Author.When.Before(*since) {
			return nil // Skip this commit
		}

		// Count files changed
		fileCount := 0
		if c.NumParents() > 0 {
			parent, err := c.Parent(0)
			if err == nil {
				patch, err := parent.Patch(c)
				if err == nil {
					fileCount = len(patch.FilePatches())
				}
			}
		} else {
			// First commit - count all files
			tree, err := c.Tree()
			if err == nil {
				_ = tree.Files().ForEach(func(_ *object.File) error {
					fileCount++
					return nil
				})
			}
		}

		logs = append(logs, CommitLog{
			Hash:    c.Hash.String(),
			Author:  fmt.Sprintf("%s <%s>", c.Author.Name, c.Author.Email),
			Email:   c.Author.Email,
			Date:    c.Author.When,
			Message: strings.TrimSpace(c.Message),
			Files:   fileCount,
		})

		count++
		return nil
	})

	if err != nil && err.Error() != "limit reached" {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return logs, nil
}

// TagInfo represents a Git tag
type TagInfo struct {
	Name    string
	Hash    string
	Message string
	Tagger  string
	Date    time.Time
	IsLight bool // true if lightweight tag (no message/tagger)
}

// CreateTag creates an annotated tag at the current HEAD
func (r *Repository) CreateTag(name, message string) error {
	head, err := r.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Check if tag already exists
	_, err = r.repo.Tag(name)
	if err == nil {
		return ErrTagExists
	}

	// Get user config for tagger
	cfg, err := r.repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	tagger := &object.Signature{
		Name:  cfg.User.Name,
		Email: cfg.User.Email,
		When:  time.Now(),
	}

	// If no user configured, use defaults
	if tagger.Name == "" {
		tagger.Name = "FoGit"
	}
	if tagger.Email == "" {
		tagger.Email = "fogit@localhost"
	}

	// Create annotated tag
	_, err = r.repo.CreateTag(name, head.Hash(), &git.CreateTagOptions{
		Tagger:  tagger,
		Message: message,
	})
	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	return nil
}

// ListTags returns all tags in the repository
func (r *Repository) ListTags() ([]TagInfo, error) {
	tagRefs, err := r.repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	var tags []TagInfo

	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		tagName := ref.Name().Short()

		// Try to get annotated tag object
		tagObj, err := r.repo.TagObject(ref.Hash())
		if err != nil {
			// Lightweight tag - no tag object, just a commit reference
			commit, err := r.repo.CommitObject(ref.Hash())
			if err != nil {
				// Could be a tag pointing to something else, skip
				return nil
			}
			tags = append(tags, TagInfo{
				Name:    tagName,
				Hash:    ref.Hash().String()[:7],
				IsLight: true,
				Date:    commit.Author.When,
			})
			return nil
		}

		// Annotated tag
		tags = append(tags, TagInfo{
			Name:    tagName,
			Hash:    tagObj.Target.String()[:7],
			Message: strings.TrimSpace(tagObj.Message),
			Tagger:  fmt.Sprintf("%s <%s>", tagObj.Tagger.Name, tagObj.Tagger.Email),
			Date:    tagObj.Tagger.When,
			IsLight: false,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate tags: %w", err)
	}

	return tags, nil
}

// DeleteTag removes a tag from the repository
func (r *Repository) DeleteTag(name string) error {
	refName := plumbing.NewTagReferenceName(name)

	// Check if tag exists
	_, err := r.repo.Reference(refName, true)
	if err != nil {
		return ErrTagNotFound
	}

	err = r.repo.Storer.RemoveReference(refName)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	return nil
}

// GetTag returns information about a specific tag
func (r *Repository) GetTag(name string) (*TagInfo, error) {
	refName := plumbing.NewTagReferenceName(name)

	ref, err := r.repo.Reference(refName, true)
	if err != nil {
		return nil, ErrTagNotFound
	}

	// Try to get annotated tag object
	tagObj, err := r.repo.TagObject(ref.Hash())
	if err != nil {
		// Lightweight tag
		commit, err := r.repo.CommitObject(ref.Hash())
		if err != nil {
			return nil, fmt.Errorf("failed to get commit for tag: %w", err)
		}
		return &TagInfo{
			Name:    name,
			Hash:    ref.Hash().String()[:7],
			IsLight: true,
			Date:    commit.Author.When,
		}, nil
	}

	// Annotated tag
	return &TagInfo{
		Name:    name,
		Hash:    tagObj.Target.String()[:7],
		Message: strings.TrimSpace(tagObj.Message),
		Tagger:  fmt.Sprintf("%s <%s>", tagObj.Tagger.Name, tagObj.Tagger.Email),
		Date:    tagObj.Tagger.When,
		IsLight: false,
	}, nil
}

// ============================================================================
// Cross-Branch Feature Discovery Methods
// Per spec/specification/07-git-integration.md#cross-branch-feature-discovery
// ============================================================================

var (
	ErrBranchNotFound = errors.New("branch not found")
	ErrFileNotFound   = errors.New("file not found on branch")
)

// ListBranches returns all local branch names
func (r *Repository) ListBranches() ([]string, error) {
	branches, err := r.repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var names []string
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		names = append(names, ref.Name().Short())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}

	return names, nil
}

// ListRemoteBranches returns all remote tracking branch names
// Returns branch names in format "origin/branch-name"
func (r *Repository) ListRemoteBranches() ([]string, error) {
	refs, err := r.repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	var names []string
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsRemote() {
			names = append(names, ref.Name().Short())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate references: %w", err)
	}

	return names, nil
}

// ListFilesOnBranch lists files in a path on a specific branch without checkout.
// Uses: git ls-tree -r --name-only <branch> -- <path>
// Returns relative paths of files found under the given path.
func (r *Repository) ListFilesOnBranch(branch, path string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", branch, "--", path)
	cmd.Dir = r.path

	output, err := cmd.Output()
	if err != nil {
		// Check if it's an exit error with specific message
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if strings.Contains(stderr, "Not a valid object name") ||
				strings.Contains(stderr, "not a tree object") {
				return nil, ErrBranchNotFound
			}
		}
		return nil, fmt.Errorf("failed to list files on branch %s: %w", branch, err)
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return []string{}, nil
	}

	files := strings.Split(outputStr, "\n")
	return files, nil
}

// ReadFileOnBranch reads a file from a specific branch without checkout.
// Uses: git show <branch>:<path>
func (r *Repository) ReadFileOnBranch(branch, path string) ([]byte, error) {
	// Validate branch name to prevent command injection
	if !isValidGitRef(branch) {
		return nil, fmt.Errorf("invalid branch name: %s", branch)
	}

	// Normalize path separators for git (always use forward slashes)
	normalizedPath := strings.ReplaceAll(path, "\\", "/")

	// Validate path to prevent traversal attacks
	if strings.Contains(normalizedPath, "..") {
		return nil, fmt.Errorf("invalid path: contains parent directory reference")
	}

	// Build the git show argument safely
	refSpec := fmt.Sprintf("%s:%s", branch, normalizedPath)
	cmd := exec.Command("git", "show", refSpec) // #nosec G204 - inputs validated above
	cmd.Dir = r.path

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.ToLower(string(output))
		// Check for file not found errors
		if strings.Contains(outputStr, "does not exist") ||
			strings.Contains(outputStr, "path") && strings.Contains(outputStr, "exist") ||
			strings.Contains(outputStr, "exists on disk, but not in") {
			return nil, ErrFileNotFound
		}
		// Check for branch not found errors
		if strings.Contains(outputStr, "not a valid object name") ||
			strings.Contains(outputStr, "invalid object name") ||
			strings.Contains(outputStr, "unknown revision") ||
			strings.Contains(outputStr, "bad revision") {
			return nil, ErrBranchNotFound
		}
		return nil, fmt.Errorf("failed to read file %s on branch %s: %w", path, branch, err)
	}

	return output, nil
}

// GetTrunkBranch returns the main/master branch name.
// Checks for existence of common trunk branch names in order: main, master.
// Falls back to "main" if neither exists.
func (r *Repository) GetTrunkBranch() (string, error) {
	branches, err := r.ListBranches()
	if err != nil {
		return "", err
	}

	// Check for common trunk branch names
	trunkCandidates := []string{"main", "master"}
	branchSet := make(map[string]bool)
	for _, b := range branches {
		branchSet[b] = true
	}

	for _, candidate := range trunkCandidates {
		if branchSet[candidate] {
			return candidate, nil
		}
	}

	// Check remote branches as fallback
	remoteBranches, err := r.ListRemoteBranches()
	if err == nil {
		for _, candidate := range trunkCandidates {
			for _, remote := range remoteBranches {
				if strings.HasSuffix(remote, "/"+candidate) {
					return candidate, nil
				}
			}
		}
	}

	// Default to "main" if nothing found
	return "main", nil
}

// BranchExists checks if a branch exists locally
func (r *Repository) BranchExists(name string) bool {
	refName := plumbing.NewBranchReferenceName(name)
	_, err := r.repo.Reference(refName, true)
	return err == nil
}

// RemoteBranchExists checks if a remote branch exists
func (r *Repository) RemoteBranchExists(remoteBranch string) bool {
	// remoteBranch should be in format "origin/branch-name"
	refName := plumbing.NewRemoteReferenceName("origin", strings.TrimPrefix(remoteBranch, "origin/"))
	_, err := r.repo.Reference(refName, true)
	return err == nil
}
