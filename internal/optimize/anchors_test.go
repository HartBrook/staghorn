package optimize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractAnchors_ToolNames(t *testing.T) {
	content := `## Python

Use pytest for testing, black for formatting, and ruff for linting.
`
	anchors := ExtractAnchors(content)

	assert.Contains(t, anchors, "pytest")
	assert.Contains(t, anchors, "black")
	assert.Contains(t, anchors, "ruff")
}

func TestExtractAnchors_FilePaths(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "absolute path",
			content: "Config is at /etc/config.yaml",
			want:    []string{"/etc/config.yaml"},
		},
		{
			name:    "home path",
			content: "Settings in ~/.config/staghorn/config.yaml",
			want:    []string{"~/.config/staghorn/config.yaml"},
		},
		{
			name:    "relative path",
			content: "Edit ./src/main.go",
			want:    []string{"./src/main.go"},
		},
		{
			name:    "dotfile",
			content: "Add .gitignore and .env files",
			want:    []string{".gitignore", ".env"},
		},
		{
			name:    "config file",
			content: "Update pyproject.toml",
			want:    []string{"pyproject.toml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anchors := ExtractAnchors(tt.content)
			for _, w := range tt.want {
				assert.Contains(t, anchors, w)
			}
		})
	}
}

func TestExtractAnchors_Commands(t *testing.T) {
	content := "Run `npm test` to test and `go build ./...` to build."

	anchors := ExtractAnchors(content)

	assert.Contains(t, anchors, "npm test")
	assert.Contains(t, anchors, "go build ./...")
}

func TestExtractAnchors_CodeBlocks(t *testing.T) {
	content := "```python\ndef calculate_total(items):\n    return sum(items)\n```"

	anchors := ExtractAnchors(content)

	assert.Contains(t, anchors, "calculate_total")
}

func TestExtractAnchors_GoFunctions(t *testing.T) {
	content := "```go\nfunc ProcessRequest(ctx context.Context) error {\n    return nil\n}\n```"

	anchors := ExtractAnchors(content)

	assert.Contains(t, anchors, "ProcessRequest")
}

func TestExtractAnchors_TypeScriptFunctions(t *testing.T) {
	content := "```typescript\nfunction validateInput(data: unknown): boolean {\n    return true;\n}\n\nconst processData = (input: string) => {};\n```"

	anchors := ExtractAnchors(content)

	// Only function declarations are extracted, not const/let/var
	// This prevents generic variable names from being treated as critical anchors
	assert.Contains(t, anchors, "validateInput")
	// processData is a const, not extracted (arrow functions with const are typically generic)
	assert.NotContains(t, anchors, "processData")
}

func TestValidateAnchors_AllPreserved(t *testing.T) {
	original := "Use pytest for testing. Config in ~/.config/app.yaml"
	optimized := "Use pytest. Config: ~/.config/app.yaml"

	preserved, missing := ValidateAnchors(original, optimized)

	assert.Contains(t, preserved, "pytest")
	assert.Empty(t, missing)
}

func TestValidateAnchors_SomeMissing(t *testing.T) {
	original := "Use pytest for testing and ruff for linting"
	optimized := "Use pytest for testing"

	preserved, missing := ValidateAnchors(original, optimized)

	assert.Contains(t, preserved, "pytest")
	assert.Contains(t, missing, "ruff")
}

func TestValidateAnchors_CaseInsensitive(t *testing.T) {
	original := "Use PyTest for testing"
	optimized := "Use pytest"

	preserved, missing := ValidateAnchors(original, optimized)

	assert.Contains(t, preserved, "pytest")
	assert.Empty(t, missing)
}

func TestExtractAnchors_EmptyContent(t *testing.T) {
	anchors := ExtractAnchors("")
	assert.Empty(t, anchors)
}

func TestExtractAnchors_NoAnchors(t *testing.T) {
	content := "This is just plain text without any special content."
	anchors := ExtractAnchors(content)
	// May have some false positives, but should be minimal
	assert.True(t, len(anchors) < 3)
}

