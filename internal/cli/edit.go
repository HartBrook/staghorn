package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/HartBrook/staghorn/internal/cache"
	"github.com/HartBrook/staghorn/internal/config"
	"github.com/spf13/cobra"
)

// NewEditCmd creates the edit command.
func NewEditCmd() *cobra.Command {
	var noApply bool

	cmd := &cobra.Command{
		Use:   "edit [layer]",
		Short: "Edit config in $EDITOR (auto-applies on save)",
		Long: `Opens a config file in your editor and automatically applies changes on save.

Layers:
  personal  Edit ~/.config/staghorn/personal.md (default)
  project   Edit .staghorn/project.md

Your changes are automatically applied after the editor closes.
Use --no-apply to edit without applying.`,
		Example: `  staghorn edit              # Edit personal config
  staghorn edit project      # Edit project config
  staghorn edit --no-apply   # Edit without auto-applying`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			layer := "personal"
			if len(args) > 0 {
				layer = args[0]
			}
			return runEdit(layer, noApply)
		},
	}

	cmd.Flags().BoolVar(&noApply, "no-apply", false, "Don't apply changes after editing")

	return cmd
}

func runEdit(layer string, noApply bool) error {
	switch layer {
	case "personal", "":
		return editPersonal(noApply)
	case "project":
		return editProject(noApply)
	case "team":
		return fmt.Errorf("team config is read-only\nUse 'staghorn info --layer team' to view it")
	default:
		return fmt.Errorf("unknown layer '%s'\nValid layers: personal, project", layer)
	}
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

	owner, repo, err := cfg.Team.ParseRepo()
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
