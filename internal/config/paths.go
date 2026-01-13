// Package config handles staghorn configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths provides all staghorn-related filesystem paths.
type Paths struct {
	ConfigDir         string // ~/.config/staghorn
	CacheDir          string // ~/.cache/staghorn
	ConfigFile        string // ~/.config/staghorn/config.yaml
	PersonalMD        string // ~/.config/staghorn/personal.md
	PersonalActions   string // ~/.config/staghorn/actions
	PersonalLanguages string // ~/.config/staghorn/languages
}

// NewPaths creates Paths using ~/.config and ~/.cache directories.
// We use these paths explicitly for cross-platform consistency rather than
// platform-specific defaults (like ~/Library/Application Support on macOS).
func NewPaths() *Paths {
	home := os.Getenv("HOME")
	configDir := filepath.Join(home, ".config", "staghorn")
	cacheDir := filepath.Join(home, ".cache", "staghorn")

	return &Paths{
		ConfigDir:         configDir,
		CacheDir:          cacheDir,
		ConfigFile:        filepath.Join(configDir, "config.yaml"),
		PersonalMD:        filepath.Join(configDir, "personal.md"),
		PersonalActions:   filepath.Join(configDir, "actions"),
		PersonalLanguages: filepath.Join(configDir, "languages"),
	}
}

// NewPathsWithOverrides allows overriding directories for testing.
func NewPathsWithOverrides(configDir, cacheDir string) *Paths {
	return &Paths{
		ConfigDir:         configDir,
		CacheDir:          cacheDir,
		ConfigFile:        filepath.Join(configDir, "config.yaml"),
		PersonalMD:        filepath.Join(configDir, "personal.md"),
		PersonalActions:   filepath.Join(configDir, "actions"),
		PersonalLanguages: filepath.Join(configDir, "languages"),
	}
}

// CacheFile returns the path for a cached team config.
func (p *Paths) CacheFile(owner, repo string) string {
	return filepath.Join(p.CacheDir, fmt.Sprintf("%s-%s.md", owner, repo))
}

// CacheMetadataFile returns the path for cache metadata sidecar.
func (p *Paths) CacheMetadataFile(owner, repo string) string {
	return filepath.Join(p.CacheDir, fmt.Sprintf("%s-%s.meta.json", owner, repo))
}

// TeamActionsDir returns the path for cached team actions.
func (p *Paths) TeamActionsDir(owner, repo string) string {
	return filepath.Join(p.CacheDir, fmt.Sprintf("%s-%s-actions", owner, repo))
}

// TeamTemplatesDir returns the path for cached team project templates.
func (p *Paths) TeamTemplatesDir(owner, repo string) string {
	return filepath.Join(p.CacheDir, fmt.Sprintf("%s-%s-templates", owner, repo))
}

// TeamLanguagesDir returns the path for cached team language configs.
func (p *Paths) TeamLanguagesDir(owner, repo string) string {
	return filepath.Join(p.CacheDir, fmt.Sprintf("%s-%s-languages", owner, repo))
}

// ProjectActionsDir returns the path for project-specific actions.
// This is relative to the project root (.staghorn/actions/).
func ProjectActionsDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".staghorn", "actions")
}

// ProjectPaths holds paths for project-level config management.
type ProjectPaths struct {
	Root         string // Project root directory
	StaghornDir  string // .staghorn/
	SourceMD     string // .staghorn/project.md (source of truth)
	OutputMD     string // ./CLAUDE.md (generated output)
	ActionsDir   string // .staghorn/actions/
	LanguagesDir string // .staghorn/languages/
	ConfigFile   string // .staghorn/config.yaml (optional project config)
}

// NewProjectPaths creates ProjectPaths for a given project root.
func NewProjectPaths(projectRoot string) *ProjectPaths {
	staghornDir := filepath.Join(projectRoot, ".staghorn")
	return &ProjectPaths{
		Root:         projectRoot,
		StaghornDir:  staghornDir,
		SourceMD:     filepath.Join(staghornDir, "project.md"),
		OutputMD:     filepath.Join(projectRoot, "CLAUDE.md"),
		ActionsDir:   filepath.Join(staghornDir, "actions"),
		LanguagesDir: filepath.Join(staghornDir, "languages"),
		ConfigFile:   filepath.Join(staghornDir, "config.yaml"),
	}
}
