package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/errors"
	"github.com/HartBrook/staghorn/internal/github"
	"github.com/spf13/cobra"
)

// NewInitCmd creates the init command.
func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize staghorn configuration",
		Long: `Interactive setup for staghorn.

This command will:
1. Prompt for your team's config repository
2. Verify authentication works
3. Create the configuration file
4. Perform an initial sync`,
		RunE: runInit,
	}
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

	// Verify auth is available, guide user through setup if needed
	authMethod := github.AuthMethod()
	if authMethod == "none" {
		authMethod = setupAuthentication()
		if authMethod == "" {
			return errors.New(errors.ErrGitHubAuthFailed, "no authentication configured", "")
		}
	}

	fmt.Printf("Using authentication: %s\n\n", info(authMethod))

	// Prompt for repo
	repoURL := promptString("Team config repository URL (e.g., https://github.com/acme/claude-standards):")
	if repoURL == "" {
		return fmt.Errorf("repository is required")
	}

	// Validate and parse repo
	cfg := &config.Config{
		Team: config.TeamConfig{
			Repo: repoURL,
		},
	}

	owner, repo, err := cfg.Team.ParseRepo()
	if err != nil {
		return err
	}

	fmt.Printf("\nVerifying access to %s/%s...\n", owner, repo)

	// Create client and verify access
	client, err := github.NewClient()
	if err != nil {
		token := github.GetTokenFromEnv()
		if token == "" {
			return errors.GitHubAuthFailed(err)
		}
		client, err = github.NewClientWithToken(token)
		if err != nil {
			return errors.GitHubAuthFailed(err)
		}
	}

	ctx := context.Background()

	// Check if repo exists
	exists, err := client.RepoExists(ctx, owner, repo)
	if err != nil {
		return errors.GitHubFetchFailed(owner+"/"+repo, err)
	}
	if !exists {
		return fmt.Errorf("repository %s/%s not found or not accessible", owner, repo)
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

	// Optional: custom branch
	fmt.Println()
	branch := promptString("Branch (leave empty for default):")
	if branch != "" {
		cfg.Team.Branch = branch
	}

	// Optional: custom path
	path := promptString(fmt.Sprintf("Config file path (default: %s):", config.DefaultPath))
	if path != "" {
		cfg.Team.Path = path
	}

	// Save config
	fmt.Println()
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	printSuccess("Config saved to %s", paths.ConfigFile)

	// Perform initial sync
	fmt.Println()
	fmt.Println("Performing initial sync...")
	syncOpts := &syncOptions{force: true}
	if err := runSync(ctx, syncOpts); err != nil {
		printWarning("Initial sync failed: %v", err)
		fmt.Println("You can run `staghorn sync` later to fetch the config.")
	}

	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  %s    - add your personal preferences\n", info("staghorn edit"))
	fmt.Println()
	fmt.Println("Periodic updates:")
	fmt.Printf("  %s              - fetch latest and apply\n", info("staghorn sync"))
	fmt.Printf("  %s --apply-only - re-apply without fetching\n", info("staghorn sync"))

	return nil
}

// setupAuthentication guides the user through setting up GitHub authentication.
// Returns the auth method string if successful, or empty string if user should re-run init.
func setupAuthentication() string {
	fmt.Println("GitHub authentication is required to fetch team configs.")
	fmt.Println()

	ghInstalled := github.IsGHCLIInstalled()

	if ghInstalled {
		// gh is installed but not authenticated
		fmt.Println("GitHub CLI is installed but not authenticated.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  1. Authenticate with GitHub CLI (recommended)")
		fmt.Println("  2. Enter a personal access token")
		fmt.Println()

		choice := promptString("Choose an option [1/2]:")

		switch choice {
		case "1", "":
			fmt.Println()
			fmt.Println("Run the following command, then re-run 'staghorn init':")
			fmt.Println()
			fmt.Printf("  %s\n", info("gh auth login"))
			fmt.Println()
			return ""

		case "2":
			return promptForToken()

		default:
			printError("Invalid option")
			return ""
		}
	}

	// gh is not installed
	fmt.Println("Options:")
	fmt.Println("  1. Install GitHub CLI (recommended)")
	fmt.Println("  2. Enter a personal access token")
	fmt.Println()

	choice := promptString("Choose an option [1/2]:")

	switch choice {
	case "1", "":
		fmt.Println()
		fmt.Println("Install GitHub CLI, then re-run 'staghorn init':")
		fmt.Println()
		fmt.Println("  macOS:               brew install gh && gh auth login")
		fmt.Println("  Windows:             winget install --id GitHub.cli && gh auth login")
		fmt.Println("  Linux (Debian):      sudo apt install gh && gh auth login")
		fmt.Println("  Linux (Fedora):      sudo dnf install gh && gh auth login")
		fmt.Println()
		fmt.Println("  More options:        https://cli.github.com/")
		fmt.Println()
		return ""

	case "2":
		return promptForToken()

	default:
		printError("Invalid option")
		return ""
	}
}

// promptForToken prompts the user to enter a GitHub personal access token.
func promptForToken() string {
	fmt.Println()
	fmt.Println("Create a personal access token at:")
	fmt.Println("  https://github.com/settings/tokens/new")
	fmt.Println()
	fmt.Println("Required scopes: repo (for private repos) or public_repo (for public repos)")
	fmt.Println()

	token := promptString("Paste your token:")
	if token == "" {
		printError("No token provided")
		return ""
	}

	// Validate token format
	if !strings.HasPrefix(token, "ghp_") && !strings.HasPrefix(token, "github_pat_") {
		printWarning("Token doesn't match expected format (ghp_* or github_pat_*)")
		if !promptYesNo("Continue anyway?") {
			return ""
		}
	}

	// Set for current process
	os.Setenv(github.EnvGitHubToken, token)

	fmt.Println()
	fmt.Println("To persist this token, add to your shell profile:")
	fmt.Printf("  export %s=%s\n", github.EnvGitHubToken, token[:10]+"...")
	fmt.Println()

	printSuccess("Token configured for this session")
	return "STAGHORN_GITHUB_TOKEN"
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
