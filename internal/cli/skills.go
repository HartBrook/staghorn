package cli

import (
	"fmt"
	"strings"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/skills"
	"github.com/HartBrook/staghorn/internal/starter"
	"github.com/spf13/cobra"
)

// NewSkillsCmd creates the skills command.
func NewSkillsCmd() *cobra.Command {
	var tag string
	var source string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "skills [name]",
		Short: "List skills or show info for a specific skill",
		Long: `Lists all available skills from team, personal, and project sources.

If a skill name is provided, shows detailed information about that skill.
Skills are directories containing SKILL.md plus optional supporting files
like templates, scripts, and references.

Skills follow the Agent Skills standard (agentskills.io) and support extended
Claude Code features like tool restrictions, subagent execution, and hooks.`,
		Example: `  staghorn skills              # List all skills
  staghorn skills -v           # List with details
  staghorn skills code-review  # Show info for specific skill
  staghorn skills --tag review # Filter by tag`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				// Show info for specific skill
				return runSkillInfo(args[0])
			}
			return runSkillsList(tag, source, verbose)
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&source, "source", "", "Filter by source (team, personal, project)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information including supporting files")

	// Add subcommands
	cmd.AddCommand(NewSkillsInitCmd())

	return cmd
}

// NewSkillsInitCmd creates the 'skills init' command to bootstrap starter skills.
func NewSkillsInitCmd() *cobra.Command {
	var project bool
	var claude bool
	var claudeProject bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Install starter skills",
		Long: `Installs staghorn's built-in starter skills to your personal or project config.

Starter skills include common workflows like code-review, test-gen, and
security-audit with enhanced features like tool restrictions and subagent execution.

Use --claude to install skills directly to Claude Code's skills directory.`,
		Example: `  staghorn skills init                  # Install to ~/.config/staghorn/skills/
  staghorn skills init --project         # Install to .staghorn/skills/
  staghorn skills init --claude          # Install to ~/.claude/skills/
  staghorn skills init --claude-project  # Install to .claude/skills/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if claude || claudeProject {
				return runSkillsInitClaude(claudeProject)
			}
			return runSkillsInit(project)
		},
	}

	cmd.Flags().BoolVar(&project, "project", false, "Install to project directory (.staghorn/skills/)")
	cmd.Flags().BoolVar(&claude, "claude", false, "Install to Claude Code skills (~/.claude/skills/)")
	cmd.Flags().BoolVar(&claudeProject, "claude-project", false, "Install to project Claude skills (.claude/skills/)")

	return cmd
}

func runSkillsInit(project bool) error {
	paths := config.NewPaths()

	var targetDir string
	var targetLabel string

	if project {
		projectRoot := findProjectRoot()
		if projectRoot == "" {
			return fmt.Errorf("no project root found (looking for .git or .staghorn directory)")
		}
		targetDir = config.ProjectSkillsDir(projectRoot)
		targetLabel = ".staghorn/skills/"
	} else {
		targetDir = paths.PersonalSkills
		targetLabel = "~/.config/staghorn/skills/"
	}

	fmt.Printf("Installing starter skills to %s\n", targetLabel)
	fmt.Println()

	// Show available skills
	skillNames := starter.SkillNames()
	fmt.Printf("Available starter skills (%d):\n", len(skillNames))
	for _, name := range skillNames {
		fmt.Printf("  - %s\n", info(name))
	}
	fmt.Println()

	// Install starter skills
	count, installed, err := starter.BootstrapSkillsWithSkip(targetDir, nil)
	if err != nil {
		return fmt.Errorf("failed to install starter skills: %w", err)
	}

	if count == 0 {
		fmt.Println(dim("All starter skills already installed."))
	} else {
		printSuccess("Installed %d starter skills:", count)
		for _, name := range installed {
			fmt.Printf("  - %s\n", info(name))
		}
	}

	fmt.Println()
	fmt.Printf("Skills are invoked via %s in Claude Code.\n", info("/skill-name"))

	return nil
}

func runSkillsInitClaude(project bool) error {
	paths := config.NewPaths()

	var targetDir string
	var targetLabel string

	if project {
		projectRoot := findProjectRoot()
		if projectRoot == "" {
			return fmt.Errorf("no project root found (looking for .git or .staghorn directory)")
		}
		targetDir = config.ProjectClaudeSkillsDir(projectRoot)
		targetLabel = ".claude/skills/"
	} else {
		targetDir = paths.ClaudeSkillsDir()
		targetLabel = "~/.claude/skills/"
	}

	fmt.Printf("Installing starter skills to %s\n", targetLabel)
	fmt.Println()

	// Show available skills
	skillNames := starter.SkillNames()
	fmt.Printf("Available starter skills (%d):\n", len(skillNames))
	for _, name := range skillNames {
		fmt.Printf("  - %s\n", info(name))
	}
	fmt.Println()

	// Install starter skills
	count, installed, err := starter.BootstrapSkillsWithSkip(targetDir, nil)
	if err != nil {
		return fmt.Errorf("failed to install starter skills: %w", err)
	}

	if count == 0 {
		fmt.Println(dim("All starter skills already installed."))
	} else {
		printSuccess("Installed %d starter skills:", count)
		for _, name := range installed {
			fmt.Printf("  - %s\n", info(name))
		}
	}

	fmt.Println()
	fmt.Printf("Skills are invoked via %s in Claude Code.\n", info("/skill-name"))

	return nil
}

func runSkillsList(tagFilter, sourceFilter string, verbose bool) error {
	registry, err := loadSkillRegistry()
	if err != nil {
		return err
	}

	if registry.Count() == 0 {
		fmt.Println("No skills found.")
		fmt.Println()
		fmt.Println("Skills are directories containing SKILL.md plus optional supporting files")
		fmt.Println("like templates, scripts, and references. They follow the Agent Skills")
		fmt.Println("standard (agentskills.io) and support Claude Code's extended features.")
		fmt.Println()
		fmt.Println(dim("To create a personal skill:"))
		fmt.Println()
		fmt.Println("  1. Create directory ~/.config/staghorn/skills/my-skill/")
		fmt.Println("  2. Add SKILL.md with YAML frontmatter:")
		fmt.Println()
		fmt.Println(dim("     ---"))
		fmt.Println(dim("     name: my-skill"))
		fmt.Println(dim("     description: What this skill does"))
		fmt.Println(dim("     allowed-tools: Read Grep Glob"))
		fmt.Println(dim("     ---"))
		fmt.Println(dim("     Instructions for the skill..."))
		fmt.Println()
		fmt.Println("  3. Optionally add templates/, scripts/, references/ directories")
		fmt.Println()
		fmt.Println(dim("Skills can also come from:"))
		fmt.Println(dim("  - Team repo (skills/ directory, synced via 'staghorn sync')"))
		fmt.Println(dim("  - Project (.staghorn/skills/)"))
		fmt.Println(dim("  - Community repos (via multi-source config)"))
		return nil
	}

	// Apply filters
	var filtered []*skills.Skill
	if tagFilter != "" {
		filtered = registry.ByTag(tagFilter)
	} else {
		filtered = registry.All()
	}

	if sourceFilter != "" {
		var src skills.Source
		switch sourceFilter {
		case "team":
			src = skills.SourceTeam
		case "personal":
			src = skills.SourcePersonal
		case "project":
			src = skills.SourceProject
		default:
			return fmt.Errorf("invalid source: %s (use team, personal, or project)", sourceFilter)
		}

		var sourceFiltered []*skills.Skill
		for _, s := range filtered {
			if s.Source == src {
				sourceFiltered = append(sourceFiltered, s)
			}
		}
		filtered = sourceFiltered
	}

	if len(filtered) == 0 {
		fmt.Println("No skills match the filter.")
		return nil
	}

	// Group by source for display
	teamSkills := filterSkillsBySource(filtered, skills.SourceTeam)
	personalSkills := filterSkillsBySource(filtered, skills.SourcePersonal)
	projectSkills := filterSkillsBySource(filtered, skills.SourceProject)

	if len(teamSkills) > 0 {
		printSkillGroup("TEAM SKILLS", teamSkills, verbose)
	}

	if len(personalSkills) > 0 {
		if len(teamSkills) > 0 {
			fmt.Println()
		}
		printSkillGroup("PERSONAL SKILLS", personalSkills, verbose)
	}

	if len(projectSkills) > 0 {
		if len(teamSkills) > 0 || len(personalSkills) > 0 {
			fmt.Println()
		}
		printSkillGroup("PROJECT SKILLS", projectSkills, verbose)
	}

	fmt.Println()
	fmt.Printf("Skills are invoked via %s in Claude Code.\n", info("/skill-name"))

	return nil
}

func filterSkillsBySource(skillList []*skills.Skill, source skills.Source) []*skills.Skill {
	var result []*skills.Skill
	for _, s := range skillList {
		if s.Source == source {
			result = append(result, s)
		}
	}
	return result
}

func printSkillGroup(title string, skillList []*skills.Skill, verbose bool) {
	fmt.Println(dim(title))
	for _, s := range skillList {
		name := s.Name
		desc := s.Description
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
			if len(s.Tags) > 0 {
				fmt.Printf("                       %s %s\n", dim("Tags:"), strings.Join(s.Tags, ", "))
			}
			// Show allowed tools
			if s.AllowedTools != "" {
				fmt.Printf("                       %s %s\n", dim("Tools:"), s.AllowedTools)
			}
			// Show context
			if s.Context != "" {
				fmt.Printf("                       %s %s\n", dim("Context:"), s.Context)
			}
			// Show supporting files count
			if len(s.SupportingFiles) > 0 {
				fmt.Printf("                       %s %d supporting files\n", dim("Files:"), len(s.SupportingFiles))
			}
			fmt.Println()
		}
	}
}

// loadSkillRegistry loads skills from all sources.
func loadSkillRegistry() (*skills.Registry, error) {
	paths := config.NewPaths()

	// Get team skills directory
	var teamSkillsDir string
	if config.Exists() {
		cfg, err := config.Load()
		if err == nil {
			owner, repo, err := cfg.DefaultOwnerRepo()
			if err == nil {
				teamSkillsDir = paths.TeamSkillsDir(owner, repo)
			}
		}
	}

	// Find project root
	projectSkillsDir := ""
	if projectRoot := findProjectRoot(); projectRoot != "" {
		projectSkillsDir = config.ProjectSkillsDir(projectRoot)
	}

	return skills.LoadRegistry(teamSkillsDir, paths.PersonalSkills, projectSkillsDir)
}

func runSkillInfo(skillName string) error {
	registry, err := loadSkillRegistry()
	if err != nil {
		return err
	}

	skill := registry.Get(skillName)
	if skill == nil {
		return fmt.Errorf("skill '%s' not found", skillName)
	}

	fmt.Println(dim("Name:"), info(skill.Name))
	fmt.Println(dim("Source:"), skill.Source.Label())

	if skill.Description != "" {
		fmt.Println(dim("Description:"), skill.Description)
	}

	if len(skill.Tags) > 0 {
		fmt.Println(dim("Tags:"), strings.Join(skill.Tags, ", "))
	}

	// Agent Skills standard fields
	if skill.License != "" {
		fmt.Println(dim("License:"), skill.License)
	}
	if skill.Compatibility != "" {
		fmt.Println(dim("Compatibility:"), skill.Compatibility)
	}

	// Claude Code extensions
	fmt.Println()
	fmt.Println(dim("Claude Code Settings:"))
	if skill.AllowedTools != "" {
		fmt.Println("  Allowed Tools:", skill.AllowedTools)
	}
	if skill.Context != "" {
		fmt.Println("  Context:", skill.Context)
	}
	if skill.Agent != "" {
		fmt.Println("  Agent:", skill.Agent)
	}
	fmt.Println("  User Invocable:", skill.IsUserInvocable())
	if skill.DisableModelInvocation {
		fmt.Println("  Model Invocation: disabled")
	}
	if skill.Hooks != nil {
		if skill.Hooks.Pre != "" {
			fmt.Println("  Pre Hook:", skill.Hooks.Pre)
		}
		if skill.Hooks.Post != "" {
			fmt.Println("  Post Hook:", skill.Hooks.Post)
		}
	}

	// Supporting files
	if len(skill.SupportingFiles) > 0 {
		fmt.Println()
		fmt.Println(dim("Supporting Files:"))
		for relPath := range skill.SupportingFiles {
			fmt.Printf("  %s\n", relPath)
		}
	}

	// Show if overridden
	versions := registry.GetAllVersions(skillName)
	if len(versions) > 1 {
		fmt.Println()
		fmt.Println(dim("Versions:"))
		for _, v := range versions {
			active := ""
			if v == skill {
				active = " (active)"
			}
			fmt.Printf("  %s%s\n", v.Source.Label(), active)
		}
	}

	return nil
}
