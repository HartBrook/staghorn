package eval

import (
	"testing"
)

func TestValidate_ValidEval(t *testing.T) {
	e := &Eval{
		Name:        "test-eval",
		Description: "A test eval",
		Tags:        []string{"security", "test"},
		Tests: []Test{
			{
				Name:        "test-case",
				Description: "A test case",
				Prompt:      "Review this code",
				Assert: []Assertion{
					{Type: "llm-rubric", Value: "Response is helpful"},
				},
			},
		},
	}

	errors := e.Validate()
	if HasErrors(errors) {
		t.Errorf("expected no errors for valid eval, got: %v", errors)
	}
}

func TestValidate_InvalidAssertionType(t *testing.T) {
	e := &Eval{
		Name:        "test-eval",
		Description: "A test eval",
		Tests: []Test{
			{
				Name:        "test-case",
				Description: "A test case",
				Prompt:      "Review this code",
				Assert: []Assertion{
					{Type: "llm_rubric", Value: "Response is helpful"},
				},
			},
		},
	}

	errors := e.Validate()
	if !HasErrors(errors) {
		t.Error("expected error for invalid assertion type")
	}

	// Check for suggestion
	found := false
	for _, err := range errors {
		if err.Level == ValidationLevelError && err.Field == "tests[0].assert[0].type" {
			found = true
			if err.Message == "" {
				t.Error("expected error message")
			}
			// Should suggest llm-rubric
			if !contains(err.Message, "llm-rubric") {
				t.Errorf("expected suggestion for llm-rubric, got: %s", err.Message)
			}
		}
	}
	if !found {
		t.Error("expected error for tests[0].assert[0].type")
	}
}

func TestValidate_MissingName(t *testing.T) {
	e := &Eval{
		Description: "A test eval",
		Tests: []Test{
			{
				Name:   "test-case",
				Prompt: "Review this code",
				Assert: []Assertion{
					{Type: "contains", Value: "test"},
				},
			},
		},
	}

	errors := e.Validate()
	if !HasErrors(errors) {
		t.Error("expected error for missing name")
	}

	found := false
	for _, err := range errors {
		if err.Field == "name" && err.Level == ValidationLevelError {
			found = true
		}
	}
	if !found {
		t.Error("expected error for name field")
	}
}

func TestValidate_MissingPrompt(t *testing.T) {
	e := &Eval{
		Name:        "test-eval",
		Description: "A test eval",
		Tests: []Test{
			{
				Name:        "test-case",
				Description: "A test case",
				Prompt:      "", // Empty prompt
				Assert: []Assertion{
					{Type: "contains", Value: "test"},
				},
			},
		},
	}

	errors := e.Validate()
	if !HasErrors(errors) {
		t.Error("expected error for missing prompt")
	}

	found := false
	for _, err := range errors {
		if err.Field == "tests[0].prompt" && err.Level == ValidationLevelError {
			found = true
		}
	}
	if !found {
		t.Error("expected error for tests[0].prompt field")
	}
}

func TestValidate_EmptyAssertions(t *testing.T) {
	e := &Eval{
		Name:        "test-eval",
		Description: "A test eval",
		Tests: []Test{
			{
				Name:        "test-case",
				Description: "A test case",
				Prompt:      "Review this code",
				Assert:      []Assertion{}, // Empty assertions
			},
		},
	}

	errors := e.Validate()
	if !HasErrors(errors) {
		t.Error("expected error for empty assertions")
	}

	found := false
	for _, err := range errors {
		if err.Field == "tests[0].assert" && err.Level == ValidationLevelError {
			found = true
		}
	}
	if !found {
		t.Error("expected error for tests[0].assert field")
	}
}

