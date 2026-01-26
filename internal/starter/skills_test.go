package starter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillNames(t *testing.T) {
	names := SkillNames()
	if len(names) == 0 {
		t.Error("expected at least one starter skill")
	}

	// Check for expected starter skills
	expected := []string{"code-review", "security-audit", "test-gen"}
	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected skill %q not found in starter skills", exp)
		}
	}
}

func TestGetSkill(t *testing.T) {
	content, err := GetSkill("code-review")
	if err != nil {
		t.Fatalf("GetSkill(code-review) error: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty content")
	}

	// Check it contains expected frontmatter
	if !strings.Contains(string(content), "name: code-review") {
		t.Error("expected skill to have name field")
	}
	if !strings.Contains(string(content), "allowed-tools:") {
		t.Error("expected skill to have allowed-tools field")
	}
}

func TestGetSkillNotFound(t *testing.T) {
	_, err := GetSkill("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestBootstrapSkills(t *testing.T) {
	tempDir := t.TempDir()

	count, err := BootstrapSkills(tempDir)
	if err != nil {
		t.Fatalf("BootstrapSkills error: %v", err)
	}

	if count == 0 {
		t.Error("expected at least one skill to be installed")
	}

	// Check that skill directories were created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != count {
		t.Errorf("expected %d directories, got %d", count, len(entries))
	}

	// Check that each directory has a SKILL.md
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillMD := filepath.Join(tempDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillMD); os.IsNotExist(err) {
			t.Errorf("expected SKILL.md in %s", entry.Name())
		}
	}
}

func TestBootstrapSkillsSkipExisting(t *testing.T) {
	tempDir := t.TempDir()

	// Create a skill that already exists
	existingSkill := filepath.Join(tempDir, "code-review")
	if err := os.MkdirAll(existingSkill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existingSkill, "SKILL.md"), []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	count, err := BootstrapSkills(tempDir)
	if err != nil {
		t.Fatalf("BootstrapSkills error: %v", err)
	}

	// Should skip the existing one
	totalSkills := len(SkillNames())
	expectedCount := totalSkills - 1
	if count != expectedCount {
		t.Errorf("expected %d skills copied (skipping existing), got %d", expectedCount, count)
	}

	// Existing skill should not be overwritten
	content, _ := os.ReadFile(filepath.Join(existingSkill, "SKILL.md"))
	if string(content) != "existing" {
		t.Error("existing skill was overwritten")
	}
}

func TestBootstrapSkillsSelective(t *testing.T) {
	tempDir := t.TempDir()

	count, installed, err := BootstrapSkillsSelective(tempDir, []string{"code-review"})
	if err != nil {
		t.Fatalf("BootstrapSkillsSelective error: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 skill, got %d", count)
	}

	if len(installed) != 1 || installed[0] != "code-review" {
		t.Errorf("expected [code-review], got %v", installed)
	}

	// Check only code-review was installed
	entries, _ := os.ReadDir(tempDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 directory, got %d", len(entries))
	}
}

func TestBootstrapSkillsWithSkip(t *testing.T) {
	tempDir := t.TempDir()

	count, installed, err := BootstrapSkillsWithSkip(tempDir, []string{"code-review"})
	if err != nil {
		t.Fatalf("BootstrapSkillsWithSkip error: %v", err)
	}

	// Should install all except code-review
	totalSkills := len(SkillNames())
	if count != totalSkills-1 {
		t.Errorf("expected %d skills (skipping 1), got %d", totalSkills-1, count)
	}

	// code-review should not be in installed list
	for _, name := range installed {
		if name == "code-review" {
			t.Error("code-review should have been skipped")
		}
	}
}

func TestLoadStarterSkills(t *testing.T) {
	skills, err := LoadStarterSkills()
	if err != nil {
		t.Fatalf("LoadStarterSkills error: %v", err)
	}

	if len(skills) == 0 {
		t.Error("expected at least one starter skill")
	}

	// Check skills are valid
	for _, skill := range skills {
		if skill.Name == "" {
			t.Error("skill has empty name")
		}
		if skill.Description == "" {
			t.Error("skill has empty description")
		}
	}
}

func TestListSkills(t *testing.T) {
	entries, err := ListSkills()
	if err != nil {
		t.Fatalf("ListSkills error: %v", err)
	}

	if len(entries) == 0 {
		t.Error("expected at least one skill entry")
	}
}
