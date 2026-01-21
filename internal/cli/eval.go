package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/eval"
	"github.com/HartBrook/staghorn/internal/merge"
	"github.com/HartBrook/staghorn/internal/starter"
	"github.com/spf13/cobra"
)

// NewEvalCmd creates the eval command.
func NewEvalCmd() *cobra.Command {
	var tag string
	var layer string
	var output string
	var verbose bool
	var debug bool
	var dryRun bool
	var testFilter string

	cmd := &cobra.Command{
		Use:   "eval [name]",
		Short: "Run behavioral evals against your CLAUDE.md config",
		Long: `Runs behavioral tests (evals) to validate that your CLAUDE.md configuration
produces the expected Claude behavior.

Evals use Promptfoo under the hood to test Claude's responses against your
merged config. This helps ensure your team's guidelines are actually followed.`,
		Example: `  staghorn eval                         # Run all evals
  staghorn eval security-secrets        # Run specific eval
  staghorn eval --tag security          # Filter by tag
  staghorn eval --layer team            # Test team config only
  staghorn eval --output json           # CI/CD output format
  staghorn eval lang-python --test uses-type-hints  # Run specific test
  staghorn eval --test "uses-*"         # Run tests matching prefix
  staghorn eval --debug                 # Show full responses and keep temp files`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return runEval(name, tag, layer, output, verbose, debug, dryRun, testFilter)
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "", "Filter evals by tag (e.g., security, quality)")
	cmd.Flags().StringVar(&layer, "layer", "merged", "Config layer to test: team, personal, project, or merged")
	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output format: table, json, or github")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output")
	cmd.Flags().BoolVar(&debug, "debug", false, "Show full Claude responses for failures and preserve temp files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be tested without running")
	cmd.Flags().StringVarP(&testFilter, "test", "t", "", "Filter by test name (supports prefix patterns like 'uses-*')")

	// Add subcommands
	cmd.AddCommand(NewEvalListCmd())
	cmd.AddCommand(NewEvalInitCmd())
	cmd.AddCommand(NewEvalInfoCmd())
	cmd.AddCommand(NewEvalValidateCmd())
	cmd.AddCommand(NewEvalCreateCmd())

	return cmd
}

// NewEvalListCmd creates the 'eval list' command.
func NewEvalListCmd() *cobra.Command {
	var tag string
	var source string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available evals",
		Long:  `Lists all available evals from team, personal, project, and starter sources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEvalList(tag, source)
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&source, "source", "", "Filter by source (team, personal, project, starter)")

	return cmd
}

// NewEvalInitCmd creates the 'eval init' command.
func NewEvalInitCmd() *cobra.Command {
	var project bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Install starter evals",
		Long: `Installs staghorn's built-in starter evals to your personal or project config.

Starter evals cover common validation scenarios like security guidelines,
code quality, and behavioral baselines.`,
		Example: `  staghorn eval init            # Install to ~/.config/staghorn/evals/
  staghorn eval init --project  # Install to .staghorn/evals/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEvalInit(project)
		},
	}

	cmd.Flags().BoolVar(&project, "project", false, "Install to project directory (.staghorn/evals/)")

	return cmd
}

// NewEvalInfoCmd creates the 'eval info' command.
func NewEvalInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <eval>",
		Short: "Show detailed information about an eval",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEvalInfo(args[0])
		},
	}
}

