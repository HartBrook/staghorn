package starter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemplateNames(t *testing.T) {
	names := TemplateNames()

	if len(names) == 0 {
		t.Error("expected at least one starter template")
	}

	// Check for expected templates
	expected := []string{"backend-service", "frontend-app", "cli-tool"}
	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected template %q not found in starter templates", exp)
		}
	}
}

func TestGetTemplate(t *testing.T) {
	content, err := GetTemplate("backend-service")
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty content for backend-service template")
	}

	// Check for expected content
	if !contains(string(content), "Backend") {
		t.Error("expected backend-service template to mention 'Backend'")
	}
}

func TestGetTemplate_NotFound(t *testing.T) {
	_, err := GetTemplate("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
}

func TestBootstrapTemplates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// First bootstrap
	count, err := BootstrapTemplates(tmpDir)
	if err != nil {
		t.Fatalf("BootstrapTemplates failed: %v", err)
	}

	if count == 0 {
		t.Error("expected at least one template to be copied")
	}

	// Verify files exist
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}

	if len(entries) != count {
		t.Errorf("expected %d files, got %d", count, len(entries))
	}

	// Check backend-service.md exists and has content
	servicePath := filepath.Join(tmpDir, "backend-service.md")
	content, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("failed to read backend-service.md: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty backend-service.md")
	}

	// Second bootstrap should skip existing files
	count2, err := BootstrapTemplates(tmpDir)
	if err != nil {
		t.Fatalf("second BootstrapTemplates failed: %v", err)
	}
	if count2 != 0 {
		t.Errorf("expected 0 files copied on second run, got %d", count2)
	}
}
