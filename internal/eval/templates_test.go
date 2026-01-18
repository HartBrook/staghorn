package eval

import (
	"strings"
	"testing"
)

func TestGetTemplate_ValidName(t *testing.T) {
	for _, name := range []string{"security", "quality", "language", "blank"} {
		tmpl, err := GetTemplate(name)
		if err != nil {
			t.Errorf("GetTemplate(%q) returned error: %v", name, err)
			continue
		}
		if tmpl == nil {
			t.Errorf("GetTemplate(%q) returned nil template", name)
			continue
		}
		if tmpl.Name != name {
			t.Errorf("GetTemplate(%q).Name = %q, expected %q", name, tmpl.Name, name)
		}
		if tmpl.Description == "" {
			t.Errorf("GetTemplate(%q) has empty description", name)
		}
		if tmpl.Content == "" {
			t.Errorf("GetTemplate(%q) has empty content", name)
		}
	}
}

func TestGetTemplate_InvalidName(t *testing.T) {
	tmpl, err := GetTemplate("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
	if tmpl != nil {
		t.Error("expected nil template for nonexistent name")
	}
}

func TestListTemplates(t *testing.T) {
	names := ListTemplates()
	if len(names) == 0 {
		t.Error("expected at least one template")
	}

	// Check that expected templates are present
	expected := []string{"security", "quality", "language", "blank"}
	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected template %q in list", exp)
		}
	}
}

func TestRenderTemplate_WithVars(t *testing.T) {
	tmpl, err := GetTemplate("blank")
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}

	vars := TemplateVars{
		Name:        "my-custom-eval",
		Description: "My custom evaluation",
		Tags:        []string{"custom", "test"},
	}

	result, err := RenderTemplate(tmpl, vars)
	if err != nil {
		t.Fatalf("RenderTemplate failed: %v", err)
	}

	// Check that variables were substituted
	if !strings.Contains(result, "name: my-custom-eval") {
		t.Error("expected name to be substituted")
	}
	if !strings.Contains(result, "description: My custom evaluation") {
		t.Error("expected description to be substituted")
	}
	if !strings.Contains(result, "- custom") {
		t.Error("expected custom tag to be present")
	}
	if !strings.Contains(result, "- test") {
		t.Error("expected test tag to be present")
	}
}

func TestRenderTemplate_EmptyTags(t *testing.T) {
	tmpl, err := GetTemplate("blank")
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}

	vars := TemplateVars{
		Name:        "my-eval",
		Description: "Test eval",
		Tags:        nil, // No tags
	}

	result, err := RenderTemplate(tmpl, vars)
	if err != nil {
		t.Fatalf("RenderTemplate failed: %v", err)
	}

	// Should still produce valid YAML with empty tags
	if !strings.Contains(result, "name: my-eval") {
		t.Error("expected name to be substituted")
	}
}

func TestRenderTemplateByName(t *testing.T) {
	vars := TemplateVars{
		Name:        "test-eval",
		Description: "Test description",
		Tags:        []string{"test"},
	}

	result, err := RenderTemplateByName("security", vars)
	if err != nil {
		t.Fatalf("RenderTemplateByName failed: %v", err)
	}

	if !strings.Contains(result, "name: test-eval") {
		t.Error("expected name to be substituted")
	}
}

func TestRenderTemplateByName_InvalidTemplate(t *testing.T) {
	vars := TemplateVars{
		Name:        "test-eval",
		Description: "Test description",
	}

	_, err := RenderTemplateByName("nonexistent", vars)
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
}

func TestAllTemplates_AreValid(t *testing.T) {
	vars := TemplateVars{
		Name:        "test-eval",
		Description: "Test eval for validation",
		Tags:        []string{"test"},
	}

	for _, tmpl := range Templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			// Render the template
			rendered, err := RenderTemplate(&tmpl, vars)
			if err != nil {
				t.Fatalf("RenderTemplate failed for %q: %v", tmpl.Name, err)
			}

			// Parse the rendered YAML as an eval
			eval, err := Parse(rendered, SourcePersonal, "test.yaml")
			if err != nil {
				t.Fatalf("Parse failed for rendered template %q: %v\nRendered:\n%s", tmpl.Name, err, rendered)
			}

			// Validate the parsed eval
			errors := eval.Validate()
			if HasErrors(errors) {
				t.Errorf("Validation errors for template %q:", tmpl.Name)
				for _, e := range errors {
					if e.Level == ValidationLevelError {
						t.Errorf("  %s: %s", e.Field, e.Message)
					}
				}
			}
		})
	}
}

func TestAllTemplates_HaveRequiredFields(t *testing.T) {
	for _, tmpl := range Templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			if tmpl.Name == "" {
				t.Error("template has empty name")
			}
			if tmpl.Description == "" {
				t.Error("template has empty description")
			}
			if tmpl.Content == "" {
				t.Error("template has empty content")
			}

			// Check that template contains required placeholders
			if !strings.Contains(tmpl.Content, "{{.Name}}") {
				t.Error("template missing {{.Name}} placeholder")
			}
			if !strings.Contains(tmpl.Content, "{{.Description}}") {
				t.Error("template missing {{.Description}} placeholder")
			}
		})
	}
}
