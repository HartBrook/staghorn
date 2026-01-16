package commands

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ClaudeCommand represents a Claude Code custom command.
type ClaudeCommand struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// ConvertToClaude converts a staghorn command to Claude Code command format.
// It adds a usage hint so Claude (the LLM) understands the {{varname}} syntax.
func ConvertToClaude(cmd *Command) string {
	var sb strings.Builder

	// Write frontmatter (simplified for Claude Code)
	sb.WriteString("---\n")
	claudeCmd := ClaudeCommand{
		Name:        cmd.Name,
		Description: cmd.Description,
	}
	yamlBytes, _ := yaml.Marshal(claudeCmd)
	sb.Write(yamlBytes)
	sb.WriteString("---\n\n")

	// Add staghorn header after frontmatter so it doesn't interfere with description display
	sb.WriteString(fmt.Sprintf("<!-- Managed by staghorn | Source: %s | Do not edit directly -->\n\n", cmd.Source.Label()))

	// Add args hint if there are arguments
	if len(cmd.Args) > 0 {
		sb.WriteString(buildArgsHint(cmd))
		sb.WriteString("\n")
	}

	// Add the body
	sb.WriteString(cmd.Body)
	sb.WriteString("\n")

	return sb.String()
}

// buildArgsHint creates a usage hint comment for Claude to understand the args.
func buildArgsHint(cmd *Command) string {
	var sb strings.Builder

	// Build args list
	var argParts []string
	for _, arg := range cmd.Args {
		part := arg.Name
		if arg.Required {
			part += " (required)"
		} else if arg.Default != "" {
			part += fmt.Sprintf(" (default: %s)", arg.Default)
		}
		argParts = append(argParts, part)
	}

	sb.WriteString(fmt.Sprintf("<!-- Args: %s -->\n", strings.Join(argParts, ", ")))

	// Build example usage
	var exampleParts []string
	for _, arg := range cmd.Args {
		val := arg.Default
		if val == "" {
			val = "<value>"
		}
		exampleParts = append(exampleParts, fmt.Sprintf("%s=%q", arg.Name, val))
	}
	sb.WriteString(fmt.Sprintf("<!-- Example: /%s %s -->\n", cmd.Name, strings.Join(exampleParts, " ")))

	return sb.String()
}
