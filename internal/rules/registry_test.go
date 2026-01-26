package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_Add(t *testing.T) {
	r := NewRegistry()

	// Add team rule
	teamRule := &Rule{
		Name:    "security",
		RelPath: "security.md",
		Source:  SourceTeam,
		Body:    "Team security rules",
	}
	r.Add(teamRule)

	// Verify it was added
	if got := r.Get("security.md"); got != teamRule {
		t.Error("team rule not added")
	}

	// Add personal rule with same relPath - should override
	personalRule := &Rule{
		Name:    "security",
		RelPath: "security.md",
		Source:  SourcePersonal,
		Body:    "Personal security rules",
	}
	r.Add(personalRule)

	// Verify personal rule overrides team
	if got := r.Get("security.md"); got != personalRule {
		t.Error("personal rule should override team rule")
	}

	// Add project rule - should override personal
	projectRule := &Rule{
		Name:    "security",
		RelPath: "security.md",
		Source:  SourceProject,
		Body:    "Project security rules",
	}
	r.Add(projectRule)

	if got := r.Get("security.md"); got != projectRule {
		t.Error("project rule should override personal rule")
	}

	// Try to add team rule again - should NOT override project
	anotherTeamRule := &Rule{
		Name:    "security",
		RelPath: "security.md",
		Source:  SourceTeam,
		Body:    "Another team rule",
	}
	r.Add(anotherTeamRule)

	if got := r.Get("security.md"); got != projectRule {
		t.Error("team rule should not override project rule")
	}
}

func TestRegistry_All(t *testing.T) {
	r := NewRegistry()

	r.Add(&Rule{RelPath: "a.md", Source: SourceTeam})
	r.Add(&Rule{RelPath: "b.md", Source: SourcePersonal})
	r.Add(&Rule{RelPath: "c.md", Source: SourceProject})
	// Override a.md with personal
	r.Add(&Rule{RelPath: "a.md", Source: SourcePersonal})

	all := r.All()
	if len(all) != 3 {
		t.Errorf("All() returned %d rules, want 3", len(all))
	}
}

func TestRegistry_BySource(t *testing.T) {
	r := NewRegistry()

	r.Add(&Rule{RelPath: "a.md", Source: SourceTeam})
	r.Add(&Rule{RelPath: "b.md", Source: SourceTeam})
	r.Add(&Rule{RelPath: "c.md", Source: SourcePersonal})

	teamRules := r.BySource(SourceTeam)
	if len(teamRules) != 2 {
		t.Errorf("BySource(team) returned %d rules, want 2", len(teamRules))
	}

	personalRules := r.BySource(SourcePersonal)
	if len(personalRules) != 1 {
		t.Errorf("BySource(personal) returned %d rules, want 1", len(personalRules))
	}

	projectRules := r.BySource(SourceProject)
	if len(projectRules) != 0 {
		t.Errorf("BySource(project) returned %d rules, want 0", len(projectRules))
	}
}

func TestRegistry_Count(t *testing.T) {
	r := NewRegistry()

	if r.Count() != 0 {
		t.Error("empty registry should have count 0")
	}

	r.Add(&Rule{RelPath: "a.md", Source: SourceTeam})
	r.Add(&Rule{RelPath: "b.md", Source: SourceTeam})
	// Override a.md
	r.Add(&Rule{RelPath: "a.md", Source: SourcePersonal})

	if r.Count() != 2 {
		t.Errorf("Count() = %d, want 2 (unique relPaths)", r.Count())
	}
}

func TestLoadRegistry(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	teamDir := filepath.Join(tmpDir, "team")
	personalDir := filepath.Join(tmpDir, "personal")
	projectDir := filepath.Join(tmpDir, "project")

	for _, dir := range []string{teamDir, personalDir, projectDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create team rule
	if err := os.WriteFile(filepath.Join(teamDir, "security.md"), []byte("# Team Security"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create personal rule (overrides team)
	if err := os.WriteFile(filepath.Join(personalDir, "security.md"), []byte("# Personal Security"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create project-only rule
	if err := os.WriteFile(filepath.Join(projectDir, "project-only.md"), []byte("# Project Only"), 0644); err != nil {
		t.Fatal(err)
	}

	registry, err := LoadRegistry(teamDir, personalDir, projectDir)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	// Should have 2 unique rules
	if registry.Count() != 2 {
		t.Errorf("Count() = %d, want 2", registry.Count())
	}

	// security.md should be from personal (overrides team)
	securityRule := registry.Get("security.md")
	if securityRule == nil {
		t.Fatal("missing security.md rule")
	}
	if securityRule.Source != SourcePersonal {
		t.Errorf("security.md source = %q, want %q", securityRule.Source, SourcePersonal)
	}

	// project-only.md should exist
	projectRule := registry.Get("project-only.md")
	if projectRule == nil {
		t.Fatal("missing project-only.md rule")
	}
	if projectRule.Source != SourceProject {
		t.Errorf("project-only.md source = %q, want %q", projectRule.Source, SourceProject)
	}
}

func TestLoadRegistry_EmptyDirs(t *testing.T) {
	registry, err := LoadRegistry("", "", "")
	if err != nil {
		t.Fatalf("LoadRegistry with empty dirs failed: %v", err)
	}
	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}
}

func TestLoadRegistry_SubdirectoryConflict(t *testing.T) {
	// Test that subdirectory rules are also properly overridden
	tmpDir := t.TempDir()
	teamDir := filepath.Join(tmpDir, "team", "api")
	personalDir := filepath.Join(tmpDir, "personal", "api")

	for _, dir := range []string{teamDir, personalDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create team api/rest.md
	if err := os.WriteFile(filepath.Join(teamDir, "rest.md"), []byte("# Team REST"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create personal api/rest.md (overrides team)
	if err := os.WriteFile(filepath.Join(personalDir, "rest.md"), []byte("# Personal REST"), 0644); err != nil {
		t.Fatal(err)
	}

	registry, err := LoadRegistry(
		filepath.Join(tmpDir, "team"),
		filepath.Join(tmpDir, "personal"),
		"",
	)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	// Should have 1 unique rule
	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}

	// api/rest.md should be from personal
	restRule := registry.Get("api/rest.md")
	if restRule == nil {
		t.Fatal("missing api/rest.md rule")
	}
	if restRule.Source != SourcePersonal {
		t.Errorf("api/rest.md source = %q, want %q", restRule.Source, SourcePersonal)
	}
}

func TestSourcePrecedence(t *testing.T) {
	// Verify precedence order: project > personal > team = starter
	tests := []struct {
		source Source
		want   int
	}{
		{SourceTeam, 1},
		{SourceStarter, 1},
		{SourcePersonal, 2},
		{SourceProject, 3},
	}

	for _, tt := range tests {
		t.Run(string(tt.source), func(t *testing.T) {
			if got := sourcePrecedence(tt.source); got != tt.want {
				t.Errorf("sourcePrecedence(%q) = %d, want %d", tt.source, got, tt.want)
			}
		})
	}
}
