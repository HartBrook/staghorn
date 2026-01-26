// Package integration provides integration testing utilities for staghorn.
package integration

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/language"
	"github.com/HartBrook/staghorn/internal/merge"
	"github.com/HartBrook/staghorn/internal/rules"
	"github.com/HartBrook/staghorn/internal/skills"
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

// SetupTeamCommand writes a team command to cache.
func (e *TestEnv) SetupTeamCommand(owner, repo, cmd, content string) error {
	cmdDir := e.Paths.TeamCommandsDir(owner, repo)
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cmdDir, cmd+".md"), []byte(content), 0644)
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

// SetupTeamRule writes a team rule to cache.
func (e *TestEnv) SetupTeamRule(owner, repo, relPath, content string) error {
	rulesDir := e.Paths.TeamRulesDir(owner, repo)
	fullPath := filepath.Join(rulesDir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// SetupPersonalRule writes a personal rule.
func (e *TestEnv) SetupPersonalRule(relPath, content string) error {
	fullPath := filepath.Join(e.Paths.PersonalRules, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// GetClaudeRulesDir returns the path to ~/.claude/rules.
func (e *TestEnv) GetClaudeRulesDir() string {
	return filepath.Join(e.ClaudeDir, "rules")
}

// ReadClaudeRule reads a rule from ~/.claude/rules.
func (e *TestEnv) ReadClaudeRule(relPath string) (string, error) {
	content, err := os.ReadFile(filepath.Join(e.GetClaudeRulesDir(), relPath))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// RunSyncRules syncs rules from team/personal sources to Claude rules directory.
func (e *TestEnv) RunSyncRules(owner, repo string) (int, error) {
	// Load rules from all sources using the registry
	registry, err := rules.LoadRegistry(
		e.Paths.TeamRulesDir(owner, repo),
		e.Paths.PersonalRules,
		"", // No project dir for global sync
	)
	if err != nil {
		return 0, err
	}

	allRules := registry.All()
	if len(allRules) == 0 {
		return 0, nil
	}

	// Create Claude rules directory
	claudeRulesDir := e.GetClaudeRulesDir()
	if err := os.MkdirAll(claudeRulesDir, 0755); err != nil {
		return 0, err
	}

	// Write each rule
	count := 0
	for _, rule := range allRules {
		outputPath := filepath.Join(claudeRulesDir, rule.RelPath)

		// Ensure parent directory exists (for subdirectories)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return count, err
		}

		content, err := rules.ConvertToClaude(rule)
		if err != nil {
			return count, err
		}
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
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

	// Write to output (RunSync)
	return os.WriteFile(e.GetOutputPath(), []byte(output), 0644)
}

// RunMultiSourceSync executes the merge for multi-source configurations.
// It reads the base config from the base repo and languages from their respective repos.
func (e *TestEnv) RunMultiSourceSync(cfg *config.Config) error {
	// Get base repo
	baseRepoStr := cfg.Source.RepoForBase()
	baseOwner, baseRepo, err := config.ParseRepo(baseRepoStr)
	if err != nil {
		return err
	}

	// Read team config from base repo cache
	teamConfig, err := os.ReadFile(e.Paths.CacheFile(baseOwner, baseRepo))
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

	// Resolve active languages
	var activeLanguages []string
	if len(cfg.Languages.Enabled) > 0 {
		activeLanguages = language.FilterDisabled(cfg.Languages.Enabled, cfg.Languages.Disabled)
	} else {
		// Collect available languages from all source repos
		availableLanguages := make(map[string]bool)
		for _, repoStr := range cfg.Source.AllRepos() {
			owner, repo, err := config.ParseRepo(repoStr)
			if err != nil {
				continue
			}
			teamLangDir := e.Paths.TeamLanguagesDir(owner, repo)
			langs, _ := language.ListAvailableLanguages(teamLangDir, "", "")
			for _, lang := range langs {
				availableLanguages[lang] = true
			}
		}
		// Also check personal languages
		personalLangs, _ := language.ListAvailableLanguages("", e.Paths.PersonalLanguages, "")
		for _, lang := range personalLangs {
			availableLanguages[lang] = true
		}
		for lang := range availableLanguages {
			activeLanguages = append(activeLanguages, lang)
		}
		// Sort for deterministic output
		sort.Strings(activeLanguages)
		activeLanguages = language.FilterDisabled(activeLanguages, cfg.Languages.Disabled)
	}

	// Load language files from their respective source repos
	var languageFiles map[string][]*language.LanguageFile
	if len(activeLanguages) > 0 {
		languageFiles = make(map[string][]*language.LanguageFile)
		for _, lang := range activeLanguages {
			// Determine the source repo for this language
			sourceRepoStr := cfg.Source.RepoForLanguage(lang)
			owner, repo, err := config.ParseRepo(sourceRepoStr)
			if err != nil {
				continue
			}
			teamLangDir := e.Paths.TeamLanguagesDir(owner, repo)

			files, err := language.LoadLanguageFiles(
				[]string{lang},
				teamLangDir,
				e.Paths.PersonalLanguages,
				"",
			)
			if err != nil {
				continue
			}
			if langFiles, ok := files[lang]; ok {
				languageFiles[lang] = langFiles
			}
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

	// Write to output (RunMultiSourceSync)
	return os.WriteFile(e.GetOutputPath(), []byte(output), 0644)
}

// SetupTeamSkill writes a team skill to cache.
// The skill is a directory containing SKILL.md and optional supporting files.
func (e *TestEnv) SetupTeamSkill(owner, repo, skillName, skillMD string) error {
	skillsDir := e.Paths.TeamSkillsDir(owner, repo)
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644)
}

// SetupTeamSkillWithFiles writes a team skill with supporting files.
func (e *TestEnv) SetupTeamSkillWithFiles(owner, repo, skillName, skillMD string, files map[string]string) error {
	skillsDir := e.Paths.TeamSkillsDir(owner, repo)
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	// Write SKILL.md
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		return err
	}

	// Write supporting files
	for relPath, content := range files {
		fullPath := filepath.Join(skillDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// SetupPersonalSkill writes a personal skill.
func (e *TestEnv) SetupPersonalSkill(skillName, skillMD string) error {
	skillDir := filepath.Join(e.Paths.PersonalSkills, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644)
}

// GetClaudeSkillsDir returns the path to ~/.claude/skills.
func (e *TestEnv) GetClaudeSkillsDir() string {
	return filepath.Join(e.ClaudeDir, "skills")
}

// ReadClaudeSkill reads SKILL.md from ~/.claude/skills/<name>/.
func (e *TestEnv) ReadClaudeSkill(skillName string) (string, error) {
	content, err := os.ReadFile(filepath.Join(e.GetClaudeSkillsDir(), skillName, "SKILL.md"))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ReadClaudeSkillFile reads a supporting file from a Claude skill directory.
func (e *TestEnv) ReadClaudeSkillFile(skillName, relPath string) (string, error) {
	content, err := os.ReadFile(filepath.Join(e.GetClaudeSkillsDir(), skillName, relPath))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// RunSyncSkills syncs skills from team/personal sources to Claude skills directory.
func (e *TestEnv) RunSyncSkills(owner, repo string) (int, error) {
	// Load skills from all sources using the registry
	registry, err := skills.LoadRegistry(
		e.Paths.TeamSkillsDir(owner, repo),
		e.Paths.PersonalSkills,
		"", // No project dir for global sync
	)
	if err != nil {
		return 0, err
	}

	allSkills := registry.All()
	if len(allSkills) == 0 {
		return 0, nil
	}

	// Create Claude skills directory
	claudeSkillsDir := e.GetClaudeSkillsDir()
	if err := os.MkdirAll(claudeSkillsDir, 0755); err != nil {
		return 0, err
	}

	// Sync each skill
	count := 0
	for _, skill := range allSkills {
		_, err := skills.SyncToClaude(skill, claudeSkillsDir)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

// RunSyncSkillsMultiSource syncs skills from multiple source repos.
func (e *TestEnv) RunSyncSkillsMultiSource(cfg *config.Config) (int, error) {
	// Collect team skills directories from all repos
	var teamSkillsDirs []string
	for _, repoStr := range cfg.Source.AllRepos() {
		owner, repo, err := config.ParseRepo(repoStr)
		if err != nil {
			continue
		}
		teamSkillsDirs = append(teamSkillsDirs, e.Paths.TeamSkillsDir(owner, repo))
	}

	// Load skills from all sources
	registry, err := skills.LoadRegistryWithMultipleDirs(
		teamSkillsDirs,
		e.Paths.PersonalSkills,
		"", // No project dir for global sync
	)
	if err != nil {
		return 0, err
	}

	allSkills := registry.All()
	if len(allSkills) == 0 {
		return 0, nil
	}

	// Create Claude skills directory
	claudeSkillsDir := e.GetClaudeSkillsDir()
	if err := os.MkdirAll(claudeSkillsDir, 0755); err != nil {
		return 0, err
	}

	// Sync each skill
	count := 0
	for _, skill := range allSkills {
		_, err := skills.SyncToClaude(skill, claudeSkillsDir)
		if err != nil {
			continue // Skip errors for multi-source (may have collision warnings)
		}
		count++
	}

	return count, nil
}

// SetupExistingClaudeSkill creates a skill in ~/.claude/skills that is NOT managed by staghorn.
// Used for testing collision detection.
func (e *TestEnv) SetupExistingClaudeSkill(skillName, skillMD string) error {
	skillDir := filepath.Join(e.GetClaudeSkillsDir(), skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644)
}
