package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/HartBrook/staghorn/internal/commands"
	"github.com/HartBrook/staghorn/internal/eval"
	"github.com/HartBrook/staghorn/internal/starter"
	"github.com/spf13/cobra"
)

// NewTeamCmd creates the team command group.
func NewTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Team repository management",
		Long: `Commands for managing team standards repositories.

A team repository is a shared GitHub repo that contains CLAUDE.md guidelines,
commands, language configs, and templates that team members sync from.`,
	}

	cmd.AddCommand(NewTeamInitCmd())
	cmd.AddCommand(NewTeamValidateCmd())

	return cmd
}

// NewTeamInitCmd creates the team init command.
func NewTeamInitCmd() *cobra.Command {
	var nonInteractive bool
	var noTemplates bool
	var noReadme bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a team standards repository",
		Long: `Initialize a team standards repository in the current directory.

This command helps you set up a shared repository that team members will
sync from using 'staghorn init'. It creates the standard directory structure
and optionally installs starter commands, language configs, and templates.

Run this in an empty or existing git repository that will become your
team's shared standards repo.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTeamInit(nonInteractive, noTemplates, noReadme)
		},
	}

	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Use defaults without prompting")
	cmd.Flags().BoolVar(&noTemplates, "no-templates", false, "Skip project templates")
	cmd.Flags().BoolVar(&noReadme, "no-readme", false, "Skip README.md generation")

	return cmd
}

// NewTeamValidateCmd creates the team validate command.
func NewTeamValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate team repository structure",
		Long: `Validate that the current directory is a valid team repository.

Checks:
- CLAUDE.md exists and is non-empty
- Commands in commands/ have valid YAML frontmatter
- Languages in languages/ are valid markdown
- Templates in templates/ are valid markdown (optional)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTeamValidate()
		},
	}
}