func runEval(name, tagFilter, layer, outputFormat string, verbose, debug, dryRun bool, testFilter string) error {
	// Check for Promptfoo
	if err := eval.CheckPromptfoo(); err != nil {
		printWarning("Promptfoo not found. Install with: npm install -g promptfoo")
		return err
	}

	// Check for API key (skip for dry-run)
	if !dryRun && os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Println(dim("ANTHROPIC_API_KEY not set."))
		fmt.Println()
		fmt.Println("Evals require an Anthropic API key to call Claude.")
		fmt.Println("Set it in your environment:")
		fmt.Println()
		fmt.Println("  " + info("export ANTHROPIC_API_KEY=sk-ant-..."))
		fmt.Println()
		fmt.Println(dim("Note: Running evals will consume API credits."))
		return fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	// Load evals
	evals, err := loadEvals()
	if err != nil {
		return err
	}

	if len(evals) == 0 {
		fmt.Println("No evals found.")
		fmt.Println()
		fmt.Println("Run " + info("staghorn eval init") + " to install starter evals.")
		return nil
	}

	// Filter evals
	var filtered []*eval.Eval
	for _, e := range evals {
		// Filter by name
		if name != "" && e.Name != name {
			continue
		}

		// Filter by tag
		if tagFilter != "" && !e.HasTag(tagFilter) {
			continue
		}

		// Filter by test name
		if testFilter != "" {
			e = e.FilterTests(testFilter)
			if e == nil {
				continue
			}
		}

		filtered = append(filtered, e)
	}

	if len(filtered) == 0 {
		if name != "" {
			return fmt.Errorf("eval '%s' not found", name)
		}
		if testFilter != "" {
			return fmt.Errorf("no tests match filter '%s'", testFilter)
		}
		fmt.Println("No evals match the filter.")
		return nil
	}

	if dryRun {
		fmt.Println(dim("Dry run - would test the following evals:"))
		fmt.Println()
		for _, e := range filtered {
			fmt.Printf("  %s (%d tests)\n", info(e.Name), e.TestCount())
		}
		fmt.Println()
		fmt.Printf("Config layer: %s\n", layer)
		fmt.Printf("Total tests: %d\n", countTests(filtered))
		return nil
	}

	// Count total tests
	totalTests := countTests(filtered)
	fmt.Printf("Running %d evals (%d tests) against %s config...\n", len(filtered), totalTests, layer)
	fmt.Println()

	// Generate merged CLAUDE.md for testing
	claudeConfig, err := generateClaudeConfig(layer)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Create runner
	tempDir, err := os.MkdirTemp("", "staghorn-eval-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	if !debug {
		defer os.RemoveAll(tempDir)
	}

	runner := eval.NewRunner(tempDir)
	runner.Verbose = verbose
	runner.Debug = debug

	// Run evals
	ctx := context.Background()
	results, err := runner.RunAll(ctx, filtered, claudeConfig)
	if err != nil {
		return err
	}

	// Format output
	var format eval.OutputFormat
	switch outputFormat {
	case "json":
		format = eval.OutputFormatJSON
	case "github":
		format = eval.OutputFormatGitHub
	default:
		format = eval.OutputFormatTable
	}

	formatter := eval.NewFormatter(os.Stdout, format)
	formatter.Debug = debug
	if err := formatter.FormatResults(results); err != nil {
		return err
	}

	// Return error if there were failures (cobra will set exit code)
	summary := eval.Summarize(results)
	if summary.Failed > 0 {
		return fmt.Errorf("%d of %d tests failed", summary.Failed, summary.TotalTests)
	}

	return nil
}

func runEvalList(tagFilter, sourceFilter string) error {
	evals, err := loadEvals()
	if err != nil {
		return err
	}

	if len(evals) == 0 {
		fmt.Println("No evals found.")
		fmt.Println()
		fmt.Println("Run " + info("staghorn eval init") + " to install starter evals.")
		return nil
	}

	// Apply filters
	var filtered []*eval.Eval
	for _, e := range evals {
		if tagFilter != "" && !e.HasTag(tagFilter) {
			continue
		}

		if sourceFilter != "" {
			var src eval.Source
			switch sourceFilter {
			case "team":
				src = eval.SourceTeam
			case "personal":
				src = eval.SourcePersonal
			case "project":
				src = eval.SourceProject
			case "starter":
				src = eval.SourceStarter
			default:
				return fmt.Errorf("invalid source: %s", sourceFilter)
			}
			if e.Source != src {
				continue
			}
		}

		filtered = append(filtered, e)
	}

	if len(filtered) == 0 {
		fmt.Println("No evals match the filter.")
		return nil
	}

	// Group by source
	teamEvals := filterEvalsBySource(filtered, eval.SourceTeam)
	personalEvals := filterEvalsBySource(filtered, eval.SourcePersonal)
	projectEvals := filterEvalsBySource(filtered, eval.SourceProject)
	starterEvals := filterEvalsBySource(filtered, eval.SourceStarter)

	if len(teamEvals) > 0 {
		printEvalGroup("TEAM EVALS", teamEvals)
	}

	if len(personalEvals) > 0 {
		if len(teamEvals) > 0 {
			fmt.Println()
		}
		printEvalGroup("PERSONAL EVALS", personalEvals)
	}

	if len(projectEvals) > 0 {
		if len(teamEvals) > 0 || len(personalEvals) > 0 {
			fmt.Println()
		}
		printEvalGroup("PROJECT EVALS", projectEvals)
	}

	if len(starterEvals) > 0 {
		if len(teamEvals) > 0 || len(personalEvals) > 0 || len(projectEvals) > 0 {
			fmt.Println()
		}
		printEvalGroup("STARTER EVALS", starterEvals)
	}

	fmt.Println()
	fmt.Printf("Use: %s\n", info("staghorn eval <name>"))

	return nil
}

