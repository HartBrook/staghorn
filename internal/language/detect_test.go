package language

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected []string
	}{
		{
			name:     "empty project",
			files:    []string{},
			expected: []string{},
		},
		{
			name:     "go project",
			files:    []string{"go.mod"},
			expected: []string{"go"},
		},
		{
			name:     "python project with pyproject.toml",
			files:    []string{"pyproject.toml"},
			expected: []string{"python"},
		},
		{
			name:     "python project with requirements.txt",
			files:    []string{"requirements.txt"},
			expected: []string{"python"},
		},
		{
			name:     "typescript project",
			files:    []string{"package.json", "tsconfig.json"},
			expected: []string{"typescript"},
		},
		{
			name:     "javascript project",
			files:    []string{"package.json"},
			expected: []string{"javascript"},
		},
		{
			name:     "rust project",
			files:    []string{"Cargo.toml"},
			expected: []string{"rust"},
		},
		{
			name:     "java maven project",
			files:    []string{"pom.xml"},
			expected: []string{"java"},
		},
		{
			name:     "ruby project",
			files:    []string{"Gemfile"},
			expected: []string{"ruby"},
		},
		{
			name:     "multi-language project",
			files:    []string{"go.mod", "pyproject.toml"},
			expected: []string{"go", "python"},
		},
		{
			name:     "typescript supersedes javascript",
			files:    []string{"package.json", "tsconfig.json"},
			expected: []string{"typescript"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Create marker files
			for _, file := range tt.files {
				path := filepath.Join(tmpDir, file)
				if err := os.WriteFile(path, []byte{}, 0644); err != nil {
					t.Fatalf("failed to create file %s: %v", file, err)
				}
			}

			// Detect languages
			detected, err := Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			// Check results
			if len(detected) != len(tt.expected) {
				t.Errorf("Detect() = %v, expected %v", detected, tt.expected)
				return
			}

			for i, lang := range detected {
				if lang != tt.expected[i] {
					t.Errorf("Detect()[%d] = %v, expected %v", i, lang, tt.expected[i])
				}
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *LanguageConfig
		files    []string
		expected []string
	}{
		{
			name: "explicit list takes precedence",
			cfg: &LanguageConfig{
				AutoDetect: true,
				Enabled:    []string{"rust", "go"},
			},
			files:    []string{"pyproject.toml"},
			expected: []string{"rust", "go"},
		},
		{
			name: "auto-detect enabled",
			cfg: &LanguageConfig{
				AutoDetect: true,
			},
			files:    []string{"go.mod"},
			expected: []string{"go"},
		},
		{
			name: "auto-detect disabled with no explicit list",
			cfg: &LanguageConfig{
				AutoDetect: false,
			},
			files:    []string{"go.mod"},
			expected: []string{},
		},
		{
			name: "disabled languages filtered",
			cfg: &LanguageConfig{
				AutoDetect: true,
				Disabled:   []string{"python"},
			},
			files:    []string{"go.mod", "pyproject.toml"},
			expected: []string{"go"},
		},
		{
			name: "explicit list with disabled",
			cfg: &LanguageConfig{
				Enabled:  []string{"go", "python", "rust"},
				Disabled: []string{"python"},
			},
			files:    []string{},
			expected: []string{"go", "rust"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Create marker files
			for _, file := range tt.files {
				path := filepath.Join(tmpDir, file)
				if err := os.WriteFile(path, []byte{}, 0644); err != nil {
					t.Fatalf("failed to create file %s: %v", file, err)
				}
			}

			// Resolve languages
			resolved, err := Resolve(tt.cfg, tmpDir)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}

			// Check results
			if len(resolved) != len(tt.expected) {
				t.Errorf("Resolve() = %v, expected %v", resolved, tt.expected)
				return
			}

			for i, lang := range resolved {
				if lang != tt.expected[i] {
					t.Errorf("Resolve()[%d] = %v, expected %v", i, lang, tt.expected[i])
				}
			}
		})
	}
}

func TestGetDisplayName(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{"python", "Python"},
		{"go", "Go"},
		{"typescript", "TypeScript"},
		{"javascript", "JavaScript"},
		{"csharp", "C#"},
		{"unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := GetDisplayName(tt.id)
			if result != tt.expected {
				t.Errorf("GetDisplayName(%q) = %q, expected %q", tt.id, result, tt.expected)
			}
		})
	}
}

func TestGetLanguage(t *testing.T) {
	// Test existing language
	lang := GetLanguage("python")
	if lang == nil {
		t.Error("GetLanguage(\"python\") returned nil")
	} else if lang.ID != "python" {
		t.Errorf("GetLanguage(\"python\").ID = %q, expected \"python\"", lang.ID)
	}

	// Test non-existent language
	lang = GetLanguage("nonexistent")
	if lang != nil {
		t.Errorf("GetLanguage(\"nonexistent\") = %v, expected nil", lang)
	}
}