func runTeamInit(nonInteractive, noTemplates, noReadme bool) error {
	fmt.Println()
	fmt.Println("Team Repository Setup")
	fmt.Println("=====================")
	fmt.Println()
	fmt.Println("This will initialize a team standards repository in the current directory.")
	fmt.Println("Team members will run 'staghorn init' and point to this repo.")
	fmt.Println()

	// Get team info
	var teamName, teamDesc string
	if nonInteractive {
		teamName = "My Team"
		teamDesc = ""
	} else {
		teamName = promptString("Team name:")
		if teamName == "" {
			teamName = "My Team"
		}
		teamDesc = promptString("Team description (optional):")
	}

	// Check for existing CLAUDE.md
	if _, err := os.Stat("CLAUDE.md"); err == nil {
		if nonInteractive {
			fmt.Println("  CLAUDE.md already exists, keeping existing file")
		} else {
			printWarning("CLAUDE.md already exists")
			if !promptYesNo("Overwrite?") {
				fmt.Println("  Keeping existing CLAUDE.md")
			} else {
				if err := writeTeamClaudeMD(teamName, teamDesc); err != nil {
					return fmt.Errorf("failed to write CLAUDE.md: %w", err)
				}
				printSuccess("Wrote CLAUDE.md")
			}
		}
	} else {
		if err := writeTeamClaudeMD(teamName, teamDesc); err != nil {
			return fmt.Errorf("failed to write CLAUDE.md: %w", err)
		}
		printSuccess("Created CLAUDE.md")
	}

	// Starter commands
	fmt.Println()
	starterCommands := starter.CommandNames()
	fmt.Printf("Starter Commands (%d available)\n", len(starterCommands))
	fmt.Println(strings.Repeat("-", 30))

	var selectedCommands []string
	if nonInteractive {
		selectedCommands = starterCommands
	} else {
		selectedCommands = promptSelection("Install starter commands?", starterCommands)
	}

	if len(selectedCommands) > 0 {
		count, _, err := starter.BootstrapCommandsSelective("commands", selectedCommands)
		if err != nil {
			return fmt.Errorf("failed to install commands: %w", err)
		}
		printSuccess("Installed %d commands to ./commands/", count)
	} else {
		// Create empty commands directory
		if err := os.MkdirAll("commands", 0755); err != nil {
			return fmt.Errorf("failed to create commands directory: %w", err)
		}
		fmt.Println("  Created empty ./commands/ directory")
	}

	// Language configs
	fmt.Println()
	starterLanguages := starter.LanguageNames()
	fmt.Printf("Language Configs (%d available)\n", len(starterLanguages))
	fmt.Println(strings.Repeat("-", 30))

	var selectedLanguages []string
	if nonInteractive {
		selectedLanguages = starterLanguages
	} else {
		selectedLanguages = promptSelection("Install language configs?", starterLanguages)
	}

	if len(selectedLanguages) > 0 {
		count, _, err := starter.BootstrapLanguagesSelective("languages", selectedLanguages)
		if err != nil {
			return fmt.Errorf("failed to install languages: %w", err)
		}
		printSuccess("Installed %d language configs to ./languages/", count)
	} else {
		// Create empty languages directory
		if err := os.MkdirAll("languages", 0755); err != nil {
			return fmt.Errorf("failed to create languages directory: %w", err)
		}
		fmt.Println("  Created empty ./languages/ directory")
	}

	// Project templates
	templatesInstalled := 0
	if !noTemplates {
		fmt.Println()
		starterTemplates := starter.TemplateNames()
		if len(starterTemplates) > 0 {
			fmt.Printf("Project Templates (%d available)\n", len(starterTemplates))
			fmt.Println(strings.Repeat("-", 30))

			installTemplates := false
			if nonInteractive {
				installTemplates = true
			} else {
				installTemplates = promptYesNo("Include example templates?")
			}

			if installTemplates {
				count, err := starter.BootstrapTemplates("templates")
				if err != nil {
					return fmt.Errorf("failed to install templates: %w", err)
				}
				templatesInstalled = count
				printSuccess("Created %d templates in ./templates/", count)
			}
		}
	}

	// README.md
	readmeCreated := false
	if !noReadme {
		fmt.Println()
		if _, err := os.Stat("README.md"); err == nil {
			if nonInteractive {
				fmt.Println("  README.md already exists, keeping existing file")
			} else {
				printWarning("README.md already exists")
				if promptYesNo("Overwrite with staghorn template?") {
					if err := writeTeamReadme(teamName, selectedCommands, selectedLanguages); err != nil {
						return fmt.Errorf("failed to write README.md: %w", err)
					}
					readmeCreated = true
					printSuccess("Wrote README.md")
				} else {
					fmt.Println("  Keeping existing README.md")
				}
			}
		} else {
			if err := writeTeamReadme(teamName, selectedCommands, selectedLanguages); err != nil {
				return fmt.Errorf("failed to write README.md: %w", err)
			}
			readmeCreated = true
			printSuccess("Created README.md")
		}
	}

	// Summary
	fmt.Println()
	fmt.Println("Created Files")
	fmt.Println(strings.Repeat("-", 30))
	printFileStatus("CLAUDE.md", "Team guidelines (customize this!)")
	if readmeCreated {
		printFileStatus("README.md", "Explains repo structure")
	}
	printDirStatus("commands/", len(selectedCommands), "starter commands")
	printDirStatus("languages/", len(selectedLanguages), "language configs")
	if templatesInstalled > 0 {
		printDirStatus("templates/", templatesInstalled, "project templates")
	}

	// Next steps
	fmt.Println()
	fmt.Println("Next Steps")
	fmt.Println(strings.Repeat("-", 30))
	fmt.Println("1. Edit CLAUDE.md with your team's guidelines")
	fmt.Println("2. Customize commands for your workflows")
	fmt.Println("3. Commit and push to GitHub")
	fmt.Println("4. Have team members run: staghorn init")
	fmt.Println()

	return nil
}

