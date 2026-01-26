package rules

import (
	"strings"
	"testing"
)

func TestConvertToClaude(t *testing.T) {
	tests := []struct {
		name    string
		rule    *Rule
		wantHas []string
		wantNot []string
	}{
		{
			name: "rule with paths frontmatter",
			rule: &Rule{
				Frontmatter: Frontmatter{
					Paths: []string{"src/api/**/*.ts", "src/routes/**/*.ts"},
				},
				Name:   "rest",
				Body:   "# REST API\n\nUse proper methods.",
				Source: SourceTeam,
			},
			wantHas: []string{
				"---\n",
				"paths:",
				"src/api/**/*.ts",
				"src/routes/**/*.ts",
				"<!-- Managed by staghorn | Source: team | Do not edit directly -->",
				"# REST API",
				"Use proper methods.",
			},
		},
		{
			name: "rule without paths",
			rule: &Rule{
				Name:   "security",
				Body:   "# Security\n\nValidate input.",
				Source: SourcePersonal,
			},
			wantHas: []string{
				"<!-- Managed by staghorn | Source: personal | Do not edit directly -->",
				"# Security",
				"Validate input.",
			},
			wantNot: []string{
				"---\n",
				"paths:",
			},
		},
		{
			name: "rule from project source",
			rule: &Rule{
				Frontmatter: Frontmatter{
					Paths: []string{"src/**/*.go"},
				},
				Name:   "go",
				Body:   "# Go Rules",
				Source: SourceProject,
			},
			wantHas: []string{
				"Source: project",
				"src/**/*.go",
			},
		},
		{
			name: "rule from starter source",
			rule: &Rule{
				Name:   "testing",
				Body:   "# Testing",
				Source: SourceStarter,
			},
			wantHas: []string{
				"Source: starter",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToClaude(tt.rule)
			if err != nil {
				t.Fatalf("ConvertToClaude failed: %v", err)
			}

			for _, want := range tt.wantHas {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\nGot:\n%s", want, got)
				}
			}

			for _, notWant := range tt.wantNot {
				if strings.Contains(got, notWant) {
					t.Errorf("output should not contain %q\nGot:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestConvertToClaude_FrontmatterFirst(t *testing.T) {
	rule := &Rule{
		Frontmatter: Frontmatter{
			Paths: []string{"src/**/*.ts"},
		},
		Name:   "test",
		Body:   "# Test",
		Source: SourceTeam,
	}

	got, err := ConvertToClaude(rule)
	if err != nil {
		t.Fatalf("ConvertToClaude failed: %v", err)
	}

	// Frontmatter should come before the managed header
	fmIdx := strings.Index(got, "---\n")
	headerIdx := strings.Index(got, "<!-- Managed by staghorn")

	if fmIdx == -1 {
		t.Fatal("missing frontmatter")
	}
	if headerIdx == -1 {
		t.Fatal("missing managed header")
	}
	if fmIdx > headerIdx {
		t.Error("frontmatter should come before managed header")
	}
}

func TestConvertToClaude_BodyLast(t *testing.T) {
	rule := &Rule{
		Frontmatter: Frontmatter{
			Paths: []string{"src/**/*.ts"},
		},
		Name:   "test",
		Body:   "# Test Content Here",
		Source: SourceTeam,
	}

	got, err := ConvertToClaude(rule)
	if err != nil {
		t.Fatalf("ConvertToClaude failed: %v", err)
	}

	// Body should come after the managed header
	headerIdx := strings.Index(got, "<!-- Managed by staghorn")
	bodyIdx := strings.Index(got, "# Test Content Here")

	if headerIdx == -1 {
		t.Fatal("missing managed header")
	}
	if bodyIdx == -1 {
		t.Fatal("missing body")
	}
	if bodyIdx < headerIdx {
		t.Error("body should come after managed header")
	}
}
