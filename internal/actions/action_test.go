package actions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantName    string
		wantDesc    string
		wantTags    []string
		wantArgs    int
		wantBody    string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid action with all fields",
			content: `---
name: security-audit
description: Scan for vulnerabilities
tags: [security, ci]
args:
  - name: path
    description: Target path
    default: "."
---

# Security Audit

Check the code at {{path}}.`,
			wantName: "security-audit",
			wantDesc: "Scan for vulnerabilities",
			wantTags: []string{"security", "ci"},
			wantArgs: 1,
			wantBody: "# Security Audit\n\nCheck the code at {{path}}.",
		},
		{
			name: "minimal action",
			content: `---
name: simple
---

Do something.`,
			wantName: "simple",
			wantBody: "Do something.",
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
			name: "action with options",
			content: `---
name: with-options
args:
  - name: severity
    options: [low, medium, high]
    default: medium
---

Check with {{severity}} severity.`,
			wantName: "with-options",
			wantArgs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := Parse(tt.content, SourceTeam, "test.md")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q doesn't contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if action.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", action.Name, tt.wantName)
			}

			if tt.wantDesc != "" && action.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", action.Description, tt.wantDesc)
			}

			if tt.wantTags != nil && len(action.Tags) != len(tt.wantTags) {
				t.Errorf("Tags = %v, want %v", action.Tags, tt.wantTags)
			}

			if len(action.Args) != tt.wantArgs {
				t.Errorf("len(Args) = %d, want %d", len(action.Args), tt.wantArgs)
			}

			if tt.wantBody != "" && action.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", action.Body, tt.wantBody)
			}
		})
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		args    []Arg
		input   map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "simple substitution",
			body: "Check {{path}} with {{severity}}.",
			args: []Arg{
				{Name: "path", Default: "."},
				{Name: "severity", Default: "medium"},
			},
			input: map[string]string{"path": "src/"},
			want:  "Check src/ with medium.",
		},
		{
			name: "all defaults",
			body: "Target: {{path}}",
			args: []Arg{
				{Name: "path", Default: "."},
			},
			input: map[string]string{},
			want:  "Target: .",
		},
		{
			name:  "unrecognized variable left as-is",
			body:  "Hello {{unknown}}!",
			args:  []Arg{},
			input: map[string]string{},
			want:  "Hello {{unknown}}!",
		},
		{
			name: "required arg missing",
			body: "Check {{path}}",
			args: []Arg{
				{Name: "path", Required: true},
			},
			input:   map[string]string{},
			wantErr: true,
		},
		{
			name: "invalid option",
			body: "Severity: {{severity}}",
			args: []Arg{
				{Name: "severity", Options: []string{"low", "medium", "high"}},
			},
			input:   map[string]string{"severity": "extreme"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &Action{
				Frontmatter: Frontmatter{
					Name: "test",
					Args: tt.args,
				},
				Body: tt.body,
			}

			result, err := action.Render(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.want {
				t.Errorf("Render() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		raw     []string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "double dash equals",
			raw:  []string{"--path=src/", "--severity=high"},
			want: map[string]string{"path": "src/", "severity": "high"},
		},
		{
			name: "double dash space",
			raw:  []string{"--path", "src/", "--severity", "high"},
			want: map[string]string{"path": "src/", "severity": "high"},
		},
		{
			name: "simple equals",
			raw:  []string{"path=src/", "severity=high"},
			want: map[string]string{"path": "src/", "severity": "high"},
		},
		{
			name: "mixed formats",
			raw:  []string{"--path=src/", "severity=high"},
			want: map[string]string{"path": "src/", "severity": "high"},
		},
		{
			name:    "missing value",
			raw:     []string{"--path"},
			wantErr: true,
		},
		{
			name:    "invalid format",
			raw:     []string{"pathvalue"},
			wantErr: true,
		},
		{
			name: "empty args",
			raw:  []string{},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseArgs(tt.raw)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tt.want) {
				t.Errorf("len(result) = %d, want %d", len(result), len(tt.want))
			}

			for k, v := range tt.want {
				if result[k] != v {
					t.Errorf("result[%q] = %q, want %q", k, result[k], v)
				}
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Add team action
	teamAction := &Action{
		Frontmatter: Frontmatter{Name: "audit", Description: "Team audit"},
		Source:      SourceTeam,
	}
	registry.Add(teamAction)

	// Add personal override
	personalAction := &Action{
		Frontmatter: Frontmatter{Name: "audit", Description: "Personal audit"},
		Source:      SourcePersonal,
	}
	registry.Add(personalAction)

	// Add project action
	projectAction := &Action{
		Frontmatter: Frontmatter{Name: "project-only", Description: "Project specific"},
		Source:      SourceProject,
	}
	registry.Add(projectAction)

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
	teamActions := registry.BySource(SourceTeam)
	if len(teamActions) != 1 {
		t.Errorf("BySource(team) = %d, want 1", len(teamActions))
	}
}

func TestLoadFromDirectory(t *testing.T) {
	// Create temp directory with test actions
	tempDir := t.TempDir()

	// Create valid action
	validAction := `---
name: valid-action
description: A valid action
---

Do something.`
	if err := os.WriteFile(filepath.Join(tempDir, "valid.md"), []byte(validAction), 0644); err != nil {
		t.Fatal(err)
	}

	// Create invalid action (should be skipped with warning)
	invalidAction := `Not a valid action file`
	if err := os.WriteFile(filepath.Join(tempDir, "invalid.md"), []byte(invalidAction), 0644); err != nil {
		t.Fatal(err)
	}

	// Create non-md file (should be ignored)
	if err := os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	actions, err := LoadFromDirectory(tempDir, SourceTeam)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(actions) != 1 {
		t.Errorf("LoadFromDirectory() returned %d actions, want 1", len(actions))
	}

	if actions[0].Name != "valid-action" {
		t.Errorf("action name = %q, want %q", actions[0].Name, "valid-action")
	}
}

func TestLoadFromNonexistentDirectory(t *testing.T) {
	actions, err := LoadFromDirectory("/nonexistent/path", SourceTeam)
	if err != nil {
		t.Errorf("expected nil error for nonexistent directory, got %v", err)
	}
	if actions != nil {
		t.Errorf("expected nil actions for nonexistent directory, got %v", actions)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