func filterEvalsBySource(evals []*eval.Eval, source eval.Source) []*eval.Eval {
	var result []*eval.Eval
	for _, e := range evals {
		if e.Source == source {
			result = append(result, e)
		}
	}
	return result
}

func printEvalGroup(title string, evals []*eval.Eval) {
	fmt.Println(dim(title))
	for _, e := range evals {
		desc := e.Description
		if desc == "" {
			desc = "(no description)"
		}
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		tags := ""
		if len(e.Tags) > 0 {
			tags = dim(" [" + strings.Join(e.Tags, ", ") + "]")
		}

		fmt.Printf("  %-25s %s%s\n", info(e.Name), desc, tags)
		fmt.Printf("  %s %d tests\n", dim("                        "), e.TestCount())
	}
}

func runEvalInit(project bool) error {
	paths := config.NewPaths()

	var targetDir string
	var targetLabel string

	if project {
		projectRoot := findProjectRoot()
		if projectRoot == "" {
			return fmt.Errorf("no project root found (looking for .git or .staghorn directory)")
		}
		targetDir = config.ProjectEvalsDir(projectRoot)
		targetLabel = ".staghorn/evals/"
	} else {
		targetDir = paths.PersonalEvals
		targetLabel = "~/.config/staghorn/evals/"
	}

	fmt.Printf("Installing starter evals to %s\n", targetLabel)
	fmt.Println()

	count, installed, err := starter.BootstrapEvals(targetDir)
	if err != nil {
		return fmt.Errorf("failed to install evals: %w", err)
	}

	if count > 0 {
		printSuccess("Installed %d evals", count)
		fmt.Println()
		fmt.Println("Installed evals:")
		for _, name := range installed {
			fmt.Printf("  %s\n", info(name))
		}
	} else {
		fmt.Println("All starter evals already installed.")
	}

	fmt.Println()
	fmt.Printf("Run %s to see all available evals.\n", info("staghorn eval list"))
	fmt.Printf("Run %s to execute all evals.\n", info("staghorn eval"))

	return nil
}

func runEvalInfo(name string) error {
	evals, err := loadEvals()
	if err != nil {
		return err
	}

	var found *eval.Eval
	for _, e := range evals {
		if e.Name == name {
			found = e
			break
		}
	}

	if found == nil {
		return fmt.Errorf("eval '%s' not found", name)
	}

	fmt.Println(dim("Name:"), info(found.Name))
	fmt.Println(dim("Source:"), found.Source.Label())

	if found.Description != "" {
		fmt.Println(dim("Description:"), found.Description)
	}

	if len(found.Tags) > 0 {
		fmt.Println(dim("Tags:"), strings.Join(found.Tags, ", "))
	}

	fmt.Println()
	fmt.Println(dim("Tests:"))
	for _, t := range found.Tests {
		fmt.Printf("  %s\n", t.Name)
		if t.Description != "" && t.Description != t.Name {
			fmt.Printf("    %s\n", dim(t.Description))
		}
		fmt.Printf("    Assertions: %d\n", len(t.Assert))
	}

	return nil
}

