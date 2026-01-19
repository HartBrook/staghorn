package optimize

import (
	"fmt"
	"strings"
)

// buildSystemPrompt creates the system prompt for optimization.
func buildSystemPrompt() string {
	return `You are a technical documentation optimizer. Your task is to compress CLAUDE.md configuration files while preserving their semantic meaning.

GOALS:
1. Preserve ALL actionable guidance - nothing should be lost
2. Eliminate redundancy and verbose explanations
3. Consolidate related rules into concise statements
4. Remove generic advice Claude already knows
5. Keep project-specific and non-obvious guidance

RULES:
- Never remove specific tool names, file paths, or commands
- Never remove project-specific conventions or patterns
- Combine similar rules: "use black" + "use isort" + "use ruff" → "Format with black, isort, ruff"
- Remove truisms: "write clean code" adds nothing
- Preserve structure (headers) but consolidate within sections
- Use bullet points for lists instead of paragraphs
- Keep code examples intact

OUTPUT FORMAT:
Return only the optimized markdown. No explanations or commentary.`
}

// buildUserPrompt creates the user prompt for optimization.
func buildUserPrompt(content string, targetTokens int, anchors []string) string {
	currentTokens := CountTokens(content)

	var anchorsSection string
	if len(anchors) > 0 {
		anchorsSection = fmt.Sprintf("\nCRITICAL ANCHORS (must be preserved exactly):\n%s\n", strings.Join(anchors, ", "))
	}

	return fmt.Sprintf(`Optimize the following CLAUDE.md configuration to approximately %d tokens while preserving all semantic meaning.

Current token count: %d
Target token count: %d
%s
CONTENT TO OPTIMIZE:
---
%s
---

Output ONLY the optimized markdown.`, targetTokens, currentTokens, targetTokens, anchorsSection, content)
}
