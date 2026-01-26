package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertToClaude(t *testing.T) {
	skill := &Skill{
		Frontmatter: Frontmatter{
			Name:         "code-review",
			Description:  "Thorough code review",
			AllowedTools: "Read Grep Glob",
			Tags:         []string{"review"},
		},
		Body:   "Review the code carefully.",
		Source: SourceTeam,
	}

	result := ConvertToClaude(skill)

	// Check frontmatter is present
	if !strings.Contains(result, "name: code-review") {
		t.Error("result should contain name field")
	}
	if !strings.Contains(result, "description: Thorough code review") {
		t.Error("result should contain description field")
	}
	if !strings.Contains(result, "allowed-tools: Read Grep Glob") {
		t.Error("result should contain allowed-tools field")
	}

	// Check staghorn header
	if !strings.Contains(result, HeaderManagedPrefix) {
		t.Error("result should contain staghorn managed header")
	}
	if !strings.Contains(result, "Source: team") {
		t.Error("result should contain source label")
	}

	// Check body is present
	if !strings.Contains(result, "Review the code carefully.") {
		t.Error("result should contain body")
	}
}

func TestConvertToClaudeWithArgs(t *testing.T) {
	skill := &Skill{
		Frontmatter: Frontmatter{
			Name:        "test-gen",
			Description: "Generate tests",
			Args: []Arg{
				{Name: "path", Default: ".", Required: true},
				{Name: "framework", Default: "jest"},
			},
		},
		Body:   "Generate tests at {{path}}.",
		Source: SourcePersonal,
	}

	result := ConvertToClaude(skill)

	// Check args hint is added
	if !strings.Contains(result, "<!-- Args:") {
		t.Error("result should contain args hint")
	}
	if !strings.Contains(result, "path (required)") {
		t.Error("result should indicate required arg")
	}
	if !strings.Contains(result, "<!-- Example:") {
		t.Error("result should contain example usage")
	}
}

func TestConvertToClaudeWithExtensions(t *testing.T) {
	falseVal := false
	skill := &Skill{
		Frontmatter: Frontmatter{
			Name:                   "security-audit",
			Description:            "Security audit",
			DisableModelInvocation: true,
			UserInvocable:          &falseVal,
			Context:                "fork",
			Agent:                  "Explore",
			Model:                  "sonnet",
			Hooks: &Hooks{
				Pre:  "./validate.sh",
				Post: "./notify.sh",
			},
		},
		Body:   "Audit the code.",
		Source: SourceTeam,
	}

	result := ConvertToClaude(skill)

	if !strings.Contains(result, "disable-model-invocation: true") {
		t.Error("result should contain disable-model-invocation")
	}
	if !strings.Contains(result, "user-invocable: false") {
		t.Error("result should contain user-invocable")
	}
	if !strings.Contains(result, "context: fork") {
		t.Error("result should contain context")
	}
	if !strings.Contains(result, "agent: Explore") {
		t.Error("result should contain agent")
	}
	if !strings.Contains(result, "model: sonnet") {
		t.Error("result should contain model")
	}
}

func TestSyncToClaude(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	claudeSkillsDir := filepath.Join(tempDir, ".claude", "skills")

	skill := &Skill{
		Frontmatter: Frontmatter{
			Name:        "test-skill",
			Description: "A test skill",
		},
		Body:    "Do something.",
		Source:  SourceTeam,
		DirPath: filepath.Join(tempDir, "source", "test-skill"),
	}

	filesWritten, err := SyncToClaude(skill, claudeSkillsDir)
	if err != nil {
		t.Fatalf("SyncToClaude() error = %v", err)
	}

	if filesWritten != 1 {
		t.Errorf("filesWritten = %d, want 1", filesWritten)
	}

	// Check skill directory was created
	destDir := filepath.Join(claudeSkillsDir, "test-skill")
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		t.Error("skill directory was not created")
	}

	// Check SKILL.md was written
	content, err := os.ReadFile(filepath.Join(destDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}
	if !strings.Contains(string(content), HeaderManagedPrefix) {
		t.Error("SKILL.md should contain staghorn header")
	}
}

