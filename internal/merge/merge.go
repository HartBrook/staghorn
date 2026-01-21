package merge

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/HartBrook/staghorn/internal/language"
)

// Layer represents a config layer with its source.
type Layer struct {
	Content string
	Source  string // "team" | "personal" | "project"
}

// MergeOptions controls merge behavior.
type MergeOptions struct {
	AnnotateSources bool                                // Add source comments
	SourceRepo      string                              // For header annotation (e.g., "acme/standards")
	Languages       []string                            // Active languages to include
	LanguageFiles   map[string][]*language.LanguageFile // Language files by language ID
}

// Merge combines layers into a single document.
// Order: team (base) -> personal -> project
// Returns the merged markdown content.
func Merge(layers []Layer, opts MergeOptions) string {
	if len(layers) == 0 {
		return ""
	}

	// Start with the first non-empty layer as base
	var baseDoc *Document
	var baseSource string
	for _, layer := range layers {
		if strings.TrimSpace(layer.Content) != "" {
			baseDoc = Parse(layer.Content)
			baseSource = layer.Source
			// Mark all base sections with their source
			for i := range baseDoc.Sections {
				baseDoc.Sections[i].Source = baseSource
			}
			break
		}
	}

	if baseDoc == nil {
		return ""
	}

	// Merge additional layers into base
	for _, layer := range layers {
		if layer.Source == baseSource || strings.TrimSpace(layer.Content) == "" {
			continue
		}
		mergeLayer(baseDoc, layer, opts.AnnotateSources)
	}

	// Render the merged document
	return render(baseDoc, opts, baseSource)
}

// mergeLayer merges a layer into the base document.
func mergeLayer(base *Document, layer Layer, annotate bool) {
	doc := Parse(layer.Content)
	label := formatAdditionLabel(layer.Source)

	// Merge sections
	for _, section := range doc.Sections {
		// Skip sections with no meaningful content
		if strings.TrimSpace(section.Content) == "" {
			continue
		}

		existingSection := base.FindSection(section.Header)
		if existingSection != nil {
			// Append to existing section with sub-header and source annotation
			existingSection.Content = appendWithSubHeader(
				existingSection.Content,
				section.Content,
				label,
				layer.Source,
				annotate,
			)
		} else {
			// Add as new section with source
			base.Sections = append(base.Sections, Section{
				Header:  section.Header,
				Content: section.Content,
				Source:  layer.Source,
			})
		}
	}
}

// formatAdditionLabel returns the sub-header label for a source.
func formatAdditionLabel(source string) string {
	switch source {
	case "personal":
		return "Personal Additions"
	case "project":
		return "Project Additions"
	default:
		return titleCase(source) + " Additions"
	}
}

// titleCase capitalizes the first letter of a string.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// appendWithSubHeader appends content under a ### sub-header.
// If annotate is true, adds a source provenance comment before the addition.
func appendWithSubHeader(base, addition, label, source string, annotate bool) string {
	if strings.TrimSpace(addition) == "" {
		return base
	}
	if annotate {
		return fmt.Sprintf("%s\n\n%s\n### %s\n\n%s",
			base,
			sourceStartComment(source),
			label,
			addition,
		)
	}
	return fmt.Sprintf("%s\n\n### %s\n\n%s", base, label, addition)
}

// sourceStartComment returns the provenance comment for a source.
// If language is non-empty, creates a language-specific marker (e.g., "team:python").
func sourceStartComment(source string) string {
	return fmt.Sprintf("<!-- staghorn:source:%s -->", source)
}

// sourceLanguageComment returns the provenance comment for a language-specific source.
func sourceLanguageComment(source, language string) string {
	return fmt.Sprintf("<!-- staghorn:source:%s:%s -->", source, language)
}

// headerRegex matches markdown headers (## Header, ### Header, etc.)
var headerRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

// demoteHeaders shifts all markdown headers down one level (## becomes ###, etc.)
// Headers at H6 remain at H6 (can't go deeper).
func demoteHeaders(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if match := headerRegex.FindStringSubmatch(line); match != nil {
			hashes := match[1]
			text := match[2]
			if len(hashes) < 6 {
				lines[i] = "#" + hashes + " " + text
			}
		}
	}
	return strings.Join(lines, "\n")
}