func TestExtractAnchors_Deduplication(t *testing.T) {
	content := "Use pytest. Then run pytest again. pytest is great."

	anchors := ExtractAnchors(content)

	// pytest should appear only once
	count := 0
	for _, a := range anchors {
		if a == "pytest" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestExtractAnchors_MultipleCodeBlocks(t *testing.T) {
	content := `
## Python Example

` + "```python\ndef handler():\n    pass\n```" + `

## Go Example

` + "```go\nfunc Server() {\n}\n```"

	anchors := ExtractAnchors(content)

	assert.Contains(t, anchors, "handler")
	assert.Contains(t, anchors, "Server")
}

func TestLooksLikeCommand(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"npm install", true},
		{"go test ./...", true},
		{"git commit -m", true},
		{"cargo build", true},
		{"python -m pytest", true},
		{"docker compose up", true},
		{"a", false},            // Too short
		{"variableName", false}, // Just identifier
		{"SomeClass", false},    // Just identifier
		{"npm", true},           // Known tool
		{"pytest", true},        // Known tool
		{"randomword", false},   // Unknown single word
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeCommand(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidPath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{".gitignore", true},
		{".env", true},
		{"./src", true},
		{"~/.config", true},
		{"/etc/config", true},
		{".1", false}, // Version number
		{"..", false}, // Parent dir only
		{"a", false},  // Too short
		{"config.yaml", true},
		{"v1.0.0", false},    // Version string
		{"v2.1", false},      // Version string
		{"1.0.0", false},     // Version without v
		{"123.json", false},  // Numeric filename
		{"456.txt", false},   // Numeric filename
		{"file123.go", true}, // Valid - starts with letter
		{"test_v2.py", true}, // Valid - version in middle is ok
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isValidPath(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractFilePaths_FiltersVersionNumbers(t *testing.T) {
	content := `Use version v1.0.0 of the library.
Check config.yaml for settings.
The API returns 123.json format.
Edit pyproject.toml for dependencies.`

	anchors := extractFilePaths(content)

	// Should include real files
	assert.Contains(t, anchors, "config.yaml")
	assert.Contains(t, anchors, "pyproject.toml")

	// Should NOT include version numbers or numeric filenames
	for _, anchor := range anchors {
		assert.NotEqual(t, "v1.0.0", anchor)
		assert.NotEqual(t, "123.json", anchor)
	}
}

func TestExtractAnchors_RealWorldExample(t *testing.T) {
	content := `## Python Guidelines

Use pytest with fixtures in conftest.py. Format with black (88 chars), isort, ruff.

Run tests with:
` + "`pytest --cov=src`" + `

Example:
` + "```python\ndef test_handler():\n    assert handler() == expected\n```"

	anchors := ExtractAnchors(content)

	// Should find tools
	assert.Contains(t, anchors, "pytest")
	assert.Contains(t, anchors, "black")
	assert.Contains(t, anchors, "isort")
	assert.Contains(t, anchors, "ruff")

	// Should find file
	assert.Contains(t, anchors, "conftest.py")

	// Should find command
	assert.Contains(t, anchors, "pytest --cov=src")

	// Should find function from code block
	assert.Contains(t, anchors, "test_handler")
}

func TestExtractAnchors_GenericVariablesNotExtracted(t *testing.T) {
	// Generic variable names in code examples should NOT be extracted as anchors
	// because they're illustrative, not project-specific
	content := `## Type Patterns

- Use ` + "`" + `satisfies` + "`" + ` to validate types while preserving literal types

` + "```typescript" + `
// satisfies preserves literal types
const config = {
  port: 3000,
  host: "localhost",
} satisfies ServerConfig;
` + "```"

	anchors := ExtractAnchors(content)
	t.Logf("Extracted anchors: %v", anchors)

	// "config" should NOT be extracted - it's a generic variable name
	assert.NotContains(t, anchors, "config")
	// Language names from code fence markers may still be extracted
	assert.Contains(t, anchors, "typescript")
}

func TestExtractAnchors_ProjectSpecificFunctionsExtracted(t *testing.T) {
	// Function definitions with specific names SHOULD be extracted
	content := "```go\nfunc ProcessPayment(ctx context.Context, amount int) error {\n    return nil\n}\n```"

	anchors := ExtractAnchors(content)

	// "ProcessPayment" should be extracted - it's a specific function name
	assert.Contains(t, anchors, "ProcessPayment")
}

func TestValidateAnchors_GenericVariableRenamedNoFalsePositive(t *testing.T) {
	// Renaming a generic variable in code example should NOT trigger validation failure
	original := "```typescript\nconst config = { port: 3000 };\n```"
	optimized := "```typescript\nconst settings = { port: 3000 };\n```"

	preserved, missing := ValidateAnchors(original, optimized)
	t.Logf("Preserved: %v", preserved)
	t.Logf("Missing: %v", missing)

	// Neither "config" nor "settings" should cause validation issues
	// since const declarations are not extracted as anchors
	assert.NotContains(t, missing, "config")
	assert.NotContains(t, missing, "settings")
}

func TestValidateAnchorsCategorized_ToolNamesAreSoft(t *testing.T) {
	// Tool names should be soft anchors (warnings, not failures)
	original := "Use pytest for testing. Edit ~/.config/app.yaml for settings."
	optimized := "Edit ~/.config/app.yaml for settings." // pytest removed

	result := ValidateAnchorsCategorized(original, optimized)

	// pytest should be in MissingSoft (tool name)
	assert.Contains(t, result.MissingSoft, "pytest")
	// File path should be preserved
	assert.Contains(t, result.Preserved, "~/.config/app.yaml")
	// No strict failures
	assert.False(t, result.HasStrictFailures())
}

func TestValidateAnchorsCategorized_FilePathsAreStrict(t *testing.T) {
	// File paths should be strict anchors (failures)
	original := "Use pytest for testing. Edit ~/.config/app.yaml for settings."
	optimized := "Use pytest for testing." // file path removed

	result := ValidateAnchorsCategorized(original, optimized)

	// pytest should be preserved
	assert.Contains(t, result.Preserved, "pytest")
	// File path should be in MissingStrict
	assert.Contains(t, result.MissingStrict, "~/.config/app.yaml")
	// Should have strict failures
	assert.True(t, result.HasStrictFailures())
}

func TestValidateAnchorsCategorized_CommandsAreStrict(t *testing.T) {
	// Commands should be strict anchors (failures)
	original := "Run `go test ./...` to test."
	optimized := "Run tests." // command removed

	result := ValidateAnchorsCategorized(original, optimized)

	// Command should be in MissingStrict
	assert.Contains(t, result.MissingStrict, "go test ./...")
	// Should have strict failures
	assert.True(t, result.HasStrictFailures())
}

func TestExtractCategorizedAnchors_Categorization(t *testing.T) {
	content := `Use pytest and ruff for Python. Edit ~/.config/app.yaml for settings. Run ` + "`npm test`" + `.`

	anchors := ExtractCategorizedAnchors(content)
	t.Logf("Soft: %v", anchors.Soft)
	t.Logf("Strict: %v", anchors.Strict)

	// Tool names should be soft
	assert.Contains(t, anchors.Soft, "pytest")
	assert.Contains(t, anchors.Soft, "ruff")

	// File paths should be strict
	assert.Contains(t, anchors.Strict, "~/.config/app.yaml")

	// Commands should be strict
	assert.Contains(t, anchors.Strict, "npm test")
}