// loadEvals loads evals from all sources.
func loadEvals() ([]*eval.Eval, error) {
	paths := config.NewPaths()
	var allEvals []*eval.Eval
	var warnings []string

	// Load team evals
	if config.Exists() {
		cfg, err := config.Load()
		if err == nil {
			owner, repo, err := cfg.DefaultOwnerRepo()
			if err == nil {
				teamEvalsDir := paths.TeamEvalsDir(owner, repo)
				teamEvals, err := eval.LoadFromDirectory(teamEvalsDir, eval.SourceTeam)
				if err != nil {
					warnings = append(warnings, fmt.Sprintf("team evals: %v", err))
				}
				allEvals = append(allEvals, teamEvals...)
			}
		}
	}

	// Load personal evals
	personalEvals, err := eval.LoadFromDirectory(paths.PersonalEvals, eval.SourcePersonal)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("personal evals: %v", err))
	}
	allEvals = append(allEvals, personalEvals...)

	// Load project evals
	if projectRoot := findProjectRoot(); projectRoot != "" {
		projectEvalsDir := config.ProjectEvalsDir(projectRoot)
		projectEvals, err := eval.LoadFromDirectory(projectEvalsDir, eval.SourceProject)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("project evals: %v", err))
		}
		allEvals = append(allEvals, projectEvals...)
	}

	// Print warnings for non-critical errors
	for _, w := range warnings {
		printWarning("Failed to load %s", w)
	}

	return allEvals, nil
}

// generateClaudeConfig generates the merged CLAUDE.md content for a given layer.
func generateClaudeConfig(layer string) (string, error) {
	paths := config.NewPaths()
	projectRoot := findProjectRoot()

	// Load team config (from source repo or cache)
	var teamContent string
	if config.IsSourceRepo(projectRoot) {
		// Read from local source repo
		sourcePaths := config.NewSourceRepoPaths(projectRoot)
		if data, err := os.ReadFile(sourcePaths.ClaudeMD); err == nil {
			teamContent = string(data)
		}
	} else if config.Exists() {
		cfg, err := config.Load()
		if err == nil {
			owner, repo, err := cfg.DefaultOwnerRepo()
			if err == nil {
				cacheFile := paths.CacheFile(owner, repo)
				if data, err := os.ReadFile(cacheFile); err == nil {
					teamContent = string(data)
				}
			}
		}
	}

	// Load personal config
	var personalContent string
	if data, err := os.ReadFile(paths.PersonalMD); err == nil {
		personalContent = string(data)
	}

	// Load project config
	var projectContent string
	if projectRoot != "" {
		projectPaths := config.NewProjectPaths(projectRoot)
		if data, err := os.ReadFile(projectPaths.SourceMD); err == nil {
			projectContent = string(data)
		}
	}

	// Select content based on layer
	switch layer {
	case "team":
		if teamContent == "" {
			return "", fmt.Errorf("no team config found")
		}
		return teamContent, nil
	case "personal":
		if personalContent == "" {
			return "", fmt.Errorf("no personal config found")
		}
		return personalContent, nil
	case "project":
		if projectContent == "" {
			return "", fmt.Errorf("no project config found")
		}
		return projectContent, nil
	case "merged":
		// Merge all layers
		result := merge.MergeSimple(teamContent, personalContent, projectContent)
		return result, nil
	default:
		return "", fmt.Errorf("invalid layer: %s (use team, personal, project, or merged)", layer)
	}
}

func countTests(evals []*eval.Eval) int {
	total := 0
	for _, e := range evals {
		total += e.TestCount()
	}
	return total
}

// NewEvalValidateCmd creates the 'eval validate' command.
func NewEvalValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [name]",
		Short: "Validate eval definitions without running them",
		Long: `Validates eval YAML files for correctness before running.

Checks for:
- Valid assertion types (llm-rubric, contains, regex, etc.)
- Required fields (name, prompt, assert)
- Proper YAML structure
- Naming conventions

Returns exit code 1 if any errors are found.`,
		Example: `  staghorn eval validate                    # Validate all evals
  staghorn eval validate security-secrets   # Validate specific eval`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return runEvalValidate(name)
		},
	}
}

