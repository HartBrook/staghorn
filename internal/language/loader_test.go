package language

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasUserContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "empty string",
			content:  "",
			expected: false,
		},
		{
			name:     "only whitespace",
			content:  "   \n\n\t  ",
			expected: false,
		},
		{
			name:     "only heading",
			content:  "## My Python Preferences\n\n",
			expected: false,
		},
		{
			name:     "multiple headings only",
			content:  "# Title\n\n## Section 1\n\n### Subsection\n",
			expected: false,
		},
		{
			name:     "only HTML comment",
			content:  "<!-- This is a comment -->",
			expected: false,
		},
		{
			name:     "only staghorn comment",
			content:  "<!-- [staghorn] Personal Python preferences -->",
			expected: false,
		},
		{
			name:     "heading and comment only",
			content:  "<!-- [staghorn] marker -->\n## My Python Preferences\n\n",
			expected: false,
		},
		{
			name:     "has actual content",
			content:  "## My Preferences\n\n- Use pytest\n",
			expected: true,
		},
		{
			name:     "content with comment and heading",
			content:  "<!-- comment -->\n## Heading\n\nSome actual content here",
			expected: true,
		},
		{
			name:     "just text",
			content:  "I prefer using dataclasses",
			expected: true,
		},
		{
			name:     "bullet list",
			content:  "## Prefs\n\n- item 1\n- item 2\n",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasUserContent(tt.content)
			if got != tt.expected {
				t.Errorf("HasUserContent(%q) = %v, expected %v", tt.content, got, tt.expected)
			}
		})
	}
}

