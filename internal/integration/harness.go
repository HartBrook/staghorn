// Package integration provides integration testing utilities for staghorn.
package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/language"
	"github.com/HartBrook/staghorn/internal/merge"
	"gopkg.in/yaml.v3"
)

// TestEnv provides an isolated test environment with overridden paths.
type TestEnv struct {
	t         *testing.T
	RootDir   string        // t.TempDir() root
	HomeDir   string        // Simulated $HOME
	ConfigDir string        // ~/.config/staghorn
	CacheDir  string        // ~/.cache/staghorn
	ClaudeDir string        // ~/.claude
	Paths     *config.Paths // Configured paths pointing to temp dirs
}

// NewTestEnv creates an isolated test environment.
// All paths are configured to use temporary directories.
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	rootDir := t.TempDir()
	homeDir := filepath.Join(rootDir, "home")
	configDir := filepath.Join(homeDir, ".config", "staghorn")
	cacheDir := filepath.Join(homeDir, ".cache", "staghorn")
	claudeDir := filepath.Join(homeDir, ".claude")

	// Create directory structure
	for _, dir := range []string{configDir, cacheDir, claudeDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	paths := config.NewPathsWithOverrides(configDir, cacheDir)

	return &TestEnv{
		t:         t,
		RootDir:   rootDir,
		HomeDir:   homeDir,
		ConfigDir: configDir,
		CacheDir:  cacheDir,
		ClaudeDir: claudeDir,
		Paths:     paths,
	}
}

// Cleanup is a no-op retained for API compatibility.
// Temporary directories are automatically cleaned up by t.TempDir().
func (e *TestEnv) Cleanup() {}

// SetupTeamConfig writes team CLAUDE.md to cache (simulates fetch).
func (e *TestEnv) SetupTeamConfig(owner, repo, content string) error {
	cacheFile := e.Paths.CacheFile(owner, repo)
	return os.WriteFile(cacheFile, []byte(content), 0644)
}

// SetupTeamLanguage writes a team language config to cache.
func (e *TestEnv) SetupTeamLanguage(owner, repo, lang, content string) error {
	langDir := e.Paths.TeamLanguagesDir(owner, repo)
	if err := os.MkdirAll(langDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(langDir, lang+".md"), []byte(content), 0644)
}

// SetupPersonalConfig writes personal.md.
func (e *TestEnv) SetupPersonalConfig(content string) error {
	return os.WriteFile(e.Paths.PersonalMD, []byte(content), 0644)
}

// SetupPersonalLanguage writes a personal language config.
func (e *TestEnv) SetupPersonalLanguage(lang, content string) error {
	if err := os.MkdirAll(e.Paths.PersonalLanguages, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.Paths.PersonalLanguages, lang+".md"), []byte(content), 0644)
}

// SetupConfig writes config.yaml.
func (e *TestEnv) SetupConfig(cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(e.Paths.ConfigFile, data, 0644)
}

// GetOutputPath returns the path to ~/.claude/CLAUDE.md.
func (e *TestEnv) GetOutputPath() string {
	return filepath.Join(e.ClaudeDir, "CLAUDE.md")
}

// ReadOutput reads the final CLAUDE.md output.
func (e *TestEnv) ReadOutput() (string, error) {
	content, err := os.ReadFile(e.GetOutputPath())
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// RunSync executes the merge and write operation with the current test environment.
// This replicates the core logic of applyConfig without the interactive prompts.
func (e *TestEnv) RunSync(owner, repo string, cfg *config.Config) error {
	// Read team config from cache
	teamConfig, err := os.ReadFile(e.Paths.CacheFile(owner, repo))
	if err != nil {
		return err
	}

	// Read personal config (optional)
	var personalConfig []byte
	if _, err := os.Stat(e.Paths.PersonalMD); err == nil {
		personalConfig, err = os.ReadFile(e.Paths.PersonalMD)
		if err != nil {
			return err
		}
	}

	// Resolve languages
	teamLangDir := e.Paths.TeamLanguagesDir(owner, repo)
	personalLangDir := e.Paths.PersonalLanguages

	var activeLanguages []string
	var languageFiles map[string][]*language.LanguageFile

	if len(cfg.Languages.Enabled) > 0 {
		activeLanguages = language.FilterDisabled(cfg.Languages.Enabled, cfg.Languages.Disabled)
	} else {
		available, listErr := language.ListAvailableLanguages(teamLangDir, personalLangDir, "")
		if listErr != nil {
			return listErr
		}
		activeLanguages = language.FilterDisabled(available, cfg.Languages.Disabled)
	}

	if len(activeLanguages) > 0 {
		var err error
		languageFiles, err = language.LoadLanguageFiles(
			activeLanguages,
			teamLangDir,
			personalLangDir,
			"",
		)
		if err != nil {
			return err
		}
	}

	// Merge configs
	layers := []merge.Layer{
		{Content: string(teamConfig), Source: "team"},
		{Content: string(personalConfig), Source: "personal"},
	}
	mergeOpts := merge.MergeOptions{
		AnnotateSources: true,
		SourceRepo:      cfg.SourceRepo(),
		Languages:       activeLanguages,
		LanguageFiles:   languageFiles,
	}
	output := merge.MergeWithLanguages(layers, mergeOpts)

	// Write to output
	return os.WriteFile(e.GetOutputPath(), []byte(output), 0644)
}
