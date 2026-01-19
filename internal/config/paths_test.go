package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPaths_OptimizedDir(t *testing.T) {
	home := os.Getenv("HOME")
	paths := NewPaths()

	got := paths.OptimizedDir()
	want := filepath.Join(home, ".config", "staghorn", "optimized")

	if got != want {
		t.Errorf("OptimizedDir() = %q, want %q", got, want)
	}
}

func TestPaths_OptimizedFile(t *testing.T) {
	paths := NewPaths()

	got := paths.OptimizedFile("acme", "standards")

	if !strings.HasSuffix(got, "acme-standards.md") {
		t.Errorf("OptimizedFile() = %q, want suffix %q", got, "acme-standards.md")
	}
	if !strings.Contains(got, "optimized") {
		t.Errorf("OptimizedFile() = %q, should contain 'optimized'", got)
	}
}

func TestPaths_OptimizedMetaFile(t *testing.T) {
	paths := NewPaths()

	got := paths.OptimizedMetaFile("acme", "standards")

	if !strings.HasSuffix(got, "acme-standards.meta.json") {
		t.Errorf("OptimizedMetaFile() = %q, want suffix %q", got, "acme-standards.meta.json")
	}
	if !strings.Contains(got, "optimized") {
		t.Errorf("OptimizedMetaFile() = %q, should contain 'optimized'", got)
	}
}

func TestPaths_OptimizedPaths_Consistency(t *testing.T) {
	paths := NewPaths()

	optimizedDir := paths.OptimizedDir()
	optimizedFile := paths.OptimizedFile("acme", "standards")
	optimizedMeta := paths.OptimizedMetaFile("acme", "standards")

	// OptimizedFile should be inside OptimizedDir
	if !strings.HasPrefix(optimizedFile, optimizedDir) {
		t.Errorf("OptimizedFile() not inside OptimizedDir()")
	}

	// OptimizedMetaFile should be inside OptimizedDir
	if !strings.HasPrefix(optimizedMeta, optimizedDir) {
		t.Errorf("OptimizedMetaFile() not inside OptimizedDir()")
	}
}

func TestNewPathsWithOverrides_OptimizedPaths(t *testing.T) {
	tempDir := t.TempDir()
	paths := NewPathsWithOverrides(tempDir, tempDir)

	optimizedDir := paths.OptimizedDir()
	want := filepath.Join(tempDir, "optimized")

	if optimizedDir != want {
		t.Errorf("OptimizedDir() with override = %q, want %q", optimizedDir, want)
	}
}
