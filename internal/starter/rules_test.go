package starter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuleNames(t *testing.T) {
	names := RuleNames()

	if len(names) == 0 {
		t.Fatal("RuleNames returned empty list")
	}

	// Check for expected rules
	expected := []string{
		"security.md",
		"testing.md",
		"error-handling.md",
		"api/rest.md",
		"frontend/react.md",
	}

	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	for _, exp := range expected {
		if !nameSet[exp] {
			t.Errorf("missing expected rule: %s", exp)
		}
	}
}

func TestGetRule(t *testing.T) {
	content, err := GetRule("security.md")
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}

	if len(content) == 0 {
		t.Error("GetRule returned empty content")
	}

	if !strings.Contains(string(content), "Security") {
		t.Error("security.md should contain 'Security'")
	}
}

func TestGetRule_Subdirectory(t *testing.T) {
	content, err := GetRule("api/rest.md")
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}

	if len(content) == 0 {
		t.Error("GetRule returned empty content")
	}

	// Should have paths frontmatter
	if !strings.Contains(string(content), "paths:") {
		t.Error("api/rest.md should have paths frontmatter")
	}
}

func TestGetRule_NotFound(t *testing.T) {
	_, err := GetRule("nonexistent.md")
	if err == nil {
		t.Error("expected error for nonexistent rule")
	}
}

func TestBootstrapRules(t *testing.T) {
	tmpDir := t.TempDir()

	count, err := BootstrapRules(tmpDir)
	if err != nil {
		t.Fatalf("BootstrapRules failed: %v", err)
	}

	if count == 0 {
		t.Error("BootstrapRules copied 0 rules")
	}

	// Check that security.md was created
	securityPath := filepath.Join(tmpDir, "security.md")
	if _, err := os.Stat(securityPath); os.IsNotExist(err) {
		t.Error("security.md was not created")
	}

	// Check that api/rest.md was created (subdirectory)
	restPath := filepath.Join(tmpDir, "api", "rest.md")
	if _, err := os.Stat(restPath); os.IsNotExist(err) {
		t.Error("api/rest.md was not created")
	}
}

func TestBootstrapRules_SkipsExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an existing file
	existingContent := "# My Custom Security"
	if err := os.WriteFile(filepath.Join(tmpDir, "security.md"), []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := BootstrapRules(tmpDir)
	if err != nil {
		t.Fatalf("BootstrapRules failed: %v", err)
	}

	// Verify existing file was not overwritten
	content, err := os.ReadFile(filepath.Join(tmpDir, "security.md"))
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != existingContent {
		t.Error("existing file was overwritten")
	}
}

func TestBootstrapRulesWithSkip(t *testing.T) {
	tmpDir := t.TempDir()

	count, installed, err := BootstrapRulesWithSkip(tmpDir, []string{"security.md", "api/rest.md"})
	if err != nil {
		t.Fatalf("BootstrapRulesWithSkip failed: %v", err)
	}

	// Check that skipped rules were not installed
	for _, name := range installed {
		if name == "security.md" || name == "api/rest.md" {
			t.Errorf("skipped rule %s was installed", name)
		}
	}

	// Check that security.md was NOT created
	securityPath := filepath.Join(tmpDir, "security.md")
	if _, err := os.Stat(securityPath); err == nil {
		t.Error("security.md should have been skipped")
	}

	// But other rules should exist
	if count == 0 {
		t.Error("no rules were installed")
	}
}

func TestLoadStarterRules(t *testing.T) {
	rules, err := LoadStarterRules()
	if err != nil {
		t.Fatalf("LoadStarterRules failed: %v", err)
	}

	if len(rules) == 0 {
		t.Fatal("LoadStarterRules returned empty list")
	}

	// Check that rules are properly parsed
	ruleMap := make(map[string]bool)
	for _, r := range rules {
		ruleMap[r.RelPath] = true

		// All rules should have starter source
		if r.Source != "starter" {
			t.Errorf("rule %s has source %q, want 'starter'", r.RelPath, r.Source)
		}
	}

	// Check expected rules exist
	expected := []string{"security.md", "testing.md", "api/rest.md", "frontend/react.md"}
	for _, exp := range expected {
		if !ruleMap[exp] {
			t.Errorf("missing expected rule: %s", exp)
		}
	}
}

func TestLoadStarterRules_PathScoped(t *testing.T) {
	rules, err := LoadStarterRules()
	if err != nil {
		t.Fatal(err)
	}

	// Find api/rest.md - it should have paths
	var restRule *struct {
		RelPath string
		Paths   []string
	}

	for _, r := range rules {
		if r.RelPath == "api/rest.md" {
			restRule = &struct {
				RelPath string
				Paths   []string
			}{r.RelPath, r.Paths}
			break
		}
	}

	if restRule == nil {
		t.Fatal("api/rest.md not found")
	}

	if len(restRule.Paths) == 0 {
		t.Error("api/rest.md should have paths for path-scoping")
	}
}
