package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/HartBrook/staghorn/internal/cache"
	"github.com/HartBrook/staghorn/internal/commands"
	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/errors"
	"github.com/HartBrook/staghorn/internal/github"
	"github.com/HartBrook/staghorn/internal/language"
	"github.com/HartBrook/staghorn/internal/merge"
	"github.com/HartBrook/staghorn/internal/optimize"
	"github.com/HartBrook/staghorn/internal/rules"
	"github.com/HartBrook/staghorn/internal/skills"
	"github.com/spf13/cobra"
)

type syncOptions struct {
	force         bool
	offline       bool
	configOnly    bool
	commandsOnly  bool
	languagesOnly bool
	rulesOnly     bool
	skillsOnly    bool
	fetchOnly     bool
	applyOnly     bool
	claudeOnly    bool
}

// shouldSyncConfig returns true if base config should be synced.
func (o *syncOptions) shouldSyncConfig() bool {
	return !o.commandsOnly && !o.languagesOnly && !o.rulesOnly && !o.claudeOnly
}

// shouldSyncCommands returns true if commands should be synced.
func (o *syncOptions) shouldSyncCommands() bool {
	return !o.configOnly && !o.languagesOnly && !o.rulesOnly && !o.claudeOnly
}

// shouldSyncLanguages returns true if languages should be synced.
func (o *syncOptions) shouldSyncLanguages() bool {
	return !o.configOnly && !o.commandsOnly && !o.rulesOnly && !o.claudeOnly
}

// shouldSyncEvals returns true if evals should be synced.
func (o *syncOptions) shouldSyncEvals() bool {
	return !o.configOnly && !o.commandsOnly && !o.languagesOnly && !o.rulesOnly && !o.claudeOnly
}

// shouldSyncRules returns true if rules should be synced.
func (o *syncOptions) shouldSyncRules() bool {
	return !o.configOnly && !o.commandsOnly && !o.languagesOnly && !o.claudeOnly
}

// shouldApplyConfig returns true if config should be applied to ~/.claude/CLAUDE.md.
func (o *syncOptions) shouldApplyConfig() bool {
	return !o.fetchOnly && !o.rulesOnly && !o.claudeOnly
}

// shouldSyncClaudeCommands returns true if commands should be synced to Claude Code.
func (o *syncOptions) shouldSyncClaudeCommands() bool {
	return !o.configOnly && !o.languagesOnly && !o.rulesOnly && !o.fetchOnly
}

// shouldSyncClaudeRules returns true if rules should be synced to Claude Code.
func (o *syncOptions) shouldSyncClaudeRules() bool {
	return !o.configOnly && !o.languagesOnly && !o.commandsOnly && !o.skillsOnly && !o.fetchOnly
}

// shouldSyncSkills returns true if skills should be synced.
func (o *syncOptions) shouldSyncSkills() bool {
	return !o.configOnly && !o.commandsOnly && !o.languagesOnly && !o.rulesOnly && !o.claudeOnly
}

// shouldSyncClaudeSkills returns true if skills should be synced to Claude Code.
func (o *syncOptions) shouldSyncClaudeSkills() bool {
	return !o.configOnly && !o.languagesOnly && !o.commandsOnly && !o.rulesOnly && !o.fetchOnly
}

// repoContext holds the branch info for a single repo.
type repoContext struct {
	owner  string
	repo   string
	branch string
}

