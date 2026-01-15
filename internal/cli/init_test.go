package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HartBrook/staghorn/internal/config"
)

func TestEnsurePersonalMD(t *testing.T) {
	// Create temp directory to act as config dir
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "staghorn")

	paths := &config.Paths{
		ConfigDir:  configDir,
		PersonalMD: filepath.Join(configDir, "personal.md"),
	}

	// First call should create the file
	err := ensurePersonalMD(paths)
	if err != nil {
		t.Fatalf("ensurePersonalMD() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(paths.PersonalMD); os.IsNotExist(err) {
		t.Error("ensurePersonalMD() did not create personal.md")
	}

	// Verify content
	content, err := os.ReadFile(paths.PersonalMD)
	if err != nil {
		t.Fatalf("failed to read personal.md: %v", err)
	}

	expectedContent := "## My Preferences\n\n"
	if string(content) != expectedContent {
		t.Errorf("ensurePersonalMD() content = %q, want %q", string(content), expectedContent)
	}

	// Second call should not error (file already exists)
	err = ensurePersonalMD(paths)
	if err != nil {
		t.Fatalf("ensurePersonalMD() second call error = %v", err)
	}

	// Verify content hasn't changed
	content2, _ := os.ReadFile(paths.PersonalMD)
	if string(content2) != expectedContent {
		t.Error("ensurePersonalMD() modified existing file")
	}
}

func TestEnsurePersonalMD_ExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "staghorn")

	// Create directory and file with custom content
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	personalMD := filepath.Join(configDir, "personal.md")
	customContent := "## My Custom Preferences\n\n- Use tabs\n"
	if err := os.WriteFile(personalMD, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom personal.md: %v", err)
	}

	paths := &config.Paths{
		ConfigDir:  configDir,
		PersonalMD: personalMD,
	}

	// ensurePersonalMD should not overwrite existing file
	err := ensurePersonalMD(paths)
	if err != nil {
		t.Fatalf("ensurePersonalMD() error = %v", err)
	}

	// Verify content is unchanged
	content, _ := os.ReadFile(personalMD)
	if string(content) != customContent {
		t.Errorf("ensurePersonalMD() overwrote existing file, got %q, want %q", string(content), customContent)
	}
}

func TestCreatePersonalLanguageFile(t *testing.T) {
	tempDir := t.TempDir()
	langDir := filepath.Join(tempDir, "languages")

	// Create file for python
	err := createPersonalLanguageFile(langDir, "python")
	if err != nil {
		t.Fatalf("createPersonalLanguageFile() error = %v", err)
	}

	// Verify file exists
	langFile := filepath.Join(langDir, "python.md")
	if _, err := os.Stat(langFile); os.IsNotExist(err) {
		t.Error("createPersonalLanguageFile() did not create python.md")
	}

	// Verify content has correct heading
	content, _ := os.ReadFile(langFile)
	if string(content) != "## My Python Preferences\n\n" {
		t.Errorf("createPersonalLanguageFile() content = %q, want heading with display name", string(content))
	}

	// Second call should return error (file exists)
	err = createPersonalLanguageFile(langDir, "python")
	if err == nil {
		t.Error("createPersonalLanguageFile() should error when file exists")
	}
}

func TestListLanguageFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create some language files
	files := []string{"python.md", "go.md", "typescript.md"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tempDir, f), []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	// Create a non-md file that should be ignored
	if err := os.WriteFile(filepath.Join(tempDir, "README.txt"), []byte("ignore"), 0644); err != nil {
		t.Fatalf("failed to create README.txt: %v", err)
	}

	// Create a directory that should be ignored
	if err := os.MkdirAll(filepath.Join(tempDir, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	langs, err := listLanguageFiles(tempDir)
	if err != nil {
		t.Fatalf("listLanguageFiles() error = %v", err)
	}

	if len(langs) != 3 {
		t.Errorf("listLanguageFiles() returned %d files, want 3", len(langs))
	}

	// Check expected languages are present (without .md suffix)
	expected := map[string]bool{"python": true, "go": true, "typescript": true}
	for _, lang := range langs {
		if !expected[lang] {
			t.Errorf("unexpected language: %s", lang)
		}
	}
}

func TestListLanguageFiles_NonexistentDir(t *testing.T) {
	langs, err := listLanguageFiles("/nonexistent/path")
	if err == nil {
		t.Error("listLanguageFiles() should error for nonexistent directory")
	}
	if langs != nil {
		t.Errorf("listLanguageFiles() should return nil for nonexistent dir, got %v", langs)
	}
}
