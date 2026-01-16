package starter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLanguageNames(t *testing.T) {
	names := LanguageNames()

	if len(names) == 0 {
		t.Error("expected at least one starter language")
	}

	// Check for expected languages
	expected := []string{"python", "go", "typescript", "rust", "java", "ruby"}
	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected language %q not found in starter languages", exp)
		}
	}
}

func TestGetLanguageConfig(t *testing.T) {
	content, err := GetLanguageConfig("python")
	if err != nil {
		t.Fatalf("GetLanguageConfig failed: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty content for python config")
	}

	// Check for expected content
	if !contains(string(content), "Python") {
		t.Error("expected python config to mention 'Python'")
	}
}

func TestGetLanguageConfig_NotFound(t *testing.T) {
	_, err := GetLanguageConfig("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent language")
	}
}

func TestBootstrapLanguages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// First bootstrap
	count, err := BootstrapLanguages(tmpDir)
	if err != nil {
		t.Fatalf("BootstrapLanguages failed: %v", err)
	}

	if count == 0 {
		t.Error("expected at least one language config to be copied")
	}

	// Verify files exist
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}

	if len(entries) != count {
		t.Errorf("expected %d files, got %d", count, len(entries))
	}

	// Check python.md exists and has content
	pythonPath := filepath.Join(tmpDir, "python.md")
	content, err := os.ReadFile(pythonPath)
	if err != nil {
		t.Fatalf("failed to read python.md: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty python.md")
	}

	// Second bootstrap should skip existing files
	count2, err := BootstrapLanguages(tmpDir)
	if err != nil {
		t.Fatalf("second BootstrapLanguages failed: %v", err)
	}
	if count2 != 0 {
		t.Errorf("expected 0 files copied on second run, got %d", count2)
	}
}

func TestBootstrapLanguagesSelective(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Install only specific languages
	selected := []string{"python", "go"}
	count, installed, err := BootstrapLanguagesSelective(tmpDir, selected)
	if err != nil {
		t.Fatalf("BootstrapLanguagesSelective failed: %v", err)
	}

	if count != len(selected) {
		t.Errorf("expected %d languages, got %d", len(selected), count)
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
	allNames := LanguageNames()
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

func TestBootstrapLanguagesSelective_EmptyList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Install with empty list should install nothing
	count, installed, err := BootstrapLanguagesSelective(tmpDir, nil)
	if err != nil {
		t.Fatalf("BootstrapLanguagesSelective failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 languages with empty list, got %d", count)
	}

	if len(installed) != 0 {
		t.Errorf("expected empty installed list, got %d items", len(installed))
	}
}

func TestBootstrapLanguagesSelective_ExistingFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Pre-create python.md
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	existingContent := "# My custom python config"
	if err := os.WriteFile(filepath.Join(tmpDir, "python.md"), []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing python.md: %v", err)
	}

	// Try to install python and go
	selected := []string{"python", "go"}
	count, installed, err := BootstrapLanguagesSelective(tmpDir, selected)
	if err != nil {
		t.Fatalf("BootstrapLanguagesSelective failed: %v", err)
	}

	// Should only install go (python already exists)
	if count != 1 {
		t.Errorf("expected 1 language installed, got %d", count)
	}

	// Verify python.md was NOT overwritten
	content, _ := os.ReadFile(filepath.Join(tmpDir, "python.md"))
	if string(content) != existingContent {
		t.Error("existing python.md should not be overwritten")
	}

	// Verify go.md was installed
	if _, err := os.Stat(filepath.Join(tmpDir, "go.md")); err != nil {
		t.Error("go.md should have been installed")
	}

	// Verify installed list only contains go
	if len(installed) != 1 || installed[0] != "go" {
		t.Errorf("expected installed list to be [go], got %v", installed)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
