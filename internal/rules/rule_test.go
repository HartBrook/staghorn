package rules

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
		source      Source
		filePath    string
		relPath     string
		wantName    string
		wantPaths   []string
		wantBody    string
		wantErr     bool
		errContains string
	}{
		{
			name: "rule with frontmatter and paths",
			content: `---
paths:
  - "src/api/**/*.ts"
  - "src/routes/**/*.ts"
---
# API Rules

Use proper status codes.`,
			source:    SourceTeam,
			filePath:  "/team/rules/api.md",
			relPath:   "api.md",
			wantName:  "api",
			wantPaths: []string{"src/api/**/*.ts", "src/routes/**/*.ts"},
			wantBody:  "# API Rules\n\nUse proper status codes.",
		},
		{
			name: "rule without frontmatter",
			content: `# Security Guidelines

Never commit secrets.`,
			source:   SourcePersonal,
			filePath: "/personal/rules/security.md",
			relPath:  "security.md",
			wantName: "security",
			wantBody: "# Security Guidelines\n\nNever commit secrets.",
		},
		{
			name: "rule with empty frontmatter",
			content: `---
---
# Testing

Write tests.`,
			source:   SourceProject,
			filePath: "/project/rules/testing.md",
			relPath:  "testing.md",
			wantName: "testing",
			wantBody: "# Testing\n\nWrite tests.",
		},
		{
			name: "rule in subdirectory",
			content: `---
paths:
  - "src/components/**/*.tsx"
---
# React Rules`,
			source:    SourceTeam,
			filePath:  "/team/rules/frontend/react.md",
			relPath:   "frontend/react.md",
			wantName:  "react",
			wantPaths: []string{"src/components/**/*.tsx"},
			wantBody:  "# React Rules",
		},
		{
			name: "unterminated frontmatter",
			content: `---
paths:
  - "src/**/*.ts"
# Missing closing ---`,
			source:      SourceTeam,
			filePath:    "/team/rules/bad.md",
			relPath:     "bad.md",
			wantErr:     true,
			errContains: "unterminated frontmatter",
		},
		{
			name: "invalid yaml frontmatter",
			content: `---
paths: [invalid yaml
---
# Content`,
			source:      SourceTeam,
			filePath:    "/team/rules/bad.md",
			relPath:     "bad.md",
			wantErr:     true,
			errContains: "invalid frontmatter YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := Parse(tt.content, tt.source, tt.filePath, tt.relPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if rule.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", rule.Name, tt.wantName)
			}

			if rule.Source != tt.source {
				t.Errorf("Source = %q, want %q", rule.Source, tt.source)
			}

			if rule.FilePath != tt.filePath {
				t.Errorf("FilePath = %q, want %q", rule.FilePath, tt.filePath)
			}

			if rule.RelPath != tt.relPath {
				t.Errorf("RelPath = %q, want %q", rule.RelPath, tt.relPath)
			}

			if rule.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", rule.Body, tt.wantBody)
			}

			if len(rule.Paths) != len(tt.wantPaths) {
				t.Errorf("Paths length = %d, want %d", len(rule.Paths), len(tt.wantPaths))
			} else {
				for i, path := range rule.Paths {
					if path != tt.wantPaths[i] {
						t.Errorf("Paths[%d] = %q, want %q", i, path, tt.wantPaths[i])
					}
				}
			}
		})
	}
}

func TestLoadFromDirectory(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create rules directory with subdirectories
	rulesDir := filepath.Join(tmpDir, "rules")
	apiDir := filepath.Join(rulesDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create rule files
	files := map[string]string{
		"security.md": `# Security

Validate input.`,
		"api/rest.md": `---
paths:
  - "src/api/**/*.ts"
---
# REST API

Use proper methods.`,
	}

	for name, content := range files {
		path := filepath.Join(rulesDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Also create a non-md file that should be ignored
	if err := os.WriteFile(filepath.Join(rulesDir, "readme.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	// Load rules
	rules, err := LoadFromDirectory(rulesDir, SourceTeam)
	if err != nil {
		t.Fatalf("LoadFromDirectory failed: %v", err)
	}

	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rules))
	}

	// Check that we got both rules with correct relative paths
	ruleMap := make(map[string]*Rule)
	for _, r := range rules {
		ruleMap[r.RelPath] = r
	}

	if _, ok := ruleMap["security.md"]; !ok {
		t.Error("missing security.md rule")
	}

	if r, ok := ruleMap["api/rest.md"]; !ok {
		t.Error("missing api/rest.md rule")
	} else {
		if r.Name != "rest" {
			t.Errorf("api/rest.md Name = %q, want %q", r.Name, "rest")
		}
		if len(r.Paths) != 1 || r.Paths[0] != "src/api/**/*.ts" {
			t.Errorf("api/rest.md Paths = %v, want [src/api/**/*.ts]", r.Paths)
		}
	}
}

func TestLoadFromDirectory_NonExistent(t *testing.T) {
	rules, err := LoadFromDirectory("/nonexistent/path", SourceTeam)
	if err != nil {
		t.Fatalf("expected nil error for nonexistent directory, got: %v", err)
	}
	if rules != nil {
		t.Errorf("expected nil rules for nonexistent directory, got: %v", rules)
	}
}

func TestHasPathScope(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  bool
	}{
		{"no paths", nil, false},
		{"empty paths", []string{}, false},
		{"has paths", []string{"src/**/*.ts"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Rule{Frontmatter: Frontmatter{Paths: tt.paths}}
			if got := r.HasPathScope(); got != tt.want {
				t.Errorf("HasPathScope() = %v, want %v", got, tt.want)
			}
		})
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
		t.Run(string(tt.source), func(t *testing.T) {
			if got := tt.source.Label(); got != tt.want {
				t.Errorf("Label() = %q, want %q", got, tt.want)
			}
		})
	}
}
