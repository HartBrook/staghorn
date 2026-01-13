// Package merge handles CLAUDE.md section parsing and merging.
package merge

import (
	"regexp"
	"strings"
)

// Section represents an H2-delimited section of markdown.
type Section struct {
	Header  string // The H2 header text (without ##)
	Content string // Everything until the next H2
}

// Document represents a parsed markdown document.
type Document struct {
	Preamble string    // Content before the first H2
	Sections []Section // H2-delimited sections
}

// h2Pattern matches markdown H2 headers (## Header).
var h2Pattern = regexp.MustCompile(`(?m)^##\s+(.+)$`)

// Parse splits markdown into sections by H2 headers.
func Parse(content string) *Document {
	doc := &Document{
		Sections: []Section{},
	}

	if content == "" {
		return doc
	}

	// Find all H2 header positions
	matches := h2Pattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		// No H2 headers, entire content is preamble
		doc.Preamble = strings.TrimSpace(content)
		return doc
	}

	// Content before first H2 is the preamble
	doc.Preamble = strings.TrimSpace(content[:matches[0][0]])

	// Extract each section
	for i, match := range matches {
		headerEnd := match[1]
		headerTextStart := match[2]
		headerTextEnd := match[3]

		// Determine where this section's content ends
		var contentEnd int
		if i+1 < len(matches) {
			contentEnd = matches[i+1][0]
		} else {
			contentEnd = len(content)
		}

		// Extract the header text and content
		header := strings.TrimSpace(content[headerTextStart:headerTextEnd])
		sectionContent := strings.TrimSpace(content[headerEnd:contentEnd])

		doc.Sections = append(doc.Sections, Section{
			Header:  header,
			Content: sectionContent,
		})
	}

	return doc
}

// FindSection finds a section by header name (case-insensitive).
func (d *Document) FindSection(header string) *Section {
	headerLower := strings.ToLower(header)
	for i := range d.Sections {
		if strings.ToLower(d.Sections[i].Header) == headerLower {
			return &d.Sections[i]
		}
	}
	return nil
}

// HasSection checks if a section exists (case-insensitive).
func (d *Document) HasSection(header string) bool {
	return d.FindSection(header) != nil
}

// SectionHeaders returns all section header names.
func (d *Document) SectionHeaders() []string {
	headers := make([]string, len(d.Sections))
	for i, s := range d.Sections {
		headers[i] = s.Header
	}
	return headers
}
