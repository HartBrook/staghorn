package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/HartBrook/staghorn/internal/cache"
	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/language"
	"github.com/spf13/cobra"
)

// NewEditCmd creates the edit command.
func NewEditCmd() *cobra.Command {
	var noApply bool
	var langFlag string

	cmd := &cobra.Command{
		Use:   "edit [layer]",
		Short: "Edit config in $EDITOR (auto-applies on save)",
		Long: `Opens a config file in your editor and automatically applies changes on save.

Layers:
  personal  Edit ~/.config/staghorn/personal.md (default)
  project   Edit .staghorn/project.md

Use --language to edit language-specific personal preferences.

Your changes are automatically applied after the editor closes.
Use --no-apply to edit without applying.`,
		Example: `  staghorn edit              # Edit personal config
  staghorn edit project      # Edit project config
  staghorn edit -l python    # Edit personal Python preferences
  staghorn edit --language go # Edit personal Go preferences
  staghorn edit --no-apply   # Edit without auto-applying`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// If --language flag is set, edit language config
			if langFlag != "" {
				return editLanguage(langFlag, noApply)
			}

			layer := "personal"
			if len(args) > 0 {
				layer = args[0]
			}
			return runEdit(layer, noApply)
		},
	}

	cmd.Flags().BoolVar(&noApply, "no-apply", false, "Don't apply changes after editing")
	cmd.Flags().StringVarP(&langFlag, "language", "l", "", "Edit personal config for a specific language (e.g., python, go, typescript)")

	return cmd
}

func runEdit(layer string, noApply bool) error {
	switch layer {
	case "personal", "":
		return editPersonal(noApply)
	case "project":
		return editProject(noApply)
	case "team":
		// Check if we're in a source repo - if so, allow editing
		projectRoot := findProjectRoot()
		if config.IsSourceRepo(projectRoot) {
			return editTeam(projectRoot, noApply)
		}
		return fmt.Errorf("team config is read-only\nUse 'staghorn info --layer team' to view it")
	default:
		return fmt.Errorf("unknown layer '%s'\nValid layers: personal, project, team (in source repo)", layer)
	}
}

func editLanguage(langID string, noApply bool) error {
	paths := config.NewPaths()

	// Validate language ID
	lang := language.GetLanguage(langID)
	displayName := langID
	if lang != nil {
		displayName = lang.DisplayName
	}

	// Ensure the languages directory exists
	if err := os.MkdirAll(paths.PersonalLanguages, 0755); err != nil {
		return fmt.Errorf("failed to create languages directory: %w", err)
	}

	langFile := filepath.Join(paths.PersonalLanguages, langID+".md")

	// Create with template if doesn't exist
	if _, err := os.Stat(langFile); os.IsNotExist(err) {
		template := fmt.Sprintf(`<!-- [staghorn] Personal %s preferences - customize to your workflow -->

## My %s Preferences

`, displayName, displayName)
		if err := os.WriteFile(langFile, []byte(template), 0644); err != nil {
			return fmt.Errorf("failed to create language config: %w", err)
		}
	}

	// Open editor
	if err := openEditor(langFile); err != nil {
		return err
	}

	fmt.Println()
	printSuccess("Personal %s config saved", displayName)

	// Auto-apply unless --no-apply
	if noApply {
		fmt.Printf("  %s Run 'staghorn sync' to apply changes\n", dim("Tip:"))
		return nil
	}

	// Apply changes
	cfg, err := config.Load()
	if err != nil {
		printWarning("Could not auto-apply: %v", err)
		fmt.Printf("  %s Run 'staghorn sync' to apply changes\n", dim("Tip:"))
		return nil
	}

	owner, repo, err := cfg.DefaultOwnerRepo()
	if err != nil {
		printWarning("Could not auto-apply: %v", err)
		fmt.Printf("  %s Run 'staghorn sync' to apply changes\n", dim("Tip:"))
		return nil
	}

	// Check if we have cached team config
	c := cache.New(paths)
	if !c.Exists(owner, repo) {
		printWarning("No cached team config, run 'staghorn sync' first")
		return nil
	}

	fmt.Println()
	if err := applyConfig(cfg, paths, owner, repo); err != nil {
		return err
	}

	return nil
}

