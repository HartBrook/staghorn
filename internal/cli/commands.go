package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HartBrook/staghorn/internal/commands"
	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/starter"
	"github.com/spf13/cobra"
)

// NewCommandsCmd creates the commands command.
func NewCommandsCmd() *cobra.Command {
	var tag string
	var source string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "commands [name]",
		Short: "List commands or show info for a specific command",
		Long: `Lists all available commands from team, personal, and project sources.

If a command name is provided, shows detailed information about that command.
Commands are reusable prompts that can be run with 'staghorn run <command>'.`,
		Example: `  staghorn commands              # List all commands
  staghorn commands -v           # List with details
  staghorn commands security-audit  # Show info for specific command
  staghorn commands --tag security  # Filter by tag`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				// Show info for specific command
				return runCommandInfo(args[0])
			}
			return runCommandsList(tag, source, verbose)
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&source, "source", "", "Filter by source (team, personal, project)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information including arguments")

	// Add subcommands
	cmd.AddCommand(NewCommandsInitCmd())

	return cmd
}

// NewCommandsInitCmd creates the 'commands init' command to bootstrap starter commands.
func NewCommandsInitCmd() *cobra.Command {
	var project bool
	var claude bool
	var claudeProject bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Install starter commands",
		Long: `Installs staghorn's built-in starter commands to your personal or project config.

Starter commands include common workflows like code-review, debug, refactor,
test-gen, and more. Commands that already exist will be skipped.

Use --claude to install commands directly as Claude Code slash commands.`,
		Example: `  staghorn commands init                  # Install to ~/.config/staghorn/commands/
  staghorn commands init --project         # Install to .staghorn/commands/
  staghorn commands init --claude          # Install to ~/.claude/commands/
  staghorn commands init --claude-project  # Install to .claude/commands/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if claude || claudeProject {
				return runCommandsInitClaude(claudeProject)
			}
			return runCommandsInit(project)
		},
	}

	cmd.Flags().BoolVar(&project, "project", false, "Install to project directory (.staghorn/commands/)")
	cmd.Flags().BoolVar(&claude, "claude", false, "Install to Claude Code commands (~/.claude/commands/)")
	cmd.Flags().BoolVar(&claudeProject, "claude-project", false, "Install to project Claude commands (.claude/commands/)")

	return cmd
}

func runCommandsInitClaude(project bool) error {
	paths := config.NewPaths()

	var targetDir string
	var targetLabel string

	if project {
		projectRoot := findProjectRoot()
		if projectRoot == "" {
			return fmt.Errorf("no project root found (looking for .git or .staghorn directory)")
		}
		targetDir = config.ProjectClaudeCommandsDir(projectRoot)
		targetLabel = ".claude/commands/"
	} else {
		targetDir = paths.ClaudeCommandsDir()
		targetLabel = "~/.claude/commands/"
	}

	// Get starter commands
	starterCommands, err := starter.LoadStarterCommands()
	if err != nil {
		return fmt.Errorf("failed to load starter commands: %w", err)
	}

	fmt.Printf("Installing %d starter commands to %s\n", len(starterCommands), targetLabel)
	fmt.Println()

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check for collisions first
	var collisions []string
	for _, cmd := range starterCommands {
		filename := cmd.Name + ".md"
		outputPath := filepath.Join(targetDir, filename)

		if existingContent, err := os.ReadFile(outputPath); err == nil {
			if !strings.Contains(string(existingContent), "Managed by staghorn") {
				collisions = append(collisions, cmd.Name)
			}
		}
	}

	// Handle collisions
	overwriteAll := false
	skipAll := false
	if len(collisions) > 0 {
		printWarning("Found %d existing commands not managed by staghorn:", len(collisions))
		for _, name := range collisions {
			fmt.Printf("  /%s\n", name)
		}
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  1. Skip these commands (keep existing)")
		fmt.Println("  2. Overwrite with staghorn versions")
		fmt.Println("  3. Abort")
		fmt.Println()

		choice := promptString("Choose an option [1/2/3]:")
		switch choice {
		case "1", "":
			skipAll = true
		case "2":
			overwriteAll = true
		case "3":
			fmt.Println("Aborted.")
			return nil
		default:
			return fmt.Errorf("invalid option")
		}
		fmt.Println()
	}

	// Write each command as a Claude command
	count := 0
	var installedNames []string
	for _, cmd := range starterCommands {
		filename := cmd.Name + ".md"
		outputPath := filepath.Join(targetDir, filename)

		// Handle existing files
		if existingContent, err := os.ReadFile(outputPath); err == nil {
			isManagedByStaghorn := strings.Contains(string(existingContent), "Managed by staghorn")

			if !isManagedByStaghorn {
				// This is a collision
				if skipAll {
					continue
				}
				if !overwriteAll {
					continue
				}
				// overwriteAll is true, proceed to write
			}
			// If managed by staghorn, always update
		}

		content := commands.ConvertToClaude(cmd)
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			printWarning("Failed to write %s: %v", cmd.Name, err)
			continue
		}
		count++
		installedNames = append(installedNames, cmd.Name)
	}

	if count > 0 {
		printSuccess("Installed %d Claude commands", count)
		fmt.Println()
		fmt.Println("Installed commands:")
		for _, name := range installedNames {
			fmt.Printf("  /%s\n", info(name))
		}
	} else {
		fmt.Println("All starter commands already installed.")
	}

	fmt.Println()
	fmt.Printf("Use %s in Claude Code to run these commands.\n", info("/code-review"))

	return nil
}

func runCommandsInit(project bool) error {
	paths := config.NewPaths()

	var targetDir string
	var targetLabel string

	if project {
		projectRoot := findProjectRoot()
		if projectRoot == "" {
			return fmt.Errorf("no project root found (looking for .git or .staghorn directory)")
		}
		targetDir = config.ProjectCommandsDir(projectRoot)
		targetLabel = ".staghorn/commands/"
	} else {
		targetDir = paths.PersonalCommands
		targetLabel = paths.PersonalCommands
	}

	fmt.Printf("Installing starter commands to %s\n", targetLabel)
	fmt.Println()

	// Check collisions only for personal installs (not project)
	result, err := InstallStarterCommands(targetDir, !project)
	if err != nil {
		return fmt.Errorf("failed to install commands: %w", err)
	}

	if result.Aborted {
		fmt.Println("Aborted.")
		return nil
	}

	if len(result.Installed) > 0 {
		printSuccess("Installed %d commands", len(result.Installed))
		fmt.Println()
		fmt.Println("Installed commands:")
		for _, name := range result.Installed {
			fmt.Printf("  %s\n", info(name))
		}
	} else {
		fmt.Println("All starter commands already installed (or skipped).")
	}

	if len(result.Skipped) > 0 {
		fmt.Println()
		fmt.Printf("Skipped %d commands (using team versions): %s\n",
			len(result.Skipped), strings.Join(result.Skipped, ", "))
	}

	fmt.Println()
	fmt.Printf("Run %s to see all available commands.\n", info("staghorn commands"))

	return nil
}

// findTeamCollisions returns starter command names that would collide with team commands.
func findTeamCollisions(starterNames []string) []string {
	paths := config.NewPaths()

	// Get team commands directory
	var teamCommandsDir string
	if config.Exists() {
		cfg, err := config.Load()
		if err == nil {
			owner, repo, err := cfg.Team.ParseRepo()
			if err == nil {
				teamCommandsDir = paths.TeamCommandsDir(owner, repo)
			}
		}
	}

	if teamCommandsDir == "" {
		return nil
	}

	// Load team commands
	teamCommands, err := commands.LoadFromDirectory(teamCommandsDir, commands.SourceTeam)
	if err != nil || len(teamCommands) == 0 {
		return nil
	}

	// Build set of team command names
	teamNames := make(map[string]bool)
	for _, cmd := range teamCommands {
		teamNames[cmd.Name] = true
	}

	// Find collisions
	var collisions []string
	for _, name := range starterNames {
		if teamNames[name] {
			collisions = append(collisions, name)
		}
	}

	return collisions
}

// StarterInstallResult contains the results of installing starter commands.
type StarterInstallResult struct {
	Installed []string // Commands that were installed
	Skipped   []string // Commands skipped due to team collision
	Existing  int      // Commands that already existed
	Aborted   bool     // User chose to abort
}

// InstallStarterCommands installs starter commands with collision detection.
// It prompts the user if there are collisions with team commands.
// Set checkCollisions=false to skip collision detection (e.g., for project installs).
func InstallStarterCommands(targetDir string, checkCollisions bool) (*StarterInstallResult, error) {
	commandNames := starter.CommandNames()
	result := &StarterInstallResult{}

	var skipCommands []string
	if checkCollisions {
		teamCollisions := findTeamCollisions(commandNames)
		if len(teamCollisions) > 0 {
			printWarning("The following starter commands would shadow team commands:")
			for _, name := range teamCollisions {
				fmt.Printf("  %s\n", name)
			}
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  1. Skip these commands (recommended - use team versions)")
			fmt.Println("  2. Install anyway (personal will override team)")
			fmt.Println("  3. Abort")
			fmt.Println()

			choice := promptString("Choose an option [1/2/3]:")
			switch choice {
			case "1", "":
				skipCommands = teamCollisions
			case "2":
				// Install all, no skipping
			case "3":
				result.Aborted = true
				return result, nil
			default:
				return nil, fmt.Errorf("invalid option")
			}
			fmt.Println()
		}
	}

	count, installed, err := starter.BootstrapCommandsWithSkip(targetDir, skipCommands)
	if err != nil {
		return nil, err
	}

	result.Installed = installed
	result.Skipped = skipCommands
	result.Existing = len(commandNames) - count - len(skipCommands)

	return result, nil
}

func runCommandsList(tagFilter, sourceFilter string, verbose bool) error {
	registry, err := loadCommandRegistry()
	if err != nil {
		return err
	}

	if registry.Count() == 0 {
		fmt.Println("No commands found.")
		fmt.Println()
		fmt.Println("Commands are reusable prompts for common workflows like code reviews,")
		fmt.Println("security audits, and documentation generation.")
		fmt.Println()
		fmt.Println(dim("To create a personal command:"))
		fmt.Println()
		fmt.Println("  1. Create ~/.config/staghorn/commands/my-command.md")
		fmt.Println("  2. Add YAML frontmatter and prompt content:")
		fmt.Println()
		fmt.Println(dim("     ---"))
		fmt.Println(dim("     name: my-command"))
		fmt.Println(dim("     description: What this command does"))
		fmt.Println(dim("     ---"))
		fmt.Println(dim("     Your prompt content here..."))
		fmt.Println()
		fmt.Println("  3. Run it with: " + info("staghorn run my-command"))
		fmt.Println()
		fmt.Println(dim("Commands can also come from:"))
		fmt.Println(dim("  - Team repo (commands/ directory, synced via 'staghorn sync')"))
		fmt.Println(dim("  - Project (.staghorn/commands/)"))
		return nil
	}

	// Apply filters
	var filtered []*commands.Command
	if tagFilter != "" {
		filtered = registry.ByTag(tagFilter)
	} else {
		filtered = registry.All()
	}

	if sourceFilter != "" {
		var src commands.Source
		switch sourceFilter {
		case "team":
			src = commands.SourceTeam
		case "personal":
			src = commands.SourcePersonal
		case "project":
			src = commands.SourceProject
		default:
			return fmt.Errorf("invalid source: %s (use team, personal, or project)", sourceFilter)
		}

		var sourceFiltered []*commands.Command
		for _, c := range filtered {
			if c.Source == src {
				sourceFiltered = append(sourceFiltered, c)
			}
		}
		filtered = sourceFiltered
	}

	if len(filtered) == 0 {
		fmt.Println("No commands match the filter.")
		return nil
	}

	// Group by source for display
	teamCommands := filterBySource(filtered, commands.SourceTeam)
	personalCommands := filterBySource(filtered, commands.SourcePersonal)
	projectCommands := filterBySource(filtered, commands.SourceProject)

	if len(teamCommands) > 0 {
		printCommandGroup("TEAM COMMANDS", teamCommands, verbose)
	}

	if len(personalCommands) > 0 {
		if len(teamCommands) > 0 {
			fmt.Println()
		}
		printCommandGroup("PERSONAL COMMANDS", personalCommands, verbose)
	}

	if len(projectCommands) > 0 {
		if len(teamCommands) > 0 || len(personalCommands) > 0 {
			fmt.Println()
		}
		printCommandGroup("PROJECT COMMANDS", projectCommands, verbose)
	}

	fmt.Println()
	fmt.Printf("Use: %s\n", info("staghorn run <command>"))

	return nil
}

func filterBySource(cmdList []*commands.Command, source commands.Source) []*commands.Command {
	var result []*commands.Command
	for _, c := range cmdList {
		if c.Source == source {
			result = append(result, c)
		}
	}
	return result
}

func printCommandGroup(title string, cmdList []*commands.Command, verbose bool) {
	fmt.Println(dim(title))
	for _, c := range cmdList {
		name := c.Name
		desc := c.Description
		if desc == "" {
			desc = "(no description)"
		}

		// Truncate description if too long (unless verbose)
		if !verbose && len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		fmt.Printf("  %-20s %s\n", info(name), desc)

		if verbose {
			// Show tags
			if len(c.Tags) > 0 {
				fmt.Printf("                       %s %s\n", dim("Tags:"), strings.Join(c.Tags, ", "))
			}
			// Show args
			if len(c.Args) > 0 {
				var argStrs []string
				for _, arg := range c.Args {
					argStr := "--" + arg.Name
					if arg.Default != "" {
						argStr += "=" + arg.Default
					}
					if arg.Required {
						argStr += " (required)"
					}
					argStrs = append(argStrs, argStr)
				}
				fmt.Printf("                       %s %s\n", dim("Args:"), strings.Join(argStrs, ", "))
			}
			fmt.Println()
		}
	}
}

// loadCommandRegistry loads commands from all sources.
func loadCommandRegistry() (*commands.Registry, error) {
	paths := config.NewPaths()

	// Get team commands directory
	var teamCommandsDir string
	if config.Exists() {
		cfg, err := config.Load()
		if err == nil {
			owner, repo, err := cfg.Team.ParseRepo()
			if err == nil {
				teamCommandsDir = paths.TeamCommandsDir(owner, repo)
			}
		}
	}

	// Find project root by looking for .git or .staghorn directory
	projectCommandsDir := ""
	if projectRoot := findProjectRoot(); projectRoot != "" {
		projectCommandsDir = config.ProjectCommandsDir(projectRoot)
	}

	return commands.LoadRegistry(teamCommandsDir, paths.PersonalCommands, projectCommandsDir)
}

// findProjectRoot walks up from cwd to find a directory containing .git or .staghorn.
func findProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := cwd
	for {
		// Check for .git directory
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		// Check for .staghorn directory
		if _, err := os.Stat(filepath.Join(dir, ".staghorn")); err == nil {
			return dir
		}

		// Move up
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding project markers
			return cwd // Fall back to cwd
		}
		dir = parent
	}
}

// NewRunCmd creates the run command.
func NewRunCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "run <command> [args]",
		Short: "Run a command",
		Long: `Renders a command's prompt template and outputs it to stdout.

Arguments can be passed as --name=value or name=value.`,
		Example: `  staghorn run security-audit
  staghorn run security-audit --path=src/
  staghorn run code-review path=. severity=high`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdName := args[0]
			cmdArgs := args[1:]
			return runCommand(cmdName, cmdArgs, dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be rendered without executing")

	return cmd
}

func runCommand(cmdName string, rawArgs []string, dryRun bool) error {
	registry, err := loadCommandRegistry()
	if err != nil {
		return err
	}

	cmd := registry.Get(cmdName)
	if cmd == nil {
		return fmt.Errorf("command '%s' not found", cmdName)
	}

	// Parse arguments
	args, err := commands.ParseArgs(rawArgs)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println(dim("Command:"), cmd.Name)
		fmt.Println(dim("Source:"), cmd.Source.Label())
		fmt.Println(dim("Args:"))
		for _, arg := range cmd.Args {
			val := cmd.GetArgWithDefault(args, arg.Name)
			fmt.Printf("  %s = %s\n", arg.Name, val)
		}
		fmt.Println()
		fmt.Println(dim("--- Preview ---"))
	}

	// Render the command
	output, err := cmd.Render(args)
	if err != nil {
		return err
	}

	fmt.Println(output)
	return nil
}

// NewCommandInfoCmd creates the 'command info' command.
func NewCommandInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <command>",
		Short: "Show detailed information about a command",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runCommandInfo(args[0])
		},
	}
}

func runCommandInfo(cmdName string) error {
	registry, err := loadCommandRegistry()
	if err != nil {
		return err
	}

	cmd := registry.Get(cmdName)
	if cmd == nil {
		return fmt.Errorf("command '%s' not found", cmdName)
	}

	fmt.Println(dim("Name:"), info(cmd.Name))
	fmt.Println(dim("Source:"), cmd.Source.Label())

	if cmd.Description != "" {
		fmt.Println(dim("Description:"), cmd.Description)
	}

	if len(cmd.Tags) > 0 {
		fmt.Println(dim("Tags:"), strings.Join(cmd.Tags, ", "))
	}

	fmt.Println()
	fmt.Println(dim("Arguments:"))
	if len(cmd.Args) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, arg := range cmd.Args {
			required := ""
			if arg.Required {
				required = " (required)"
			}
			fmt.Printf("  --%s%s\n", arg.Name, required)
			if arg.Description != "" {
				fmt.Printf("      %s\n", arg.Description)
			}
			if arg.Default != "" {
				fmt.Printf("      Default: %s\n", arg.Default)
			}
			if len(arg.Options) > 0 {
				fmt.Printf("      Options: %s\n", strings.Join(arg.Options, ", "))
			}
		}
	}

	// Show if overridden
	versions := registry.GetAllVersions(cmdName)
	if len(versions) > 1 {
		fmt.Println()
		fmt.Println(dim("Versions:"))
		for _, v := range versions {
			active := ""
			if v == cmd {
				active = " (active)"
			}
			fmt.Printf("  %s%s\n", v.Source.Label(), active)
		}
	}

	return nil
}
