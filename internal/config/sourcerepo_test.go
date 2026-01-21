package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSourceRepo(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		expected bool
	}{
		{
			name: "with source_repo true",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".staghorn"), 0755); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".staghorn", "source.yaml"),
					[]byte("source_repo: true"), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			expected: true,
		},
		{
			name: "with source_repo false",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".staghorn"), 0755); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".staghorn", "source.yaml"),
					[]byte("source_repo: false"), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			expected: false,
		},
		{
			name: "no source.yaml file",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".staghorn"), 0755); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			expected: false,
		},
		{
			name:     "no .staghorn directory",
			setup:    func(t *testing.T, dir string) {},
			expected: false,
		},
		{
			name:     "empty project root",
			setup:    nil, // will pass empty string
			expected: false,
		},
		{
			name: "invalid yaml",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".staghorn"), 0755); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".staghorn", "source.yaml"),
					[]byte("invalid: [yaml: content"), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup == nil {
				// Test empty string case
				if IsSourceRepo("") != tt.expected {
					t.Errorf("IsSourceRepo(\"\") = %v, want %v", !tt.expected, tt.expected)
				}
				return
			}

			// Create temp directory
			dir := t.TempDir()
			tt.setup(t, dir)

			result := IsSourceRepo(dir)
			if result != tt.expected {
				t.Errorf("IsSourceRepo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewSourceRepoPaths(t *testing.T) {
	root := filepath.Join("test", "project")
	paths := NewSourceRepoPaths(root)

	if paths.Root != root {
		t.Errorf("Root = %q, want %q", paths.Root, root)
	}
	wantConfigFile := filepath.Join(root, ".staghorn", "source.yaml")
	if paths.ConfigFile != wantConfigFile {
		t.Errorf("ConfigFile = %q, want %q", paths.ConfigFile, wantConfigFile)
	}
	wantClaudeMD := filepath.Join(root, "CLAUDE.md")
	if paths.ClaudeMD != wantClaudeMD {
		t.Errorf("ClaudeMD = %q, want %q", paths.ClaudeMD, wantClaudeMD)
	}
	wantCommandsDir := filepath.Join(root, "commands")
	if paths.CommandsDir != wantCommandsDir {
		t.Errorf("CommandsDir = %q, want %q", paths.CommandsDir, wantCommandsDir)
	}
	wantLanguagesDir := filepath.Join(root, "languages")
	if paths.LanguagesDir != wantLanguagesDir {
		t.Errorf("LanguagesDir = %q, want %q", paths.LanguagesDir, wantLanguagesDir)
	}
	wantTemplatesDir := filepath.Join(root, "templates")
	if paths.TemplatesDir != wantTemplatesDir {
		t.Errorf("TemplatesDir = %q, want %q", paths.TemplatesDir, wantTemplatesDir)
	}
	wantEvalsDir := filepath.Join(root, "evals")
	if paths.EvalsDir != wantEvalsDir {
		t.Errorf("EvalsDir = %q, want %q", paths.EvalsDir, wantEvalsDir)
	}
}

func TestLoadSourceRepoConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".staghorn"), 0755); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".staghorn", "source.yaml"),
			[]byte("source_repo: true"), 0644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		cfg, err := LoadSourceRepoConfig(dir)
		if err != nil {
			t.Fatalf("LoadSourceRepoConfig() error = %v", err)
		}
		if !cfg.SourceRepo {
			t.Error("SourceRepo should be true")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		dir := t.TempDir()

		_, err := LoadSourceRepoConfig(dir)
		if err == nil {
			t.Error("LoadSourceRepoConfig() should return error for missing file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".staghorn"), 0755); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".staghorn", "source.yaml"),
			[]byte("invalid: [yaml"), 0644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		_, err := LoadSourceRepoConfig(dir)
		if err == nil {
			t.Error("LoadSourceRepoConfig() should return error for invalid YAML")
		}
	})
}

func TestIsSourceRepo_Subdirectory(t *testing.T) {
	// Verify that IsSourceRepo returns false for a subdirectory of a source repo
	// (it only checks the exact directory passed, not parents)
	dir := t.TempDir()

	// Create source repo marker in root
	if err := os.MkdirAll(filepath.Join(dir, ".staghorn"), 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".staghorn", "source.yaml"),
		[]byte("source_repo: true"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Create a subdirectory
	subdir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Root should be a source repo
	if !IsSourceRepo(dir) {
		t.Error("IsSourceRepo(root) should return true")
	}

	// Subdirectory should NOT be a source repo (no .staghorn/source.yaml there)
	if IsSourceRepo(subdir) {
		t.Error("IsSourceRepo(subdir) should return false - it doesn't have its own source.yaml")
	}
}

func TestWriteSourceRepoConfig(t *testing.T) {
	dir := t.TempDir()

	err := WriteSourceRepoConfig(dir)
	if err != nil {
		t.Fatalf("WriteSourceRepoConfig() error = %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(dir, ".staghorn", "source.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// Verify content
	expectedContent := "# This marks this repository as a staghorn source repo\nsource_repo: true\n"
	if string(content) != expectedContent {
		t.Errorf("Config content = %q, want %q", string(content), expectedContent)
	}

	// Verify IsSourceRepo returns true after writing
	if !IsSourceRepo(dir) {
		t.Error("IsSourceRepo() should return true after WriteSourceRepoConfig()")
	}
}

func TestWriteSourceRepoConfig_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()

	// Verify .staghorn doesn't exist yet
	staghornDir := filepath.Join(dir, ".staghorn")
	if _, err := os.Stat(staghornDir); err == nil {
		t.Fatal(".staghorn directory should not exist initially")
	}

	err := WriteSourceRepoConfig(dir)
	if err != nil {
		t.Fatalf("WriteSourceRepoConfig() error = %v", err)
	}

	// Verify .staghorn was created
	if _, err := os.Stat(staghornDir); err != nil {
		t.Error(".staghorn directory should be created")
	}
}
