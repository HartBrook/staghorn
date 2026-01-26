package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantName    string
		wantDesc    string
		wantTags    []string
		wantTools   string
		wantBody    string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid skill with all fields",
			content: `---
name: code-review
description: Thorough code review
tags: [review, quality]
allowed-tools: Read Grep Glob
context: normal
---

# Code Review

Review the code carefully.`,
			wantName:  "code-review",
			wantDesc:  "Thorough code review",
			wantTags:  []string{"review", "quality"},
			wantTools: "Read Grep Glob",
			wantBody:  "# Code Review\n\nReview the code carefully.",
		},
		{
			name: "minimal skill",
			content: `---
name: simple
description: A simple skill
---

Do something.`,
			wantName: "simple",
			wantDesc: "A simple skill",
			wantBody: "Do something.",
		},
		{
			name: "skill with claude code extensions",
			content: `---
name: security-audit
description: Scan for vulnerabilities
disable-model-invocation: true
user-invocable: false
context: fork
agent: Explore
---

Audit the code.`,
			wantName: "security-audit",
			wantDesc: "Scan for vulnerabilities",
			wantBody: "Audit the code.",
		},
		{
			name:        "missing frontmatter start",
			content:     "name: test\n---\nBody",
			wantErr:     true,
			errContains: "must start with YAML frontmatter",
		},
		{
			name:        "unterminated frontmatter",
			content:     "---\nname: test\nBody without closing",
			wantErr:     true,
			errContains: "unterminated frontmatter",
		},
		{
			name: "missing name field",
			content: `---
description: No name provided
---

Body here.`,
			wantErr:     true,
			errContains: "must have a 'name' field",
		},
		{
			name: "missing description field",
			content: `---
name: no-desc
---

Body here.`,
			wantErr:     true,
			errContains: "must have a 'description' field",
		},
		{
			name: "invalid name - uppercase",
			content: `---
name: CodeReview
description: Has uppercase
---

Body.`,
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name: "invalid name - starts with hyphen",
			content: `---
name: -review
description: Starts with hyphen
---

Body.`,
			wantErr:     true,
			errContains: "cannot start or end with hyphen",
		},
		{
			name: "invalid name - consecutive hyphens",
			content: `---
name: code--review
description: Has consecutive hyphens
---

Body.`,
			wantErr:     true,
			errContains: "cannot contain consecutive hyphens",
		},
		{
			name: "skill with hooks",
			content: `---
name: deploy
description: Deploy the application
hooks:
  pre: ./scripts/validate.sh
  post: ./scripts/notify.sh
---

Deploy steps here.`,
			wantName: "deploy",
			wantDesc: "Deploy the application",
			wantBody: "Deploy steps here.",
		},
		{
			name: "skill with args",
			content: `---
name: test-gen
description: Generate tests
args:
  - name: path
    description: Target path
    default: "."
  - name: framework
    options: [jest, vitest, pytest]
    default: jest
---

Generate tests at {{path}}.`,
			wantName: "test-gen",
			wantDesc: "Generate tests",
			wantBody: "Generate tests at {{path}}.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill, err := Parse(tt.content, SourceTeam, "")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q doesn't contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if skill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", skill.Name, tt.wantName)
			}

			if tt.wantDesc != "" && skill.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", skill.Description, tt.wantDesc)
			}

			if tt.wantTags != nil && len(skill.Tags) != len(tt.wantTags) {
				t.Errorf("Tags = %v, want %v", skill.Tags, tt.wantTags)
			}

			if tt.wantTools != "" && skill.AllowedTools != tt.wantTools {
				t.Errorf("AllowedTools = %q, want %q", skill.AllowedTools, tt.wantTools)
			}

			if tt.wantBody != "" && skill.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", skill.Body, tt.wantBody)
			}
		})
	}
}

