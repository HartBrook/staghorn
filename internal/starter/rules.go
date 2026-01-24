package starter

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/HartBrook/staghorn/internal/rules"
)

//go:embed rules/*.md rules/**/*.md
var rulesFS embed.FS

// RuleNames returns the list of available starter rule names (relative paths).
func RuleNames() []string {
	var names []string
	_ = fs.WalkDir(rulesFS, "rules", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			// Get relative path from rules/
			relPath := strings.TrimPrefix(path, "rules/")
			names = append(names, relPath)
		}
		return nil
	})
	return names
}

// BootstrapRules copies starter rules to the target directory.
// It skips files that already exist. Returns the number of rules copied.
func BootstrapRules(targetDir string) (int, error) {
	count, _, err := BootstrapRulesWithSkip(targetDir, nil)
	return count, err
}

// BootstrapRulesWithSkip copies starter rules to the target directory,
// skipping rules in the skip list. Returns the count and names of installed rules.
func BootstrapRulesWithSkip(targetDir string, skip []string) (int, []string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, nil, fmt.Errorf("failed to create rules directory: %w", err)
	}

	// Build skip set
	skipSet := make(map[string]bool)
	for _, name := range skip {
		skipSet[name] = true
	}

	copied := 0
	var installed []string

	err := fs.WalkDir(rulesFS, "rules", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		// Get relative path from rules/
		relPath := strings.TrimPrefix(path, "rules/")

		// Skip if in skip list
		if skipSet[relPath] {
			return nil
		}

		targetPath := filepath.Join(targetDir, relPath)

		// Skip if file already exists
		if _, err := os.Stat(targetPath); err == nil {
			return nil
		}

		content, err := rulesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", relPath, err)
		}

		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", relPath, err)
		}

		copied++
		installed = append(installed, relPath)
		return nil
	})

	if err != nil {
		return copied, installed, err
	}

	return copied, installed, nil
}

// GetRule returns the content of a starter rule by relative path.
func GetRule(relPath string) ([]byte, error) {
	return rulesFS.ReadFile(filepath.Join("rules", relPath))
}

// ListRules returns all embedded rule files.
func ListRules() ([]fs.DirEntry, error) {
	return rulesFS.ReadDir("rules")
}

// LoadStarterRules loads and parses all embedded starter rules.
func LoadStarterRules() ([]*rules.Rule, error) {
	var result []*rules.Rule

	err := fs.WalkDir(rulesFS, "rules", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		content, err := rulesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Get relative path from rules/
		relPath := strings.TrimPrefix(path, "rules/")

		rule, err := rules.Parse(string(content), rules.SourceStarter, "", relPath)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		result = append(result, rule)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
