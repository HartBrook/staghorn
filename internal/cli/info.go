package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HartBrook/staghorn/internal/cache"
	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/github"
	"github.com/HartBrook/staghorn/internal/language"
	"github.com/HartBrook/staghorn/internal/merge"
	"github.com/spf13/cobra"
)

type infoOptions struct {
	content   bool
	layer     string
	sources   bool
	languages string
	verbose   bool
}

// NewInfoCmd creates the info command.
func NewInfoCmd() *cobra.Command {
	opts := &infoOptions{}

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show current config state",
		Long: `Displays information about your staghorn configuration.

By default, shows a compact status overview. Use flags to customize output.`,
		Example: `  staghorn info              # Compact status
  staghorn info --content    # Show full merged config
  staghorn info --layer team # Show only team config
  staghorn info --verbose    # Detailed status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.content, "content", false, "Output the merged config content")
	cmd.Flags().StringVar(&opts.layer, "layer", "", "Show specific layer: team, personal, or project")
	cmd.Flags().BoolVar(&opts.sources, "sources", false, "Annotate content with source information (requires --content)")
	cmd.Flags().StringVar(&opts.languages, "languages", "auto", "Languages to include: auto, none, or comma-separated list")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Show detailed status information")

	return cmd
}

func runInfo(opts *infoOptions) error {
	// If --content flag or --layer is specified, show content
	if opts.content || opts.layer != "" {
		return showContent(opts)
	}

	// Otherwise show status
	return showStatus(opts.verbose)
}

// showContent outputs the merged configuration (replaces `show` command)
func showContent(opts *infoOptions) error {
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

	layer := opts.layer
	if layer == "" {
		layer = "all"
	}

	// Collect layers
	var layers []merge.Layer

	// Team layer (from cache)
	if layer == "all" || layer == "team" {
		c := cache.New(paths)
		teamContent, _, err := c.Read(owner, repo)
		if err != nil {
			if layer == "team" {
				return err
			}
			// For "all", continue without team layer
			printWarning("Team config not cached, run `staghorn sync` to fetch")
		} else {
			layers = append(layers, merge.Layer{
				Content: teamContent,
				Source:  "team",
			})
		}
	}

	// Personal layer
	if layer == "all" || layer == "personal" {
		if content, err := os.ReadFile(paths.PersonalMD); err == nil {
			layers = append(layers, merge.Layer{
				Content: string(content),
				Source:  "personal",
			})
		} else if layer == "personal" {
			return fmt.Errorf("personal config not found at %s", paths.PersonalMD)
		}
	}

	// Project layer
	if layer == "all" || layer == "project" {
		projectPath := findProjectConfig()
		if projectPath != "" {
			if content, err := os.ReadFile(projectPath); err == nil {
				layers = append(layers, merge.Layer{
					Content: string(content),
					Source:  "project",
				})
			}
		} else if layer == "project" {
			return fmt.Errorf("no project CLAUDE.md found")
		}
	}

	if len(layers) == 0 {
		return fmt.Errorf("no config layers found")
	}

	// Resolve languages
	var activeLanguages []string
	var languageFiles map[string][]*language.LanguageFile

	if opts.languages != "none" {
		projectRoot := findProjectRoot()

		// Build language config from options
		langCfg := language.LanguageConfig{
			AutoDetect: cfg.Languages.AutoDetect,
			Enabled:    cfg.Languages.Enabled,
			Disabled:   cfg.Languages.Disabled,
		}
		if opts.languages != "" && opts.languages != "auto" {
			// Override with explicit list
			langCfg.Enabled = strings.Split(opts.languages, ",")
			langCfg.AutoDetect = false
		}

		activeLanguages, _ = language.Resolve(&langCfg, projectRoot)

		if len(activeLanguages) > 0 {
			projectPaths := config.NewProjectPaths(projectRoot)
			languageFiles, _ = language.LoadLanguageFiles(
				activeLanguages,
				paths.TeamLanguagesDir(owner, repo),
				paths.PersonalLanguages,
				projectPaths.LanguagesDir,
			)
		}
	}

	// Merge and output
	mergeOpts := merge.MergeOptions{
		AnnotateSources: opts.sources,
		SourceRepo:      fmt.Sprintf("%s/%s", owner, repo),
		Languages:       activeLanguages,
		LanguageFiles:   languageFiles,
	}

	output := merge.MergeWithLanguages(layers, mergeOpts)
	fmt.Println(output)

	return nil
}

// showStatus displays the configuration state (replaces `status` command)
func showStatus(verbose bool) error {
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

	owner, repo, err := cfg.DefaultOwnerRepo()
	if err != nil {
		return err
	}

	// Compact format by default
	if !verbose {
		return showCompactStatus(cfg, paths, owner, repo)
	}

	// Verbose format
	return showVerboseStatus(cfg, paths, owner, repo)
}

func showCompactStatus(cfg *config.Config, paths *config.Paths, owner, repo string) error {
	c := cache.New(paths)

	// Team status
	sourceStatus := warning("not synced")
	if c.Exists(owner, repo) {
		meta, err := c.GetMetadata(owner, repo)
		if err == nil {
			if meta.IsStale(cfg.Cache.TTLDuration()) {
				sourceStatus = fmt.Sprintf("%s %s", meta.Age(), warning("(stale)"))
			} else {
				sourceStatus = success(meta.Age())
			}
		}
	}

	// Personal status
	personalStatus := dim("not created")
	if _, err := os.Stat(paths.PersonalMD); err == nil {
		content, _ := os.ReadFile(paths.PersonalMD)
		lines := bytes.Count(content, []byte("\n")) + 1
		personalStatus = fmt.Sprintf("%d lines", lines)
	}

	// Project status
	projectStatus := dim("not found")
	projectPath := findProjectConfig()
	if projectPath != "" {
		content, _ := os.ReadFile(projectPath)
		lines := bytes.Count(content, []byte("\n")) + 1
		projectStatus = fmt.Sprintf("%d lines", lines)
	}

	// Languages
	projectRoot := findProjectRoot()
	langCfg := language.LanguageConfig{
		AutoDetect: cfg.Languages.AutoDetect,
		Enabled:    cfg.Languages.Enabled,
		Disabled:   cfg.Languages.Disabled,
	}
	activeLanguages, _ := language.Resolve(&langCfg, projectRoot)
	langStatus := dim("none")
	if len(activeLanguages) > 0 {
		langStatus = strings.Join(activeLanguages, ", ")
	}

	// Output
	fmt.Printf("  %s: %s/%s (%s)\n", dim("Source"), owner, repo, sourceStatus)
	fmt.Printf("  %s: %s\n", dim("Personal"), personalStatus)
	fmt.Printf("  %s: %s\n", dim("Project"), projectStatus)
	fmt.Printf("  %s: %s\n", dim("Languages"), langStatus)

	return nil
}

func showVerboseStatus(cfg *config.Config, paths *config.Paths, owner, repo string) error {
	fmt.Println("Source config:")
	printInfo("Repository", fmt.Sprintf("%s/%s", owner, repo))
	printInfo("Path", config.DefaultPath)

	// Cache status
	c := cache.New(paths)
	fmt.Println()
	if c.Exists(owner, repo) {
		meta, err := c.GetMetadata(owner, repo)
		if err == nil {
			stale := meta.IsStale(cfg.Cache.TTLDuration())
			ageStr := meta.Age()
			if stale {
				fmt.Printf("  %s: %s %s\n", dim("Cache"), ageStr, warning("(stale)"))
				fmt.Printf("          Run %s to update.\n", info("staghorn sync"))
			} else {
				fmt.Printf("  %s: %s\n", dim("Cache"), ageStr)
			}
		}
	} else {
		fmt.Printf("  %s: %s\n", dim("Cache"), warning("not synced"))
		fmt.Printf("          Run %s to fetch team config.\n", info("staghorn sync"))
	}

	// Personal config
	fmt.Println()
	fmt.Println("Personal config:")
	if _, err := os.Stat(paths.PersonalMD); err == nil {
		content, _ := os.ReadFile(paths.PersonalMD)
		lines := bytes.Count(content, []byte("\n")) + 1
		printInfo("Location", paths.PersonalMD)
		printInfo("Size", fmt.Sprintf("%d lines", lines))
	} else {
		printInfo("Location", dim("not created"))
		fmt.Printf("          Create %s to add personal preferences.\n", info(paths.PersonalMD))
	}

	// Project config
	fmt.Println()
	fmt.Println("Project config:")
	projectPath := findProjectConfig()
	if projectPath != "" {
		content, _ := os.ReadFile(projectPath)
		lines := bytes.Count(content, []byte("\n")) + 1
		printInfo("Location", projectPath)
		printInfo("Size", fmt.Sprintf("%d lines", lines))
	} else {
		printInfo("Location", dim("not found"))
	}

	// Language detection
	fmt.Println()
	fmt.Println("Languages:")
	projectRoot := findProjectRoot()
	langCfg := language.LanguageConfig{
		AutoDetect: cfg.Languages.AutoDetect,
		Enabled:    cfg.Languages.Enabled,
		Disabled:   cfg.Languages.Disabled,
	}
	activeLanguages, _ := language.Resolve(&langCfg, projectRoot)

	if len(activeLanguages) == 0 {
		printInfo("Detected", dim("none"))
	} else {
		printInfo("Detected", strings.Join(activeLanguages, ", "))

		// Show which have team configs
		teamLangDir := paths.TeamLanguagesDir(owner, repo)
		var withTeamConfig []string
		for _, lang := range activeLanguages {
			if _, err := os.Stat(filepath.Join(teamLangDir, lang+".md")); err == nil {
				withTeamConfig = append(withTeamConfig, lang)
			}
		}

		if len(withTeamConfig) > 0 {
			printInfo("Team configs", strings.Join(withTeamConfig, ", "))
		}
	}

	// Auth status
	fmt.Println()
	fmt.Println("Authentication:")
	printInfo("Method", github.AuthMethod())

	return nil
}

// findProjectConfig walks up from CWD to find CLAUDE.md.
func findProjectConfig() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		claudePath := filepath.Join(dir, "CLAUDE.md")
		if _, err := os.Stat(claudePath); err == nil {
			return claudePath
		}

		// Stop at git root
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return ""
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