func runEvalValidate(name string) error {
	evals, err := loadEvals()
	if err != nil {
		return err
	}

	if len(evals) == 0 {
		fmt.Println("No evals found.")
		fmt.Println()
		fmt.Println("Run " + info("staghorn eval init") + " to install starter evals.")
		return nil
	}

	// Filter by name if provided
	if name != "" {
		var filtered []*eval.Eval
		for _, e := range evals {
			if e.Name == name {
				filtered = append(filtered, e)
				break
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("eval '%s' not found", name)
		}
		evals = filtered
	}

	fmt.Printf("Validating %d eval(s)...\n", len(evals))
	fmt.Println()

	totalErrors := 0
	totalWarnings := 0
	validCount := 0
	invalidCount := 0

	for _, e := range evals {
		errors := e.Validate()
		errorCount, warningCount := eval.CountByLevel(errors)
		totalErrors += errorCount
		totalWarnings += warningCount

		if errorCount > 0 {
			invalidCount++
			fmt.Printf("%s %s (%d tests)\n", danger("✗"), e.Name, e.TestCount())
			for _, err := range errors {
				prefix := "  "
				if err.Level == eval.ValidationLevelError {
					prefix += danger("error: ")
				} else {
					prefix += warning("warning: ")
				}
				fmt.Printf("%s%s: %s\n", prefix, err.Field, err.Message)
			}
		} else if warningCount > 0 {
			validCount++
			fmt.Printf("%s %s (%d tests)\n", success("✓"), e.Name, e.TestCount())
			for _, err := range errors {
				fmt.Printf("  %s%s: %s\n", warning("warning: "), err.Field, err.Message)
			}
		} else {
			validCount++
			fmt.Printf("%s %s (%d tests)\n", success("✓"), e.Name, e.TestCount())
		}
	}

	fmt.Println()
	summary := fmt.Sprintf("%d valid", validCount)
	if invalidCount > 0 {
		summary += fmt.Sprintf(", %s", danger(fmt.Sprintf("%d invalid", invalidCount)))
	}
	if totalWarnings > 0 {
		summary += fmt.Sprintf(", %s", warning(fmt.Sprintf("%d warning(s)", totalWarnings)))
	}
	fmt.Println(summary)

	if totalErrors > 0 {
		return fmt.Errorf("%d eval(s) have validation errors", invalidCount)
	}

	return nil
}

// NewEvalCreateCmd creates the 'eval create' command.
func NewEvalCreateCmd() *cobra.Command {
	var project bool
	var team bool
	var templateName string
	var fromEval string
	var evalName string
	var description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new eval from a template",
		Long: `Creates a new eval file from a template or existing eval.

Available templates:
- security: Security-focused eval for testing security guidelines
- quality: Code quality eval for testing style and best practices
- language: Language-specific eval template
- blank: Minimal blank template to start from scratch

Destination options:
- Default: ~/.config/staghorn/evals/ (personal evals)
- --project: .staghorn/evals/ (project-specific evals)
- --team: ./evals/ (team/community evals for sharing via git)`,
		Example: `  staghorn eval create                              # Interactive wizard
  staghorn eval create --template security          # Use security template
  staghorn eval create --from security-secrets      # Copy existing eval
  staghorn eval create --project                    # Save to project directory
  staghorn eval create --team                       # Save to ./evals/ for team sharing`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEvalCreate(project, team, templateName, fromEval, evalName, description)
		},
	}

	cmd.Flags().BoolVar(&project, "project", false, "Save to project directory (.staghorn/evals/) instead of personal")
	cmd.Flags().BoolVar(&team, "team", false, "Save to ./evals/ for team/community sharing via git")
	cmd.Flags().StringVar(&templateName, "template", "", "Template to use (security, quality, language, blank)")
	cmd.Flags().StringVar(&fromEval, "from", "", "Copy from an existing eval")
	cmd.Flags().StringVar(&evalName, "name", "", "Name for the new eval")
	cmd.Flags().StringVar(&description, "description", "", "Description for the new eval")

	return cmd
}

