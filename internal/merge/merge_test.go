package merge

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantPreamble string
		wantSections int
		wantHeaders  []string
	}{
		{
			name:         "empty document",
			content:      "",
			wantPreamble: "",
			wantSections: 0,
		},
		{
			name:         "preamble only",
			content:      "This is just a preamble with no headers.",
			wantPreamble: "This is just a preamble with no headers.",
			wantSections: 0,
		},
		{
			name:         "single section",
			content:      "## Code Style\n\nUse consistent formatting.",
			wantPreamble: "",
			wantSections: 1,
			wantHeaders:  []string{"Code Style"},
		},
		{
			name:         "multiple sections",
			content:      "## Code Style\n\nFormat code.\n\n## Review Guidelines\n\nReview carefully.",
			wantPreamble: "",
			wantSections: 2,
			wantHeaders:  []string{"Code Style", "Review Guidelines"},
		},
		{
			name:         "preamble and sections",
			content:      "# Title\n\nIntro text.\n\n## Section One\n\nContent one.\n\n## Section Two\n\nContent two.",
			wantPreamble: "# Title\n\nIntro text.",
			wantSections: 2,
			wantHeaders:  []string{"Section One", "Section Two"},
		},
		{
			name:         "nested headers preserved",
			content:      "## Main Section\n\nContent\n\n### Subsection\n\nMore content.",
			wantPreamble: "",
			wantSections: 1,
			wantHeaders:  []string{"Main Section"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := Parse(tt.content)

			if doc.Preamble != tt.wantPreamble {
				t.Errorf("Preamble = %q, want %q", doc.Preamble, tt.wantPreamble)
			}

			if len(doc.Sections) != tt.wantSections {
				t.Errorf("Sections count = %d, want %d", len(doc.Sections), tt.wantSections)
			}

			if tt.wantHeaders != nil {
				for i, wantHeader := range tt.wantHeaders {
					if i >= len(doc.Sections) {
						t.Errorf("Missing section %d with header %q", i, wantHeader)
						continue
					}
					if doc.Sections[i].Header != wantHeader {
						t.Errorf("Section[%d].Header = %q, want %q", i, doc.Sections[i].Header, wantHeader)
					}
				}
			}
		})
	}
}

func TestFindSection(t *testing.T) {
	doc := Parse("## Code Style\n\nFormat code.\n\n## Review Guidelines\n\nReview carefully.")

	tests := []struct {
		header string
		want   bool
	}{
		{"Code Style", true},
		{"code style", true}, // case-insensitive
		{"CODE STYLE", true}, // case-insensitive
		{"Review Guidelines", true},
		{"Nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			section := doc.FindSection(tt.header)
			found := section != nil
			if found != tt.want {
				t.Errorf("FindSection(%q) found = %v, want %v", tt.header, found, tt.want)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name           string
		layers         []Layer
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "single layer",
			layers: []Layer{
				{Content: "## Code Style\n\nUse formatting.", Source: "team"},
			},
			wantContains: []string{"## Code Style", "Use formatting"},
		},
		{
			name: "team plus personal with matching section",
			layers: []Layer{
				{Content: "## Code Style\n\nUse formatting.", Source: "team"},
				{Content: "## Code Style\n\nI prefer verbose comments.", Source: "personal"},
			},
			wantContains: []string{
				"## Code Style",
				"Use formatting",
				"### Personal Additions",
				"I prefer verbose comments",
			},
		},
		{
			name: "personal adds new section",
			layers: []Layer{
				{Content: "## Code Style\n\nUse formatting.", Source: "team"},
				{Content: "## Verbosity\n\nBe concise.", Source: "personal"},
			},
			wantContains: []string{
				"## Code Style",
				"Use formatting",
				"## Verbosity",
				"Be concise",
			},
			wantNotContain: []string{"Personal Additions"},
		},
		{
			name: "three layers",
			layers: []Layer{
				{Content: "## Code Style\n\nTeam rules.", Source: "team"},
				{Content: "## Code Style\n\nPersonal prefs.", Source: "personal"},
				{Content: "## Code Style\n\nProject specific.", Source: "project"},
			},
			wantContains: []string{
				"Team rules",
				"### Personal Additions",
				"Personal prefs",
				"### Project Additions",
				"Project specific",
			},
		},
		{
			name: "empty layers skipped",
			layers: []Layer{
				{Content: "## Code Style\n\nTeam rules.", Source: "team"},
				{Content: "", Source: "personal"},
				{Content: "## Code Style\n\nProject rules.", Source: "project"},
			},
			wantContains: []string{
				"Team rules",
				"### Project Additions",
				"Project rules",
			},
			wantNotContain: []string{"Personal Additions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Merge(tt.layers, MergeOptions{})

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Merge() result should contain %q\nGot:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(result, notWant) {
					t.Errorf("Merge() result should NOT contain %q\nGot:\n%s", notWant, result)
				}
			}
		})
	}
}

func TestMergeWithSources(t *testing.T) {
	layers := []Layer{
		{Content: "## Code Style\n\nUse formatting.", Source: "team"},
	}

	opts := MergeOptions{
		AnnotateSources: true,
		TeamRepo:        "acme/standards",
	}

	result := Merge(layers, opts)

	if !strings.Contains(result, "Generated by staghorn") {
		t.Error("Merge() with AnnotateSources should include generation comment")
	}
	if !strings.Contains(result, "acme/standards") {
		t.Error("Merge() with TeamRepo should include repo name")
	}
}

func TestMergeSimple(t *testing.T) {
	result := MergeSimple(
		"## A\n\nTeam content.",
		"## A\n\nPersonal content.",
		"## A\n\nProject content.",
	)

	if !strings.Contains(result, "Team content") {
		t.Error("MergeSimple() should include team content")
	}
	if !strings.Contains(result, "Personal Additions") {
		t.Error("MergeSimple() should include personal additions header")
	}
	if !strings.Contains(result, "Project Additions") {
		t.Error("MergeSimple() should include project additions header")
	}
}
