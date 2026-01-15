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