func runEvalCreate(project, team bool, templateName, fromEval, evalName, description string) error {
	paths := config.NewPaths()

	// Check for conflicting flags
	if project && team {
		return fmt.Errorf("cannot use both --project and --team flags")
	}

	// Determine target directory
	var targetDir string
	var targetLabel string
	if team {
		// Team evals go to ./evals/ in the current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		targetDir = cwd + "/evals"
		targetLabel = "./evals/"
	} else if project {
		projectRoot := findProjectRoot()
		if projectRoot == "" {
			return fmt.Errorf("no project root found (looking for .git or .staghorn directory)")
		}
		targetDir = config.ProjectEvalsDir(projectRoot)
		targetLabel = ".staghorn/evals/"
	} else {
		targetDir = paths.PersonalEvals
		targetLabel = "~/.config/staghorn/evals/"
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	var content string
	var err error

	if fromEval != "" {
		// Copy from existing eval
		content, err = copyFromExistingEval(fromEval, evalName, description)
		if err != nil {
			return err
		}
	} else {
		// Use template (interactive if not specified)
		content, err = createFromTemplate(templateName, evalName, description)
		if err != nil {
			return err
		}
	}

	// Parse to get the name for the filename
	parsedEval, err := eval.Parse(content, eval.SourcePersonal, "")
	if err != nil {
		return fmt.Errorf("generated invalid eval: %w", err)
	}

	// Check if file already exists
	filename := parsedEval.Name + ".yaml"
	filepath := targetDir + "/" + filename
	if _, err := os.Stat(filepath); err == nil {
		return fmt.Errorf("eval file already exists: %s", filepath)
	}

	// Write file
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	printSuccess("Created %s%s", targetLabel, filename)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Edit the eval file to customize tests\n")
	fmt.Printf("  2. Run: %s\n", info("staghorn eval validate "+parsedEval.Name))
	fmt.Printf("  3. Run: %s\n", info("staghorn eval "+parsedEval.Name))

	return nil
}

func copyFromExistingEval(fromName, newName, description string) (string, error) {
	evals, err := loadEvals()
	if err != nil {
		return "", err
	}

	var source *eval.Eval
	for _, e := range evals {
		if e.Name == fromName {
			source = e
			break
		}
	}
	if source == nil {
		return "", fmt.Errorf("eval '%s' not found", fromName)
	}

	// Read the original file
	content, err := os.ReadFile(source.FilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source eval: %w", err)
	}

	// Get new name interactively if not provided
	if newName == "" {
		fmt.Print("Name for new eval: ")
		if _, err := fmt.Scanln(&newName); err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		if newName == "" {
			return "", fmt.Errorf("name is required")
		}
	}

	// Get description interactively if not provided
	if description == "" {
		fmt.Print("Description (press enter to keep original): ")
		var input string
		// Ignore error here - empty input on enter is expected
		_, _ = fmt.Scanln(&input)
		if input != "" {
			description = input
		}
	}

	// Replace name in content
	result := strings.Replace(string(content), "name: "+source.Name, "name: "+newName, 1)

	// Replace description if provided
	if description != "" && source.Description != "" {
		result = strings.Replace(result, "description: "+source.Description, "description: "+description, 1)
	}

	return result, nil
}

func createFromTemplate(templateName, evalName, description string) (string, error) {
	// Interactive mode if template not specified
	if templateName == "" {
		fmt.Println("Select a template:")
		for i, t := range eval.Templates {
			fmt.Printf("  %d. %s - %s\n", i+1, t.Name, t.Description)
		}
		fmt.Print("Choice (1-4): ")
		var choice int
		if _, err := fmt.Scanln(&choice); err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		if choice < 1 || choice > len(eval.Templates) {
			return "", fmt.Errorf("invalid choice")
		}
		templateName = eval.Templates[choice-1].Name
	}

	// Get name interactively if not provided
	if evalName == "" {
		fmt.Print("Name for new eval: ")
		if _, err := fmt.Scanln(&evalName); err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		if evalName == "" {
			return "", fmt.Errorf("name is required")
		}
	}

	// Get description interactively if not provided
	if description == "" {
		fmt.Print("Description: ")
		// Ignore error - empty input on enter uses default
		_, _ = fmt.Scanln(&description)
		if description == "" {
			description = "Custom eval"
		}
	}

	// Get tags
	fmt.Print("Tags (comma-separated, or press enter for none): ")
	var tagsInput string
	// Ignore error - empty input on enter is expected
	_, _ = fmt.Scanln(&tagsInput)
	var tags []string
	if tagsInput != "" {
		for _, tag := range strings.Split(tagsInput, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// Render template
	vars := eval.TemplateVars{
		Name:        evalName,
		Description: description,
		Tags:        tags,
	}

	content, err := eval.RenderTemplateByName(templateName, vars)
	if err != nil {
		return "", err
	}

	return content, nil
}
