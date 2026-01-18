package eval

import (
	"bytes"
	"fmt"
	"text/template"
)

// EvalTemplate represents a starter template for creating evals.
type EvalTemplate struct {
	Name        string // Template identifier (e.g., "security")
	Description string // Human-readable description
	Content     string // YAML template with Go template placeholders
}

// TemplateVars holds variables for template rendering.
type TemplateVars struct {
	Name        string
	Description string
	Tags        []string
}

// Templates contains all available eval templates.
var Templates = []EvalTemplate{
	{
		Name:        "security",
		Description: "Security-focused eval for testing security guidelines",
		Content: `name: {{.Name}}
description: {{.Description}}
tags:
{{- range .Tags}}
  - {{.}}
{{- end}}

tests:
  - name: warns-about-hardcoded-secrets
    description: Verifies Claude warns about hardcoded API keys and secrets
    prompt: |
      Review this code:
      API_KEY = "sk-12345abcdef"
      db_password = "super_secret_123"
    assert:
      - type: llm-rubric
        value: Response warns about hardcoded secrets and suggests using environment variables

  - name: identifies-injection-risks
    description: Verifies Claude identifies potential injection vulnerabilities
    prompt: |
      Review this code:
      query = f"SELECT * FROM users WHERE id = {user_input}"
    assert:
      - type: llm-rubric
        value: Response identifies SQL injection risk and suggests parameterized queries
`,
	},
	{
		Name:        "quality",
		Description: "Code quality eval for testing style and best practices",
		Content: `name: {{.Name}}
description: {{.Description}}
tags:
{{- range .Tags}}
  - {{.}}
{{- end}}

tests:
  - name: suggests-clear-naming
    description: Verifies Claude suggests better variable names
    prompt: |
      Review this code:
      def f(x, y, z):
          a = x + y
          b = a * z
          return b
    assert:
      - type: llm-rubric
        value: Response suggests more descriptive function and variable names

  - name: identifies-code-duplication
    description: Verifies Claude identifies duplicated code
    prompt: |
      Review this code:
      def process_user(user):
          if user.age > 18:
              print("Adult")
          return user.name.upper()

      def process_admin(admin):
          if admin.age > 18:
              print("Adult")
          return admin.name.upper()
    assert:
      - type: llm-rubric
        value: Response identifies code duplication and suggests refactoring
`,
	},
	{
		Name:        "language",
		Description: "Language-specific eval template",
		Content: `name: {{.Name}}
description: {{.Description}}
tags:
{{- range .Tags}}
  - {{.}}
{{- end}}

tests:
  - name: follows-language-conventions
    description: Verifies Claude follows language-specific conventions
    prompt: |
      Write a function that calculates the factorial of a number.
    assert:
      - type: llm-rubric
        value: Response follows idiomatic patterns for the target language

  - name: uses-appropriate-error-handling
    description: Verifies Claude uses proper error handling
    prompt: |
      Write code to read a file and parse its JSON contents.
    assert:
      - type: llm-rubric
        value: Response includes proper error handling for file and JSON operations
`,
	},
	{
		Name:        "blank",
		Description: "Minimal blank template to start from scratch",
		Content: `name: {{.Name}}
description: {{.Description}}
tags:
{{- range .Tags}}
  - {{.}}
{{- end}}

tests:
  - name: example-test
    description: Replace with your test description
    prompt: |
      Your prompt here
    assert:
      - type: llm-rubric
        value: Your assertion here
`,
	},
}

// GetTemplate returns a template by name.
func GetTemplate(name string) (*EvalTemplate, error) {
	for _, t := range Templates {
		if t.Name == name {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("template %q not found", name)
}

// ListTemplates returns all available template names.
func ListTemplates() []string {
	names := make([]string, len(Templates))
	for i, t := range Templates {
		names[i] = t.Name
	}
	return names
}

// RenderTemplate renders a template with the given variables.
func RenderTemplate(t *EvalTemplate, vars TemplateVars) (string, error) {
	// Ensure tags is not nil
	if vars.Tags == nil {
		vars.Tags = []string{}
	}

	tmpl, err := template.New(t.Name).Parse(t.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderTemplateByName gets a template by name and renders it.
func RenderTemplateByName(name string, vars TemplateVars) (string, error) {
	t, err := GetTemplate(name)
	if err != nil {
		return "", err
	}
	return RenderTemplate(t, vars)
}
