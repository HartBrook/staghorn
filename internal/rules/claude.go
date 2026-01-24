package rules

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConvertToClaude converts a staghorn rule to Claude Code rules format.
// It preserves the frontmatter (required for path-scoping) and adds a staghorn header.
func ConvertToClaude(rule *Rule) (string, error) {
	var sb strings.Builder

	// Preserve original frontmatter if present (required for path-scoping)
	if len(rule.Frontmatter.Paths) > 0 {
		sb.WriteString("---\n")
		yamlBytes, err := yaml.Marshal(rule.Frontmatter)
		if err != nil {
			return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
		}
		sb.Write(yamlBytes)
		sb.WriteString("---\n\n")
	}

	// Add staghorn header after frontmatter so it doesn't interfere with Claude's parsing
	sb.WriteString(fmt.Sprintf("<!-- Managed by staghorn | Source: %s | Do not edit directly -->\n\n", rule.Source.Label()))

	// Add the body
	sb.WriteString(rule.Body)
	sb.WriteString("\n")

	return sb.String(), nil
}
