package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/language"
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

	return cmd
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

	projectRoot := findProjectRoot()

	// Detect languages
	detected, _ := language.Detect(projectRoot)

	// Resolve active languages
	langCfg := language.LanguageConfig{
		AutoDetect: cfg.Languages.AutoDetect,
		Enabled:    cfg.Languages.Enabled,
		Disabled:   cfg.Languages.Disabled,
	}
	active, _ := language.Resolve(&langCfg, projectRoot)

	fmt.Println("Language Detection")
	fmt.Println()

	// Mode
	if len(cfg.Languages.Enabled) > 0 {
		printInfo("Mode", "explicit")
		printInfo("Configured", strings.Join(cfg.Languages.Enabled, ", "))
	} else if cfg.Languages.AutoDetect {
		printInfo("Mode", "auto-detect")
	} else {
		printInfo("Mode", "disabled")
	}

	// Detected
	if len(detected) > 0 {
		printInfo("Detected", strings.Join(detected, ", "))
	} else {
		printInfo("Detected", dim("none"))
	}

	// Disabled
	if len(cfg.Languages.Disabled) > 0 {
		printInfo("Disabled", strings.Join(cfg.Languages.Disabled, ", "))
	}

	fmt.Println()
	fmt.Println("Active Languages")
	fmt.Println()

	if len(active) == 0 {
		fmt.Println("  No languages active")
		return nil
	}

	teamLangDir := paths.TeamLanguagesDir(owner, repo)
	personalLangDir := paths.PersonalLanguages
	projectPaths := config.NewProjectPaths(projectRoot)

	for _, lang := range active {
		sources := []string{}

		if _, err := os.Stat(filepath.Join(teamLangDir, lang+".md")); err == nil {
			sources = append(sources, "team")
		}
		if _, err := os.Stat(filepath.Join(personalLangDir, lang+".md")); err == nil {
			sources = append(sources, "personal")
		}
		if _, err := os.Stat(filepath.Join(projectPaths.LanguagesDir, lang+".md")); err == nil {
			sources = append(sources, "project")
		}

		displayName := language.GetDisplayName(lang)
		sourceStr := dim("(no config files)")
		if len(sources) > 0 {
			sourceStr = strings.Join(sources, ", ")
		}

		fmt.Printf("  %-15s %s\n", info(displayName), sourceStr)
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
