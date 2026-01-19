package optimize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreprocess_NormalizesWhitespace(t *testing.T) {
	input := "Line 1\n\n\n\n\nLine 2\n\n\n\nLine 3"
	result, stats := Preprocess(input)

	// Should have at most one blank line between content
	assert.NotContains(t, result, "\n\n\n")
	assert.Greater(t, stats.BlankLinesRemoved, 0)
}

func TestPreprocess_RemovesDuplicateRules(t *testing.T) {
	input := `## Code Style

- Use meaningful variable names
- Follow PEP 8
- Use meaningful variable names
- Add type hints

## Testing

- Write unit tests
- Write unit tests
- Use pytest
`
	result, stats := Preprocess(input)

	// Count occurrences of "Use meaningful variable names"
	count := strings.Count(result, "Use meaningful variable names")
	assert.Equal(t, 1, count, "Should have only one instance of duplicate bullet")

	// Count occurrences of "Write unit tests"
	count = strings.Count(result, "Write unit tests")
	assert.Equal(t, 1, count, "Should have only one instance of duplicate bullet")

	assert.Equal(t, 2, stats.DuplicatesRemoved)
}

func TestPreprocess_StripsVerbosePhrases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "You should",
			input: "You should use type hints",
			want:  "Use type hints",
		},
		{
			name:  "Make sure to",
			input: "Make sure to handle errors",
			want:  "Handle errors",
		},
		{
			name:  "It is important to",
			input: "It is important to test your code",
			want:  "Test your code",
		},
		{
			name:  "in bullet point",
			input: "- You should always use descriptive names",
			want:  "- Use descriptive names",
		},
		{
			name:  "Remember to",
			input: "Remember to commit often",
			want:  "Commit often",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, stats := Preprocess(tt.input)
			result = strings.TrimSpace(result)
			assert.Equal(t, tt.want, result)
			assert.Greater(t, stats.PhrasesStripped, 0)
		})
	}
}

func TestPreprocess_PreservesStructure(t *testing.T) {
	input := `## Code Style

Keep functions focused.

### Naming

Use descriptive names.

## Testing

Write comprehensive tests.
`
	result, _ := Preprocess(input)

	// All headers should be preserved
	assert.Contains(t, result, "## Code Style")
	assert.Contains(t, result, "### Naming")
	assert.Contains(t, result, "## Testing")

	// Content should be preserved
	assert.Contains(t, result, "Keep functions focused")
	assert.Contains(t, result, "Use descriptive names")
	assert.Contains(t, result, "Write comprehensive tests")
}

func TestPreprocess_EmptyInput(t *testing.T) {
	result, stats := Preprocess("")

	assert.Equal(t, "\n", result)
	assert.Equal(t, 0, stats.BlankLinesRemoved)
	assert.Equal(t, 0, stats.DuplicatesRemoved)
	assert.Equal(t, 0, stats.PhrasesStripped)
}

func TestPreprocess_SingleLine(t *testing.T) {
	result, _ := Preprocess("Single line content")
	assert.Equal(t, "Single line content\n", result)
}

func TestPreprocessStats(t *testing.T) {
	input := `## Section


- Item 1
- Item 1
- You should do something



## Another


- Item 2
`
	_, stats := Preprocess(input)

	assert.Greater(t, stats.BlankLinesRemoved, 0, "Should remove extra blank lines")
	assert.Equal(t, 1, stats.DuplicatesRemoved, "Should remove duplicate item")
	assert.Equal(t, 1, stats.PhrasesStripped, "Should strip verbose phrase")
}

func TestPreprocess_PreservesCodeBlocks(t *testing.T) {
	input := "## Example\n\n```python\ndef hello():\n    print('world')\n```\n"
	result, _ := Preprocess(input)

	assert.Contains(t, result, "```python")
	assert.Contains(t, result, "def hello():")
	assert.Contains(t, result, "print('world')")
	assert.Contains(t, result, "```")
}

func TestPreprocess_TrimsTrailingWhitespace(t *testing.T) {
	input := "Line with trailing spaces   \nAnother line\t\t\n"
	result, _ := Preprocess(input)

	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if line != "" {
			assert.Equal(t, strings.TrimRight(line, " \t"), line)
		}
	}
}

func TestPreprocess_WindowsLineEndings(t *testing.T) {
	input := "Line 1\r\nLine 2\r\nLine 3\r\n"
	result, _ := Preprocess(input)

	// Should not contain carriage returns
	assert.NotContains(t, result, "\r")
	assert.Contains(t, result, "Line 1\nLine 2\nLine 3")
}

func TestCollapseBlankLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "three newlines to two",
			input: "a\n\n\nb",
			want:  "a\n\nb",
		},
		{
			name:  "five newlines to two",
			input: "a\n\n\n\n\nb",
			want:  "a\n\nb",
		},
		{
			name:  "two newlines unchanged",
			input: "a\n\nb",
			want:  "a\n\nb",
		},
		{
			name:  "single newline unchanged",
			input: "a\nb",
			want:  "a\nb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collapseBlankLines(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRemoveDuplicateBullets_DifferentSections(t *testing.T) {
	// Same bullet in different sections should NOT be removed
	input := `## Section A

- Common item

## Section B

- Common item
`
	result, duplicatesRemoved := removeDuplicateBullets(input)

	count := strings.Count(result, "Common item")
	assert.Equal(t, 2, count, "Same bullet in different sections should be kept")
	assert.Equal(t, 0, duplicatesRemoved)
}

func TestStripVerbosePhrases_OnlyStripsOnePhrasePerLine(t *testing.T) {
	// Even if a line matches multiple phrases, only one should be stripped
	// and count should only increment by 1
	input := "You should make sure to do something"
	result, count := stripVerbosePhrases(input)

	// Should strip "You should " but not try to strip again
	assert.Equal(t, "Make sure to do something", result)
	assert.Equal(t, 1, count, "Should only count one strip per line")
}

func TestStripVerbosePhrases_BulletLineAccurate(t *testing.T) {
	// A bullet line should only increment count once
	input := "- You should do this"
	result, count := stripVerbosePhrases(input)

	assert.Equal(t, "- Do this", result)
	assert.Equal(t, 1, count, "Bullet line should only count once")
}

func TestStripVerbosePhrases_MultipleLines(t *testing.T) {
	input := `You should test your code
- Make sure to handle errors
Remember to commit often`

	result, count := stripVerbosePhrases(input)

	assert.Contains(t, result, "Test your code")
	assert.Contains(t, result, "- Handle errors")
	assert.Contains(t, result, "Commit often")
	assert.Equal(t, 3, count, "Should strip exactly 3 phrases")
}
