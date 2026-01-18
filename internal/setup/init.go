package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/pkg/fogit"
)

// InitializeRepository initializes a new FoGit repository
func InitializeRepository(cwd string) error {
	fogitDir := filepath.Join(cwd, ".fogit")

	// Check if already initialized
	if _, err := os.Stat(fogitDir); err == nil {
		return fmt.Errorf("fogit repository already initialized in %s", cwd)
	}

	// Create .fogit directory
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		return fmt.Errorf("failed to create .fogit directory: %w", err)
	}

	// Create features directory
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		return fmt.Errorf("failed to create features directory: %w", err)
	}

	// Create hooks directory
	hooksDir := filepath.Join(fogitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Create metadata directory
	metadataDir := filepath.Join(fogitDir, "metadata")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Create default config file using spec-compliant defaults
	defaultCfg := fogit.DefaultConfig()
	defaultCfg.Repository.Name = filepath.Base(cwd)

	if err := config.Save(fogitDir, defaultCfg); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	// Create .gitignore
	gitignorePath := filepath.Join(fogitDir, ".gitignore")
	gitignoreContent := `# Ignore generated metadata and caches
metadata/
*.cache
`
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0600); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	return nil
}