func TestValidateSkillName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "code-review", false},
		{"valid with numbers", "test-123", false},
		{"valid short", "a", false},
		{"valid long", "this-is-a-valid-skill-name-with-many-parts", false},
		{"empty", "", true},
		{"too long", "this-skill-name-is-way-too-long-and-exceeds-the-sixty-four-character-limit-set-by-standard", true},
		{"uppercase", "CodeReview", true},
		{"spaces", "code review", true},
		{"underscores", "code_review", true},
		{"starts with hyphen", "-review", true},
		{"ends with hyphen", "review-", true},
		{"consecutive hyphens", "code--review", true},
		{"special chars", "code@review", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSkillName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSkillName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestAllowedToolsList(t *testing.T) {
	tests := []struct {
		name  string
		tools string
		want  []string
	}{
		{"empty", "", nil},
		{"single tool", "Read", []string{"Read"}},
		{"multiple tools", "Read Grep Glob", []string{"Read", "Grep", "Glob"}},
		{"extra spaces", "Read   Grep  Glob", []string{"Read", "Grep", "Glob"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &Skill{Frontmatter: Frontmatter{AllowedTools: tt.tools}}
			got := skill.AllowedToolsList()

			if len(got) != len(tt.want) {
				t.Errorf("AllowedToolsList() = %v, want %v", got, tt.want)
				return
			}

			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("AllowedToolsList()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestIsUserInvocable(t *testing.T) {
	tests := []struct {
		name  string
		value *bool
		want  bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &Skill{Frontmatter: Frontmatter{UserInvocable: tt.value}}
			if got := skill.IsUserInvocable(); got != tt.want {
				t.Errorf("IsUserInvocable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Add team skill
	teamSkill := &Skill{
		Frontmatter: Frontmatter{Name: "audit", Description: "Team audit"},
		Source:      SourceTeam,
	}
	registry.Add(teamSkill)

	// Add personal override
	personalSkill := &Skill{
		Frontmatter: Frontmatter{Name: "audit", Description: "Personal audit"},
		Source:      SourcePersonal,
	}
	registry.Add(personalSkill)

	// Add project skill
	projectSkill := &Skill{
		Frontmatter: Frontmatter{Name: "project-only", Description: "Project specific"},
		Source:      SourceProject,
	}
	registry.Add(projectSkill)

	// Test Get returns highest precedence
	got := registry.Get("audit")
	if got.Description != "Personal audit" {
		t.Errorf("Get(audit) returned %q, want personal override", got.Description)
	}

	// Test Count
	if registry.Count() != 2 {
		t.Errorf("Count() = %d, want 2", registry.Count())
	}

	// Test GetAllVersions
	versions := registry.GetAllVersions("audit")
	if len(versions) != 2 {
		t.Errorf("GetAllVersions(audit) = %d, want 2", len(versions))
	}

	// Test BySource
	teamSkills := registry.BySource(SourceTeam)
	if len(teamSkills) != 1 {
		t.Errorf("BySource(team) = %d, want 1", len(teamSkills))
	}
}

func TestRegistryPrecedence(t *testing.T) {
	// Test that project > personal > team > starter
	tests := []struct {
		name       string
		sources    []Source
		wantSource Source
	}{
		{"team only", []Source{SourceTeam}, SourceTeam},
		{"personal overrides team", []Source{SourceTeam, SourcePersonal}, SourcePersonal},
		{"project overrides personal", []Source{SourcePersonal, SourceProject}, SourceProject},
		{"project overrides all", []Source{SourceStarter, SourceTeam, SourcePersonal, SourceProject}, SourceProject},
		{"starter lowest", []Source{SourceStarter, SourceTeam}, SourceTeam},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			for _, source := range tt.sources {
				registry.Add(&Skill{
					Frontmatter: Frontmatter{Name: "test", Description: string(source)},
					Source:      source,
				})
			}

			got := registry.Get("test")
			if got.Source != tt.wantSource {
				t.Errorf("Get() source = %v, want %v", got.Source, tt.wantSource)
			}
		})
	}
}

func TestByTag(t *testing.T) {
	registry := NewRegistry()
	registry.Add(&Skill{
		Frontmatter: Frontmatter{Name: "a", Description: "A", Tags: []string{"review", "quality"}},
		Source:      SourceTeam,
	})
	registry.Add(&Skill{
		Frontmatter: Frontmatter{Name: "b", Description: "B", Tags: []string{"security"}},
		Source:      SourceTeam,
	})
	registry.Add(&Skill{
		Frontmatter: Frontmatter{Name: "c", Description: "C", Tags: []string{"review"}},
		Source:      SourceTeam,
	})

	got := registry.ByTag("review")
	if len(got) != 2 {
		t.Errorf("ByTag(review) = %d skills, want 2", len(got))
	}

	got = registry.ByTag("security")
	if len(got) != 1 {
		t.Errorf("ByTag(security) = %d skills, want 1", len(got))
	}

	got = registry.ByTag("nonexistent")
	if len(got) != 0 {
		t.Errorf("ByTag(nonexistent) = %d skills, want 0", len(got))
	}
}

func TestParseDirWithSupportingFiles(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()
	skillDir := filepath.Join(tempDir, "my-skill")
	if err := os.MkdirAll(filepath.Join(skillDir, "templates"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create SKILL.md
	skillMD := `---
name: my-skill
description: Test skill with supporting files
---

Instructions here.`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	// Create supporting files
	if err := os.WriteFile(filepath.Join(skillDir, "templates", "review.md"), []byte("template"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "script.sh"), []byte("echo hello"), 0644); err != nil {
		t.Fatal(err)
	}

	skill, err := ParseDir(skillDir, SourceTeam)
	if err != nil {
		t.Fatalf("ParseDir() error = %v", err)
	}

	if skill.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "my-skill")
	}

	if len(skill.SupportingFiles) != 2 {
		t.Errorf("SupportingFiles count = %d, want 2", len(skill.SupportingFiles))
	}

	// Check that relative paths are correct
	if _, ok := skill.SupportingFiles["templates/review.md"]; !ok {
		t.Error("expected templates/review.md in SupportingFiles")
	}
	if _, ok := skill.SupportingFiles["script.sh"]; !ok {
		t.Error("expected script.sh in SupportingFiles")
	}
}

func TestLoadFromDirectory(t *testing.T) {
	// Create temp directory with test skills
	tempDir := t.TempDir()

	// Create valid skill
	validSkillDir := filepath.Join(tempDir, "valid-skill")
	if err := os.MkdirAll(validSkillDir, 0755); err != nil {
		t.Fatal(err)
	}
	validSkill := `---
name: valid-skill
description: A valid skill
---

Do something.`
	if err := os.WriteFile(filepath.Join(validSkillDir, "SKILL.md"), []byte(validSkill), 0644); err != nil {
		t.Fatal(err)
	}

	// Create directory without SKILL.md (should be ignored)
	notASkill := filepath.Join(tempDir, "not-a-skill")
	if err := os.MkdirAll(notASkill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(notASkill, "README.md"), []byte("not a skill"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create regular file (should be ignored)
	if err := os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	skills, err := LoadFromDirectory(tempDir, SourceTeam)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("LoadFromDirectory() returned %d skills, want 1", len(skills))
	}

	if skills[0].Name != "valid-skill" {
		t.Errorf("skill name = %q, want %q", skills[0].Name, "valid-skill")
	}
}

func TestLoadFromNonexistentDirectory(t *testing.T) {
	skills, err := LoadFromDirectory("/nonexistent/path", SourceTeam)
	if err != nil {
		t.Errorf("expected nil error for nonexistent directory, got %v", err)
	}
	if skills != nil {
		t.Errorf("expected nil skills for nonexistent directory, got %v", skills)
	}
}

func TestLoadRegistryWithMultipleDirs(t *testing.T) {
	// Create temp directories for multiple team sources
	tempDir := t.TempDir()

	// Team dir 1
	teamDir1 := filepath.Join(tempDir, "team1")
	skillDir1 := filepath.Join(teamDir1, "skill-from-team1")
	if err := os.MkdirAll(skillDir1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir1, "SKILL.md"), []byte(`---
name: skill-from-team1
description: From team 1
---

Body.`), 0644); err != nil {
		t.Fatal(err)
	}

	// Team dir 2
	teamDir2 := filepath.Join(tempDir, "team2")
	skillDir2 := filepath.Join(teamDir2, "skill-from-team2")
	if err := os.MkdirAll(skillDir2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir2, "SKILL.md"), []byte(`---
name: skill-from-team2
description: From team 2
---

Body.`), 0644); err != nil {
		t.Fatal(err)
	}

	// Personal dir with override
	personalDir := filepath.Join(tempDir, "personal")
	personalSkillDir := filepath.Join(personalDir, "skill-from-team1")
	if err := os.MkdirAll(personalSkillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(personalSkillDir, "SKILL.md"), []byte(`---
name: skill-from-team1
description: Personal override
---

Body.`), 0644); err != nil {
		t.Fatal(err)
	}

	// Load with multiple team dirs
	registry, err := LoadRegistryWithMultipleDirs(
		[]string{teamDir1, teamDir2},
		personalDir,
		"", // no project dir
	)
	if err != nil {
		t.Fatalf("LoadRegistryWithMultipleDirs() error = %v", err)
	}

	// Should have 2 unique skills
	if registry.Count() != 2 {
		t.Errorf("Count() = %d, want 2", registry.Count())
	}

	// skill-from-team1 should be the personal override
	skill1 := registry.Get("skill-from-team1")
	if skill1 == nil {
		t.Fatal("expected skill-from-team1 to exist")
	}
	if skill1.Description != "Personal override" {
		t.Errorf("skill-from-team1 description = %q, want %q", skill1.Description, "Personal override")
	}
	if skill1.Source != SourcePersonal {
		t.Errorf("skill-from-team1 source = %v, want %v", skill1.Source, SourcePersonal)
	}

	// skill-from-team2 should be from team
	skill2 := registry.Get("skill-from-team2")
	if skill2 == nil {
		t.Fatal("expected skill-from-team2 to exist")
	}
	if skill2.Source != SourceTeam {
		t.Errorf("skill-from-team2 source = %v, want %v", skill2.Source, SourceTeam)
	}
}

func TestLoadRegistryWithMultipleDirsNonexistent(t *testing.T) {
	// Should handle nonexistent team dirs gracefully (logs warning, continues)
	registry, err := LoadRegistryWithMultipleDirs(
		[]string{"/nonexistent/team1", "/nonexistent/team2"},
		"",
		"",
	)
	if err != nil {
		t.Fatalf("LoadRegistryWithMultipleDirs() error = %v", err)
	}
	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}
}

func TestHasArg(t *testing.T) {
	skill := &Skill{
		Frontmatter: Frontmatter{
			Name:        "test",
			Description: "Test",
			Args: []Arg{
				{Name: "path", Default: "."},
				{Name: "format", Options: []string{"json", "yaml"}},
			},
		},
	}

	if !skill.HasArg("path") {
		t.Error("expected HasArg(path) to be true")
	}
	if !skill.HasArg("format") {
		t.Error("expected HasArg(format) to be true")
	}
	if skill.HasArg("nonexistent") {
		t.Error("expected HasArg(nonexistent) to be false")
	}
}

func TestGetArg(t *testing.T) {
	skill := &Skill{
		Frontmatter: Frontmatter{
			Name:        "test",
			Description: "Test",
			Args: []Arg{
				{Name: "path", Default: "."},
			},
		},
	}

	arg := skill.GetArg("path")
	if arg == nil {
		t.Fatal("expected GetArg(path) to return non-nil")
	}
	if arg.Default != "." {
		t.Errorf("arg.Default = %q, want %q", arg.Default, ".")
	}

	if skill.GetArg("nonexistent") != nil {
		t.Error("expected GetArg(nonexistent) to return nil")
	}
}

func TestSourceLabel(t *testing.T) {
	tests := []struct {
		source Source
		want   string
	}{
		{SourceTeam, "team"},
		{SourcePersonal, "personal"},
		{SourceProject, "project"},
		{SourceStarter, "starter"},
		{Source("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.source.Label(); got != tt.want {
				t.Errorf("Label() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewSkillTemplate(t *testing.T) {
	template := NewSkillTemplate("code-review", "Review code for issues")

	// Check it contains expected content
	if !strings.Contains(template, "name: code-review") {
		t.Error("template should contain name field")
	}
	if !strings.Contains(template, "description: Review code for issues") {
		t.Error("template should contain description field")
	}
	if !strings.Contains(template, "# Code Review") {
		t.Error("template should contain title")
	}
	if !strings.Contains(template, "allowed-tools:") {
		t.Error("template should contain allowed-tools field")
	}
}

// Helper functions

func boolPtr(b bool) *bool {
	return &b
}
