package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/language"
	"github.com/HartBrook/staghorn/internal/starter"
	"github.com/spf13/cobra"
)

// NewLanguagesCmd creates the languages command.
func NewLanguagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "languages",
		Short: "Show detected and configured languages",
		Long: `Displays the languages detected in the current project and their configuration status.

Languages are auto-detected from marker files (e.g., go.mod, pyproject.toml, package.json)
and can be explicitly configured in your config file.`,
		Example: `  staghorn languages`,
		RunE:    runLanguages,
	}

	// Add subcommands
	cmd.AddCommand(NewLanguagesInitCmd())

	return cmd
}

// NewLanguagesInitCmd creates the 'languages init' command to bootstrap starter language configs.
func NewLanguagesInitCmd() *cobra.Command {
	var project bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Install starter language configs",
		Long: `Installs staghorn's built-in starter language configs to your personal or project config.

Starter configs include personal preferences templates for common languages like
Python, Go, TypeScript, Rust, Java, and Ruby. Files that already exist will be skipped.`,
		Example: `  staghorn languages init           # Install to ~/.config/staghorn/languages/
  staghorn languages init --project  # Install to .staghorn/languages/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLanguagesInit(project)
		},
	}

	cmd.Flags().BoolVar(&project, "project", false, "Install to project directory (.staghorn/languages/)")

	return cmd
}

func runLanguagesInit(project bool) error {
	paths := config.NewPaths()

	var targetDir string
	var targetLabel string

	if project {
		projectRoot := findProjectRoot()
		if projectRoot == "" {
			return fmt.Errorf("no project root found (looking for .git or .staghorn directory)")
		}
		projectPaths := config.NewProjectPaths(projectRoot)
		targetDir = projectPaths.LanguagesDir
		targetLabel = ".staghorn/languages/"
	} else {
		targetDir = paths.PersonalLanguages
		targetLabel = paths.PersonalLanguages
	}

	// Show available languages
	langNames := starter.LanguageNames()
	fmt.Printf("Installing %d starter language configs to %s\n", len(langNames), targetLabel)
	fmt.Println()

	count, err := starter.BootstrapLanguages(targetDir)
	if err != nil {
		return fmt.Errorf("failed to install language configs: %w", err)
	}

	if count > 0 {
		printSuccess("Installed %d language configs", count)
		fmt.Println()
		fmt.Println("Installed languages:")
		for _, name := range langNames {
			fmt.Printf("  %s\n", info(name))
		}
	} else {
		fmt.Println("All starter language configs already installed.")
	}

	fmt.Println()
	fmt.Printf("Run %s to see language status.\n", info("staghorn languages"))
	fmt.Printf("Edit with %s\n", info("staghorn edit --language <lang>"))

	return nil
}

func runLanguages(cmd *cobra.Command, args []string) error {
	paths := config.NewPaths()

	// Check if configured
	if !config.Exists() {
		fmt.Println("Staghorn is not configured.")
		fmt.Println()
		fmt.Printf("  Run %s to get started.\n", info("staghorn init"))
		return nil
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	owner, repo, err := cfg.Team.ParseRepo()
	if err != nil {
		return err
	}

	teamLangDir := paths.TeamLanguagesDir(owner, repo)
	personalLangDir := paths.PersonalLanguages

	// For global config: get all available languages from team + personal
	var globalActive []string
	if len(cfg.Languages.Enabled) > 0 {
		globalActive = language.FilterDisabled(cfg.Languages.Enabled, cfg.Languages.Disabled)
	} else {
		globalActive, _ = language.ListAvailableLanguages(teamLangDir, personalLangDir, "")
		globalActive = language.FilterDisabled(globalActive, cfg.Languages.Disabled)
	}

	fmt.Println("Global Config (~/.claude/CLAUDE.md)")
	fmt.Println()

	// Mode
	if len(cfg.Languages.Enabled) > 0 {
		printInfo("Mode", "explicit")
		printInfo("Configured", strings.Join(cfg.Languages.Enabled, ", "))
	} else {
		printInfo("Mode", "all available")
	}

	// Disabled
	if len(cfg.Languages.Disabled) > 0 {
		printInfo("Disabled", strings.Join(cfg.Languages.Disabled, ", "))
	}

	fmt.Println()
	fmt.Println("Active Languages (Global)")
	fmt.Println()

	if len(globalActive) == 0 {
		fmt.Println("  No languages active")
	} else {
		for _, lang := range globalActive {
			sources := []string{}

			if _, err := os.Stat(filepath.Join(teamLangDir, lang+".md")); err == nil {
				sources = append(sources, "team")
			}
			if _, err := os.Stat(filepath.Join(personalLangDir, lang+".md")); err == nil {
				sources = append(sources, "personal")
			}

			displayName := language.GetDisplayName(lang)
			sourceStr := dim("(no config files)")
			if len(sources) > 0 {
				sourceStr = strings.Join(sources, ", ")
			}

			fmt.Printf("  %-15s %s\n", info(displayName), sourceStr)
		}
	}

	// Project detection (if in a project)
	projectRoot := findProjectRoot()
	if projectRoot != "" {
		detected, _ := language.Detect(projectRoot)

		fmt.Println()
		fmt.Println("Project Detection (./CLAUDE.md)")
		fmt.Println()

		if len(detected) > 0 {
			printInfo("Detected", strings.Join(detected, ", "))
		} else {
			printInfo("Detected", dim("none"))
		}
	}

	// Show supported languages
	fmt.Println()
	fmt.Println("Supported Languages")
	fmt.Println()

	var supported []string
	for _, lang := range language.SupportedLanguages {
		supported = append(supported, lang.ID)
	}
	fmt.Printf("  %s\n", dim(strings.Join(supported, ", ")))

	return nil
}
