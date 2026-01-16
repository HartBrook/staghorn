package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HartBrook/staghorn/internal/cache"
	"github.com/HartBrook/staghorn/internal/commands"
	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/errors"
	"github.com/HartBrook/staghorn/internal/github"
	"github.com/HartBrook/staghorn/internal/language"
	"github.com/HartBrook/staghorn/internal/merge"
	"github.com/spf13/cobra"
)

type syncOptions struct {
	force         bool
	offline       bool
	configOnly    bool
	commandsOnly  bool
	languagesOnly bool
	fetchOnly     bool
	applyOnly     bool
	claudeOnly    bool
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
	cmd.Flags().BoolVar(&opts.configOnly, "config-only", false, "Only sync config, skip commands and languages")
	cmd.Flags().BoolVar(&opts.commandsOnly, "commands-only", false, "Only sync commands, skip config and languages")
	cmd.Flags().BoolVar(&opts.languagesOnly, "languages-only", false, "Only sync languages, skip config and commands")
	cmd.Flags().BoolVar(&opts.claudeOnly, "claude-only", false, "Only sync commands to ~/.claude/commands/, skip config apply")

	return cmd
}

func runSync(ctx context.Context, opts *syncOptions) error {
	paths := config.NewPaths()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	owner, repo, err := cfg.Team.ParseRepo()
	if err != nil {
		return err
	}

	c := cache.New(paths)

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

	// Determine branch
	branch := cfg.Team.Branch
	if branch == "" {
		branch, err = client.GetDefaultBranch(ctx, owner, repo)
		if err != nil {
			return errors.GitHubFetchFailed(owner+"/"+repo, err)
		}
	}

	fmt.Printf("Fetching %s/%s...\n", owner, repo)

	// Sync config unless --commands-only, --languages-only, or --claude-only was specified
	if !opts.commandsOnly && !opts.languagesOnly && !opts.claudeOnly {
		result, err := client.FetchFile(ctx, owner, repo, cfg.Team.Path, branch)
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
		printInfo("File", cfg.Team.Path)
		printInfo("SHA", result.SHA[:8])
	}

	// Sync commands unless --config-only, --languages-only, or --claude-only was specified
	if !opts.configOnly && !opts.languagesOnly && !opts.claudeOnly {
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

	// Sync languages unless --config-only, --commands-only, or --claude-only was specified
	if !opts.configOnly && !opts.commandsOnly && !opts.claudeOnly {
		languageCount, err := syncLanguages(ctx, client, owner, repo, branch, paths)
		if err != nil {
			printWarning("Failed to sync languages: %v", err)
		} else if languageCount > 0 {
			printSuccess("Synced %d language configs", languageCount)
		} else if opts.languagesOnly {
			fmt.Println("No language configs found in team repository")
		}
	}

	// Apply to ~/.claude/CLAUDE.md unless --fetch-only was specified
	if !opts.fetchOnly && !opts.claudeOnly {
		fmt.Println()
		if err := applyConfig(cfg, paths, owner, repo); err != nil {
			return err
		}
	}

	// Sync commands to Claude Code (always runs unless --fetch-only, --config-only, or --languages-only)
	if !opts.configOnly && !opts.languagesOnly && !opts.fetchOnly {
		claudeCount, err := syncClaudeCommands(paths, owner, repo)
		if err != nil {
			printWarning("Failed to sync Claude commands: %v", err)
		} else if claudeCount > 0 {
			printSuccess("Synced %d commands to Claude Code", claudeCount)
			fmt.Printf("  %s Use /%s in Claude Code\n", dim("Tip:"), "code-review")
		}
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
			if !strings.Contains(string(existingContent), "Managed by staghorn") {
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
	var personalConfig []byte
	if _, err := os.Stat(paths.PersonalMD); err == nil {
		personalConfig, err = os.ReadFile(paths.PersonalMD)
		if err != nil {
			return fmt.Errorf("failed to read personal config: %w", err)
		}
		// Strip instructional comments from personal config
		personalConfig = []byte(stripInstructionalComments(string(personalConfig)))
	}

	// Resolve languages for global config
	// For global ~/.claude/CLAUDE.md, include ALL available language configs from team + personal
	// (auto-detect only makes sense for project-level configs)
	teamLangDir := paths.TeamLanguagesDir(owner, repo)
	personalLangDir := paths.PersonalLanguages

	var activeLanguages []string
	var languageFiles map[string][]*language.LanguageFile

	if len(cfg.Languages.Enabled) > 0 {
		// Explicit list takes precedence
		activeLanguages = language.FilterDisabled(cfg.Languages.Enabled, cfg.Languages.Disabled)
	} else {
		// Default: include all available languages from team + personal
		activeLanguages, _ = language.ListAvailableLanguages(teamLangDir, personalLangDir, "")
		activeLanguages = language.FilterDisabled(activeLanguages, cfg.Languages.Disabled)
	}

	if len(activeLanguages) > 0 {
		languageFiles, _ = language.LoadLanguageFiles(
			activeLanguages,
			teamLangDir,
			personalLangDir,
			"", // No project dir for global config
		)
	}

	// Merge configs with language support
	layers := []merge.Layer{
		{Content: string(teamConfig), Source: "team"},
		{Content: string(personalConfig), Source: "personal"},
	}
	mergeOpts := merge.MergeOptions{
		Languages:     activeLanguages,
		LanguageFiles: languageFiles,
	}
	merged := merge.MergeWithLanguages(layers, mergeOpts)

	// Add header comment
	header := fmt.Sprintf("<!-- Managed by staghorn | Team: %s | Do not edit directly -->\n\n", cfg.Team.Repo)
	output := header + merged

	// Ensure ~/.claude directory exists
	claudeDir := filepath.Join(os.Getenv("HOME"), ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create ~/.claude directory: %w", err)
	}

	// Check for existing unmanaged content
	outputPath := filepath.Join(claudeDir, "CLAUDE.md")
	if existingContent, err := os.ReadFile(outputPath); err == nil {
		// File exists - check if it's managed by staghorn
		if !strings.Contains(string(existingContent), "Managed by staghorn") {
			printWarning("Found existing ~/.claude/CLAUDE.md not managed by staghorn")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  1. Migrate content to personal config (recommended)")
			fmt.Println("  2. Back up existing file and continue")
			fmt.Println("  3. Abort")
			fmt.Println()

			choice := promptString("Choose an option [1/2/3]:")

			switch choice {
			case "1", "":
				// Migrate to personal config
				existingPersonal, _ := os.ReadFile(paths.PersonalMD)
				newPersonal := string(existingPersonal)
				if len(newPersonal) > 0 {
					newPersonal += "\n\n"
				}
				newPersonal += "<!-- [staghorn] Migrated from ~/.claude/CLAUDE.md -->\n\n" + string(existingContent)

				if err := os.MkdirAll(paths.ConfigDir, 0755); err != nil {
					return fmt.Errorf("failed to create config directory: %w", err)
				}
				if err := os.WriteFile(paths.PersonalMD, []byte(newPersonal), 0644); err != nil {
					return fmt.Errorf("failed to write personal config: %w", err)
				}
				printSuccess("Migrated content to %s", paths.PersonalMD)
				fmt.Printf("  %s Run 'staghorn edit' to review and organize\n", dim("Tip:"))
				fmt.Println()

				// Re-read personal config for merge
				personalConfig, _ = os.ReadFile(paths.PersonalMD)
				layers[1] = merge.Layer{Content: string(personalConfig), Source: "personal"}
				merged = merge.MergeWithLanguages(layers, mergeOpts)
				output = header + merged

			case "2":
				// Back up and continue
				backupPath := outputPath + ".backup"
				if err := os.WriteFile(backupPath, existingContent, 0644); err != nil {
					return fmt.Errorf("failed to backup existing file: %w", err)
				}
				printSuccess("Backed up to %s", backupPath)
				fmt.Println()

			case "3":
				fmt.Println("Aborted.")
				return nil

			default:
				return fmt.Errorf("invalid option")
			}
		}
	}

	// Write to ~/.claude/CLAUDE.md
	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	printSuccess("Applied to %s", outputPath)

	// Show what was merged
	hasPersonal := len(personalConfig) > 0
	if hasPersonal {
		fmt.Printf("  %s Team config + personal additions\n", dim("Merged:"))
	} else {
		fmt.Printf("  %s Team config only (no personal additions)\n", dim("Merged:"))
		fmt.Printf("  %s Run 'staghorn edit' to add personal preferences\n", dim("Tip:"))
	}

	return nil
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