func runTeamValidate() error {
	fmt.Println()
	fmt.Println("Validating team repository...")
	fmt.Println()

	errors := 0
	warnings := 0

	// Check CLAUDE.md
	if info, err := os.Stat("CLAUDE.md"); err != nil {
		printError("CLAUDE.md not found")
		errors++
	} else if info.Size() == 0 {
		printError("CLAUDE.md is empty")
		errors++
	} else {
		printSuccess("CLAUDE.md exists (%.1f KB)", float64(info.Size())/1024)
	}

	// Check commands/
	commandsValid, commandsTotal, commandErrs := validateCommands("commands")
	if commandsTotal == 0 {
		fmt.Printf("%s commands/ - directory not found or empty (optional)\n", warningIcon)
		warnings++
	} else if len(commandErrs) > 0 {
		for _, e := range commandErrs {
			printError("%s", e)
		}
		errors += len(commandErrs)
		if commandsValid > 0 {
			fmt.Printf("  %d of %d commands valid\n", commandsValid, commandsTotal)
		}
	} else {
		printSuccess("commands/ - %d valid commands", commandsTotal)
	}

	// Check languages/
	langsValid, langsTotal, langErrs := validateLanguages("languages")
	if langsTotal == 0 {
		fmt.Printf("%s languages/ - directory not found or empty (optional)\n", warningIcon)
		warnings++
	} else if len(langErrs) > 0 {
		for _, e := range langErrs {
			printError("%s", e)
		}
		errors += len(langErrs)
		if langsValid > 0 {
			fmt.Printf("  %d of %d configs valid\n", langsValid, langsTotal)
		}
	} else {
		printSuccess("languages/ - %d valid configs", langsTotal)
	}

	// Check templates/ (optional)
	if _, err := os.Stat("templates"); err == nil {
		templatesValid, templatesTotal, templateErrs := validateTemplates("templates")
		if templatesTotal == 0 {
			fmt.Printf("%s templates/ - directory empty\n", warningIcon)
			warnings++
		} else if len(templateErrs) > 0 {
			for _, e := range templateErrs {
				printError("%s", e)
			}
			errors += len(templateErrs)
		} else {
			printSuccess("templates/ - %d valid templates", templatesValid)
		}
	} else {
		fmt.Printf("%s templates/ - directory not found (optional)\n", warningIcon)
	}

	// Check evals/ (optional)
	if _, err := os.Stat("evals"); err == nil {
		evalsValid, evalsTotal, evalErrs := validateEvals("evals")
		if evalsTotal == 0 {
			fmt.Printf("%s evals/ - directory empty\n", warningIcon)
			warnings++
		} else if len(evalErrs) > 0 {
			for _, e := range evalErrs {
				printError("%s", e)
			}
			errors += len(evalErrs)
		} else {
			printSuccess("evals/ - %d valid evals", evalsValid)
		}
	} else {
		fmt.Printf("%s evals/ - directory not found (optional)\n", warningIcon)
	}

	// Summary
	fmt.Println()
	if errors > 0 {
		fmt.Printf("Found %d error(s). Fix issues above before sharing with team.\n", errors)
		return fmt.Errorf("validation failed with %d errors", errors)
	}

	if warnings > 0 {
		fmt.Printf("Team repository is valid with %d warning(s).\n", warnings)
	} else {
		printSuccess("Team repository is valid!")
	}

	return nil
}

// promptSelection handles the all/some/none selection pattern.
// Returns the list of selected items.
func promptSelection(prompt string, items []string) []string {
	fmt.Printf("%s [all/some/none]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "all", "a", "":
		return items
	case "none", "n":
		return nil
	case "some", "s":
		return promptToggleSelection(items)
	default:
		fmt.Println("  Invalid option, using 'none'")
		return nil
	}
}

// promptToggleSelection shows an interactive toggle list.
// Starts with none selected (opt-in).
func promptToggleSelection(items []string) []string {
	selected := make(map[int]bool)
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println()
		for i, item := range items {
			checkbox := "[ ]"
			if selected[i] {
				checkbox = "[x]"
			}
			fmt.Printf("  %d. %s %s\n", i+1, checkbox, item)
		}
		fmt.Println()
		fmt.Print("Enter numbers to toggle, 'done' when finished: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if strings.ToLower(input) == "done" || input == "" {
			break
		}

		// Parse space or comma separated numbers
		parts := strings.FieldsFunc(input, func(r rune) bool {
			return r == ' ' || r == ','
		})

		for _, part := range parts {
			if num, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
				if num >= 1 && num <= len(items) {
					idx := num - 1
					selected[idx] = !selected[idx]
				}
			}
		}
	}

	var result []string
	for i, item := range items {
		if selected[i] {
			result = append(result, item)
		}
	}
	return result
}

