package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/logger"
	"github.com/eg3r/fogit/pkg/fogit"
)

// Build information - injected via ldflags at build time
var (
	// Version is the semantic version (e.g., "1.0.0")
	Version = "dev"
	// Commit is the git commit SHA
	Commit = "none"
	// Date is the build date
	Date = "unknown"
)

var (
	// globalConfig is loaded before each command runs
	globalConfig *fogit.Config

	// Global flags
	debugMode   bool
	verboseMode bool

	// workDir is the directory override from -C flag (empty means use cwd)
	workDir string

	// originalDir stores the directory before -C flag is applied (for restoration)
	originalDir string
)

// BuildVersion returns the full version string
func BuildVersion() string {
	return fmt.Sprintf("%s (commit: %.7s, built: %s)", Version, Commit, Date)
}

var rootCmd = &cobra.Command{
	Use:           "fogit",
	Short:         "FoGit (feature oriented git)",
	Long:          `FoGit (feature oriented git) - Git-native feature tracking system`,
	Version:       Version,
	SilenceUsage:  true, // Don't print usage on error - we handle this ourselves
	SilenceErrors: true, // Don't print errors - main.go handles error output
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Handle -C flag: resolve and change to specified directory
		if workDir != "" {
			// Save original directory for restoration in PersistentPostRunE
			var err error
			originalDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			// Resolve relative paths from current directory
			absPath, err := filepath.Abs(workDir)
			if err != nil {
				return fmt.Errorf("invalid path for -C: %w", err)
			}

			// Verify directory exists
			info, err := os.Stat(absPath)
			if err != nil {
				return fmt.Errorf("cannot access directory '%s': %w", workDir, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("'%s' is not a directory", workDir)
			}

			// Change to the specified directory
			if err := os.Chdir(absPath); err != nil {
				return fmt.Errorf("failed to change to directory '%s': %w", workDir, err)
			}

			// Reset workDir for potential chained -C calls
			workDir = ""
		}

		// Initialize logger based on flags
		initLogger()

		logger.Debug("starting command",
			"command", cmd.Name(),
			"args", args,
			"version", Version,
		)

		// Skip config loading for init and help commands
		if cmd.Name() == "init" || cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		// Find .fogit directory
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		fogitDir := filepath.Join(cwd, ".fogit")

		// Check if .fogit exists
		if _, statErr := os.Stat(fogitDir); os.IsNotExist(statErr) {
			// For some commands like init, it's OK if .fogit doesn't exist
			// But we should still try to provide defaults
			globalConfig = fogit.DefaultConfig()
			return nil
		}

		// Load config
		cfg, err := config.Load(fogitDir)
		if err != nil {
			// If config fails to load, use defaults and continue
			// (don't fail the command, just warn)
			logger.Warn("failed to load config, using defaults", "error", err, "path", fogitDir)
			globalConfig = fogit.DefaultConfig()
			return nil
		}

		logger.Debug("config loaded", "path", fogitDir)
		globalConfig = cfg
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Restore original directory if -C flag was used
		if originalDir != "" {
			if err := os.Chdir(originalDir); err != nil {
				return fmt.Errorf("failed to restore directory to '%s': %w", originalDir, err)
			}
			originalDir = ""
		}
		return nil
	},
}

// Execute runs the root command
func Execute() error {
	err := rootCmd.Execute()
	// Always restore original directory after command completes (even on error)
	// This is critical because PersistentPostRunE is not called on command failure
	restoreOriginalDir()
	return err
}

// ExecuteContext runs the root command with the provided context.
// This enables signal handling for graceful shutdown - when the context
// is canceled (e.g., via Ctrl+C), the cancellation propagates to all
// commands via cmd.Context().
func ExecuteContext(ctx context.Context) error {
	err := rootCmd.ExecuteContext(ctx)
	// Always restore original directory after command completes (even on error)
	restoreOriginalDir()
	return err
}

// restoreOriginalDir restores the working directory to the original location
// if it was changed by the -C flag. This must be called after command execution.
func restoreOriginalDir() {
	if originalDir != "" {
		_ = os.Chdir(originalDir) // Best effort - ignore errors
		originalDir = ""
	}
}

// ExecuteRootCmd executes the root command with the given arguments.
// This is for testing purposes - it ensures proper cleanup of directory state.
// Tests should use this instead of rootCmd.Execute() directly.
func ExecuteRootCmd() error {
	err := rootCmd.Execute()
	restoreOriginalDir()
	return err
}

// ResetFlags resets global flags to their default values.
// This is useful for testing where flags may persist between test runs.
func ResetFlags() {
	workDir = ""
	originalDir = ""
	debugMode = false
	verboseMode = false
	globalConfig = nil

	// Reset all persistent flags on rootCmd
	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		_ = f.Value.Set(f.DefValue)
	})

	// Reset persistent flags
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		_ = f.Value.Set(f.DefValue)
	})

	// Reset flags on all subcommands (recursively)
	resetCommandFlags(rootCmd)
}

// resetCommandFlags recursively resets flags on a command and all its subcommands
func resetCommandFlags(cmd *cobra.Command) {
	for _, subCmd := range cmd.Commands() {
		subCmd.Flags().VisitAll(func(f *pflag.Flag) {
			f.Changed = false
			// For slices, we need to explicitly clear them
			// as Set("") doesn't work properly for slice types
			if sv, ok := f.Value.(pflag.SliceValue); ok {
				_ = sv.Replace([]string{})
			} else {
				_ = f.Value.Set(f.DefValue)
			}
		})
		// Recursively reset nested subcommands
		resetCommandFlags(subCmd)
	}
}

func init() {
	// Set version template to include build info
	rootCmd.SetVersionTemplate(fmt.Sprintf("fogit %s\n", BuildVersion()))

	// Disable auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&workDir, "directory", "C", "", "Run as if fogit was started in `<path>` instead of the current directory")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging (shows all diagnostic messages)")
	rootCmd.PersistentFlags().BoolVar(&verboseMode, "verbose", false, "Enable verbose output (shows info-level messages)")

	// Per spec: -v is shorthand for --version
	rootCmd.Flags().BoolP("version", "v", false, "version for fogit")
}

// GetConfig returns the loaded global config (available to all commands)
func GetConfig() *fogit.Config {
	if globalConfig == nil {
		return fogit.DefaultConfig()
	}
	return globalConfig
}

// initLogger initializes the logger based on command-line flags.
func initLogger() {
	level := logger.LevelWarn // Default: only warnings and errors

	if debugMode {
		level = logger.LevelDebug
	} else if verboseMode {
		level = logger.LevelInfo
	}

	logger.Init(logger.Options{
		Level: level,
	})
}
