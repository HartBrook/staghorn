package integration

import (
	"slices"
	"strings"
	"testing"

	"github.com/HartBrook/staghorn/internal/merge"
)

// Asserter provides assertion helpers for CLAUDE.md output.
type Asserter struct {
	t       *testing.T
	content string
}

// NewAsserter creates an asserter for the given content.
func NewAsserter(t *testing.T, content string) *Asserter {
	return &Asserter{t: t, content: content}
}

// HasManagedHeader checks for "<!-- Managed by staghorn" header.
func (a *Asserter) HasManagedHeader() bool {
	return strings.Contains(a.content, merge.HeaderManagedPrefix)
}

// HasSourceRepo checks if the header contains the expected source repo.
func (a *Asserter) HasSourceRepo(repo string) bool {
	return strings.Contains(a.content, "Source: "+repo)
}

// HasProvenanceMarker checks for a specific provenance marker.
func (a *Asserter) HasProvenanceMarker(source string) bool {
	marker := "<!-- staghorn:source:" + source + " -->"
	return strings.Contains(a.content, marker)
}

// ProvenanceOrder returns the layers in order of appearance.
func (a *Asserter) ProvenanceOrder() []string {
	return merge.ListLayers(a.content)
}

// ContainsSection checks if a ## section header exists.
func (a *Asserter) ContainsSection(header string) bool {
	return strings.Contains(a.content, header)
}

// ContainsText checks if the content contains a substring.
func (a *Asserter) ContainsText(text string) bool {
	return strings.Contains(a.content, text)
}

// GetContent returns the raw content.
func (a *Asserter) GetContent() string {
	return a.content
}

// RunAssertions runs all assertions from a fixture definition.
func (a *Asserter) RunAssertions(assertions FixtureAssertions) {
	a.t.Helper()

	// Check header
	if assertions.Header != nil {
		if assertions.Header.ManagedBy {
			if !a.HasManagedHeader() {
				a.t.Error("expected managed header, but not found")
			}
		}
		if assertions.Header.SourceRepo != "" {
			if !a.HasSourceRepo(assertions.Header.SourceRepo) {
				a.t.Errorf("expected source repo %q in header, but not found", assertions.Header.SourceRepo)
			}
		}
	}

	// Check provenance markers
	if assertions.Provenance != nil {
		if assertions.Provenance.HasTeam {
			if !a.HasProvenanceMarker("team") {
				a.t.Error("expected team provenance marker, but not found")
			}
		}
		if assertions.Provenance.HasPersonal {
			if !a.HasProvenanceMarker("personal") {
				a.t.Error("expected personal provenance marker, but not found")
			}
		}
		if len(assertions.Provenance.Order) > 0 {
			actual := a.ProvenanceOrder()
			if !slices.Equal(assertions.Provenance.Order, actual) {
				a.t.Errorf("expected provenance order %v, got %v", assertions.Provenance.Order, actual)
			}
		}
	}

	// Check contains
	for _, text := range assertions.Contains {
		if !a.ContainsText(text) {
			a.t.Errorf("expected content to contain %q, but not found", text)
		}
	}

	// Check not contains
	for _, text := range assertions.NotContains {
		if a.ContainsText(text) {
			a.t.Errorf("expected content NOT to contain %q, but found", text)
		}
	}

	// Check sections
	for _, section := range assertions.Sections {
		if !a.ContainsSection(section) {
			a.t.Errorf("expected section %q, but not found", section)
		}
	}

	// Check language sections
	for _, langCheck := range assertions.Languages {
		a.checkLanguage(langCheck)
	}
}

// checkLanguage verifies a language section.
func (a *Asserter) checkLanguage(check LanguageCheck) {
	a.t.Helper()

	if check.HasTeamContent {
		marker := "<!-- staghorn:source:team:" + check.Name + " -->"
		if !strings.Contains(a.content, marker) {
			a.t.Errorf("expected team content for language %q, but not found", check.Name)
		}
	}

	if check.HasPersonalContent {
		marker := "<!-- staghorn:source:personal:" + check.Name + " -->"
		if !strings.Contains(a.content, marker) {
			a.t.Errorf("expected personal content for language %q, but not found", check.Name)
		}
	}

	for _, text := range check.Contains {
		if !strings.Contains(a.content, text) {
			a.t.Errorf("expected language %q to contain %q, but not found", check.Name, text)
		}
	}
}
