package commands

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
)

// Updater interface for dependency injection (enables testing)
type Updater interface {
	DetectLatest(ctx context.Context, slug string) (*Release, bool, error)
	UpdateTo(ctx context.Context, release *Release, cmdPath string) error
}

// Release holds version info (abstraction over selfupdate.Release)
type Release struct {
	Version   string
	AssetURL  string
	AssetName string
}

// LessOrEqual checks if this release version is less than or equal to the given version
func (r *Release) LessOrEqual(version string) bool {
	// Simple semver comparison: strip 'v' prefix and compare
	rv := stripV(r.Version)
	cv := stripV(version)

	// Parse versions as major.minor.patch
	rParts := parseVersion(rv)
	cParts := parseVersion(cv)

	// Compare each part
	for i := 0; i < 3; i++ {
		if rParts[i] < cParts[i] {
			return true
		}
		if rParts[i] > cParts[i] {
			return false
		}
	}
	return true // Equal
}

func stripV(v string) string {
	if len(v) > 0 && v[0] == 'v' {
		return v[1:]
	}
	return v
}

func parseVersion(v string) [3]int {
	var parts [3]int
	var idx int
	var num int
	for _, c := range v {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		} else if c == '.' {
			if idx < 3 {
				parts[idx] = num
				idx++
				num = 0
			}
		} else {
			break // Stop at prerelease suffix like -rc.1
		}
	}
	if idx < 3 {
		parts[idx] = num
	}
	return parts
}

// GitHubUpdater is the real implementation using go-selfupdate
type GitHubUpdater struct{}

func (g *GitHubUpdater) DetectLatest(ctx context.Context, slug string) (*Release, bool, error) {
	latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug(slug))
	if err != nil || !found {
		return nil, found, err
	}
	return &Release{
		Version:   latest.Version(),
		AssetURL:  latest.AssetURL,
		AssetName: latest.AssetName,
	}, true, nil
}

func (g *GitHubUpdater) UpdateTo(ctx context.Context, release *Release, cmdPath string) error {
	return selfupdate.UpdateTo(ctx, release.AssetURL, release.AssetName, cmdPath)
}

// Package-level updater (can be replaced in tests)
var updater Updater = &GitHubUpdater{}

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update fogit to the latest version",
	Long:  `Check for and install the latest version of fogit from GitHub Releases.`,
	RunE:  runSelfUpdate,
}

var selfUpdateCheckOnly bool

func init() {
	rootCmd.AddCommand(selfUpdateCmd)
	selfUpdateCmd.Flags().BoolVar(&selfUpdateCheckOnly, "check", false, "Only check for updates, don't install")
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	return doSelfUpdate(cmd.Context(), Version, selfUpdateCheckOnly, updater)
}

// doSelfUpdate contains the testable logic
func doSelfUpdate(ctx context.Context, currentVersion string, checkOnly bool, u Updater) error {
	if currentVersion == "dev" {
		return fmt.Errorf("cannot update development build, please install a release version")
	}

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Println("Checking for updates...")

	latest, found, err := u.DetectLatest(ctx, "eg3r/fogit")
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	if !found {
		fmt.Println("No release found")
		return nil
	}

	if latest.LessOrEqual(currentVersion) {
		fmt.Printf("Already up to date (latest: %s)\n", latest.Version)
		return nil
	}

	fmt.Printf("New version available: %s\n", latest.Version)

	if checkOnly {
		return nil
	}

	fmt.Printf("Downloading %s for %s/%s...\n", latest.Version, runtime.GOOS, runtime.GOARCH)

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to locate executable: %w", err)
	}

	if err := u.UpdateTo(ctx, latest, exe); err != nil {
		// Check if this is a permission error on Windows (Program Files)
		if runtime.GOOS == "windows" && isPermissionError(err, exe) {
			return fmt.Errorf("permission denied: fogit is installed in a protected location (%s)\n\n"+
				"To update, either:\n"+
				"  1. Run PowerShell as Administrator and try again\n"+
				"  2. Download manually from: https://github.com/eg3r/fogit/releases/latest", exe)
		}
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Printf("Successfully updated to %s\n", latest.Version)
	return nil
}

// isPermissionError checks if the error is a permission/access denied error
// and if the executable is in a protected Windows location
func isPermissionError(err error, exePath string) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for common permission error patterns
	if !strings.Contains(errStr, "Access is denied") &&
		!strings.Contains(errStr, "permission denied") &&
		!os.IsPermission(err) {
		return false
	}
	// Check if in protected location
	lowerPath := strings.ToLower(exePath)
	return strings.Contains(lowerPath, "program files") ||
		strings.Contains(lowerPath, "windows") ||
		strings.Contains(lowerPath, "system32")
}