// NewSyncCmd creates the sync command.
func NewSyncCmd() *cobra.Command {
	opts := &syncOptions{}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Fetch team config and apply to Claude Code",
		Long: `Fetches the team CLAUDE.md configuration from GitHub and applies it to
~/.claude/CLAUDE.md, where Claude Code will automatically pick it up.

This is the main command for keeping your Claude Code config up to date.`,
		Example: `  staghorn sync
  staghorn sync --force
  staghorn sync --fetch-only
  staghorn sync --apply-only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVar(&opts.force, "force", false, "Re-fetch even if cache is fresh")
	cmd.Flags().BoolVar(&opts.offline, "offline", false, "Use cached version only, skip fetch")
	cmd.Flags().BoolVar(&opts.fetchOnly, "fetch-only", false, "Only fetch, don't apply to ~/.claude/CLAUDE.md")
	cmd.Flags().BoolVar(&opts.applyOnly, "apply-only", false, "Only apply cached config, skip fetch")
	cmd.Flags().BoolVar(&opts.configOnly, "config-only", false, "Only sync config, skip commands, languages, and rules")
	cmd.Flags().BoolVar(&opts.commandsOnly, "commands-only", false, "Only sync commands, skip config, languages, and rules")
	cmd.Flags().BoolVar(&opts.languagesOnly, "languages-only", false, "Only sync languages, skip config, commands, and rules")
	cmd.Flags().BoolVar(&opts.rulesOnly, "rules-only", false, "Only sync rules, skip config, commands, and languages")
	cmd.Flags().BoolVar(&opts.skillsOnly, "skills-only", false, "Only sync skills, skip config, commands, languages, and rules")
	cmd.Flags().BoolVar(&opts.claudeOnly, "claude-only", false, "Only sync commands, rules, and skills to ~/.claude/, skip config apply")

	return cmd
}

func runSync(ctx context.Context, opts *syncOptions) error {
	paths := config.NewPaths()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	owner, repo, err := cfg.DefaultOwnerRepo()
	if err != nil {
		return err
	}

	c := cache.New(paths)

	// Check for multi-source configuration
	// We need the GitHub client for multi-source, so check after client creation
	isMultiSource := cfg.Source.IsMultiSource()

	// Apply-only mode: skip fetch, just apply from cache
	if opts.applyOnly {
		if !c.Exists(owner, repo) {
			return errors.CacheNotFound(owner + "/" + repo)
		}
		return applyConfig(cfg, paths, owner, repo)
	}

	// Offline mode
	if opts.offline {
		if c.Exists(owner, repo) {
			meta, err := c.GetMetadata(owner, repo)
			if err != nil {
				return fmt.Errorf("failed to read cache metadata: %w", err)
			}
			printSuccess("Using cached config from %s", meta.Age())
			return nil
		}
		return errors.CacheNotFound(owner + "/" + repo)
	}

	// Check if we need to sync
	if !opts.force && c.Exists(owner, repo) {
		meta, err := c.GetMetadata(owner, repo)
		if err == nil && !meta.IsStale(cfg.Cache.TTLDuration()) {
			printSuccess("Cache is fresh (%s)", meta.Age())
			fmt.Println("  Use --force to re-fetch anyway.")
			return nil
		}
		// If metadata read failed or cache is stale, proceed with sync
	}

	// Create GitHub client
	client, err := github.NewClient()
	if err != nil {
		// Try with token fallback
		token := github.GetTokenFromEnv()
		if token == "" {
			return errors.GitHubAuthFailed(err)
		}
		client, err = github.NewClientWithToken(token)
		if err != nil {
			return errors.GitHubAuthFailed(err)
		}
	}

	// Use multi-source sync if configured
	if isMultiSource {
		return runMultiSourceSync(ctx, cfg, paths, opts, client, c)
	}

	// Determine branch (use default branch from repo)
	branch, err := client.GetDefaultBranch(ctx, owner, repo)
	if err != nil {
		return errors.GitHubFetchFailed(owner+"/"+repo, err)
	}

	fmt.Printf("Fetching %s/%s...\n", owner, repo)

	// Sync config
	if opts.shouldSyncConfig() {
		result, err := client.FetchFile(ctx, owner, repo, config.DefaultPath, branch)
		if err != nil {
			return errors.GitHubFetchFailed(owner+"/"+repo, err)
		}

		// Save to cache
		meta := &cache.Metadata{
			Owner:       owner,
			Repo:        repo,
			SHA:         result.SHA,
			LastFetched: time.Now(),
		}

		if err := c.Write(owner, repo, result.Content, meta); err != nil {
			return fmt.Errorf("failed to write cache: %w", err)
		}

		printSuccess("Synced config")
		printInfo("File", config.DefaultPath)
		printInfo("SHA", result.SHA[:8])
	}

	// Sync commands
	if opts.shouldSyncCommands() {
		commandCount, err := syncCommands(ctx, client, owner, repo, branch, paths)
		if err != nil {
			printWarning("Failed to sync commands: %v", err)
		} else if commandCount > 0 {
			printSuccess("Synced %d commands", commandCount)
		} else if opts.commandsOnly {
			fmt.Println("No commands found in team repository")
		}

		// Also sync templates
		templateCount, err := syncTemplates(ctx, client, owner, repo, branch, paths)
		if err != nil {
			printWarning("Failed to sync templates: %v", err)
		} else if templateCount > 0 {
			printSuccess("Synced %d templates", templateCount)
		}
	}

	// Sync languages
	if opts.shouldSyncLanguages() {
		languageCount, err := syncLanguages(ctx, client, owner, repo, branch, paths)
		if err != nil {
			printWarning("Failed to sync languages: %v", err)
		} else if languageCount > 0 {
			printSuccess("Synced %d language configs", languageCount)
		} else if opts.languagesOnly {
			fmt.Println("No language configs found in team repository")
		}
	}

	// Sync evals
	if opts.shouldSyncEvals() {
		evalCount, err := syncEvals(ctx, client, owner, repo, branch, paths)
		if err != nil {
			printWarning("Failed to sync evals: %v", err)
		} else if evalCount > 0 {
			printSuccess("Synced %d evals", evalCount)
		}
	}

	// Sync rules
	if opts.shouldSyncRules() {
		ruleCount, err := syncRules(ctx, client, owner, repo, branch, paths)
		if err != nil {
			printWarning("Failed to sync rules: %v", err)
		} else if ruleCount > 0 {
			printSuccess("Synced %d rules", ruleCount)
		} else if opts.rulesOnly {
			fmt.Println("No rules found in team repository")
		}
	}

	// Sync skills
	if opts.shouldSyncSkills() {
		skillCount, err := syncSkills(ctx, client, owner, repo, branch, paths)
		if err != nil {
			printWarning("Failed to sync skills: %v", err)
		} else if skillCount > 0 {
			printSuccess("Synced %d skills", skillCount)
		} else if opts.skillsOnly {
			fmt.Println("No skills found in team repository")
		}
	}

	// Apply to ~/.claude/CLAUDE.md
	if opts.shouldApplyConfig() {
		fmt.Println()
		if err := applyConfig(cfg, paths, owner, repo); err != nil {
			return err
		}
	}

	// Sync commands to Claude Code
	if opts.shouldSyncClaudeCommands() {
		claudeCount, err := syncClaudeCommands(paths, owner, repo)
		if err != nil {
			printWarning("Failed to sync Claude commands: %v", err)
		} else if claudeCount > 0 {
			printSuccess("Synced %d commands to Claude Code", claudeCount)
			fmt.Printf("  %s Use /%s in Claude Code\n", dim("Tip:"), "code-review")
		}
	}

	// Sync rules to Claude Code
	if opts.shouldSyncClaudeRules() {
		claudeRuleCount, err := syncClaudeRules(paths, owner, repo)
		if err != nil {
			printWarning("Failed to sync Claude rules: %v", err)
		} else if claudeRuleCount > 0 {
			printSuccess("Synced %d rules to Claude Code", claudeRuleCount)
		}
	}

	// Sync skills to Claude Code
	if opts.shouldSyncClaudeSkills() {
		claudeSkillCount, err := syncClaudeSkills(paths, owner, repo)
		if err != nil {
			printWarning("Failed to sync Claude skills: %v", err)
		} else if claudeSkillCount > 0 {
			printSuccess("Synced %d skills to Claude Code", claudeSkillCount)
			fmt.Printf("  %s Skills are available via /skill-name in Claude Code\n", dim("Tip:"))
		}
	}

	// Check merged config size and suggest optimization if large
	if !opts.fetchOnly {
		checkConfigSizeAndSuggestOptimize(cfg, paths, owner, repo)
	}

	return nil
}

// syncCommands fetches commands from the team repo's commands/ directory.
func syncCommands(ctx context.Context, client *github.Client, owner, repo, branch string, paths *config.Paths) (int, error) {
	// List commands directory
	entries, err := client.ListDirectory(ctx, owner, repo, "commands", branch)
	if err != nil {
		return 0, err
	}

	if entries == nil {
		// No commands directory
		return 0, nil
	}

	// Create local commands cache directory
	commandsDir := paths.TeamCommandsDir(owner, repo)
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create commands directory: %w", err)
	}

	// Fetch each .md file
	count := 0
	for _, entry := range entries {
		if entry.Type != "file" || !strings.HasSuffix(entry.Name, ".md") {
			continue
		}

		result, err := client.FetchFile(ctx, owner, repo, entry.Path, branch)
		if err != nil {
			printWarning("Failed to fetch command %s: %v", entry.Name, err)
			continue
		}

		localPath := filepath.Join(commandsDir, entry.Name)
		if err := os.WriteFile(localPath, []byte(result.Content), 0644); err != nil {
			printWarning("Failed to write command %s: %v", entry.Name, err)
			continue
		}

		count++
	}

	return count, nil
}

// syncTemplates fetches project templates from the team repo's templates/ directory.
func syncTemplates(ctx context.Context, client *github.Client, owner, repo, branch string, paths *config.Paths) (int, error) {
	// List templates directory
	entries, err := client.ListDirectory(ctx, owner, repo, "templates", branch)
	if err != nil {
		return 0, err
	}

	if entries == nil {
		// No templates directory
		return 0, nil
	}

	// Create local templates cache directory
	templatesDir := paths.TeamTemplatesDir(owner, repo)
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Fetch each .md file
	count := 0
	for _, entry := range entries {
		if entry.Type != "file" || !strings.HasSuffix(entry.Name, ".md") {
			continue
		}

		result, err := client.FetchFile(ctx, owner, repo, entry.Path, branch)
		if err != nil {
			printWarning("Failed to fetch template %s: %v", entry.Name, err)
			continue
		}

		localPath := filepath.Join(templatesDir, entry.Name)
		if err := os.WriteFile(localPath, []byte(result.Content), 0644); err != nil {
			printWarning("Failed to write template %s: %v", entry.Name, err)
			continue
		}

		count++
	}

	return count, nil
}

// syncLanguages fetches language configs from the team repo's languages/ directory.
func syncLanguages(ctx context.Context, client *github.Client, owner, repo, branch string, paths *config.Paths) (int, error) {
	// List languages directory
	entries, err := client.ListDirectory(ctx, owner, repo, "languages", branch)
	if err != nil {
		return 0, err
	}

	if entries == nil {
		// No languages directory
		return 0, nil
	}

	// Create local languages cache directory
	languagesDir := paths.TeamLanguagesDir(owner, repo)
	if err := os.MkdirAll(languagesDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create languages directory: %w", err)
	}

	// Fetch each .md file
	count := 0
	for _, entry := range entries {
		if entry.Type != "file" || !strings.HasSuffix(entry.Name, ".md") {
			continue
		}

		result, err := client.FetchFile(ctx, owner, repo, entry.Path, branch)
		if err != nil {
			printWarning("Failed to fetch language config %s: %v", entry.Name, err)
			continue
		}

		localPath := filepath.Join(languagesDir, entry.Name)
		if err := os.WriteFile(localPath, []byte(result.Content), 0644); err != nil {
			printWarning("Failed to write language config %s: %v", entry.Name, err)
			continue
		}

		count++
	}

	return count, nil
}

// syncEvals fetches evals from the team repo's evals/ directory.
func syncEvals(ctx context.Context, client *github.Client, owner, repo, branch string, paths *config.Paths) (int, error) {
	// List evals directory
	entries, err := client.ListDirectory(ctx, owner, repo, "evals", branch)
	if err != nil {
		return 0, err
	}

	if entries == nil {
		// No evals directory
		return 0, nil
	}

	// Create local evals cache directory
	evalsDir := paths.TeamEvalsDir(owner, repo)
	if err := os.MkdirAll(evalsDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create evals directory: %w", err)
	}

	// Fetch each .yaml/.yml file
	count := 0
	for _, entry := range entries {
		if entry.Type != "file" {
			continue
		}
		if !strings.HasSuffix(entry.Name, ".yaml") && !strings.HasSuffix(entry.Name, ".yml") {
			continue
		}

		result, err := client.FetchFile(ctx, owner, repo, entry.Path, branch)
		if err != nil {
			printWarning("Failed to fetch eval %s: %v", entry.Name, err)
			continue
		}

		localPath := filepath.Join(evalsDir, entry.Name)
		if err := os.WriteFile(localPath, []byte(result.Content), 0644); err != nil {
			printWarning("Failed to write eval %s: %v", entry.Name, err)
			continue
		}

		count++
	}

	return count, nil
}

// syncRules fetches rules from the team repo's rules/ directory (recursive).
func syncRules(ctx context.Context, client *github.Client, owner, repo, branch string, paths *config.Paths) (int, error) {
	rulesDir := paths.TeamRulesDir(owner, repo)

	// Clear existing cache to handle deletions
	if err := os.RemoveAll(rulesDir); err != nil {
		return 0, fmt.Errorf("failed to clear rules cache: %w", err)
	}

	count, err := syncRulesRecursive(ctx, client, owner, repo, branch, "rules", rulesDir, "")
	if err != nil {
		return 0, err
	}

	return count, nil
}

// syncRulesRecursive handles recursive directory fetching for rules.
func syncRulesRecursive(ctx context.Context, client *github.Client, owner, repo, branch, remotePath, localBase, relPath string) (int, error) {
	entries, err := client.ListDirectory(ctx, owner, repo, remotePath, branch)
	if err != nil {
		return 0, err
	}

	if entries == nil {
		return 0, nil
	}

	count := 0
	for _, entry := range entries {
		entryRelPath := entry.Name
		if relPath != "" {
			entryRelPath = filepath.Join(relPath, entry.Name)
		}

		if entry.Type == "dir" {
			// Recurse into subdirectory
			subCount, err := syncRulesRecursive(ctx, client, owner, repo, branch,
				entry.Path, localBase, entryRelPath)
			if err != nil {
				printWarning("Failed to sync rules subdirectory %s: %v", entry.Name, err)
				continue
			}
			count += subCount
		} else if entry.Type == "file" && strings.HasSuffix(entry.Name, ".md") {
			// Fetch and cache rule file
			result, err := client.FetchFile(ctx, owner, repo, entry.Path, branch)
			if err != nil {
				printWarning("Failed to fetch rule %s: %v", entry.Name, err)
				continue
			}

			localPath := filepath.Join(localBase, entryRelPath)
			if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
				printWarning("Failed to create directory for rule %s: %v", entry.Name, err)
				continue
			}

			if err := os.WriteFile(localPath, []byte(result.Content), 0644); err != nil {
				printWarning("Failed to write rule %s: %v", entry.Name, err)
				continue
			}

			count++
		}
	}

	return count, nil
}

// syncClaudeRules syncs staghorn rules to Claude Code rules directory.
func syncClaudeRules(paths *config.Paths, owner, repo string) (int, error) {
	// Load rules from all sources using the registry
	registry, err := rules.LoadRegistry(
		paths.TeamRulesDir(owner, repo),
		paths.PersonalRules,
		"", // No project dir for global sync
	)
	if err != nil {
		return 0, fmt.Errorf("failed to load rules: %w", err)
	}

	allRules := registry.All()
	if len(allRules) == 0 {
		return 0, nil
	}

	// Create Claude rules directory
	claudeDir := paths.ClaudeRulesDir()
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create Claude rules directory: %w", err)
	}

	// Write each rule
	count := 0
	for _, rule := range allRules {
		outputPath := filepath.Join(claudeDir, rule.RelPath)

		// Check for collision with non-staghorn file
		if existingContent, err := os.ReadFile(outputPath); err == nil {
			if !strings.Contains(string(existingContent), merge.HeaderManagedPrefix) {
				// File exists and is not managed by staghorn - skip with warning
				printWarning("Skipping rule %s: existing rule not managed by staghorn", rule.RelPath)
				continue
			}
		}

		// Ensure parent directory exists (for subdirectories)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			printWarning("Failed to create directory for rule %s: %v", rule.RelPath, err)
			continue
		}

		content, err := rules.ConvertToClaude(rule)
		if err != nil {
			printWarning("Failed to convert rule %s: %v", rule.RelPath, err)
			continue
		}
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			printWarning("Failed to write Claude rule %s: %v", rule.RelPath, err)
			continue
		}
		count++
	}

	return count, nil
}

// syncClaudeCommands syncs staghorn commands to Claude Code custom commands directory.
func syncClaudeCommands(paths *config.Paths, owner, repo string) (int, error) {
	// Load commands from all sources using the registry
	registry, err := commands.LoadRegistry(
		paths.TeamCommandsDir(owner, repo),
		paths.PersonalCommands,
		"", // No project dir for global sync
	)
	if err != nil {
		return 0, fmt.Errorf("failed to load commands: %w", err)
	}

	allCommands := registry.All()
	if len(allCommands) == 0 {
		return 0, nil
	}

	// Create Claude commands directory
	claudeDir := paths.ClaudeCommandsDir()
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create Claude commands directory: %w", err)
	}

	// Write each command as a Claude command
	count := 0
	for _, cmd := range allCommands {
		filename := cmd.Name + ".md"
		outputPath := filepath.Join(claudeDir, filename)

		// Check for collision with non-staghorn file
		if existingContent, err := os.ReadFile(outputPath); err == nil {
			if !strings.Contains(string(existingContent), merge.HeaderManagedPrefix) {
				// File exists and is not managed by staghorn - skip with warning
				printWarning("Skipping /%s: existing command not managed by staghorn", cmd.Name)
				continue
			}
		}

		content := commands.ConvertToClaude(cmd)
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			printWarning("Failed to write Claude command %s: %v", cmd.Name, err)
			continue
		}
		count++
	}

	return count, nil
}

// readPersonalConfig reads and processes the personal config file.
func readPersonalConfig(paths *config.Paths) ([]byte, error) {
	if _, err := os.Stat(paths.PersonalMD); err != nil {
		return nil, nil // No personal config is not an error
	}
	personalConfig, err := os.ReadFile(paths.PersonalMD)
	if err != nil {
		return nil, fmt.Errorf("failed to read personal config: %w", err)
	}
	return []byte(stripInstructionalComments(string(personalConfig))), nil
}

// resolveActiveLanguages determines which languages are active based on config and available files.
// It returns a sorted list to ensure deterministic output.
func resolveActiveLanguages(cfg *config.Config, teamLangDirs []string, personalLangDir string) []string {
	var activeLanguages []string

	if len(cfg.Languages.Enabled) > 0 {
		// Explicit list takes precedence
		activeLanguages = language.FilterDisabled(cfg.Languages.Enabled, cfg.Languages.Disabled)
	} else {
		// Collect available languages from all team directories and personal
		availableLanguages := make(map[string]bool)
		for _, teamLangDir := range teamLangDirs {
			if teamLangDir == "" {
				continue
			}
			langs, _ := language.ListAvailableLanguages(teamLangDir, "", "")
			for _, lang := range langs {
				availableLanguages[lang] = true
			}
		}
		// Also check personal languages
		personalLangs, _ := language.ListAvailableLanguages("", personalLangDir, "")
		for _, lang := range personalLangs {
			availableLanguages[lang] = true
		}
		for lang := range availableLanguages {
			activeLanguages = append(activeLanguages, lang)
		}
		activeLanguages = language.FilterDisabled(activeLanguages, cfg.Languages.Disabled)
	}

	// Sort for deterministic output
	sort.Strings(activeLanguages)
	return activeLanguages
}

// handleExistingConfigMigration checks if the output file needs migration or backup.
// Returns (shouldContinue, updatedPersonalConfig, error).
func handleExistingConfigMigration(cfg *config.Config, paths *config.Paths, outputPath string, personalConfig []byte) (bool, []byte, error) {
	existingContent, err := os.ReadFile(outputPath)
	if err != nil {
		// File doesn't exist - no migration needed
		return true, personalConfig, nil
	}

	existingStr := string(existingContent)
	needsPrompt := false
	promptReason := ""

	if !strings.Contains(existingStr, merge.HeaderManagedPrefix) {
		needsPrompt = true
		promptReason = "Found existing ~/.claude/CLAUDE.md not managed by staghorn"
	} else {
		// Check if switching sources
		currentSource := cfg.SourceRepo()
		if idx := strings.Index(existingStr, "Source: "); idx != -1 {
			end := strings.Index(existingStr[idx:], " |")
			if end != -1 {
				previousSource := existingStr[idx+8 : idx+end]
				if previousSource != currentSource {
					needsPrompt = true
					promptReason = fmt.Sprintf("Switching source from %s to %s", previousSource, currentSource)
				}
			}
		}
	}

	if !needsPrompt {
		return true, personalConfig, nil
	}

	printWarning("%s", promptReason)
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  1. Migrate content to personal config (recommended)")
	fmt.Println("  2. Back up existing file and continue")
	fmt.Println("  3. Abort")
	fmt.Println()

	choice := promptString("Choose an option [1/2/3]:")

	switch choice {
	case "1", "":
		contentToMigrate := existingStr
		if strings.Contains(existingStr, merge.HeaderManagedPrefix) {
			if idx := strings.Index(existingStr, "-->\n"); idx != -1 {
				contentToMigrate = strings.TrimLeft(existingStr[idx+4:], "\n")
			}
		}

		existingPersonal, _ := os.ReadFile(paths.PersonalMD)
		newPersonal := string(existingPersonal)
		if len(newPersonal) > 0 {
			newPersonal += "\n\n"
		}
		newPersonal += "<!-- [staghorn] Migrated from ~/.claude/CLAUDE.md -->\n\n" + contentToMigrate

		if err := os.MkdirAll(paths.ConfigDir, 0755); err != nil {
			return false, nil, fmt.Errorf("failed to create config directory: %w", err)
		}
		if err := os.WriteFile(paths.PersonalMD, []byte(newPersonal), 0644); err != nil {
			return false, nil, fmt.Errorf("failed to write personal config: %w", err)
		}
		printSuccess("Migrated content to %s", paths.PersonalMD)
		fmt.Printf("  %s Run 'staghorn edit' to review and organize\n", dim("Tip:"))
		fmt.Println()

		// Re-read personal config
		updatedPersonal, _ := os.ReadFile(paths.PersonalMD)
		return true, updatedPersonal, nil

	case "2":
		backupPath := outputPath + ".backup"
		if err := os.WriteFile(backupPath, existingContent, 0644); err != nil {
			return false, nil, fmt.Errorf("failed to backup existing file: %w", err)
		}
		printSuccess("Backed up to %s", backupPath)
		fmt.Println()
		return true, personalConfig, nil

	case "3":
		fmt.Println("Aborted.")
		return false, nil, nil

	default:
		return false, nil, fmt.Errorf("invalid option")
	}
}

// writeConfigOutput writes the merged config to the output file and prints status.
func writeConfigOutput(outputPath string, output string, hasPersonal bool) error {
	claudeDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create ~/.claude directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	printSuccess("Applied to %s", outputPath)

	if hasPersonal {
		fmt.Printf("  %s Team config + personal additions\n", dim("Merged:"))
	} else {
		fmt.Printf("  %s Team config only (no personal additions)\n", dim("Merged:"))
		fmt.Printf("  %s Run 'staghorn edit' to add personal preferences\n", dim("Tip:"))
	}

	return nil
}

// mergeAndWriteConfig handles migration, merging, and writing for both single and multi-source configs.
func mergeAndWriteConfig(cfg *config.Config, paths *config.Paths, teamConfig, personalConfig []byte, activeLanguages []string, languageFiles map[string][]*language.LanguageFile) error {
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

	outputPath := filepath.Join(os.Getenv("HOME"), ".claude", "CLAUDE.md")
	shouldContinue, updatedPersonal, err := handleExistingConfigMigration(cfg, paths, outputPath, personalConfig)
	if err != nil {
		return err
	}
	if !shouldContinue {
		return nil
	}
	if updatedPersonal != nil {
		personalConfig = updatedPersonal
		layers[1] = merge.Layer{Content: string(personalConfig), Source: "personal"}
	}

	output := merge.MergeWithLanguages(layers, mergeOpts)
	return writeConfigOutput(outputPath, output, len(personalConfig) > 0)
}

// applyConfig merges team config with personal additions and writes to ~/.claude/CLAUDE.md.
func applyConfig(cfg *config.Config, paths *config.Paths, owner, repo string) error {
	// Get team config from cache
	teamConfig, err := os.ReadFile(paths.CacheFile(owner, repo))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no cached team config found")
		}
		return fmt.Errorf("failed to read cached config: %w", err)
	}

	// Get personal config (optional)
	personalConfig, err := readPersonalConfig(paths)
	if err != nil {
		return err
	}

	// Resolve languages
	teamLangDir := paths.TeamLanguagesDir(owner, repo)
	activeLanguages := resolveActiveLanguages(cfg, []string{teamLangDir}, paths.PersonalLanguages)

	var languageFiles map[string][]*language.LanguageFile
	if len(activeLanguages) > 0 {
		languageFiles, _ = language.LoadLanguageFiles(
			activeLanguages,
			teamLangDir,
			paths.PersonalLanguages,
			"",
		)
	}

	return mergeAndWriteConfig(cfg, paths, teamConfig, personalConfig, activeLanguages, languageFiles)
}

// stripInstructionalComments removes HTML comments marked with [staghorn] prefix
// and collapses consecutive blank lines.
func stripInstructionalComments(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	prevBlank := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip lines that are staghorn instructional comments
		if strings.HasPrefix(trimmed, "<!-- [staghorn]") && strings.HasSuffix(trimmed, "-->") {
			continue
		}

		// Collapse consecutive blank lines
		isBlank := trimmed == ""
		if isBlank && prevBlank {
			continue
		}
		prevBlank = isBlank

		result = append(result, line)
	}

	// Clean up any resulting empty lines at the start
	for len(result) > 0 && strings.TrimSpace(result[0]) == "" {
		result = result[1:]
	}

	// Clean up any resulting empty lines at the end
	for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
		result = result[:len(result)-1]
	}

	return strings.Join(result, "\n")
}

// checkConfigSizeAndSuggestOptimize checks merged config size and suggests optimization if large.
func checkConfigSizeAndSuggestOptimize(cfg *config.Config, paths *config.Paths, owner, repo string) {
	// Build merged content to calculate size
	var layers []merge.Layer

	// Team layer
	c := cache.New(paths)
	if teamContent, _, err := c.Read(owner, repo); err == nil && teamContent != "" {
		layers = append(layers, merge.Layer{Content: teamContent, Source: "team"})
	}

	// Personal layer
	if personalContent, err := os.ReadFile(paths.PersonalMD); err == nil {
		layers = append(layers, merge.Layer{Content: string(personalContent), Source: "personal"})
	}

	if len(layers) == 0 {
		return
	}

	// Get active languages
	teamLangDir := paths.TeamLanguagesDir(owner, repo)
	personalLangDir := paths.PersonalLanguages

	var activeLanguages []string
	var languageFiles map[string][]*language.LanguageFile

	if len(cfg.Languages.Enabled) > 0 {
		activeLanguages = language.FilterDisabled(cfg.Languages.Enabled, cfg.Languages.Disabled)
	} else {
		activeLanguages, _ = language.ListAvailableLanguages(teamLangDir, personalLangDir, "")
		activeLanguages = language.FilterDisabled(activeLanguages, cfg.Languages.Disabled)
	}

	if len(activeLanguages) > 0 {
		languageFiles, _ = language.LoadLanguageFiles(
			activeLanguages,
			teamLangDir,
			personalLangDir,
			"",
		)
	}

	mergeOpts := merge.MergeOptions{
		AnnotateSources: true,
		Languages:       activeLanguages,
		LanguageFiles:   languageFiles,
	}

	merged := merge.MergeWithLanguages(layers, mergeOpts)
	tokens := optimize.CountTokens(merged)

	// Warn if over threshold
	if tokens > 3000 {
		fmt.Println()
		printWarning("Merged config is %d tokens (threshold: 3,000)", tokens)
		fmt.Printf("  Large configs may reduce Claude Code effectiveness.\n")
		fmt.Printf("  Run %s to compress.\n", info("staghorn optimize"))
	}
}

// buildRepoContexts creates repo contexts for all repos in a multi-source config.
func buildRepoContexts(ctx context.Context, client *github.Client, cfg *config.Config) (map[string]*repoContext, error) {
	allRepos := cfg.Source.AllRepos()
	contexts := make(map[string]*repoContext, len(allRepos))

	for _, repoStr := range allRepos {
		owner, repo, err := config.ParseRepo(repoStr)
		if err != nil {
			return nil, fmt.Errorf("invalid repo %s: %w", repoStr, err)
		}

		branch, err := client.GetDefaultBranch(ctx, owner, repo)
		if err != nil {
			return nil, errors.GitHubFetchFailed(repoStr, err)
		}

		contexts[repoStr] = &repoContext{
			owner:  owner,
			repo:   repo,
			branch: branch,
		}
	}

	return contexts, nil
}

// runMultiSourceSync handles sync when multiple source repos are configured.
func runMultiSourceSync(ctx context.Context, cfg *config.Config, paths *config.Paths, opts *syncOptions, client *github.Client, c *cache.Cache) error {
	// Build contexts for all repos
	repoContexts, err := buildRepoContexts(ctx, client, cfg)
	if err != nil {
		return err
	}

	// Get the base and default repo contexts - these should always exist after buildRepoContexts
	baseRepoStr := cfg.Source.RepoForBase()
	baseCtx := repoContexts[baseRepoStr]
	if baseCtx == nil {
		// This indicates a bug in buildRepoContexts - it should have created this context
		return fmt.Errorf("internal error: no context for base repo %s after building contexts", baseRepoStr)
	}

	defaultRepoStr := cfg.Source.DefaultRepo()
	defaultCtx := repoContexts[defaultRepoStr]
	if defaultCtx == nil {
		// This indicates a bug in buildRepoContexts - it should have created this context
		return fmt.Errorf("internal error: no context for default repo %s after building contexts", defaultRepoStr)
	}

	fmt.Printf("Fetching from %d source(s)...\n", len(repoContexts))

	// Sync base config
	if opts.shouldSyncConfig() {
		fmt.Printf("  Base config from %s/%s\n", baseCtx.owner, baseCtx.repo)
		result, err := client.FetchFile(ctx, baseCtx.owner, baseCtx.repo, config.DefaultPath, baseCtx.branch)
		if err != nil {
			return errors.GitHubFetchFailed(baseRepoStr, err)
		}

		meta := &cache.Metadata{
			Owner:       baseCtx.owner,
			Repo:        baseCtx.repo,
			SHA:         result.SHA,
			LastFetched: time.Now(),
		}

		if err := c.Write(baseCtx.owner, baseCtx.repo, result.Content, meta); err != nil {
			return fmt.Errorf("failed to write cache: %w", err)
		}

		printSuccess("Synced config from %s/%s", baseCtx.owner, baseCtx.repo)
	}

	// Sync commands with multi-source support
	if opts.shouldSyncCommands() {
		commandCount, err := syncCommandsMultiSource(ctx, client, cfg, repoContexts, paths)
		if err != nil {
			printWarning("Failed to sync commands: %v", err)
		} else if commandCount > 0 {
			printSuccess("Synced %d commands", commandCount)
		}

		// Sync templates from default repo
		templateCount, err := syncTemplates(ctx, client, defaultCtx.owner, defaultCtx.repo, defaultCtx.branch, paths)
		if err != nil {
			printWarning("Failed to sync templates: %v", err)
		} else if templateCount > 0 {
			printSuccess("Synced %d templates", templateCount)
		}
	}

	// Sync languages with multi-source support
	if opts.shouldSyncLanguages() {
		languageCount, err := syncLanguagesMultiSource(ctx, client, cfg, repoContexts, paths)
		if err != nil {
			printWarning("Failed to sync languages: %v", err)
		} else if languageCount > 0 {
			printSuccess("Synced %d language configs", languageCount)
		}
	}

	// Sync evals from default repo
	if opts.shouldSyncEvals() {
		evalCount, err := syncEvals(ctx, client, defaultCtx.owner, defaultCtx.repo, defaultCtx.branch, paths)
		if err != nil {
			printWarning("Failed to sync evals: %v", err)
		} else if evalCount > 0 {
			printSuccess("Synced %d evals", evalCount)
		}
	}

	// Sync rules from default repo (no per-rule multi-source support)
	if opts.shouldSyncRules() {
		ruleCount, err := syncRules(ctx, client, defaultCtx.owner, defaultCtx.repo, defaultCtx.branch, paths)
		if err != nil {
			printWarning("Failed to sync rules: %v", err)
		} else if ruleCount > 0 {
			printSuccess("Synced %d rules", ruleCount)
		}
	}

	// Sync skills with multi-source support
	if opts.shouldSyncSkills() {
		skillCount, err := syncSkillsMultiSource(ctx, client, cfg, repoContexts, paths)
		if err != nil {
			printWarning("Failed to sync skills: %v", err)
		} else if skillCount > 0 {
			printSuccess("Synced %d skills", skillCount)
		}
	}

	// Apply config
	if opts.shouldApplyConfig() {
		fmt.Println()
		if err := applyConfigFromMultiSource(cfg, paths, repoContexts); err != nil {
			return err
		}
	}

	// Sync commands to Claude Code
	if opts.shouldSyncClaudeCommands() {
		claudeCount, err := syncClaudeCommands(paths, defaultCtx.owner, defaultCtx.repo)
		if err != nil {
			printWarning("Failed to sync Claude commands: %v", err)
		} else if claudeCount > 0 {
			printSuccess("Synced %d commands to Claude Code", claudeCount)
		}
	}

	// Sync rules to Claude Code
	if opts.shouldSyncClaudeRules() {
		claudeRuleCount, err := syncClaudeRules(paths, defaultCtx.owner, defaultCtx.repo)
		if err != nil {
			printWarning("Failed to sync Claude rules: %v", err)
		} else if claudeRuleCount > 0 {
			printSuccess("Synced %d rules to Claude Code", claudeRuleCount)
		}
	}

	// Sync skills to Claude Code
	if opts.shouldSyncClaudeSkills() {
		claudeSkillCount, err := syncClaudeSkills(paths, defaultCtx.owner, defaultCtx.repo)
		if err != nil {
			printWarning("Failed to sync Claude skills: %v", err)
		} else if claudeSkillCount > 0 {
			printSuccess("Synced %d skills to Claude Code", claudeSkillCount)
			fmt.Printf("  %s Skills are available via /skill-name in Claude Code\n", dim("Tip:"))
		}
	}

	// Check config size
	if !opts.fetchOnly {
		checkConfigSizeAndSuggestOptimize(cfg, paths, defaultCtx.owner, defaultCtx.repo)
	}

	return nil
}

// isExplicitlyConfiguredLanguage returns true if the language has an explicit source configured.
func isExplicitlyConfiguredLanguage(cfg *config.Config, lang string) bool {
	if cfg.Source.Multi != nil && cfg.Source.Multi.Languages != nil {
		_, ok := cfg.Source.Multi.Languages[lang]
		return ok
	}
	return false
}

// isExplicitlyConfiguredCommand returns true if the command has an explicit source configured.
func isExplicitlyConfiguredCommand(cfg *config.Config, cmd string) bool {
	if cfg.Source.Multi != nil && cfg.Source.Multi.Commands != nil {
		_, ok := cfg.Source.Multi.Commands[cmd]
		return ok
	}
	return false
}

// handleMultiSourceFetchError logs appropriate warnings for fetch errors.
// For explicitly configured items, always warn. For items using default repo,
// only warn on non-404 errors (silently skip if not found).
func handleMultiSourceFetchError(itemType, name, sourceRepo string, err error, isExplicit bool) {
	if github.IsNotFoundError(err) {
		if isExplicit {
			printWarning("%s %s not found in explicitly configured source %s", itemType, name, sourceRepo)
		}
		// Silently skip 404s for non-explicit items
		return
	}
	// Always warn on non-404 errors (network issues, auth problems, etc.)
	printWarning("Failed to fetch %s %s from %s: %v", itemType, name, sourceRepo, err)
}

// syncLanguagesMultiSource fetches languages from their configured source repos.
func syncLanguagesMultiSource(ctx context.Context, client *github.Client, cfg *config.Config, repoContexts map[string]*repoContext, paths *config.Paths) (int, error) {
	// First, discover all languages from the default repo
	defaultRepoStr := cfg.Source.DefaultRepo()
	defaultCtx := repoContexts[defaultRepoStr]
	if defaultCtx == nil {
		return 0, fmt.Errorf("no context for default repo %s", defaultRepoStr)
	}

	// Get languages from default repo
	allLanguages := make(map[string]bool)
	entries, err := client.ListDirectory(ctx, defaultCtx.owner, defaultCtx.repo, "languages", defaultCtx.branch)
	if err == nil && entries != nil {
		for _, entry := range entries {
			if entry.Type == "file" && strings.HasSuffix(entry.Name, ".md") {
				lang := strings.TrimSuffix(entry.Name, ".md")
				allLanguages[lang] = true
			}
		}
	}

	// Add any explicitly configured language sources
	if cfg.Source.Multi != nil && cfg.Source.Multi.Languages != nil {
		for lang := range cfg.Source.Multi.Languages {
			allLanguages[lang] = true
		}
	}

	// Sync each language from its configured source
	count := 0
	for lang := range allLanguages {
		sourceRepoStr := cfg.Source.RepoForLanguage(lang)
		repoCtx := repoContexts[sourceRepoStr]
		if repoCtx == nil {
			printWarning("No context for language %s source %s", lang, sourceRepoStr)
			continue
		}

		// Fetch this language from its source
		langPath := fmt.Sprintf("languages/%s.md", lang)
		result, err := client.FetchFile(ctx, repoCtx.owner, repoCtx.repo, langPath, repoCtx.branch)
		if err != nil {
			handleMultiSourceFetchError("language", lang, sourceRepoStr, err, isExplicitlyConfiguredLanguage(cfg, lang))
			continue
		}

		// Store in the repo-specific cache directory
		langDir := paths.TeamLanguagesDir(repoCtx.owner, repoCtx.repo)
		if err := os.MkdirAll(langDir, 0755); err != nil {
			printWarning("Failed to create language directory for %s: %v", lang, err)
			continue
		}

		localPath := filepath.Join(langDir, lang+".md")
		if err := os.WriteFile(localPath, []byte(result.Content), 0644); err != nil {
			printWarning("Failed to write language %s: %v", lang, err)
			continue
		}

		count++
	}

	return count, nil
}

// syncCommandsMultiSource fetches commands from their configured source repos.
func syncCommandsMultiSource(ctx context.Context, client *github.Client, cfg *config.Config, repoContexts map[string]*repoContext, paths *config.Paths) (int, error) {
	// First, discover all commands from the default repo
	defaultRepoStr := cfg.Source.DefaultRepo()
	defaultCtx := repoContexts[defaultRepoStr]
	if defaultCtx == nil {
		return 0, fmt.Errorf("no context for default repo %s", defaultRepoStr)
	}

	// Get commands from default repo
	allCommands := make(map[string]bool)
	entries, err := client.ListDirectory(ctx, defaultCtx.owner, defaultCtx.repo, "commands", defaultCtx.branch)
	if err == nil && entries != nil {
		for _, entry := range entries {
			if entry.Type == "file" && strings.HasSuffix(entry.Name, ".md") {
				cmd := strings.TrimSuffix(entry.Name, ".md")
				allCommands[cmd] = true
			}
		}
	}

	// Add any explicitly configured command sources
	if cfg.Source.Multi != nil && cfg.Source.Multi.Commands != nil {
		for cmd := range cfg.Source.Multi.Commands {
			allCommands[cmd] = true
		}
	}

	// Sync each command from its configured source
	count := 0
	for cmd := range allCommands {
		sourceRepoStr := cfg.Source.RepoForCommand(cmd)
		repoCtx := repoContexts[sourceRepoStr]
		if repoCtx == nil {
			printWarning("No context for command %s source %s", cmd, sourceRepoStr)
			continue
		}

		// Fetch this command from its source
		cmdPath := fmt.Sprintf("commands/%s.md", cmd)
		result, err := client.FetchFile(ctx, repoCtx.owner, repoCtx.repo, cmdPath, repoCtx.branch)
		if err != nil {
			handleMultiSourceFetchError("command", cmd, sourceRepoStr, err, isExplicitlyConfiguredCommand(cfg, cmd))
			continue
		}

		// Store in the repo-specific cache directory
		cmdDir := paths.TeamCommandsDir(repoCtx.owner, repoCtx.repo)
		if err := os.MkdirAll(cmdDir, 0755); err != nil {
			printWarning("Failed to create commands directory for %s: %v", cmd, err)
			continue
		}

		localPath := filepath.Join(cmdDir, cmd+".md")
		if err := os.WriteFile(localPath, []byte(result.Content), 0644); err != nil {
			printWarning("Failed to write command %s: %v", cmd, err)
			continue
		}

		count++
	}

	return count, nil
}

// applyConfigFromMultiSource merges configs from multiple source repos.
func applyConfigFromMultiSource(cfg *config.Config, paths *config.Paths, repoContexts map[string]*repoContext) error {
	// Get base config from the base source repo
	baseRepoStr := cfg.Source.RepoForBase()
	baseCtx := repoContexts[baseRepoStr]
	if baseCtx == nil {
		// This should not happen if buildRepoContexts was called correctly
		return fmt.Errorf("internal error: no context for base repo %s", baseRepoStr)
	}

	teamConfig, err := os.ReadFile(paths.CacheFile(baseCtx.owner, baseCtx.repo))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no cached team config found for %s/%s", baseCtx.owner, baseCtx.repo)
		}
		return fmt.Errorf("failed to read cached config: %w", err)
	}

	// Get personal config (optional)
	personalConfig, err := readPersonalConfig(paths)
	if err != nil {
		return err
	}

	// Collect all team language directories from all repos (sorted for deterministic output)
	var repoKeys []string
	for k := range repoContexts {
		repoKeys = append(repoKeys, k)
	}
	sort.Strings(repoKeys)
	var teamLangDirs []string
	for _, k := range repoKeys {
		ctx := repoContexts[k]
		teamLangDirs = append(teamLangDirs, paths.TeamLanguagesDir(ctx.owner, ctx.repo))
	}

	// Resolve active languages (sorted for deterministic output)
	activeLanguages := resolveActiveLanguages(cfg, teamLangDirs, paths.PersonalLanguages)

	// Load language files from their respective source repos
	languageFiles := loadMultiSourceLanguageFiles(cfg, paths, repoContexts, activeLanguages)

	return mergeAndWriteConfig(cfg, paths, teamConfig, personalConfig, activeLanguages, languageFiles)
}

// loadMultiSourceLanguageFiles loads language files from their respective source repos.
func loadMultiSourceLanguageFiles(cfg *config.Config, paths *config.Paths, repoContexts map[string]*repoContext, activeLanguages []string) map[string][]*language.LanguageFile {
	if len(activeLanguages) == 0 {
		return nil
	}

	languageFiles := make(map[string][]*language.LanguageFile)
	for _, lang := range activeLanguages {
		// Determine the source repo for this language
		sourceRepoStr := cfg.Source.RepoForLanguage(lang)
		repoCtx := repoContexts[sourceRepoStr]
		var teamLangDir string
		if repoCtx != nil {
			teamLangDir = paths.TeamLanguagesDir(repoCtx.owner, repoCtx.repo)
		}

		files, err := language.LoadLanguageFiles(
			[]string{lang},
			teamLangDir,
			paths.PersonalLanguages,
			"",
		)
		if err != nil {
			printWarning("Failed to load language files for %s: %v", lang, err)
			continue
		}
		if langFiles, ok := files[lang]; ok {
			languageFiles[lang] = langFiles
		}
	}
	return languageFiles
}

// syncSkills fetches skills from the team repo's skills/ directory.
// Skills are directories containing SKILL.md plus optional supporting files.
func syncSkills(ctx context.Context, client *github.Client, owner, repo, branch string, paths *config.Paths) (int, error) {
	// List skills directory (top-level entries are skill directories)
	entries, err := client.ListDirectory(ctx, owner, repo, "skills", branch)
	if err != nil {
		return 0, err
	}

	if entries == nil {
		return 0, nil
	}

	// Create local skills cache directory
	skillsDir := paths.TeamSkillsDir(owner, repo)

	// Clear existing cache to handle deletions
	if err := os.RemoveAll(skillsDir); err != nil {
		return 0, fmt.Errorf("failed to clear skills cache: %w", err)
	}

	// Sync each skill directory
	count := 0
	for _, entry := range entries {
		if entry.Type != "dir" {
			continue
		}

		// Sync this skill directory recursively
		skillLocalDir := filepath.Join(skillsDir, entry.Name)
		fileCount, err := syncSkillDir(ctx, client, owner, repo, branch, entry.Path, skillLocalDir)
		if err != nil {
			printWarning("Failed to sync skill %s: %v", entry.Name, err)
			continue
		}

		if fileCount > 0 {
			count++
		}
	}

	return count, nil
}

// syncSkillDir syncs a single skill directory recursively.
func syncSkillDir(ctx context.Context, client *github.Client, owner, repo, branch, remotePath, localDir string) (int, error) {
	entries, err := client.ListDirectory(ctx, owner, repo, remotePath, branch)
	if err != nil {
		return 0, err
	}

	if entries == nil {
		return 0, nil
	}

	// Ensure local directory exists
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create skill directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.Type == "dir" {
			// Recurse into subdirectory
			subLocalDir := filepath.Join(localDir, entry.Name)
			subCount, err := syncSkillDir(ctx, client, owner, repo, branch, entry.Path, subLocalDir)
			if err != nil {
				printWarning("Failed to sync skill subdirectory %s: %v", entry.Name, err)
				continue
			}
			count += subCount
		} else if entry.Type == "file" {
			// Fetch and cache file
			result, err := client.FetchFile(ctx, owner, repo, entry.Path, branch)
			if err != nil {
				printWarning("Failed to fetch skill file %s: %v", entry.Name, err)
				continue
			}

			localPath := filepath.Join(localDir, entry.Name)
			if err := os.WriteFile(localPath, []byte(result.Content), 0644); err != nil {
				printWarning("Failed to write skill file %s: %v", entry.Name, err)
				continue
			}

			count++
		}
	}

	return count, nil
}

// syncClaudeSkills syncs staghorn skills to Claude Code skills directory.
func syncClaudeSkills(paths *config.Paths, owner, repo string) (int, error) {
	// Load skills from all sources using the registry
	registry, err := skills.LoadRegistry(
		paths.TeamSkillsDir(owner, repo),
		paths.PersonalSkills,
		"", // No project dir for global sync
	)
	if err != nil {
		return 0, fmt.Errorf("failed to load skills: %w", err)
	}

	allSkills := registry.All()
	if len(allSkills) == 0 {
		return 0, nil
	}

	// Create Claude skills directory
	claudeDir := paths.ClaudeSkillsDir()
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create Claude skills directory: %w", err)
	}

	// Sync each skill
	count := 0
	for _, skill := range allSkills {
		filesWritten, err := skills.SyncToClaude(skill, claudeDir)
		if err != nil {
			if strings.Contains(err.Error(), "not managed by staghorn") {
				printWarning("Skipping skill %s: existing skill not managed by staghorn", skill.Name)
			} else {
				printWarning("Failed to sync skill %s: %v", skill.Name, err)
			}
			continue
		}
		if filesWritten > 0 {
			count++
		}
	}

	return count, nil
}

// isExplicitlyConfiguredSkill returns true if the skill has an explicit source configured.
func isExplicitlyConfiguredSkill(cfg *config.Config, skill string) bool {
	if cfg.Source.Multi != nil && cfg.Source.Multi.Skills != nil {
		_, ok := cfg.Source.Multi.Skills[skill]
		return ok
	}
	return false
}

// syncSkillsMultiSource fetches skills from their configured source repos.
func syncSkillsMultiSource(ctx context.Context, client *github.Client, cfg *config.Config, repoContexts map[string]*repoContext, paths *config.Paths) (int, error) {
	// First, discover all skills from the default repo
	defaultRepoStr := cfg.Source.DefaultRepo()
	defaultCtx := repoContexts[defaultRepoStr]
	if defaultCtx == nil {
		return 0, fmt.Errorf("no context for default repo %s", defaultRepoStr)
	}

	// Get skills from default repo
	allSkills := make(map[string]bool)
	entries, err := client.ListDirectory(ctx, defaultCtx.owner, defaultCtx.repo, "skills", defaultCtx.branch)
	if err == nil && entries != nil {
		for _, entry := range entries {
			if entry.Type == "dir" {
				allSkills[entry.Name] = true
			}
		}
	}

	// Add any explicitly configured skill sources
	if cfg.Source.Multi != nil && cfg.Source.Multi.Skills != nil {
		for skill := range cfg.Source.Multi.Skills {
			allSkills[skill] = true
		}
	}

	// Sync each skill from its configured source
	count := 0
	for skill := range allSkills {
		sourceRepoStr := cfg.Source.RepoForSkill(skill)
		repoCtx := repoContexts[sourceRepoStr]
		if repoCtx == nil {
			printWarning("No context for skill %s source %s", skill, sourceRepoStr)
			continue
		}

		// Sync this skill from its source
		skillPath := fmt.Sprintf("skills/%s", skill)
		skillLocalDir := filepath.Join(paths.TeamSkillsDir(repoCtx.owner, repoCtx.repo), skill)

		// Check if skill exists in remote
		entries, err := client.ListDirectory(ctx, repoCtx.owner, repoCtx.repo, skillPath, repoCtx.branch)
		if err != nil {
			handleMultiSourceFetchError("skill", skill, sourceRepoStr, err, isExplicitlyConfiguredSkill(cfg, skill))
			continue
		}
		if entries == nil {
			if isExplicitlyConfiguredSkill(cfg, skill) {
				printWarning("Skill %s not found in explicitly configured source %s", skill, sourceRepoStr)
			}
			continue
		}

		// Sync the skill directory
		fileCount, err := syncSkillDir(ctx, client, repoCtx.owner, repoCtx.repo, repoCtx.branch, skillPath, skillLocalDir)
		if err != nil {
			printWarning("Failed to sync skill %s from %s: %v", skill, sourceRepoStr, err)
			continue
		}

		if fileCount > 0 {
			count++
		}
	}

	return count, nil
}
