package merge

import (
	"strings"
	"testing"

	"github.com/HartBrook/staghorn/internal/language"
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
		SourceRepo:      "acme/standards",
	}

	result := Merge(layers, opts)

	if !strings.Contains(result, "Managed by staghorn") {
		t.Error("Merge() with AnnotateSources should include managed comment")
	}
	if !strings.Contains(result, "acme/standards") {
		t.Error("Merge() with SourceRepo should include repo name")
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

func TestMergeWithProvenanceComments(t *testing.T) {
	layers := []Layer{
		{Content: "## Code Style\n\nTeam rules.", Source: "team"},
		{Content: "## Code Style\n\nPersonal prefs.", Source: "personal"},
	}

	opts := MergeOptions{
		AnnotateSources: true,
		SourceRepo:      "acme/standards",
	}

	result := Merge(layers, opts)

	// Should have managed header
	if !strings.Contains(result, "Managed by staghorn") {
		t.Error("Should include managed comment")
	}

	// Should have team source markers for the section
	if !strings.Contains(result, "<!-- staghorn:source:team -->") {
		t.Error("Should include team source start marker")
	}

	// Should have personal source marker for the addition
	if !strings.Contains(result, "<!-- staghorn:source:personal -->") {
		t.Error("Should include personal source marker")
	}
}

func TestMergeWithProvenanceComments_NewSection(t *testing.T) {
	layers := []Layer{
		{Content: "## Code Style\n\nTeam rules.", Source: "team"},
		{Content: "## My Preferences\n\nPersonal section.", Source: "personal"},
	}

	opts := MergeOptions{
		AnnotateSources: true,
	}

	result := Merge(layers, opts)

	// New section from personal should have personal source markers
	if !strings.Contains(result, "<!-- staghorn:source:personal -->") {
		t.Error("New section should have personal source marker")
	}

	// Should contain both sections
	if !strings.Contains(result, "## Code Style") {
		t.Error("Should contain team section")
	}
	if !strings.Contains(result, "## My Preferences") {
		t.Error("Should contain personal section")
	}
}

func TestMergeWithoutProvenance(t *testing.T) {
	layers := []Layer{
		{Content: "## Code Style\n\nTeam rules.", Source: "team"},
		{Content: "## Code Style\n\nPersonal prefs.", Source: "personal"},
	}

	// Without AnnotateSources, no provenance comments
	opts := MergeOptions{
		AnnotateSources: false,
	}

	result := Merge(layers, opts)

	// Should NOT have any staghorn source markers
	if strings.Contains(result, "<!-- staghorn:source:") {
		t.Error("Should NOT include source markers when AnnotateSources is false")
	}

	// But should still have the content
	if !strings.Contains(result, "Team rules") {
		t.Error("Should include team content")
	}
	if !strings.Contains(result, "Personal prefs") {
		t.Error("Should include personal content")
	}
}

func TestSourceCommentFormat(t *testing.T) {
	// Verify the comment format is machine-parseable
	marker := sourceStartComment("team")

	if marker != "<!-- staghorn:source:team -->" {
		t.Errorf("Source comment format wrong: %s", marker)
	}
}

func TestMergeProvenanceOrdering(t *testing.T) {
	// Verify that markers appear in correct order: team section first, then personal additions
	layers := []Layer{
		{Content: "## Code Style\n\nTeam rules.", Source: "team"},
		{Content: "## Code Style\n\nPersonal prefs.", Source: "personal"},
	}

	opts := MergeOptions{
		AnnotateSources: true,
	}

	result := Merge(layers, opts)

	// Both markers should be present
	teamStart := strings.Index(result, "<!-- staghorn:source:team -->")
	personalStart := strings.Index(result, "<!-- staghorn:source:personal -->")

	if teamStart == -1 || personalStart == -1 {
		t.Fatalf("Missing markers in result:\n%s", result)
	}

	// Team marker should appear before personal marker
	if teamStart >= personalStart {
		t.Errorf("Team marker should appear before personal: teamStart=%d, personalStart=%d", teamStart, personalStart)
	}
}

func TestBuildLanguageSectionWithProvenance(t *testing.T) {
	files := map[string][]*language.LanguageFile{
		"go": {
			{Language: "go", Content: "Go team rules", Source: "team"},
			{Language: "go", Content: "Go personal prefs", Source: "personal"},
		},
	}

	// With annotations
	result := buildLanguageSection([]string{"go"}, files, true)

	// Should have language-specific source markers
	if !strings.Contains(result, "<!-- staghorn:source:team:go -->") {
		t.Errorf("Should include team:go source marker, got:\n%s", result)
	}
	if !strings.Contains(result, "<!-- staghorn:source:personal:go -->") {
		t.Errorf("Should include personal:go source marker, got:\n%s", result)
	}

	// Should have content
	if !strings.Contains(result, "Go team rules") {
		t.Error("Should include team content")
	}
	if !strings.Contains(result, "Go personal prefs") {
		t.Error("Should include personal content")
	}
}

func TestBuildLanguageSectionWithProvenance_MultipleLanguages(t *testing.T) {
	files := map[string][]*language.LanguageFile{
		"go": {
			{Language: "go", Content: "Go team rules", Source: "team"},
		},
		"python": {
			{Language: "python", Content: "Python team rules", Source: "team"},
			{Language: "python", Content: "Python personal prefs", Source: "personal"},
		},
	}

	result := buildLanguageSection([]string{"go", "python"}, files, true)

	// Should have distinct markers for each language
	if !strings.Contains(result, "<!-- staghorn:source:team:go -->") {
		t.Error("Should include team:go marker")
	}
	if !strings.Contains(result, "<!-- staghorn:source:team:python -->") {
		t.Error("Should include team:python marker")
	}
	if !strings.Contains(result, "<!-- staghorn:source:personal:python -->") {
		t.Error("Should include personal:python marker")
	}
}

func TestBuildLanguageSectionWithoutProvenance(t *testing.T) {
	files := map[string][]*language.LanguageFile{
		"python": {
			{Language: "python", Content: "Python rules", Source: "team"},
		},
	}

	// Without annotations
	result := buildLanguageSection([]string{"python"}, files, false)

	// Should NOT have source markers
	if strings.Contains(result, "<!-- staghorn:source:") {
		t.Error("Should NOT include source markers when annotate is false")
	}

	// Should have content
	if !strings.Contains(result, "Python rules") {
		t.Error("Should include content")
	}
}

func TestDemoteHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "demotes H2 to H3",
			input:    "## Header",
			expected: "### Header",
		},
		{
			name:     "demotes H3 to H4",
			input:    "### Header",
			expected: "#### Header",
		},
		{
			name:     "demotes multiple headers",
			input:    "## First\n\nContent\n\n### Second\n\nMore content",
			expected: "### First\n\nContent\n\n#### Second\n\nMore content",
		},
		{
			name:     "H6 stays at H6",
			input:    "###### Header",
			expected: "###### Header",
		},
		{
			name:     "preserves non-header content",
			input:    "Just some text\n\nMore text",
			expected: "Just some text\n\nMore text",
		},
		{
			name:     "handles H1",
			input:    "# Title",
			expected: "## Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := demoteHeaders(tt.input)
			if result != tt.expected {
				t.Errorf("demoteHeaders() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildLanguageSection_FlatH2Structure(t *testing.T) {
	files := map[string][]*language.LanguageFile{
		"python": {
			{Language: "python", Content: "## Python Guidelines\n\nUse type hints.\n\n## Type Safety\n\nUse mypy.", Source: "team"},
		},
	}

	result := buildLanguageSection([]string{"python"}, files, false)

	// Should have Python as H2 header
	if !strings.Contains(result, "## Python\n") {
		t.Errorf("Should have Python as H2 section, got:\n%s", result)
	}

	// Content headers should be demoted: ## -> ###
	if !strings.Contains(result, "### Python Guidelines") {
		t.Errorf("Content H2 should be demoted to H3, got:\n%s", result)
	}
	if !strings.Contains(result, "### Type Safety") {
		t.Errorf("Content H2 should be demoted to H3, got:\n%s", result)
	}

	// Should NOT have the old nested structure
	if strings.Contains(result, "## Language-Specific Guidelines") {
		t.Error("Should NOT have Language-Specific Guidelines container")
	}
	if strings.Contains(result, "### Python\n") {
		t.Error("Python should be H2, not H3")
	}
}

func TestBuildLanguageSection_PersonalAdditionsAsH3(t *testing.T) {
	files := map[string][]*language.LanguageFile{
		"go": {
			{Language: "go", Content: "Team Go rules", Source: "team"},
			{Language: "go", Content: "## My Preferences\n\nPersonal stuff", Source: "personal"},
		},
	}

	result := buildLanguageSection([]string{"go"}, files, false)

	// Personal additions should be ### (under H2 Go)
	if !strings.Contains(result, "### Personal Additions") {
		t.Errorf("Personal additions should be H3, got:\n%s", result)
	}

	// Personal content headers should also be demoted
	if !strings.Contains(result, "### My Preferences") {
		t.Errorf("Personal content headers should be demoted to H3, got:\n%s", result)
	}
}
