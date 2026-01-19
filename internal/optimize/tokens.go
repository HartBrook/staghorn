// Package optimize provides LLM-powered config compression.
package optimize

import "unicode/utf8"

// CountTokens estimates the token count for content using runes/4 approximation.
// This provides a reasonable estimate for Claude's tokenizer without requiring
// external dependencies. Uses rune count (not byte count) to handle unicode correctly.
func CountTokens(content string) int {
	if len(content) == 0 {
		return 0
	}
	// Use rune count for accurate character counting with unicode
	runeCount := utf8.RuneCountInString(content)
	return runeCount / 4
}

// TokenStats holds before/after token statistics.
type TokenStats struct {
	Before int
	After  int
}

// Saved returns the number of tokens saved.
func (s TokenStats) Saved() int {
	return s.Before - s.After
}

// PercentReduction returns the percentage reduction (0-100).
func (s TokenStats) PercentReduction() float64 {
	if s.Before == 0 {
		return 0
	}
	return float64(s.Saved()) / float64(s.Before) * 100
}
