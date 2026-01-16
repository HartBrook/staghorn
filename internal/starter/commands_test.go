package starter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommandNames(t *testing.T) {
	names := CommandNames()

	if len(names) == 0 {
		t.Error("expected at least one starter command")
	}

	// Check for expected commands
	expected := []string{"code-review", "debug", "refactor", "test-gen", "security-audit"}
	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected command %q not found in starter commands", exp)
		}
	}
}

func TestGetCommand(t *testing.T) {
	content, err := GetCommand("code-review")
	if err != nil {
		t.Fatalf("GetCommand failed: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty content for code-review command")
	}

	// Check for expected content (frontmatter)
	if !contains(string(content), "name:") {
		t.Error("expected code-review command to have frontmatter with 'name:'")
	}
}

func TestGetCommand_NotFound(t *testing.T) {
	_, err := GetCommand("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

func TestBootstrapCommands(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// First bootstrap
	count, err := BootstrapCommands(tmpDir)
	if err != nil {
		t.Fatalf("BootstrapCommands failed: %v", err)
	}

	if count == 0 {
		t.Error("expected at least one command to be copied")
	}

	// Verify files exist
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}

	if len(entries) != count {
		t.Errorf("expected %d files, got %d", count, len(entries))
	}

	// Check code-review.md exists and has content
	reviewPath := filepath.Join(tmpDir, "code-review.md")
	content, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("failed to read code-review.md: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty code-review.md")
	}

	// Second bootstrap should skip existing files
	count2, err := BootstrapCommands(tmpDir)
	if err != nil {
		t.Fatalf("second BootstrapCommands failed: %v", err)
	}
	if count2 != 0 {
		t.Errorf("expected 0 files copied on second run, got %d", count2)
	}
}

func TestBootstrapCommandsWithSkip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Get all command names to calculate expected count
	allNames := CommandNames()
	skipList := []string{"code-review", "debug"}

	// Bootstrap with skip list
	count, installed, err := BootstrapCommandsWithSkip(tmpDir, skipList)
	if err != nil {
		t.Fatalf("BootstrapCommandsWithSkip failed: %v", err)
	}

	expectedCount := len(allNames) - len(skipList)
	if count != expectedCount {
		t.Errorf("expected %d commands, got %d", expectedCount, count)
	}

	// Verify skipped files don't exist
	for _, skip := range skipList {
		path := filepath.Join(tmpDir, skip+".md")
		if _, err := os.Stat(path); err == nil {
			t.Errorf("expected %s to be skipped, but it exists", skip)
		}
	}

	// Verify installed list doesn't contain skipped commands
	for _, skip := range skipList {
		for _, name := range installed {
			if name == skip {
				t.Errorf("installed list should not contain skipped command %s", skip)
			}
		}
	}

	// Verify installed list has correct length
	if len(installed) != count {
		t.Errorf("installed list length %d doesn't match count %d", len(installed), count)
	}
}

func TestBootstrapCommandsWithSkip_EmptySkipList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Bootstrap with empty skip list should install all
	count, installed, err := BootstrapCommandsWithSkip(tmpDir, nil)
	if err != nil {
		t.Fatalf("BootstrapCommandsWithSkip failed: %v", err)
	}

	allNames := CommandNames()
	if count != len(allNames) {
		t.Errorf("expected %d commands with empty skip list, got %d", len(allNames), count)
	}

	if len(installed) != count {
		t.Errorf("installed list length %d doesn't match count %d", len(installed), count)
	}
}

func TestLoadStarterCommands(t *testing.T) {
	cmds, err := LoadStarterCommands()
	if err != nil {
		t.Fatalf("LoadStarterCommands failed: %v", err)
	}

	if len(cmds) == 0 {
		t.Error("expected at least one starter command")
	}

	// Verify all commands have required fields
	for _, cmd := range cmds {
		if cmd.Name == "" {
			t.Error("command has empty name")
		}
		if cmd.Body == "" {
			t.Errorf("command %s has empty body", cmd.Name)
		}
	}
}

func TestBootstrapCommandsSelective(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Install only specific commands
	selected := []string{"code-review", "debug"}
	count, installed, err := BootstrapCommandsSelective(tmpDir, selected)
	if err != nil {
		t.Fatalf("BootstrapCommandsSelective failed: %v", err)
	}

	if count != len(selected) {
		t.Errorf("expected %d commands, got %d", len(selected), count)
	}

	if len(installed) != count {
		t.Errorf("installed list length %d doesn't match count %d", len(installed), count)
	}

	// Verify only selected files exist
	for _, name := range selected {
		path := filepath.Join(tmpDir, name+".md")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to exist", name)
		}
	}

	// Verify unselected files don't exist
	allNames := CommandNames()
	for _, name := range allNames {
		isSelected := false
		for _, sel := range selected {
			if name == sel {
				isSelected = true
				break
			}
		}
		if !isSelected {
			path := filepath.Join(tmpDir, name+".md")
			if _, err := os.Stat(path); err == nil {
				t.Errorf("expected %s to not exist (not selected)", name)
			}
		}
	}
}

func TestBootstrapCommandsSelective_EmptyList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Install with empty list should install nothing
	count, installed, err := BootstrapCommandsSelective(tmpDir, nil)
	if err != nil {
		t.Fatalf("BootstrapCommandsSelective failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 commands with empty list, got %d", count)
	}

	if len(installed) != 0 {
		t.Errorf("expected empty installed list, got %d items", len(installed))
	}
}
