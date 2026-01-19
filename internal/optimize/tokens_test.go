package optimize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountTokens(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "simple text",
			content: "hello world",
			want:    2, // 11 chars / 4 = 2
		},
		{
			name:    "longer text",
			content: "This is a longer piece of text that should have more tokens.",
			want:    15, // 60 chars / 4 = 15
		},
		{
			name:    "markdown content",
			content: "## Header\n\n- Item 1\n- Item 2\n",
			want:    7, // 29 chars / 4 = 7
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountTokens(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountTokens_EmptyString(t *testing.T) {
	assert.Equal(t, 0, CountTokens(""))
}

func TestCountTokens_Unicode(t *testing.T) {
	// Unicode characters should be counted as single runes, not bytes
	content := "Hello 世界 emoji 🎉"
	tokens := CountTokens(content)
	assert.Greater(t, tokens, 0)

	// "Hello 世界 emoji 🎉" = 17 runes (including spaces)
	// 17 / 4 = 4 tokens
	assert.Equal(t, 4, tokens)
}

func TestCountTokens_UnicodeAccuracy(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "chinese characters",
			content: "你好世界", // 4 runes, should be 1 token
			want:    1,
		},
		{
			name:    "emoji",
			content: "🎉🎊🎁🎈", // 4 runes, should be 1 token
			want:    1,
		},
		{
			name:    "mixed content",
			content: "Hello 世界!", // 9 runes, should be 2 tokens
			want:    2,
		},
		{
			name:    "cyrillic",
			content: "Привет мир", // 10 runes, should be 2 tokens
			want:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountTokens(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountTokens_LargeContent(t *testing.T) {
	// 4000 chars should give us ~1000 tokens
	content := strings.Repeat("word ", 800) // 4000 chars
	tokens := CountTokens(content)
	assert.Equal(t, 1000, tokens)
}

func TestTokenStats_Saved(t *testing.T) {
	stats := TokenStats{Before: 1000, After: 600}
	assert.Equal(t, 400, stats.Saved())
}

func TestTokenStats_Saved_NoReduction(t *testing.T) {
	stats := TokenStats{Before: 1000, After: 1000}
	assert.Equal(t, 0, stats.Saved())
}

func TestTokenStats_PercentReduction(t *testing.T) {
	tests := []struct {
		name  string
		stats TokenStats
		want  float64
	}{
		{
			name:  "50% reduction",
			stats: TokenStats{Before: 1000, After: 500},
			want:  50.0,
		},
		{
			name:  "no reduction",
			stats: TokenStats{Before: 1000, After: 1000},
			want:  0.0,
		},
		{
			name:  "full reduction",
			stats: TokenStats{Before: 1000, After: 0},
			want:  100.0,
		},
		{
			name:  "zero before",
			stats: TokenStats{Before: 0, After: 0},
			want:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.stats.PercentReduction()
			assert.Equal(t, tt.want, got)
		})
	}
}
