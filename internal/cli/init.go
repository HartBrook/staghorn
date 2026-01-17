package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/errors"
	"github.com/HartBrook/staghorn/internal/github"
	"github.com/HartBrook/staghorn/internal/language"
	"github.com/HartBrook/staghorn/internal/starter"
	"github.com/spf13/cobra"
)

const (
	// maxSearchResultsDisplay is the maximum number of search results to show in interactive mode.
	maxSearchResultsDisplay = 5
)

// NewInitCmd creates the init command.
func NewInitCmd() *cobra.Command {
	var fromRepo string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize staghorn configuration",
		Long: `Interactive setup for staghorn.

This command will help you set up staghorn by either:
1. Browsing and selecting a public config from the community
2. Connecting to a private repository
3. Starting fresh with just the starter commands

You can also install directly from a specific repository:
  staghorn init --from owner/repo`,
		Example: `  staghorn init
  staghorn init --from staghorn-io/python-standards
  staghorn init --from https://github.com/acme/claude-standards`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromRepo != "" {
				return runInitFrom(cmd.Context(), fromRepo)
			}
			return runInit(cmd, args)
		},
	}

	cmd.Flags().StringVar(&fromRepo, "from", "", "Install directly from owner/repo")

	return cmd
}

// runInitFrom handles direct installation from a specific repo.
func runInitFrom(ctx context.Context, repoStr string) error {
	paths := config.NewPaths()

	// Check if already configured
	if config.Exists() {
		fmt.Println("Staghorn is already configured.")
		fmt.Printf("Config file: %s\n\n", paths.ConfigFile)

		if !promptYesNo("Do you want to reconfigure?") {
			return nil
		}
		fmt.Println()
	}

	// Parse and validate repo
	owner, repo, err := config.ParseRepo(repoStr)
	if err != nil {
		return err
	}

	fullRepo := owner + "/" + repo

	// Try to create a client (prefer authenticated, fall back to unauthenticated for public repos)
	client, err := createClient()
	if err != nil {
		return err
	}

	fmt.Printf("Verifying access to %s...\n", fullRepo)

	// Check if repo exists
	exists, err := client.RepoExists(ctx, owner, repo)
	if err != nil {
		return errors.GitHubFetchFailed(fullRepo, err)
	}
	if !exists {
		return fmt.Errorf("repository %s not found or not accessible", fullRepo)
	}

	// Check if CLAUDE.md exists
	fileExists, err := client.FileExists(ctx, owner, repo, config.DefaultPath, "")
	if err != nil {
		printWarning("Could not verify %s exists: %v", config.DefaultPath, err)
	} else if !fileExists {
		printWarning("%s not found in repository", config.DefaultPath)
		if !promptYesNo("Continue anyway?") {
			return nil
		}
	}

	printSuccess("Repository verified")

	// Check trust and warn if needed
	cfg := config.NewSimpleConfig(fullRepo)
	if !cfg.IsTrustedSource(fullRepo) {
		fmt.Println()
		fmt.Println(config.TrustWarning(fullRepo))
		if !promptYesNo("Proceed with installation?") {
			return nil
		}
	}

	// Save config
	fmt.Println()
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	printSuccess("Config saved to %s", paths.ConfigFile)

	// Perform initial fetch (but don't apply yet - we need language selection first)
	fmt.Println()
	fmt.Println("Fetching config from source...")
	syncOpts := &syncOptions{force: true, fetchOnly: true}
	if err := runSync(ctx, syncOpts); err != nil {
		printWarning("Initial fetch failed: %v", err)
		fmt.Println("You can run `staghorn sync` later to fetch the config.")
	}

	// Create personal.md if it doesn't exist
	if err := ensurePersonalMD(paths); err != nil {
		printWarning("Failed to create personal config: %v", err)
	}

	// Offer language selection BEFORE applying (so sync respects the selection)
	offerLanguageConfigs(paths, owner, repo)

	// Offer starter commands
	offerStarterCommands(paths)

	// Now apply with the language selection in place
	fmt.Println()
	fmt.Println("Applying config...")
	applyOpts := &syncOptions{applyOnly: true}
	if err := runSync(ctx, applyOpts); err != nil {
		printWarning("Failed to apply config: %v", err)
	}

	printInitComplete()
	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	paths := config.NewPaths()

	// Check if already configured
	if config.Exists() {
		fmt.Println("Staghorn is already configured.")
		fmt.Printf("Config file: %s\n\n", paths.ConfigFile)

		if !promptYesNo("Do you want to reconfigure?") {
			return nil
		}
		fmt.Println()
	}

	fmt.Println("Welcome to Staghorn!")
	fmt.Println()
	fmt.Println("How would you like to configure Claude Code?")
	fmt.Println()
	fmt.Println("  1. Browse public configs")
	fmt.Println("  2. Connect to a repository (public or private)")
	fmt.Println("  3. Start fresh with just starter commands")
	fmt.Println()

	choice := promptString("Choice [1/2/3]:")

	switch choice {
	case "1", "":
		return initFromPublic(cmd.Context(), paths)
	case "2":
		return initFromRepo(cmd.Context(), paths)
	case "3":
		return initFresh(paths)
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

// initFromPublic handles browsing and selecting a public config.
func initFromPublic(ctx context.Context, paths *config.Paths) error {
	fmt.Println()
	fmt.Println("Searching for public configs...")

	// Try unauthenticated first for public repos
	client, err := github.NewUnauthenticatedClient()
	if err != nil {
		// Fall back to authenticated
		client, err = github.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}
	}

	results, err := client.SearchConfigs(ctx, "")
	if err != nil {
		printWarning("Search failed: %v", err)
		fmt.Println()
		fmt.Println("You can install directly with: staghorn init --from owner/repo")
		return nil
	}

	if len(results) == 0 {
		fmt.Println("No public configs found with the 'staghorn-config' topic.")
		fmt.Println()
		fmt.Println("You can:")
		fmt.Println("  - Install directly: staghorn init --from owner/repo")
		fmt.Println("  - Connect to a private repo: staghorn init (choose option 2)")
		return nil
	}

	// Display results
	fmt.Println()
	fmt.Println("Available public configs:")
	fmt.Println()

	maxDisplay := maxSearchResultsDisplay
	if len(results) < maxDisplay {
		maxDisplay = len(results)
	}

	for i := 0; i < maxDisplay; i++ {
		r := results[i]
		fmt.Printf("  %d. %s/%s", i+1, r.Owner, r.Repo)
		if r.Stars > 0 {
			fmt.Printf(" ★ %d", r.Stars)
		}
		fmt.Println()
		if r.Description != "" {
			fmt.Printf("     %s\n", truncate(r.Description, 60))
		}
		fmt.Println()
	}

	if len(results) > maxDisplay {
		fmt.Printf("  ... and %d more. Use 'staghorn search' to see all.\n\n", len(results)-maxDisplay)
	}

	// Prompt for selection
	fmt.Println("Enter a number to install, or type a search query:")
	input := promptString("Selection:")

	// Check if it's a number
	var selectedIdx int
	if _, err := fmt.Sscanf(input, "%d", &selectedIdx); err == nil {
		if selectedIdx < 1 || selectedIdx > maxDisplay {
			return fmt.Errorf("invalid selection: %d", selectedIdx)
		}
		selected := results[selectedIdx-1]
		return runInitFrom(ctx, selected.FullName())
	}

	// Treat as search query
	if input != "" {
		results, err = client.SearchConfigs(ctx, input)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		if len(results) == 0 {
			fmt.Println("No configs found matching your query.")
			return nil
		}

		// Show filtered results and prompt again
		fmt.Println()
		fmt.Printf("Found %d configs matching '%s':\n\n", len(results), input)
		displayCount := min(len(results), maxSearchResultsDisplay)
		for i := 0; i < displayCount; i++ {
			r := results[i]
			fmt.Printf("  %d. %s/%s", i+1, r.Owner, r.Repo)
			if r.Stars > 0 {
				fmt.Printf(" ★ %d", r.Stars)
			}
			fmt.Println()
		}
		fmt.Println()

		input = promptString("Enter number to install:")
		if _, err := fmt.Sscanf(input, "%d", &selectedIdx); err == nil {
			if selectedIdx >= 1 && selectedIdx <= displayCount {
				selected := results[selectedIdx-1]
				return runInitFrom(ctx, selected.FullName())
			}
		}
		return fmt.Errorf("invalid selection: please enter a number between 1 and %d", displayCount)
	}

	return nil
}

// initFromRepo handles connecting to any repository (public or private).
func initFromRepo(ctx context.Context, paths *config.Paths) error {
	fmt.Println()
	repoURL := promptString("Repository (e.g., owner/repo or https://github.com/owner/repo):")
	if repoURL == "" {
		return fmt.Errorf("repository is required")
	}

	// Use runInitFrom which handles both public and private repos
	return runInitFrom(ctx, repoURL)
}

// initFresh sets up staghorn with just starter commands, no remote source.
func initFresh(paths *config.Paths) error {
	fmt.Println()
	fmt.Println("Starting fresh with starter commands only.")
	fmt.Println()

	// Create a minimal config file (empty source is valid for local-only mode)
	cfg := &config.Config{
		Version: config.DefaultVersion,
		Cache: config.CacheConfig{
			TTL: config.DefaultCacheTTL,
		},
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	printSuccess("Config saved to %s", paths.ConfigFile)

	// Create personal.md
	if err := ensurePersonalMD(paths); err != nil {
		printWarning("Failed to create personal config: %v", err)
	}

	// Install starter commands
	starterCommands := starter.CommandNames()
	if len(starterCommands) > 0 {
		fmt.Printf("Installing %d starter commands...\n", len(starterCommands))
		result, err := InstallStarterCommands(paths.PersonalCommands, false)
		if err != nil {
			printWarning("Failed to install starter commands: %v", err)
		} else if len(result.Installed) > 0 {
			printSuccess("Installed %d starter commands", len(result.Installed))
		}
	}

	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Println()
	fmt.Println("You're set up without a remote source. To add one later:")
	fmt.Printf("  %s --from owner/repo\n", info("staghorn init"))
	fmt.Println()
	fmt.Println("Or edit your config directly:")
	fmt.Printf("  %s\n", info(paths.ConfigFile))

	return nil
}

// Helper functions

func createClient() (*github.Client, error) {
	// Try authenticated first
	client, err := github.NewClient()
	if err == nil {
		return client, nil
	}

	// Try with token from env
	token := github.GetTokenFromEnv()
	if token != "" {
		return github.NewClientWithToken(token)
	}

	// Fall back to unauthenticated for public repos
	return github.NewUnauthenticatedClient()
}

func offerStarterCommands(paths *config.Paths) {
	starterCommands := starter.CommandNames()
	if len(starterCommands) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("Staghorn includes %d starter commands (code-review, debug, refactor, etc.)\n", len(starterCommands))
	if promptYesNo("Install starter commands to your personal config?") {
		result, err := InstallStarterCommands(paths.PersonalCommands, true)
		if err != nil {
			printWarning("Failed to install starter commands: %v", err)
		} else if result.Aborted {
			fmt.Println("  Skipped starter commands installation")
		} else if len(result.Installed) > 0 {
			printSuccess("Installed %d starter commands to %s", len(result.Installed), paths.PersonalCommands)
			if len(result.Skipped) > 0 {
				fmt.Printf("  Skipped %d (using source versions): %s\n", len(result.Skipped), strings.Join(result.Skipped, ", "))
			}
			fmt.Printf("  %s Run '%s' to see them\n", dim("Tip:"), info("staghorn commands"))
		} else if len(result.Skipped) > 0 {
			fmt.Printf("  Skipped %d commands (using source versions)\n", len(result.Skipped))
		} else {
			fmt.Println("  Starter commands already installed")
		}
	}
}

func offerLanguageConfigs(paths *config.Paths, owner, repo string) {
	sourceLangDir := paths.TeamLanguagesDir(owner, repo)
	sourceLangs, _ := listLanguageFiles(sourceLangDir)
	if len(sourceLangs) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("Your source has %d language configs: %s\n", len(sourceLangs), strings.Join(sourceLangs, ", "))
	fmt.Println()

	// Ask which languages to enable
	fmt.Println("Which languages do you want to enable?")
	fmt.Println("  1. All of them")
	fmt.Println("  2. Let me choose")
	fmt.Println("  3. None (skip language configs)")
	fmt.Println()

	choice := promptString("Choice [1/2/3]:")

	var enabledLangs []string

	var useAll bool

	switch choice {
	case "1", "":
		enabledLangs = sourceLangs
		useAll = true
		printSuccess("Enabled all %d languages", len(enabledLangs))
	case "2":
		enabledLangs = selectLanguages(sourceLangs)
		useAll = false
		if len(enabledLangs) > 0 {
			printSuccess("Enabled %d languages: %s", len(enabledLangs), strings.Join(enabledLangs, ", "))
		} else {
			fmt.Println("  No languages selected")
		}
	case "3":
		fmt.Println("  Skipping language configs")
		// Save empty list to disable all
		enabledLangs = []string{}
		useAll = false
	default:
		// Invalid choice, default to all
		enabledLangs = sourceLangs
		useAll = true
		printSuccess("Enabled all %d languages", len(enabledLangs))
	}

	// Save enabled languages to config
	if err := saveEnabledLanguages(enabledLangs, useAll); err != nil {
		printWarning("Failed to save language selection: %v", err)
	}

	// Offer to create personal language files for selected languages
	if len(enabledLangs) > 0 && promptYesNo("Create personal language configs to customize them?") {
		created := 0
		for _, lang := range enabledLangs {
			if err := createPersonalLanguageFile(paths.PersonalLanguages, lang); err == nil {
				created++
			}
		}
		if created > 0 {
			printSuccess("Created %d personal language configs in %s", created, paths.PersonalLanguages)
			fmt.Printf("  %s Run '%s' to edit them\n", dim("Tip:"), info("staghorn edit --language <lang>"))
		} else {
			fmt.Println("  Personal language configs already exist")
		}
	}
}

// selectLanguages prompts the user to select which languages to enable.
func selectLanguages(available []string) []string {
	fmt.Println()
	fmt.Println("Enter the numbers of languages to enable (comma-separated), or 'all':")
	for i, lang := range available {
		fmt.Printf("  %d. %s\n", i+1, lang)
	}
	fmt.Println()

	input := promptString("Selection:")
	input = strings.TrimSpace(input)

	if input == "" || strings.ToLower(input) == "all" {
		return available
	}

	// Parse comma-separated numbers
	var selected []string
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err == nil {
			if idx >= 1 && idx <= len(available) {
				selected = append(selected, available[idx-1])
			}
		}
	}

	return selected
}

// saveEnabledLanguages updates the config with the selected languages.
// Pass nil for "all" (auto-detect), empty slice for "none", or specific list.
func saveEnabledLanguages(langs []string, useAll bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if useAll {
		// Clear enabled list to use auto-detect (includes all available)
		cfg.Languages.Enabled = nil
		cfg.Languages.AutoDetect = true
	} else if len(langs) == 0 {
		// Explicitly disable all languages by setting empty enabled list
		cfg.Languages.Enabled = []string{}
		cfg.Languages.AutoDetect = false
	} else {
		// Specific selection
		cfg.Languages.Enabled = langs
		cfg.Languages.AutoDetect = false
	}

	return config.Save(cfg)
}

func printInitComplete() {
	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  %s    - add your personal preferences\n", info("staghorn edit"))
	fmt.Printf("  %s - list available commands\n", info("staghorn commands"))
	fmt.Println()
	fmt.Println("Periodic updates:")
	fmt.Printf("  %s              - fetch latest and apply\n", info("staghorn sync"))
	fmt.Printf("  %s --apply-only - re-apply without fetching\n", info("staghorn sync"))
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// promptString prompts for a string input.
func promptString(prompt string) string {
	fmt.Printf("%s ", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// promptYesNo prompts for a yes/no input.
func promptYesNo(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes"
}

// listLanguageFiles returns the language IDs from .md files in a directory.
func listLanguageFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var langs []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") {
			langs = append(langs, strings.TrimSuffix(name, ".md"))
		}
	}
	return langs, nil
}

// createPersonalLanguageFile creates a personal language config file if it doesn't exist.
// The file is created with a template heading. Files without user content beyond
// headings and comments are automatically skipped during merge.
func createPersonalLanguageFile(dir, langID string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(dir, langID+".md")

	// Skip if already exists
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("file already exists")
	}

	// Get display name
	displayName := language.GetDisplayName(langID)

	template := fmt.Sprintf(`## My %s Preferences

`, displayName)

	return os.WriteFile(filePath, []byte(template), 0644)
}

// ensurePersonalMD creates the personal.md file if it doesn't exist.
// The file is created with a minimal template that gets skipped during merge
// until the user adds actual content.
func ensurePersonalMD(paths *config.Paths) error {
	if _, err := os.Stat(paths.PersonalMD); err == nil {
		return nil // Already exists
	}

	if err := os.MkdirAll(paths.ConfigDir, 0755); err != nil {
		return err
	}

	template := `## My Preferences

`
	return os.WriteFile(paths.PersonalMD, []byte(template), 0644)
}
