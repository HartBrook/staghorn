package skills

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Header prefix used to identify staghorn-managed skills.
const HeaderManagedPrefix = "<!-- Managed by staghorn"

// ConvertToClaude converts a staghorn skill's SKILL.md to Claude Code format.
// This adds the staghorn header while preserving the original frontmatter and body.
func ConvertToClaude(skill *Skill) string {
	var sb strings.Builder

	// Write frontmatter (preserve all fields for Claude Code compatibility)
	sb.WriteString("---\n")

	// Build frontmatter map to preserve field order and include all fields
	fm := buildClaudeFrontmatter(skill)
	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		// Fallback to minimal frontmatter if marshal fails
		sb.WriteString(fmt.Sprintf("name: %s\ndescription: %s\n", skill.Name, skill.Description))
	} else {
		sb.Write(yamlBytes)
	}
	sb.WriteString("---\n\n")

	// Add staghorn header after frontmatter
	sb.WriteString(fmt.Sprintf("%s | Source: %s | Do not edit directly -->\n\n", HeaderManagedPrefix, skill.Source.Label()))

	// Add args hint if there are arguments
	if len(skill.Args) > 0 {
		sb.WriteString(buildArgsHint(skill))
		sb.WriteString("\n")
	}

	// Add the body
	sb.WriteString(skill.Body)
	sb.WriteString("\n")

	return sb.String()
}

// buildClaudeFrontmatter creates the frontmatter map for Claude Code.
func buildClaudeFrontmatter(skill *Skill) map[string]any {
	fm := make(map[string]any)

	// Required fields
	fm["name"] = skill.Name
	fm["description"] = skill.Description

	// Optional Agent Skills standard fields
	if skill.License != "" {
		fm["license"] = skill.License
	}
	if skill.Compatibility != "" {
		fm["compatibility"] = skill.Compatibility
	}
	if len(skill.Metadata) > 0 {
		fm["metadata"] = skill.Metadata
	}
	if skill.AllowedTools != "" {
		fm["allowed-tools"] = skill.AllowedTools
	}

	// Claude Code extensions
	if skill.DisableModelInvocation {
		fm["disable-model-invocation"] = true
	}
	if skill.UserInvocable != nil {
		fm["user-invocable"] = *skill.UserInvocable
	}
	if skill.Context != "" {
		fm["context"] = skill.Context
	}
	if skill.Agent != "" {
		fm["agent"] = skill.Agent
	}
	if skill.ArgumentHint != "" {
		fm["argument-hint"] = skill.ArgumentHint
	}
	if skill.Model != "" {
		fm["model"] = skill.Model
	}
	if skill.Hooks != nil {
		fm["hooks"] = skill.Hooks
	}

	return fm
}

// buildArgsHint creates a usage hint comment for Claude to understand the args.
func buildArgsHint(skill *Skill) string {
	var sb strings.Builder

	// Build args list
	var argParts []string
	for _, arg := range skill.Args {
		part := arg.Name
		if arg.Required {
			part += " (required)"
		} else if arg.Default != "" {
			part += fmt.Sprintf(" (default: %s)", arg.Default)
		}
		argParts = append(argParts, part)
	}

	sb.WriteString(fmt.Sprintf("<!-- Args: %s -->\n", strings.Join(argParts, ", ")))

	// Build example usage
	var exampleParts []string
	for _, arg := range skill.Args {
		val := arg.Default
		if val == "" {
			val = "<value>"
		}
		exampleParts = append(exampleParts, fmt.Sprintf("%s=%q", arg.Name, val))
	}
	sb.WriteString(fmt.Sprintf("<!-- Example: /%s %s -->\n", skill.Name, strings.Join(exampleParts, " ")))

	return sb.String()
}

// SyncToClaude syncs a skill to Claude Code's skills directory.
// This copies the entire skill directory, preserving structure.
// Returns the number of files written.
func SyncToClaude(skill *Skill, claudeSkillsDir string) (int, error) {
	destDir := filepath.Join(claudeSkillsDir, skill.Name)

	// Check for collision with non-staghorn skill
	destSkillMD := filepath.Join(destDir, "SKILL.md")
	if existingContent, err := os.ReadFile(destSkillMD); err == nil {
		if !strings.Contains(string(existingContent), HeaderManagedPrefix) {
			return 0, fmt.Errorf("existing skill not managed by staghorn")
		}
	}

	// Create the skill directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create skill directory: %w", err)
	}

	filesWritten := 0

	// Write the converted SKILL.md
	content := ConvertToClaude(skill)
	if err := os.WriteFile(destSkillMD, []byte(content), 0644); err != nil {
		return filesWritten, fmt.Errorf("failed to write SKILL.md: %w", err)
	}
	filesWritten++

	// Copy supporting files
	for relPath, srcPath := range skill.SupportingFiles {
		destPath := filepath.Join(destDir, relPath)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return filesWritten, fmt.Errorf("failed to create directory for %s: %w", relPath, err)
		}

		// Copy the file
		if err := copyFile(srcPath, destPath); err != nil {
			return filesWritten, fmt.Errorf("failed to copy %s: %w", relPath, err)
		}
		filesWritten++
	}

	return filesWritten, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// RemoveSkill removes a skill directory from Claude Code's skills directory.
func RemoveSkill(name, claudeSkillsDir string) error {
	destDir := filepath.Join(claudeSkillsDir, name)

	// Check if it exists and is managed by staghorn
	destSkillMD := filepath.Join(destDir, "SKILL.md")
	if existingContent, err := os.ReadFile(destSkillMD); err == nil {
		if !strings.Contains(string(existingContent), HeaderManagedPrefix) {
			return fmt.Errorf("skill not managed by staghorn")
		}
	} else if os.IsNotExist(err) {
		return nil // Nothing to remove
	} else {
		return err
	}

	return os.RemoveAll(destDir)
}

// ListManagedSkills returns a list of skill names in the Claude skills directory
// that are managed by staghorn.
func ListManagedSkills(claudeSkillsDir string) ([]string, error) {
	entries, err := os.ReadDir(claudeSkillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillMD := filepath.Join(claudeSkillsDir, entry.Name(), "SKILL.md")
		content, err := os.ReadFile(skillMD)
		if err != nil {
			continue
		}

		if strings.Contains(string(content), HeaderManagedPrefix) {
			names = append(names, entry.Name())
		}
	}

	return names, nil
}