func TestSyncToClaudeWithSupportingFiles(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source", "my-skill")
	claudeSkillsDir := filepath.Join(tempDir, ".claude", "skills")

	// Create source structure
	if err := os.MkdirAll(filepath.Join(sourceDir, "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	templateContent := "# Template\nReview template here."
	if err := os.WriteFile(filepath.Join(sourceDir, "templates", "review.md"), []byte(templateContent), 0644); err != nil {
		t.Fatal(err)
	}
	scriptContent := "#!/bin/bash\necho hello"
	if err := os.WriteFile(filepath.Join(sourceDir, "validate.sh"), []byte(scriptContent), 0644); err != nil {
		t.Fatal(err)
	}

	skill := &Skill{
		Frontmatter: Frontmatter{
			Name:        "my-skill",
			Description: "Skill with supporting files",
		},
		Body:    "Instructions.",
		Source:  SourceTeam,
		DirPath: sourceDir,
		SupportingFiles: map[string]string{
			"templates/review.md": filepath.Join(sourceDir, "templates", "review.md"),
			"validate.sh":         filepath.Join(sourceDir, "validate.sh"),
		},
	}

	filesWritten, err := SyncToClaude(skill, claudeSkillsDir)
	if err != nil {
		t.Fatalf("SyncToClaude() error = %v", err)
	}

	if filesWritten != 3 { // SKILL.md + 2 supporting files
		t.Errorf("filesWritten = %d, want 3", filesWritten)
	}

	// Check supporting files were copied
	destDir := filepath.Join(claudeSkillsDir, "my-skill")

	copiedTemplate, err := os.ReadFile(filepath.Join(destDir, "templates", "review.md"))
	if err != nil {
		t.Fatalf("failed to read copied template: %v", err)
	}
	if string(copiedTemplate) != templateContent {
		t.Error("template content doesn't match")
	}

	copiedScript, err := os.ReadFile(filepath.Join(destDir, "validate.sh"))
	if err != nil {
		t.Fatalf("failed to read copied script: %v", err)
	}
	if string(copiedScript) != scriptContent {
		t.Error("script content doesn't match")
	}
}

func TestSyncToClaudeCollisionDetection(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	claudeSkillsDir := filepath.Join(tempDir, ".claude", "skills")
	destDir := filepath.Join(claudeSkillsDir, "existing-skill")

	// Create existing skill NOT managed by staghorn
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	existingContent := `---
name: existing-skill
description: Not managed by staghorn
---

User's custom skill.`
	if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	skill := &Skill{
		Frontmatter: Frontmatter{
			Name:        "existing-skill",
			Description: "Trying to overwrite",
		},
		Body:   "New content.",
		Source: SourceTeam,
	}

	_, err := SyncToClaude(skill, claudeSkillsDir)
	if err == nil {
		t.Error("expected error when trying to overwrite non-staghorn skill")
	}
	if !strings.Contains(err.Error(), "not managed by staghorn") {
		t.Errorf("error message should mention 'not managed by staghorn', got: %v", err)
	}
}

func TestSyncToClaudeUpdateExisting(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	claudeSkillsDir := filepath.Join(tempDir, ".claude", "skills")
	destDir := filepath.Join(claudeSkillsDir, "my-skill")

	// Create existing skill managed by staghorn
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	existingContent := `---
name: my-skill
description: Old version
---

` + HeaderManagedPrefix + ` | Source: team | Do not edit directly -->

Old content.`
	if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	skill := &Skill{
		Frontmatter: Frontmatter{
			Name:        "my-skill",
			Description: "New version",
		},
		Body:   "New content.",
		Source: SourceTeam,
	}

	filesWritten, err := SyncToClaude(skill, claudeSkillsDir)
	if err != nil {
		t.Fatalf("SyncToClaude() error = %v", err)
	}
	if filesWritten != 1 {
		t.Errorf("filesWritten = %d, want 1", filesWritten)
	}

	// Check content was updated
	content, err := os.ReadFile(filepath.Join(destDir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "New version") {
		t.Error("content should be updated")
	}
}

func TestRemoveSkill(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	claudeSkillsDir := filepath.Join(tempDir, ".claude", "skills")
	destDir := filepath.Join(claudeSkillsDir, "my-skill")

	// Create managed skill
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `---
name: my-skill
description: Test
---

` + HeaderManagedPrefix + ` | Source: team -->

Content.`
	if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := RemoveSkill("my-skill", claudeSkillsDir)
	if err != nil {
		t.Fatalf("RemoveSkill() error = %v", err)
	}

	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		t.Error("skill directory should be removed")
	}
}

func TestRemoveSkillNonManaged(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	claudeSkillsDir := filepath.Join(tempDir, ".claude", "skills")
	destDir := filepath.Join(claudeSkillsDir, "user-skill")

	// Create non-managed skill
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `---
name: user-skill
description: User's own skill
---

Not managed by staghorn.`
	if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := RemoveSkill("user-skill", claudeSkillsDir)
	if err == nil {
		t.Error("expected error when trying to remove non-managed skill")
	}

	// Directory should still exist
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		t.Error("non-managed skill should not be removed")
	}
}

func TestRemoveSkillNonexistent(t *testing.T) {
	tempDir := t.TempDir()
	claudeSkillsDir := filepath.Join(tempDir, ".claude", "skills")

	// Should not error for nonexistent skill
	err := RemoveSkill("nonexistent", claudeSkillsDir)
	if err != nil {
		t.Errorf("RemoveSkill() for nonexistent skill should not error, got: %v", err)
	}
}

func TestListManagedSkills(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	claudeSkillsDir := filepath.Join(tempDir, ".claude", "skills")

	// Create managed skill
	managedDir := filepath.Join(claudeSkillsDir, "managed-skill")
	if err := os.MkdirAll(managedDir, 0755); err != nil {
		t.Fatal(err)
	}
	managedContent := `---
name: managed-skill
description: Managed
---

` + HeaderManagedPrefix + ` -->

Content.`
	if err := os.WriteFile(filepath.Join(managedDir, "SKILL.md"), []byte(managedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create non-managed skill
	userDir := filepath.Join(claudeSkillsDir, "user-skill")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		t.Fatal(err)
	}
	userContent := `---
name: user-skill
description: User
---

Not managed.`
	if err := os.WriteFile(filepath.Join(userDir, "SKILL.md"), []byte(userContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create regular file (should be ignored)
	if err := os.WriteFile(filepath.Join(claudeSkillsDir, "readme.txt"), []byte("ignore"), 0644); err != nil {
		t.Fatal(err)
	}

	names, err := ListManagedSkills(claudeSkillsDir)
	if err != nil {
		t.Fatalf("ListManagedSkills() error = %v", err)
	}

	if len(names) != 1 {
		t.Errorf("ListManagedSkills() = %d, want 1", len(names))
	}
	if len(names) > 0 && names[0] != "managed-skill" {
		t.Errorf("ListManagedSkills()[0] = %q, want %q", names[0], "managed-skill")
	}
}

func TestListManagedSkillsNonexistent(t *testing.T) {
	names, err := ListManagedSkills("/nonexistent/path")
	if err != nil {
		t.Errorf("expected nil error for nonexistent directory, got %v", err)
	}
	if names != nil {
		t.Errorf("expected nil names for nonexistent directory, got %v", names)
	}
}
