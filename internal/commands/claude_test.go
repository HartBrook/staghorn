package commands

import (
	"strings"
	"testing"
)

func TestConvertToClaude(t *testing.T) {
	tests := []struct {
		name           string
		cmd            *Command
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "basic command with args",
			cmd: &Command{
				Frontmatter: Frontmatter{
					Name:        "code-review",
					Description: "Perform a code review",
					Args: []Arg{
						{Name: "path", Default: ".", Description: "Target path"},
						{Name: "focus", Required: true, Description: "Focus area"},
					},
				},
				Body:   "Review the code at {{path}} with focus on {{focus}}.",
				Source: SourceTeam,
			},
			wantContains: []string{
				"<!-- Managed by staghorn | Source: team | Do not edit directly -->",
				"name: code-review",
				"description: Perform a code review",
				"<!-- Args: path (default: .), focus (required) -->",
				"<!-- Example: /code-review",
				"Review the code at {{path}}",
			},
		},
		{
			name: "command without args",
			cmd: &Command{
				Frontmatter: Frontmatter{
					Name:        "simple",
					Description: "A simple command",
				},
				Body:   "Do something simple.",
				Source: SourcePersonal,
			},
			wantContains: []string{
				"<!-- Managed by staghorn | Source: personal | Do not edit directly -->",
				"name: simple",
				"Do something simple.",
			},
			wantNotContain: []string{
				"<!-- Args:",
				"<!-- Example:",
			},
		},
		{
			name: "command from project source",
			cmd: &Command{
				Frontmatter: Frontmatter{
					Name: "project-command",
				},
				Body:   "Project specific.",
				Source: SourceProject,
			},
			wantContains: []string{
				"Source: project",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToClaude(tt.cmd)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("result missing %q\nGot:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(result, notWant) {
					t.Errorf("result should not contain %q\nGot:\n%s", notWant, result)
				}
			}
		})
	}
}

func TestBuildArgsHint(t *testing.T) {
	tests := []struct {
		name         string
		args         []Arg
		wantContains []string
	}{
		{
			name: "required and default args",
			args: []Arg{
				{Name: "path", Default: "."},
				{Name: "target", Required: true},
			},
			wantContains: []string{
				"path (default: .)",
				"target (required)",
			},
		},
		{
			name: "arg without default or required",
			args: []Arg{
				{Name: "optional"},
			},
			wantContains: []string{
				"<!-- Args: optional -->",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Command{
				Frontmatter: Frontmatter{
					Name: "test",
					Args: tt.args,
				},
			}

			result := buildArgsHint(cmd)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("result missing %q\nGot:\n%s", want, result)
				}
			}
		})
	}
}
