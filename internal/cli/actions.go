package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HartBrook/staghorn/internal/actions"
	"github.com/HartBrook/staghorn/internal/config"
	"github.com/spf13/cobra"
)

// NewActionsCmd creates the actions command.
func NewActionsCmd() *cobra.Command {
	var tag string
	var source string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "actions [name]",
		Short: "List actions or show info for a specific action",
		Long: `Lists all available actions from team, personal, and project sources.

If an action name is provided, shows detailed information about that action.
Actions are reusable prompts that can be run with 'staghorn run <action>'.`,
		Example: `  staghorn actions              # List all actions
  staghorn actions -v           # List with details
  staghorn actions security-audit  # Show info for specific action
  staghorn actions --tag security  # Filter by tag`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				// Show info for specific action
				return runActionInfo(args[0])
			}
			return runActionsList(tag, source, verbose)
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&source, "source", "", "Filter by source (team, personal, project)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information including arguments")

	return cmd
}

func runActionsList(tagFilter, sourceFilter string, verbose bool) error {
	registry, err := loadActionRegistry()
	if err != nil {
		return err
	}

	if registry.Count() == 0 {
		fmt.Println("No actions found.")
		fmt.Println()
		fmt.Println("Actions are reusable prompts for common workflows like code reviews,")
		fmt.Println("security audits, and documentation generation.")
		fmt.Println()
		fmt.Println(dim("To create a personal action:"))
		fmt.Println()
		fmt.Println("  1. Create ~/.config/staghorn/actions/my-action.md")
		fmt.Println("  2. Add YAML frontmatter and prompt content:")
		fmt.Println()
		fmt.Println(dim("     ---"))
		fmt.Println(dim("     name: my-action"))
		fmt.Println(dim("     description: What this action does"))
		fmt.Println(dim("     ---"))
		fmt.Println(dim("     Your prompt content here..."))
		fmt.Println()
		fmt.Println("  3. Run it with: " + info("staghorn run my-action"))
		fmt.Println()
		fmt.Println(dim("Actions can also come from:"))
		fmt.Println(dim("  - Team repo (actions/ directory, synced via 'staghorn sync')"))
		fmt.Println(dim("  - Project (.staghorn/actions/)"))
		return nil
	}

	// Apply filters
	var filtered []*actions.Action
	if tagFilter != "" {
		filtered = registry.ByTag(tagFilter)
	} else {
		filtered = registry.All()
	}

	if sourceFilter != "" {
		var src actions.Source
		switch sourceFilter {
		case "team":
			src = actions.SourceTeam
		case "personal":
			src = actions.SourcePersonal
		case "project":
			src = actions.SourceProject
		default:
			return fmt.Errorf("invalid source: %s (use team, personal, or project)", sourceFilter)
		}

		var sourceFiltered []*actions.Action
		for _, a := range filtered {
			if a.Source == src {
				sourceFiltered = append(sourceFiltered, a)
			}
		}
		filtered = sourceFiltered
	}

	if len(filtered) == 0 {
		fmt.Println("No actions match the filter.")
		return nil
	}

	// Group by source for display
	teamActions := filterBySource(filtered, actions.SourceTeam)
	personalActions := filterBySource(filtered, actions.SourcePersonal)
	projectActions := filterBySource(filtered, actions.SourceProject)

	if len(teamActions) > 0 {
		printActionGroup("TEAM ACTIONS", teamActions, verbose)
	}

	if len(personalActions) > 0 {
		if len(teamActions) > 0 {
			fmt.Println()
		}
		printActionGroup("PERSONAL ACTIONS", personalActions, verbose)
	}

	if len(projectActions) > 0 {
		if len(teamActions) > 0 || len(personalActions) > 0 {
			fmt.Println()
		}
		printActionGroup("PROJECT ACTIONS", projectActions, verbose)
	}

	fmt.Println()
	fmt.Printf("Use: %s\n", info("staghorn run <action>"))

	return nil
}

func filterBySource(actionList []*actions.Action, source actions.Source) []*actions.Action {
	var result []*actions.Action
	for _, a := range actionList {
		if a.Source == source {
			result = append(result, a)
		}
	}
	return result
}