func writeTeamClaudeMD(teamName, teamDesc string) error {
	desc := teamDesc
	if desc == "" {
		desc = fmt.Sprintf("Guidelines for Claude Code across all %s projects.", teamName)
	}

	content := fmt.Sprintf(`# %s Engineering Standards

%s

## Code Style

- Write clear, self-documenting code
- Prefer explicit over implicit
- Use meaningful variable and function names

## Code Review

- Review for correctness, readability, and maintainability
- Check for edge cases and error handling
- Ensure tests cover new functionality

## Git Conventions

- Write clear, descriptive commit messages
- Keep commits focused and atomic
- Reference issue numbers when applicable

## Security

- Never commit secrets or credentials
- Validate all external input
- Follow principle of least privilege

## Testing

- Write tests for new functionality
- Maintain existing test coverage
- Test edge cases and error conditions
`, teamName, desc)

	return os.WriteFile("CLAUDE.md", []byte(content), 0644)
}

func writeTeamReadme(teamName string, cmdNames, langNames []string) error {
	var commandsList, languagesList string

	if len(cmdNames) > 0 {
		var sb strings.Builder
		for _, name := range cmdNames {
			sb.WriteString(fmt.Sprintf("- `%s`\n", name))
		}
		commandsList = sb.String()
	} else {
		commandsList = "No commands installed yet.\n"
	}

	if len(langNames) > 0 {
		var sb strings.Builder
		for _, name := range langNames {
			sb.WriteString(fmt.Sprintf("- `%s`\n", name))
		}
		languagesList = sb.String()
	} else {
		languagesList = "No language configs installed yet.\n"
	}

	content := fmt.Sprintf(`# %s Claude Code Standards

This repository contains shared Claude Code configuration for %s.

## For Team Members

To set up your local environment:

%s
staghorn init
%s

When prompted, enter this repository URL.

## Repository Structure

%s
.
├── CLAUDE.md           # Main team guidelines
├── commands/           # Shared commands (code-review, debug, etc.)
├── languages/          # Language-specific configs
└── templates/          # Project templates (optional)
%s

## Available Commands

%s

## Language Configs

%s

## Customization

Team members can add personal customizations that layer on top of these
team standards. Personal configs are stored in ~/.config/staghorn/.

## Updating

Team members can pull the latest changes with:

%s
staghorn sync
%s
`, teamName, teamName, "```bash", "```", "```", "```", commandsList, languagesList, "```bash", "```")

	return os.WriteFile("README.md", []byte(content), 0644)
}

func printFileStatus(name, desc string) {
	fmt.Printf("%s %-20s # %s\n", successIcon, name, desc)
}

func printDirStatus(name string, count int, itemType string) {
	if count > 0 {
		fmt.Printf("%s %-20s # %d %s\n", successIcon, name, count, itemType)
	} else {
		fmt.Printf("%s %-20s # empty\n", successIcon, name)
	}
}

func validateCommands(dir string) (valid, total int, errs []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, nil
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		total++

		path := filepath.Join(dir, entry.Name())
		_, err := commands.ReadFrontmatterOnly(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s - %v", path, err))
		} else {
			valid++
		}
	}

	return valid, total, errs
}

func validateLanguages(dir string) (valid, total int, errs []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, nil
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		total++

		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s - failed to read: %v", path, err))
			continue
		}

		if len(content) == 0 {
			errs = append(errs, fmt.Sprintf("%s - file is empty", path))
			continue
		}

		valid++
	}

	return valid, total, errs
}

func validateTemplates(dir string) (valid, total int, errs []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, nil
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		total++

		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s - failed to read: %v", path, err))
			continue
		}

		if len(content) == 0 {
			errs = append(errs, fmt.Sprintf("%s - file is empty", path))
			continue
		}

		valid++
	}

	return valid, total, errs
}

func validateEvals(dir string) (valid, total int, errs []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		total++

		path := filepath.Join(dir, entry.Name())
		_, err := eval.ParseFile(path, eval.SourceTeam)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s - %v", path, err))
		} else {
			valid++
		}
	}

	return valid, total, errs
}
