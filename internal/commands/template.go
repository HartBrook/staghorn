package commands

import (
	"fmt"
	"regexp"
	"strings"
)

// templateVarPattern matches {{varname}} in templates.
// This is a user-friendly syntax that doesn't require the Go template dot prefix.
var templateVarPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

// Render renders a command's body with the provided arguments.
// Uses a simple {{varname}} syntax for user-friendliness.
func (c *Command) Render(args map[string]string) (string, error) {
	// Validate args first
	if err := c.ValidateArgs(args); err != nil {
		return "", err
	}

	// Build complete args map with defaults
	fullArgs := make(map[string]string)
	for _, arg := range c.Args {
		if arg.Default != "" {
			fullArgs[arg.Name] = arg.Default
		}
	}
	for k, v := range args {
		fullArgs[k] = v
	}

	// Replace template variables
	result := templateVarPattern.ReplaceAllStringFunc(c.Body, func(match string) string {
		// Extract variable name from {{varname}}
		varName := match[2 : len(match)-2]

		if val, ok := fullArgs[varName]; ok {
			return val
		}

		// Leave unrecognized variables as-is
		return match
	})

	return result, nil
}

// RenderWithValidation renders and also returns warnings for undefined variables.
func (c *Command) RenderWithValidation(args map[string]string) (string, []string, error) {
	// Validate args first
	if err := c.ValidateArgs(args); err != nil {
		return "", nil, err
	}

	// Build complete args map with defaults
	fullArgs := make(map[string]string)
	for _, arg := range c.Args {
		if arg.Default != "" {
			fullArgs[arg.Name] = arg.Default
		}
	}
	for k, v := range args {
		fullArgs[k] = v
	}

	// Find all template variables in the body
	matches := templateVarPattern.FindAllStringSubmatch(c.Body, -1)
	definedVars := make(map[string]bool)
	for _, arg := range c.Args {
		definedVars[arg.Name] = true
	}

	var warnings []string
	seenWarnings := make(map[string]bool)
	for _, match := range matches {
		varName := match[1]
		if !definedVars[varName] && !seenWarnings[varName] {
			seenWarnings[varName] = true
			warnings = append(warnings, fmt.Sprintf("undefined variable {{%s}} in template", varName))
		}
	}

	// Replace template variables
	result := templateVarPattern.ReplaceAllStringFunc(c.Body, func(match string) string {
		varName := match[2 : len(match)-2]
		if val, ok := fullArgs[varName]; ok {
			return val
		}
		return match
	})

	return result, warnings, nil
}

// ExtractVariables returns all variable names used in the command body.
func (c *Command) ExtractVariables() []string {
	matches := templateVarPattern.FindAllStringSubmatch(c.Body, -1)
	seen := make(map[string]bool)
	var vars []string

	for _, match := range matches {
		varName := match[1]
		if !seen[varName] {
			seen[varName] = true
			vars = append(vars, varName)
		}
	}

	return vars
}

// ParseArgs parses command-line style arguments into a map.
// Supports formats:
//   - --name value
//   - --name=value
//   - name=value
func ParseArgs(rawArgs []string) (map[string]string, error) {
	args := make(map[string]string)

	for i := 0; i < len(rawArgs); i++ {
		arg := rawArgs[i]

		// Handle --name=value format
		if strings.HasPrefix(arg, "--") {
			arg = arg[2:] // Remove --
			if idx := strings.Index(arg, "="); idx != -1 {
				args[arg[:idx]] = arg[idx+1:]
				continue
			}

			// Handle --name value format
			if i+1 < len(rawArgs) && !strings.HasPrefix(rawArgs[i+1], "--") {
				args[arg] = rawArgs[i+1]
				i++
				continue
			}

			return nil, fmt.Errorf("missing value for argument --%s", arg)
		}

		// Handle name=value format
		if idx := strings.Index(arg, "="); idx != -1 {
			args[arg[:idx]] = arg[idx+1:]
			continue
		}

		return nil, fmt.Errorf("invalid argument format: %s (use --name=value or name=value)", arg)
	}

	return args, nil
}

// FormatArgsHelp formats the arguments help text for a command.
func (c *Command) FormatArgsHelp() string {
	if len(c.Args) == 0 {
		return "  No arguments"
	}

	var lines []string
	for _, arg := range c.Args {
		line := fmt.Sprintf("  --%s", arg.Name)

		if arg.Required {
			line += " (required)"
		} else if arg.Default != "" {
			line += fmt.Sprintf(" (default: %q)", arg.Default)
		}

		if arg.Description != "" {
			line += "\n      " + arg.Description
		}

		if len(arg.Options) > 0 {
			line += fmt.Sprintf("\n      Options: %s", strings.Join(arg.Options, ", "))
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