func TestValidate_MissingDescription(t *testing.T) {
	e := &Eval{
		Name: "test-eval",
		// Missing description
		Tests: []Test{
			{
				Name:   "test-case",
				Prompt: "Review this code",
				Assert: []Assertion{
					{Type: "contains", Value: "test"},
				},
			},
		},
	}

	errors := e.Validate()
	// Should not have errors, only warnings
	if HasErrors(errors) {
		t.Error("expected only warnings for missing description, not errors")
	}

	// Should have warnings
	_, warningCount := CountByLevel(errors)
	if warningCount == 0 {
		t.Error("expected warnings for missing descriptions")
	}

	// Check for eval description warning
	foundEvalWarning := false
	foundTestWarning := false
	for _, err := range errors {
		if err.Field == "description" && err.Level == ValidationLevelWarning {
			foundEvalWarning = true
		}
		if err.Field == "tests[0]" && err.Level == ValidationLevelWarning {
			foundTestWarning = true
		}
	}
	if !foundEvalWarning {
		t.Error("expected warning for eval description")
	}
	if !foundTestWarning {
		t.Error("expected warning for test description")
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	e := &Eval{
		Name: "", // Error 1: missing name
		Tests: []Test{
			{
				Name:   "",  // Error 2: missing test name
				Prompt: "",  // Error 3: missing prompt
				Assert: nil, // Error 4: missing assertions
			},
		},
	}

	errors := e.Validate()
	errorCount, _ := CountByLevel(errors)

	// Should have at least 4 errors
	if errorCount < 4 {
		t.Errorf("expected at least 4 errors, got %d", errorCount)
	}
}

func TestValidate_AllAssertionTypes(t *testing.T) {
	for _, assertType := range ValidAssertionTypes {
		e := &Eval{
			Name:        "test-eval",
			Description: "A test eval",
			Tests: []Test{
				{
					Name:        "test-case",
					Description: "A test case",
					Prompt:      "Review this code",
					Assert: []Assertion{
						{Type: assertType, Value: "test value"},
					},
				},
			},
		}

		errors := e.Validate()
		if HasErrors(errors) {
			t.Errorf("expected no errors for valid assertion type %q, got: %v", assertType, errors)
		}
	}
}

func TestSuggestAssertionType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"llm_rubric", "llm-rubric"},
		{"LLM-RUBRIC", "llm-rubric"},
		{"rubric", "llm-rubric"},
		{"contain", "contains"},
		{"not_contains", "not-contains"},
		{"contains_all", "contains-all"},
		{"contains_any", "contains-any"},
		{"js", "javascript"},
		{"regexp", "regex"},
		{"invalid", ""},
		{"xyz", ""},
	}

	for _, tc := range tests {
		result := suggestAssertionType(tc.input)
		if result != tc.expected {
			t.Errorf("suggestAssertionType(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"security-secrets", true},
		{"test", true},
		{"test-123", true},
		{"a", true},
		{"Test", false},      // Uppercase
		{"test_name", false}, // Underscore
		{"-test", false},     // Leading hyphen
		{"test-", false},     // Trailing hyphen
		{"test name", false}, // Space
		{"test.name", false}, // Dot
	}

	for _, tc := range tests {
		result := isValidName(tc.name)
		if result != tc.valid {
			t.Errorf("isValidName(%q) = %v, expected %v", tc.name, result, tc.valid)
		}
	}
}

func TestIsValidTag(t *testing.T) {
	tests := []struct {
		tag   string
		valid bool
	}{
		{"security", true},
		{"python", true},
		{"code-quality", true},
		{"Security", false},     // Uppercase
		{"code_quality", false}, // Underscore
	}

	for _, tc := range tests {
		result := isValidTag(tc.tag)
		if result != tc.valid {
			t.Errorf("isValidTag(%q) = %v, expected %v", tc.tag, result, tc.valid)
		}
	}
}

func TestCountByLevel(t *testing.T) {
	errors := []ValidationError{
		{Level: ValidationLevelError},
		{Level: ValidationLevelError},
		{Level: ValidationLevelWarning},
		{Level: ValidationLevelWarning},
		{Level: ValidationLevelWarning},
	}

	errorCount, warningCount := CountByLevel(errors)
	if errorCount != 2 {
		t.Errorf("expected 2 errors, got %d", errorCount)
	}
	if warningCount != 3 {
		t.Errorf("expected 3 warnings, got %d", warningCount)
	}
}

func TestHasErrors(t *testing.T) {
	// Only warnings
	warnings := []ValidationError{
		{Level: ValidationLevelWarning},
	}
	if HasErrors(warnings) {
		t.Error("expected HasErrors to return false for warnings only")
	}

	// With errors
	withErrors := []ValidationError{
		{Level: ValidationLevelWarning},
		{Level: ValidationLevelError},
	}
	if !HasErrors(withErrors) {
		t.Error("expected HasErrors to return true when errors present")
	}

	// Empty
	if HasErrors(nil) {
		t.Error("expected HasErrors to return false for nil")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
