// Package starter provides embedded starter commands that ship with staghorn.
package starter

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/HartBrook/staghorn/internal/commands"
)

//go:embed commands/*.md
var commandsFS embed.FS

// CommandNames returns the list of available starter command names.
func CommandNames() []string {
	entries, err := commandsFS.ReadDir("commands")
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			name := entry.Name()[:len(entry.Name())-3] // strip .md
			names = append(names, name)
		}
	}
	return names
}

// BootstrapCommands copies starter commands to the target directory.
// It skips files that already exist. Returns the number of commands copied.
func BootstrapCommands(targetDir string) (int, error) {
	count, _, err := BootstrapCommandsWithSkip(targetDir, nil)
	return count, err
}

// BootstrapCommandsWithSkip copies starter commands to the target directory,
// skipping commands in the skip list. Returns the count and names of installed commands.
func BootstrapCommandsWithSkip(targetDir string, skip []string) (int, []string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, nil, fmt.Errorf("failed to create commands directory: %w", err)
	}

	// Build skip set
	skipSet := make(map[string]bool)
	for _, name := range skip {
		skipSet[name] = true
	}

	entries, err := commandsFS.ReadDir("commands")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read embedded commands: %w", err)
	}

	copied := 0
	var installed []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Get command name (strip .md extension)
		name := entry.Name()
		if filepath.Ext(name) == ".md" {
			name = name[:len(name)-3]
		}

		// Skip if in skip list
		if skipSet[name] {
			continue
		}

		targetPath := filepath.Join(targetDir, entry.Name())

		// Skip if file already exists
		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		content, err := commandsFS.ReadFile(filepath.Join("commands", entry.Name()))
		if err != nil {
			return copied, installed, fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return copied, installed, fmt.Errorf("failed to write %s: %w", entry.Name(), err)
		}

		copied++
		installed = append(installed, name)
	}

	return copied, installed, nil
}

// GetCommand returns the content of a starter command by name.
func GetCommand(name string) ([]byte, error) {
	return commandsFS.ReadFile(filepath.Join("commands", name+".md"))
}

// ListCommands returns all embedded command files.
func ListCommands() ([]fs.DirEntry, error) {
	return commandsFS.ReadDir("commands")
}

// LoadStarterCommands loads and parses all embedded starter commands.
func LoadStarterCommands() ([]*commands.Command, error) {
	entries, err := commandsFS.ReadDir("commands")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded commands: %w", err)
	}

	var result []*commands.Command
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		content, err := commandsFS.ReadFile(filepath.Join("commands", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		cmd, err := commands.Parse(string(content), commands.SourceStarter, "")
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", entry.Name(), err)
		}

		result = append(result, cmd)
	}

	return result, nil
}