func editPersonal(noApply bool) error {
	paths := config.NewPaths()

	// Ensure the personal config file exists
	if _, err := os.Stat(paths.PersonalMD); os.IsNotExist(err) {
		// Create with minimal template - users add their own sections
		template := `<!-- [staghorn] Personal additions - add ## sections below -->

`
		if err := os.MkdirAll(paths.ConfigDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		if err := os.WriteFile(paths.PersonalMD, []byte(template), 0644); err != nil {
			return fmt.Errorf("failed to create personal config: %w", err)
		}
	}

	// Open editor
	if err := openEditor(paths.PersonalMD); err != nil {
		return err
	}

	fmt.Println()
	printSuccess("Personal config saved")

	// Auto-apply unless --no-apply
	if noApply {
		fmt.Printf("  %s Run 'staghorn sync' to apply changes\n", dim("Tip:"))
		return nil
	}

	// Apply changes
	cfg, err := config.Load()
	if err != nil {
		printWarning("Could not auto-apply: %v", err)
		fmt.Printf("  %s Run 'staghorn sync' to apply changes\n", dim("Tip:"))
		return nil
	}

	owner, repo, err := cfg.DefaultOwnerRepo()
	if err != nil {
		printWarning("Could not auto-apply: %v", err)
		fmt.Printf("  %s Run 'staghorn sync' to apply changes\n", dim("Tip:"))
		return nil
	}

	// Check if we have cached team config
	c := cache.New(paths)
	if !c.Exists(owner, repo) {
		printWarning("No cached team config, run 'staghorn sync' first")
		return nil
	}

	fmt.Println()
	if err := applyConfig(cfg, paths, owner, repo); err != nil {
		return err
	}

	return nil
}

func editProject(noApply bool) error {
	projectRoot := findProjectRoot()
	projectPaths := config.NewProjectPaths(projectRoot)

	// Check if initialized
	if _, err := os.Stat(projectPaths.SourceMD); os.IsNotExist(err) {
		return fmt.Errorf("project not initialized\nRun 'staghorn project init' first")
	}

	// Open editor
	if err := openEditor(projectPaths.SourceMD); err != nil {
		return err
	}

	fmt.Println()
	printSuccess("Project config saved")

	// Auto-apply unless --no-apply
	if noApply {
		fmt.Printf("  %s Run 'staghorn project info' to preview changes\n", dim("Tip:"))
		return nil
	}

	// Apply changes
	if err := generateProjectOutput(projectPaths); err != nil {
		return err
	}

	printSuccess("Applied to %s", relativePath(projectPaths.OutputMD))

	return nil
}

func editTeam(projectRoot string, noApply bool) error {
	sourcePaths := config.NewSourceRepoPaths(projectRoot)

	// Check if CLAUDE.md exists
	if _, err := os.Stat(sourcePaths.ClaudeMD); os.IsNotExist(err) {
		return fmt.Errorf("CLAUDE.md not found\nRun 'staghorn team init' first")
	}

	// Open editor
	if err := openEditor(sourcePaths.ClaudeMD); err != nil {
		return err
	}

	fmt.Println()
	printSuccess("Team config saved")

	// For team layer in source repo, there's no "apply" step needed
	// The file is edited directly
	if !noApply {
		fmt.Printf("  %s Changes are saved directly to %s\n", dim("Note:"), relativePath(sourcePaths.ClaudeMD))
	}

	return nil
}

func openEditor(filepath string) error {
	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors
		for _, e := range []string{"code", "vim", "nano", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found, set $EDITOR environment variable")
	}

	// Open editor
	editorCmd := exec.Command(editor, filepath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	return nil
}
