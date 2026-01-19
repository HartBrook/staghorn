package optimize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt(t *testing.T) {
	prompt := buildSystemPrompt()

	// Should contain key instructions (case-insensitive checks where needed)
	assert.Contains(t, prompt, "technical documentation optimizer")
	assert.Contains(t, prompt, "CLAUDE.md")
	assert.Contains(t, strings.ToLower(prompt), "preserve")
	assert.Contains(t, prompt, "RULES")
	assert.Contains(t, prompt, "OUTPUT FORMAT")
}

func TestBuildSystemPrompt_ContainsRules(t *testing.T) {
	prompt := buildSystemPrompt()

	// Should contain key optimization rules
	rules := []string{
		"tool names",
		"file paths",
		"commands",
		"bullet points",
		"code examples",
	}

	for _, rule := range rules {
		assert.Contains(t, strings.ToLower(prompt), rule,
			"System prompt should mention '%s'", rule)
	}
}

func TestBuildUserPrompt(t *testing.T) {
	content := "## Testing\n\nUse pytest for testing."
	targetTokens := 50
	anchors := []string{"pytest"}

	prompt := buildUserPrompt(content, targetTokens, anchors)

	// Should contain the content
	assert.Contains(t, prompt, content)

	// Should contain target token count
	assert.Contains(t, prompt, "50")

	// Should contain anchors
	assert.Contains(t, prompt, "pytest")
	assert.Contains(t, prompt, "CRITICAL ANCHORS")
}

func TestBuildUserPrompt_NoAnchors(t *testing.T) {
	content := "## Testing\n\nWrite tests."
	targetTokens := 50

	prompt := buildUserPrompt(content, targetTokens, nil)

	// Should contain the content
	assert.Contains(t, prompt, content)

	// Should NOT contain anchors section when no anchors
	assert.NotContains(t, prompt, "CRITICAL ANCHORS")
}

func TestBuildUserPrompt_MultipleAnchors(t *testing.T) {
	content := "## Testing\n\nUse pytest with ruff and black."
	targetTokens := 100
	anchors := []string{"pytest", "ruff", "black", "conftest.py"}

	prompt := buildUserPrompt(content, targetTokens, anchors)

	// All anchors should be listed
	for _, anchor := range anchors {
		assert.Contains(t, prompt, anchor)
	}
}

func TestBuildUserPrompt_TokenCounts(t *testing.T) {
	content := strings.Repeat("word ", 100) // ~500 chars = ~125 tokens
	targetTokens := 50

	prompt := buildUserPrompt(content, targetTokens, nil)

	// Should show current token count
	assert.Contains(t, prompt, "Current token count:")

	// Should show target token count
	assert.Contains(t, prompt, "Target token count: 50")
}
