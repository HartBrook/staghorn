package starter

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HartBrook/staghorn/internal/eval"
)

//go:embed evals/*.yaml
var evalsFS embed.FS

// EvalNames returns the list of available starter eval names.
func EvalNames() []string {
	entries, err := evalsFS.ReadDir("evals")
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			ext := filepath.Ext(entry.Name())
			if ext == ".yaml" || ext == ".yml" {
				name := entry.Name()[:len(entry.Name())-len(ext)]
				names = append(names, name)
			}
		}
	}
	return names
}

// BootstrapEvals copies starter evals to the target directory.
// It skips files that already exist. Returns the number of files copied and their names.
func BootstrapEvals(targetDir string) (int, []string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, nil, fmt.Errorf("failed to create evals directory: %w", err)
	}

	entries, err := evalsFS.ReadDir("evals")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read embedded evals: %w", err)
	}

	copied := 0
	var installed []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		name := entry.Name()[:len(entry.Name())-len(ext)]
		targetPath := filepath.Join(targetDir, entry.Name())

		// Skip if file already exists
		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		content, err := evalsFS.ReadFile(filepath.Join("evals", entry.Name()))
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

// BootstrapEvalsSelective copies only the specified starter evals to the target directory.
func BootstrapEvalsSelective(targetDir string, names []string) (int, []string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, nil, fmt.Errorf("failed to create evals directory: %w", err)
	}

	// Build set of requested names
	requested := make(map[string]bool)
	for _, name := range names {
		requested[name] = true
	}

	entries, err := evalsFS.ReadDir("evals")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read embedded evals: %w", err)
	}

	copied := 0
	var installed []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		name := entry.Name()[:len(entry.Name())-len(ext)]

		// Skip if not in requested list
		if !requested[name] {
			continue
		}

		targetPath := filepath.Join(targetDir, entry.Name())

		// Skip if file already exists
		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		content, err := evalsFS.ReadFile(filepath.Join("evals", entry.Name()))
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

// GetEval returns the content of a starter eval by name.
func GetEval(name string) ([]byte, error) {
	// Try .yaml first, then .yml
	content, err := evalsFS.ReadFile(filepath.Join("evals", name+".yaml"))
	if err != nil {
		content, err = evalsFS.ReadFile(filepath.Join("evals", name+".yml"))
	}
	return content, err
}

// LoadStarterEvals loads and parses all embedded starter evals.
func LoadStarterEvals() ([]*eval.Eval, error) {
	entries, err := evalsFS.ReadDir("evals")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded evals: %w", err)
	}

	var result []*eval.Eval
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		content, err := evalsFS.ReadFile(filepath.Join("evals", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		e, err := eval.Parse(string(content), eval.SourceStarter, "")
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", entry.Name(), err)
		}

		result = append(result, e)
	}

	return result, nil
}
