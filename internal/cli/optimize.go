package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/HartBrook/staghorn/internal/cache"
	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/language"
	"github.com/HartBrook/staghorn/internal/merge"
	"github.com/HartBrook/staghorn/internal/optimize"
	"github.com/spf13/cobra"
)

type optimizeOptions struct {
	layer         string
	target        int
	dryRun        bool
	showDiff      bool
	output        string
	apply         bool
	force         bool
	deterministic bool
	verbose       bool
	noCache       bool
}

// NewOptimizeCmd creates the optimize command.
func NewOptimizeCmd() *cobra.Command {
	opts := &optimizeOptions{}

	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Analyze and compress config to reduce token usage",
		Long: `Analyzes your config and shows potential token savings from optimization.

By default, this command is informational only - it shows before/after token
counts without modifying any files. Use --apply to save the optimized content
back to the source layer, or -o to write to a custom file.

The optimization process:
1. Pre-processes content (normalizes whitespace, removes duplicates)
2. Extracts critical anchors (tool names, file paths, commands)
3. Uses Claude to intelligently compress content (unless --deterministic)
4. Validates that critical content is preserved

Note: CLAUDE.md files are managed by staghorn and regenerated on sync.
Use --apply with --layer team or --layer personal to optimize source files.
The --layer merged option is for analysis only (no source file to update).

Use --deterministic for fast, repeatable optimization without LLM calls.`,
		Example: `  staghorn optimize                          # Analyze merged config (informational)
  staghorn optimize --diff                   # Show before/after diff
  staghorn optimize --layer personal --apply # Optimize and save personal config
  staghorn optimize --layer team --apply     # Optimize and save team config
  staghorn optimize --deterministic          # No LLM, just cleanup
  staghorn optimize --target 2000            # Target ~2000 tokens
  staghorn optimize -o optimized.md          # Write to custom file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOptimize(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.layer, "layer", "merged", "Layer to optimize: team, personal, or merged")
	cmd.Flags().IntVar(&opts.target, "target", 0, "Target token count (0 = auto ~50% reduction)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Alias for default behavior (informational only)")
	cmd.Flags().BoolVar(&opts.showDiff, "diff", false, "Show before/after diff")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Write optimized content to file")
	cmd.Flags().BoolVar(&opts.apply, "apply", false, "Save optimized content back to source layer")
	cmd.Flags().BoolVar(&opts.force, "force", false, "Re-optimize even if cache is valid")
	cmd.Flags().BoolVar(&opts.deterministic, "deterministic", false, "Only apply deterministic transforms (no LLM)")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Show detailed progress")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "Skip cache read/write")

	return cmd
}

func runOptimize(ctx context.Context, opts *optimizeOptions) error {
	// Validate --apply with --layer merged
	if opts.apply && opts.layer == "merged" {
		fmt.Println(dim("Cannot apply optimization to merged layer."))
		fmt.Println()
		fmt.Println("The merged layer is derived from team and personal configs and has no")
		fmt.Println("source file to update. Use one of these options instead:")
		fmt.Println()
		fmt.Println("  " + info("stag optimize --layer personal --apply") + "  # Optimize personal config")
		fmt.Println("  " + info("stag optimize --layer team --apply") + "      # Optimize team config")
		fmt.Println("  " + info("stag optimize -o output.md") + "              # Write to custom file")
		return fmt.Errorf("cannot apply to merged layer")
	}

	// Check for API key (skip for deterministic mode or informational runs)
	needsAPI := !opts.deterministic && (opts.apply || opts.output != "")
	if needsAPI && os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Println(dim("ANTHROPIC_API_KEY not set."))
		fmt.Println()
		fmt.Println("LLM optimization requires an Anthropic API key to call Claude.")
		fmt.Println("Set it in your environment:")
		fmt.Println()
		fmt.Println("  " + info("export ANTHROPIC_API_KEY=<your-api-key>"))
		fmt.Println()
		fmt.Println("Or use " + info("--deterministic") + " for fast cleanup without API calls.")
		return fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

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

	// Get content to optimize
	content, err := getContentToOptimize(cfg, paths, owner, repo, opts.layer)
	if err != nil {
		return err
	}

	if opts.verbose {
		fmt.Println("Analyzing config...")
		fmt.Printf("  Original: %d tokens\n", optimize.CountTokens(content))
	}

	// Create optimizer
	optimizer := optimize.NewOptimizer(paths)

	// Run optimization
	optimizerOpts := optimize.Options{
		Target:        opts.target,
		Deterministic: opts.deterministic,
		Force:         opts.force,
		NoCache:       opts.noCache,
	}

	result, err := optimizer.Optimize(ctx, content, owner, repo, optimizerOpts)
	if err != nil {
		return err
	}

	// Display results
	displayOptimizationResult(result, opts)

	// Show diff if requested
	if opts.showDiff {
		displayDiff(result.OriginalContent, result.OptimizedContent)
	}

	// Handle output
	if opts.output != "" {
		// Write to custom output file
		if err := os.WriteFile(opts.output, []byte(result.OptimizedContent), 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		printSuccess("Wrote optimized config to %s", opts.output)
		return nil
	}

	if opts.apply {
		// Apply optimization to source layer
		if err := applyOptimization(paths, owner, repo, opts.layer, result.OptimizedContent); err != nil {
			return err
		}
		printSuccess("Applied optimization to %s layer", opts.layer)
		fmt.Println()
		fmt.Println(dim("Run 'stag sync' to regenerate CLAUDE.md with optimized content."))
		return nil
	}

	// Default: informational only
	fmt.Println()
	if result.FromCache {
		fmt.Println(dim("(from cache - use --force to re-optimize)"))
	}
	fmt.Println(dim("No changes applied. Use --apply to save to source, or -o to write to file."))

	return nil
}

// getContentToOptimize retrieves the content to optimize based on layer.
func getContentToOptimize(cfg *config.Config, paths *config.Paths, owner, repo, layer string) (string, error) {
	c := cache.New(paths)

	switch layer {
	case "team":
		content, _, err := c.Read(owner, repo)
		if err != nil {
			return "", fmt.Errorf("team config not cached: %w", err)
		}
		return content, nil

	case "personal":
		content, err := os.ReadFile(paths.PersonalMD)
		if err != nil {
			return "", fmt.Errorf("personal config not found: %w", err)
		}
		return string(content), nil

	case "merged":
		// Build merged content
		var layers []merge.Layer

		// Team layer
		teamContent, _, err := c.Read(owner, repo)
		if err == nil && teamContent != "" {
			layers = append(layers, merge.Layer{
				Content: teamContent,
				Source:  "team",
			})
		}

		// Personal layer
		if personalContent, err := os.ReadFile(paths.PersonalMD); err == nil {
			layers = append(layers, merge.Layer{
				Content: string(personalContent),
				Source:  "personal",
			})
		}

		if len(layers) == 0 {
			return "", fmt.Errorf("no config layers found to optimize")
		}

		// Get active languages
		projectRoot := findProjectRoot()
		langCfg := language.LanguageConfig{
			AutoDetect: cfg.Languages.AutoDetect,
			Enabled:    cfg.Languages.Enabled,
			Disabled:   cfg.Languages.Disabled,
		}
		activeLanguages, _ := language.Resolve(&langCfg, projectRoot)

		var languageFiles map[string][]*language.LanguageFile
		if len(activeLanguages) > 0 {
			projectPaths := config.NewProjectPaths(projectRoot)
			languageFiles, _ = language.LoadLanguageFiles(
				activeLanguages,
				paths.TeamLanguagesDir(owner, repo),
				paths.PersonalLanguages,
				projectPaths.LanguagesDir,
			)
		}

		mergeOpts := merge.MergeOptions{
			SourceRepo:    fmt.Sprintf("%s/%s", owner, repo),
			Languages:     activeLanguages,
			LanguageFiles: languageFiles,
		}

		return merge.MergeWithLanguages(layers, mergeOpts), nil

	default:
		return "", fmt.Errorf("unknown layer: %s (use team, personal, or merged)", layer)
	}
}

// applyOptimization saves optimized content back to the source layer.
func applyOptimization(paths *config.Paths, owner, repo, layer, content string) error {
	switch layer {
	case "team":
		// Update the team cache file
		c := cache.New(paths)
		// Read existing metadata to preserve it
		_, existingMeta, err := c.Read(owner, repo)
		if err != nil {
			return fmt.Errorf("failed to read team cache: %w", err)
		}
		// Preserve existing metadata or create new
		meta := existingMeta
		if meta == nil {
			meta = &cache.Metadata{
				Owner: owner,
				Repo:  repo,
			}
		}
		if err := c.Write(owner, repo, content, meta); err != nil {
			return fmt.Errorf("failed to write team cache: %w", err)
		}
		return nil

	case "personal":
		// Update the personal.md file
		if err := os.WriteFile(paths.PersonalMD, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write personal config: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("cannot apply to layer: %s", layer)
	}
}

// displayOptimizationResult shows the optimization results.
func displayOptimizationResult(result *optimize.Result, opts *optimizeOptions) {
	fmt.Println()

	// Token stats
	fmt.Printf("  %s: %d tokens\n", dim("Before"), result.Stats.Before)
	fmt.Printf("  %s: %d tokens\n", dim("After"), result.Stats.After)
	fmt.Printf("  %s: %d tokens (%.0f%%)\n", dim("Saved"), result.Stats.Saved(), result.Stats.PercentReduction())

	// Preprocessing stats
	if opts.verbose && !result.FromCache {
		fmt.Println()
		fmt.Printf("  %s:\n", dim("Pre-processing"))
		if result.PreprocessStats.BlankLinesRemoved > 0 {
			fmt.Printf("    Blank lines removed: %d\n", result.PreprocessStats.BlankLinesRemoved)
		}
		if result.PreprocessStats.DuplicatesRemoved > 0 {
			fmt.Printf("    Duplicates removed: %d\n", result.PreprocessStats.DuplicatesRemoved)
		}
		if result.PreprocessStats.PhrasesStripped > 0 {
			fmt.Printf("    Verbose phrases stripped: %d\n", result.PreprocessStats.PhrasesStripped)
		}
	}

	// Validation warnings
	if len(result.MissingSoft) > 0 {
		fmt.Println()
		fmt.Printf("  %s: %s\n", dim("Tool names consolidated"), strings.Join(result.MissingSoft, ", "))
	}
	if len(result.MissingStrict) > 0 {
		fmt.Println()
		printWarning("Critical anchors missing: %s", strings.Join(result.MissingStrict, ", "))
	}

	// Source info
	if result.FromCache {
		fmt.Printf("\n  %s\n", dim("(from cache)"))
	} else if result.Deterministic {
		fmt.Printf("\n  %s\n", dim("(deterministic mode)"))
	}
}

// displayDiff shows a simple diff between original and optimized content.
func displayDiff(original, optimized string) {
	fmt.Println()
	fmt.Println(dim("--- original"))
	fmt.Println(dim("+++ optimized"))
	fmt.Println()

	// Simple line-by-line comparison
	origLines := strings.Split(original, "\n")
	optLines := strings.Split(optimized, "\n")

	// Show first differences (limited output for readability)
	shown := 0
	maxDiff := 20

	for i := 0; i < len(origLines) && shown < maxDiff; i++ {
		if i >= len(optLines) || origLines[i] != optLines[i] {
			if i < len(origLines) {
				fmt.Printf("%s %s\n", danger("-"), origLines[i])
			}
			if i < len(optLines) {
				fmt.Printf("%s %s\n", success("+"), optLines[i])
			}
			shown++
		}
	}

	if shown >= maxDiff {
		fmt.Printf("\n%s\n", dim("(diff truncated, showing first 20 changes)"))
	}
}