// render converts a Document back to markdown.
// baseSource is the source of the base layer (used for section provenance).
func render(doc *Document, opts MergeOptions, baseSource string) string {
	var b strings.Builder

	// Header comment
	if opts.AnnotateSources {
		timestamp := time.Now().Format("2006-01-02")
		if opts.SourceRepo != "" {
			b.WriteString(fmt.Sprintf("%s | Source: %s | Do not edit directly | %s -->\n\n", HeaderManagedPrefix, opts.SourceRepo, timestamp))
		} else {
			b.WriteString(fmt.Sprintf("%s | Do not edit directly | %s -->\n\n", HeaderManagedPrefix, timestamp))
		}
	}

	// Preamble (from base source)
	if doc.Preamble != "" {
		if opts.AnnotateSources && baseSource != "" {
			b.WriteString(sourceStartComment(baseSource))
			b.WriteString("\n")
		}
		b.WriteString(doc.Preamble)
		b.WriteString("\n\n")
	}

	// Sections - only emit source marker when source changes
	currentSource := ""
	if doc.Preamble != "" && baseSource != "" {
		currentSource = baseSource // preamble already set the source
	}

	for i, section := range doc.Sections {
		source := section.Source
		if source == "" {
			source = baseSource
		}

		// Only add source annotation when source changes
		if opts.AnnotateSources && source != "" && source != currentSource {
			b.WriteString(sourceStartComment(source))
			b.WriteString("\n")
			currentSource = source
		}

		b.WriteString(fmt.Sprintf("## %s\n\n", section.Header))
		b.WriteString(section.Content)

		if i < len(doc.Sections)-1 {
			b.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(b.String())
}

// MergeSimple merges multiple markdown contents without layer metadata.
// Useful for simple cases where source tracking isn't needed.
func MergeSimple(contents ...string) string {
	layers := make([]Layer, len(contents))
	sources := []string{"team", "personal", "project"}

	for i, content := range contents {
		source := "unknown"
		if i < len(sources) {
			source = sources[i]
		}
		layers[i] = Layer{Content: content, Source: source}
	}

	return Merge(layers, MergeOptions{})
}

// MergeWithLanguages performs the full merge including language configs.
// First merges base layers, then appends a language-specific section.
func MergeWithLanguages(layers []Layer, opts MergeOptions) string {
	// First, merge the base layers (team -> personal -> project)
	baseResult := Merge(layers, opts)

	if len(opts.Languages) == 0 || len(opts.LanguageFiles) == 0 {
		return baseResult
	}

	// Build and append language section
	languageSection := buildLanguageSection(opts.Languages, opts.LanguageFiles, opts.AnnotateSources)
	if languageSection == "" {
		return baseResult
	}

	return baseResult + "\n\n" + languageSection
}

// buildLanguageSection creates language sections as top-level H2 headers.
// Each language becomes its own ## section with content headers demoted one level.
func buildLanguageSection(languages []string, files map[string][]*language.LanguageFile, annotate bool) string {
	if len(files) == 0 {
		return ""
	}

	// Sort languages for consistent output
	sortedLangs := make([]string, 0, len(languages))
	for _, lang := range languages {
		if _, ok := files[lang]; ok {
			sortedLangs = append(sortedLangs, lang)
		}
	}
	sort.Strings(sortedLangs)

	if len(sortedLangs) == 0 {
		return ""
	}

	var b strings.Builder

	for langIdx, lang := range sortedLangs {
		langFiles := files[lang]
		if len(langFiles) == 0 {
			continue
		}

		// Add spacing between language sections
		if langIdx > 0 {
			b.WriteString("\n\n")
		}

		// Each language is a top-level H2 section
		displayName := language.GetDisplayName(lang)
		b.WriteString(fmt.Sprintf("## %s\n\n", displayName))

		// Merge language files from all layers
		for i, file := range langFiles {
			content := strings.TrimSpace(file.Content)
			if content == "" {
				continue
			}

			// Demote all headers in content by one level (## -> ###, etc.)
			content = demoteHeaders(content)

			// Add language-specific source marker for each layer's content
			if annotate {
				if i > 0 {
					b.WriteString("\n\n")
				}
				b.WriteString(sourceLanguageComment(file.Source, lang))
				b.WriteString("\n")
			}

			if i == 0 {
				// First layer (team) - write demoted content directly
				b.WriteString(content)
			} else {
				// Personal/project additions get sub-headers (### under the H2 language)
				label := formatAdditionLabel(file.Source)
				if !annotate {
					// No marker was written, need spacing
					b.WriteString("\n\n")
				}
				b.WriteString(fmt.Sprintf("### %s\n\n%s", label, content))
			}
		}
	}

	return strings.TrimSpace(b.String())
}