func TestLoadLanguageFiles(t *testing.T) {
	// Create temp directories for each layer
	tmpDir := t.TempDir()
	teamDir := filepath.Join(tmpDir, "team")
	personalDir := filepath.Join(tmpDir, "personal")
	projectDir := filepath.Join(tmpDir, "project")

	if err := os.MkdirAll(teamDir, 0755); err != nil {
		t.Fatalf("failed to create team dir: %v", err)
	}
	if err := os.MkdirAll(personalDir, 0755); err != nil {
		t.Fatalf("failed to create personal dir: %v", err)
	}
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create language files (personal needs actual content to be included)
	if err := os.WriteFile(filepath.Join(teamDir, "python.md"), []byte("Team Python config"), 0644); err != nil {
		t.Fatalf("failed to write team python.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(personalDir, "python.md"), []byte("## My Prefs\n\nPersonal Python config"), 0644); err != nil {
		t.Fatalf("failed to write personal python.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(teamDir, "go.md"), []byte("Team Go config"), 0644); err != nil {
		t.Fatalf("failed to write team go.md: %v", err)
	}

	// Load language files
	files, err := LoadLanguageFiles([]string{"python", "go", "rust"}, teamDir, personalDir, projectDir)
	if err != nil {
		t.Fatalf("LoadLanguageFiles() error = %v", err)
	}

	// Check python files (should have team and personal)
	pythonFiles, ok := files["python"]
	if !ok {
		t.Error("LoadLanguageFiles() missing python files")
	} else if len(pythonFiles) != 2 {
		t.Errorf("LoadLanguageFiles() python files count = %d, expected 2", len(pythonFiles))
	} else {
		// Check order: team first, then personal
		if pythonFiles[0].Source != "team" {
			t.Errorf("LoadLanguageFiles() python[0].Source = %q, expected \"team\"", pythonFiles[0].Source)
		}
		if pythonFiles[0].Content != "Team Python config" {
			t.Errorf("LoadLanguageFiles() python[0].Content = %q, expected \"Team Python config\"", pythonFiles[0].Content)
		}
		if pythonFiles[1].Source != "personal" {
			t.Errorf("LoadLanguageFiles() python[1].Source = %q, expected \"personal\"", pythonFiles[1].Source)
		}
	}

	// Check go files (should have team only)
	goFiles, ok := files["go"]
	if !ok {
		t.Error("LoadLanguageFiles() missing go files")
	} else if len(goFiles) != 1 {
		t.Errorf("LoadLanguageFiles() go files count = %d, expected 1", len(goFiles))
	} else if goFiles[0].Source != "team" {
		t.Errorf("LoadLanguageFiles() go[0].Source = %q, expected \"team\"", goFiles[0].Source)
	}

	// Check rust files (should be empty since no files exist)
	rustFiles, ok := files["rust"]
	if ok && len(rustFiles) > 0 {
		t.Errorf("LoadLanguageFiles() rust files = %v, expected none", rustFiles)
	}
}

func TestLoadLanguageFiles_SkipsEmptyPersonalFiles(t *testing.T) {
	tmpDir := t.TempDir()
	teamDir := filepath.Join(tmpDir, "team")
	personalDir := filepath.Join(tmpDir, "personal")

	if err := os.MkdirAll(teamDir, 0755); err != nil {
		t.Fatalf("failed to create team dir: %v", err)
	}
	if err := os.MkdirAll(personalDir, 0755); err != nil {
		t.Fatalf("failed to create personal dir: %v", err)
	}

	// Team file with content
	if err := os.WriteFile(filepath.Join(teamDir, "python.md"), []byte("Team Python config"), 0644); err != nil {
		t.Fatalf("failed to write team python.md: %v", err)
	}

	// Personal file with only heading (should be skipped)
	if err := os.WriteFile(filepath.Join(personalDir, "python.md"), []byte("## My Python Preferences\n\n"), 0644); err != nil {
		t.Fatalf("failed to write personal python.md: %v", err)
	}

	files, err := LoadLanguageFiles([]string{"python"}, teamDir, personalDir, "")
	if err != nil {
		t.Fatalf("LoadLanguageFiles() error = %v", err)
	}

	pythonFiles := files["python"]
	if len(pythonFiles) != 1 {
		t.Errorf("LoadLanguageFiles() should skip empty personal file, got %d files", len(pythonFiles))
	}
	if len(pythonFiles) == 1 && pythonFiles[0].Source != "team" {
		t.Errorf("LoadLanguageFiles() expected team file only, got %q", pythonFiles[0].Source)
	}
}

func TestLoadLanguageFiles_EmptyDirs(t *testing.T) {
	// Test with non-existent directories
	files, err := LoadLanguageFiles([]string{"python"}, "/nonexistent1", "/nonexistent2", "/nonexistent3")
	if err != nil {
		t.Fatalf("LoadLanguageFiles() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("LoadLanguageFiles() with empty dirs = %v, expected empty map", files)
	}
}

func TestLoadLanguageFiles_EmptyLanguageList(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	if err := os.WriteFile(filepath.Join(tmpDir, "python.md"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Load with empty language list
	files, err := LoadLanguageFiles([]string{}, tmpDir, "", "")
	if err != nil {
		t.Fatalf("LoadLanguageFiles() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("LoadLanguageFiles() with empty languages = %v, expected empty map", files)
	}
}

func TestListAvailableLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	teamDir := filepath.Join(tmpDir, "team")
	personalDir := filepath.Join(tmpDir, "personal")

	if err := os.MkdirAll(teamDir, 0755); err != nil {
		t.Fatalf("failed to create team dir: %v", err)
	}
	if err := os.MkdirAll(personalDir, 0755); err != nil {
		t.Fatalf("failed to create personal dir: %v", err)
	}

	// Create language files in different directories
	if err := os.WriteFile(filepath.Join(teamDir, "python.md"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(teamDir, "go.md"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(personalDir, "rust.md"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// List available languages
	languages, err := ListAvailableLanguages(teamDir, personalDir, "")
	if err != nil {
		t.Fatalf("ListAvailableLanguages() error = %v", err)
	}

	// Should have 3 unique languages
	if len(languages) != 3 {
		t.Errorf("ListAvailableLanguages() = %v, expected 3 languages", languages)
	}

	// Check all expected languages are present
	langSet := make(map[string]bool)
	for _, lang := range languages {
		langSet[lang] = true
	}

	for _, expected := range []string{"python", "go", "rust"} {
		if !langSet[expected] {
			t.Errorf("ListAvailableLanguages() missing %q", expected)
		}
	}
}