func printActionGroup(title string, actionList []*actions.Action, verbose bool) {
	fmt.Println(dim(title))
	for _, a := range actionList {
		name := a.Name
		desc := a.Description
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
			if len(a.Tags) > 0 {
				fmt.Printf("                       %s %s\n", dim("Tags:"), strings.Join(a.Tags, ", "))
			}
			// Show args
			if len(a.Args) > 0 {
				var argStrs []string
				for _, arg := range a.Args {
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

// loadActionRegistry loads actions from all sources.
func loadActionRegistry() (*actions.Registry, error) {
	paths := config.NewPaths()

	// Get team actions directory
	var teamActionsDir string
	if config.Exists() {
		cfg, err := config.Load()
		if err == nil {
			owner, repo, err := cfg.Team.ParseRepo()
			if err == nil {
				teamActionsDir = paths.TeamActionsDir(owner, repo)
			}
		}
	}

	// Find project root by looking for .git or .staghorn directory
	projectActionsDir := ""
	if projectRoot := findProjectRoot(); projectRoot != "" {
		projectActionsDir = config.ProjectActionsDir(projectRoot)
	}

	return actions.LoadRegistry(teamActionsDir, paths.PersonalActions, projectActionsDir)
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
		Use:   "run <action> [args]",
		Short: "Run an action",
		Long: `Renders an action's prompt template and outputs it to stdout.

Arguments can be passed as --name=value or name=value.`,
		Example: `  staghorn run security-audit
  staghorn run security-audit --path=src/
  staghorn run code-review path=. severity=high`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			actionName := args[0]
			actionArgs := args[1:]
			return runAction(actionName, actionArgs, dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be rendered without executing")

	return cmd
}

func runAction(actionName string, rawArgs []string, dryRun bool) error {
	registry, err := loadActionRegistry()
	if err != nil {
		return err
	}

	action := registry.Get(actionName)
	if action == nil {
		return fmt.Errorf("action '%s' not found", actionName)
	}

	// Parse arguments
	args, err := actions.ParseArgs(rawArgs)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println(dim("Action:"), action.Name)
		fmt.Println(dim("Source:"), action.Source.Label())
		fmt.Println(dim("Args:"))
		for _, arg := range action.Args {
			val := action.GetArgWithDefault(args, arg.Name)
			fmt.Printf("  %s = %s\n", arg.Name, val)
		}
		fmt.Println()
		fmt.Println(dim("--- Preview ---"))
	}

	// Render the action
	output, err := action.Render(args)
	if err != nil {
		return err
	}

	fmt.Println(output)
	return nil
}

// NewActionCmd creates the action parent command for management subcommands.
func NewActionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "action",
		Short: "Manage actions",
		Long:  `Commands for managing actions (info, new, edit, override).`,
	}

	cmd.AddCommand(NewActionInfoCmd())

	return cmd
}

// NewActionInfoCmd creates the 'action info' command.
func NewActionInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <action>",
		Short: "Show detailed information about an action",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runActionInfo(args[0])
		},
	}
}

func runActionInfo(actionName string) error {
	registry, err := loadActionRegistry()
	if err != nil {
		return err
	}

	action := registry.Get(actionName)
	if action == nil {
		return fmt.Errorf("action '%s' not found", actionName)
	}

	fmt.Println(dim("Name:"), info(action.Name))
	fmt.Println(dim("Source:"), action.Source.Label())

	if action.Description != "" {
		fmt.Println(dim("Description:"), action.Description)
	}

	if len(action.Tags) > 0 {
		fmt.Println(dim("Tags:"), strings.Join(action.Tags, ", "))
	}

	fmt.Println()
	fmt.Println(dim("Arguments:"))
	if len(action.Args) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, arg := range action.Args {
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
	versions := registry.GetAllVersions(actionName)
	if len(versions) > 1 {
		fmt.Println()
		fmt.Println(dim("Versions:"))
		for _, v := range versions {
			active := ""
			if v == action {
				active = " (active)"
			}
			fmt.Printf("  %s%s\n", v.Source.Label(), active)
		}
	}

	return nil
}
