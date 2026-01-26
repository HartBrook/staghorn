// Package starter provides embedded starter skills that ship with staghorn.
package starter

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/HartBrook/staghorn/internal/skills"
)

//go:embed skills/*/SKILL.md
var skillsFS embed.FS

// SkillNames returns the list of available starter skill names.
func SkillNames() []string {
	entries, err := skillsFS.ReadDir("skills")
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if it has a SKILL.md
			skillMD := filepath.Join("skills", entry.Name(), "SKILL.md")
			if _, err := skillsFS.ReadFile(skillMD); err == nil {
				names = append(names, entry.Name())
			}
		}
	}
	return names
}

// BootstrapSkills copies starter skills to the target directory.
// It skips skills that already exist. Returns the number of skills copied.
func BootstrapSkills(targetDir string) (int, error) {
	count, _, err := BootstrapSkillsWithSkip(targetDir, nil)
	return count, err
}

// BootstrapSkillsWithSkip copies starter skills to the target directory,
// skipping skills in the skip list. Returns the count and names of installed skills.
func BootstrapSkillsWithSkip(targetDir string, skip []string) (int, []string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, nil, fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Build skip set
	skipSet := make(map[string]bool)
	for _, name := range skip {
		skipSet[name] = true
	}

	entries, err := skillsFS.ReadDir("skills")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read embedded skills: %w", err)
	}

	copied := 0
	var installed []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip if in skip list
		if skipSet[name] {
			continue
		}

		// Check if skill already exists
		targetPath := filepath.Join(targetDir, name)
		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		// Create skill directory
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return copied, installed, fmt.Errorf("failed to create skill directory %s: %w", name, err)
		}

		// Copy SKILL.md
		skillMDPath := filepath.Join("skills", name, "SKILL.md")
		content, err := skillsFS.ReadFile(skillMDPath)
		if err != nil {
			return copied, installed, fmt.Errorf("failed to read %s/SKILL.md: %w", name, err)
		}

		if err := os.WriteFile(filepath.Join(targetPath, "SKILL.md"), content, 0644); err != nil {
			return copied, installed, fmt.Errorf("failed to write %s/SKILL.md: %w", name, err)
		}

		// Copy any supporting files (if they exist)
		supportingFiles, _ := listSupportingFiles(name)
		for _, relPath := range supportingFiles {
			srcPath := filepath.Join("skills", name, relPath)
			destPath := filepath.Join(targetPath, relPath)

			// Create parent directory if needed
			if dir := filepath.Dir(destPath); dir != targetPath {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return copied, installed, fmt.Errorf("failed to create directory for %s: %w", relPath, err)
				}
			}

			fileContent, err := skillsFS.ReadFile(srcPath)
			if err != nil {
				return copied, installed, fmt.Errorf("failed to read %s: %w", srcPath, err)
			}

			if err := os.WriteFile(destPath, fileContent, 0644); err != nil {
				return copied, installed, fmt.Errorf("failed to write %s: %w", destPath, err)
			}
		}

		copied++
		installed = append(installed, name)
	}

	return copied, installed, nil
}

// BootstrapSkillsSelective copies only the specified starter skills to the target directory.
// It skips skills that already exist. Returns the count and names of installed skills.
func BootstrapSkillsSelective(targetDir string, names []string) (int, []string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, nil, fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Build set of requested names
	requested := make(map[string]bool)
	for _, name := range names {
		requested[name] = true
	}

	entries, err := skillsFS.ReadDir("skills")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read embedded skills: %w", err)
	}

	copied := 0
	var installed []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip if not in requested list
		if !requested[name] {
			continue
		}

		// Check if skill already exists
		targetPath := filepath.Join(targetDir, name)
		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		// Create skill directory
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return copied, installed, fmt.Errorf("failed to create skill directory %s: %w", name, err)
		}

		// Copy SKILL.md
		skillMDPath := filepath.Join("skills", name, "SKILL.md")
		content, err := skillsFS.ReadFile(skillMDPath)
		if err != nil {
			return copied, installed, fmt.Errorf("failed to read %s/SKILL.md: %w", name, err)
		}

		if err := os.WriteFile(filepath.Join(targetPath, "SKILL.md"), content, 0644); err != nil {
			return copied, installed, fmt.Errorf("failed to write %s/SKILL.md: %w", name, err)
		}

		copied++
		installed = append(installed, name)
	}

	return copied, installed, nil
}

// GetSkill returns the SKILL.md content for a starter skill by name.
func GetSkill(name string) ([]byte, error) {
	return skillsFS.ReadFile(filepath.Join("skills", name, "SKILL.md"))
}

// ListSkills returns all embedded skill directories.
func ListSkills() ([]fs.DirEntry, error) {
	return skillsFS.ReadDir("skills")
}

// LoadStarterSkills loads and parses all embedded starter skills.
func LoadStarterSkills() ([]*skills.Skill, error) {
	entries, err := skillsFS.ReadDir("skills")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded skills: %w", err)
	}

	var result []*skills.Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillMDPath := filepath.Join("skills", entry.Name(), "SKILL.md")
		content, err := skillsFS.ReadFile(skillMDPath)
		if err != nil {
			continue // Skip if no SKILL.md
		}

		skill, err := skills.Parse(string(content), skills.SourceStarter, "")
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", entry.Name(), err)
		}

		result = append(result, skill)
	}

	return result, nil
}

// listSupportingFiles returns all files in a skill directory except SKILL.md.
func listSupportingFiles(skillName string) ([]string, error) {
	var files []string

	skillDir := filepath.Join("skills", skillName)
	err := fs.WalkDir(skillsFS, skillDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root and directories
		if path == skillDir || d.IsDir() {
			return nil
		}

		// Skip SKILL.md
		if d.Name() == "SKILL.md" {
			return nil
		}

		// Get relative path from skill directory
		relPath := strings.TrimPrefix(path, skillDir+"/")
		files = append(files, relPath)

		return nil
	})

	return files, err
}
