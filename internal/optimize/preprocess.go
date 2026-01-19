package optimize

import (
	"regexp"
	"strings"
)

// PreprocessStats tracks what changes were made during preprocessing.
type PreprocessStats struct {
	BlankLinesRemoved int
	DuplicatesRemoved int
	PhrasesStripped   int
}

// Preprocess performs deterministic cleanup on content without using an LLM.
// It normalizes whitespace, removes duplicate rules, and strips verbose phrases.
func Preprocess(content string) (string, PreprocessStats) {
	var stats PreprocessStats

	// Step 1: Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Step 2: Collapse multiple blank lines into double newlines
	blanksBefore := countBlankLines(content)
	content = collapseBlankLines(content)
	blanksAfter := countBlankLines(content)
	stats.BlankLinesRemoved = blanksBefore - blanksAfter

	// Step 3: Remove duplicate bullet points within sections
	content, stats.DuplicatesRemoved = removeDuplicateBullets(content)

	// Step 4: Strip verbose phrases
	content, stats.PhrasesStripped = stripVerbosePhrases(content)

	// Step 5: Trim trailing whitespace from lines
	content = trimTrailingWhitespace(content)

	// Step 6: Ensure single trailing newline
	content = strings.TrimRight(content, "\n") + "\n"

	return content, stats
}

// collapseBlankLines reduces multiple consecutive blank lines to a single blank line.
func collapseBlankLines(content string) string {
	// Match 3+ consecutive newlines and replace with 2 newlines (one blank line)
	re := regexp.MustCompile(`\n{3,}`)
	return re.ReplaceAllString(content, "\n\n")
}

// countBlankLines counts the number of blank lines in content.
func countBlankLines(content string) int {
	lines := strings.Split(content, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			count++
		}
	}
	return count
}

// removeDuplicateBullets removes exact duplicate bullet points within each section.
func removeDuplicateBullets(content string) (string, int) {
	lines := strings.Split(content, "\n")
	var result []string
	seen := make(map[string]bool)
	duplicatesRemoved := 0
	inSection := false
	currentSection := ""

	bulletPattern := regexp.MustCompile(`^(\s*[-*+]\s+)(.+)$`)

	for _, line := range lines {
		// Check if this is a new section header
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			// Reset seen bullets for new section
			seen = make(map[string]bool)
			inSection = true
			currentSection = line
			result = append(result, line)
			continue
		}

		// Check if this is a bullet point
		if matches := bulletPattern.FindStringSubmatch(line); matches != nil && inSection {
			bulletContent := strings.TrimSpace(matches[2])
			key := currentSection + ":" + bulletContent

			if seen[key] {
				duplicatesRemoved++
				continue
			}
			seen[key] = true
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n"), duplicatesRemoved
}

// verbosePhrases are common filler phrases that add no value.
var verbosePhrases = []string{
	"You should always ",
	"You should ",
	"Always make sure to ",
	"Make sure to ",
	"Make sure that ",
	"Please make sure to ",
	"Please ensure that ",
	"Please ensure ",
	"It is important to ",
	"It's important to ",
	"Remember to always ",
	"Remember to ",
	"Be sure to ",
	"Don't forget to ",
}

// stripVerbosePhrases removes common filler phrases from the start of sentences.
func stripVerbosePhrases(content string) (string, int) {
	count := 0
	lines := strings.Split(content, "\n")
	var result []string

	bulletPattern := regexp.MustCompile(`^(\s*[-*+]\s+)`)

	for _, line := range lines {
		modified := line
		strippedThisLine := false

		for _, phrase := range verbosePhrases {
			if strippedThisLine {
				break // Only strip one phrase per line
			}

			lowerPhrase := strings.ToLower(phrase)

			// Check for bullet point first (more specific case)
			if matches := bulletPattern.FindStringSubmatch(modified); matches != nil {
				prefix := matches[1]
				rest := modified[len(prefix):]
				lowerRest := strings.ToLower(rest)

				if strings.HasPrefix(lowerRest, lowerPhrase) {
					remaining := rest[len(phrase):]
					if len(remaining) > 0 {
						modified = prefix + strings.ToUpper(string(remaining[0])) + remaining[1:]
						count++
						strippedThisLine = true
					}
				}
			} else {
				// Check for phrase at start of non-bullet line
				lowerLine := strings.ToLower(modified)
				if strings.HasPrefix(lowerLine, lowerPhrase) {
					remaining := modified[len(phrase):]
					if len(remaining) > 0 {
						modified = strings.ToUpper(string(remaining[0])) + remaining[1:]
						count++
						strippedThisLine = true
					}
				}
			}
		}
		result = append(result, modified)
	}

	return strings.Join(result, "\n"), count
}

// trimTrailingWhitespace removes trailing spaces/tabs from each line.
func trimTrailingWhitespace(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}
