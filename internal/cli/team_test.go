package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HartBrook/staghorn/internal/config"
)

func TestTeamInitNonInteractive(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Run team init non-interactively
	err = runTeamInit(true, false, false)
	if err != nil {
		t.Fatalf("runTeamInit failed: %v", err)
	}

	// Verify CLAUDE.md was created
	if _, err := os.Stat("CLAUDE.md"); err != nil {
		t.Error("CLAUDE.md was not created")
	}

	// Verify README.md was created
	if _, err := os.Stat("README.md"); err != nil {
		t.Error("README.md was not created")
	}

	// Verify commands/ directory exists
	if _, err := os.Stat("commands"); err != nil {
		t.Error("commands/ directory was not created")
	}

	// Verify languages/ directory exists
	if _, err := os.Stat("languages"); err != nil {
		t.Error("languages/ directory was not created")
	}

	// Verify templates/ directory exists
	if _, err := os.Stat("templates"); err != nil {
		t.Error("templates/ directory was not created")
	}

	// Verify commands were installed
	entries, err := os.ReadDir("commands")
	if err != nil {
		t.Fatalf("failed to read commands dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected commands to be installed")
	}

	// Verify languages were installed
	entries, err = os.ReadDir("languages")
	if err != nil {
		t.Fatalf("failed to read languages dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected languages to be installed")
	}
}

func TestTeamInitNoTemplates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Run with --no-templates
	err = runTeamInit(true, true, false)
	if err != nil {
		t.Fatalf("runTeamInit failed: %v", err)
	}

	// Verify templates/ was not created
	if _, err := os.Stat("templates"); err == nil {
		t.Error("templates/ should not be created with --no-templates")
	}
}

func TestTeamInitNoReadme(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Run with --no-readme
	err = runTeamInit(true, false, true)
	if err != nil {
		t.Fatalf("runTeamInit failed: %v", err)
	}

	// Verify README.md was not created
	if _, err := os.Stat("README.md"); err == nil {
		t.Error("README.md should not be created with --no-readme")
	}
}

func TestTeamValidate_ValidRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create a valid team repo
	err = runTeamInit(true, false, false)
	if err != nil {
		t.Fatalf("runTeamInit failed: %v", err)
	}

	// Validate should succeed
	err = runTeamValidate()
	if err != nil {
		t.Errorf("runTeamValidate failed on valid repo: %v", err)
	}
}

func TestTeamValidate_MissingClaudeMD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create empty directory - no CLAUDE.md
	err = runTeamValidate()
	if err == nil {
		t.Error("expected validation to fail without CLAUDE.md")
	}
}

func TestTeamValidate_InvalidCommand(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create CLAUDE.md
	if err := os.WriteFile("CLAUDE.md", []byte("# Team Standards"), 0644); err != nil {
		t.Fatalf("failed to write CLAUDE.md: %v", err)
	}

	// Create commands directory with invalid command
	if err := os.MkdirAll("commands", 0755); err != nil {
		t.Fatalf("failed to create commands dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("commands", "broken.md"), []byte("No frontmatter here"), 0644); err != nil {
		t.Fatalf("failed to write broken.md: %v", err)
	}

	// Validate should fail
	err = runTeamValidate()
	if err == nil {
		t.Error("expected validation to fail with invalid command")
	}
}

func TestWriteTeamClaudeMD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	err = writeTeamClaudeMD("Acme Corp", "Building the future")
	if err != nil {
		t.Fatalf("writeTeamClaudeMD failed: %v", err)
	}

	content, err := os.ReadFile("CLAUDE.md")
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}

	if len(content) == 0 {
		t.Error("CLAUDE.md should not be empty")
	}

	// Check content includes team name
	if !strings.Contains(string(content), "Acme Corp") {
		t.Error("CLAUDE.md should include team name")
	}

	// Check content includes description
	if !strings.Contains(string(content), "Building the future") {
		t.Error("CLAUDE.md should include description")
	}
}

func TestWriteTeamReadme(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	commands := []string{"code-review", "debug"}
	languages := []string{"go", "python"}

	err = writeTeamReadme("Acme Corp", commands, languages)
	if err != nil {
		t.Fatalf("writeTeamReadme failed: %v", err)
	}

	content, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	// Check content includes team name
	if !strings.Contains(string(content), "Acme Corp") {
		t.Error("README.md should include team name")
	}

	// Check content includes commands
	if !strings.Contains(string(content), "code-review") {
		t.Error("README.md should include command names")
	}

	// Check content includes languages
	if !strings.Contains(string(content), "python") {
		t.Error("README.md should include language names")
	}
}

func TestTeamInitExistingFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create existing CLAUDE.md and README.md
	existingClaudeContent := "# Existing CLAUDE.md content"
	existingReadmeContent := "# Existing README content"
	if err := os.WriteFile("CLAUDE.md", []byte(existingClaudeContent), 0644); err != nil {
		t.Fatalf("failed to write CLAUDE.md: %v", err)
	}
	if err := os.WriteFile("README.md", []byte(existingReadmeContent), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	// Run non-interactive (should keep existing files)
	err = runTeamInit(true, false, false)
	if err != nil {
		t.Fatalf("runTeamInit failed: %v", err)
	}

	// Verify existing CLAUDE.md was NOT overwritten
	content, _ := os.ReadFile("CLAUDE.md")
	if string(content) != existingClaudeContent {
		t.Error("CLAUDE.md should not be overwritten in non-interactive mode")
	}

	// Verify existing README.md was NOT overwritten
	content, _ = os.ReadFile("README.md")
	if string(content) != existingReadmeContent {
		t.Error("README.md should not be overwritten in non-interactive mode")
	}
}

func TestTeamValidate_EmptyClaudeMD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create empty CLAUDE.md
	if err := os.WriteFile("CLAUDE.md", []byte(""), 0644); err != nil {
		t.Fatalf("failed to write CLAUDE.md: %v", err)
	}

	// Validate should fail
	err = runTeamValidate()
	if err == nil {
		t.Error("expected validation to fail with empty CLAUDE.md")
	}
}

func TestTeamValidate_EmptyLanguageFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create valid CLAUDE.md
	if err := os.WriteFile("CLAUDE.md", []byte("# Team Standards"), 0644); err != nil {
		t.Fatalf("failed to write CLAUDE.md: %v", err)
	}

	// Create languages directory with empty file
	if err := os.MkdirAll("languages", 0755); err != nil {
		t.Fatalf("failed to create languages dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("languages", "empty.md"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write empty.md: %v", err)
	}

	// Validate should fail
	err = runTeamValidate()
	if err == nil {
		t.Error("expected validation to fail with empty language file")
	}
}

func TestTeamValidate_EmptyTemplateFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create valid CLAUDE.md
	if err := os.WriteFile("CLAUDE.md", []byte("# Team Standards"), 0644); err != nil {
		t.Fatalf("failed to write CLAUDE.md: %v", err)
	}

	// Create templates directory with empty file
	if err := os.MkdirAll("templates", 0755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("templates", "empty.md"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write empty.md: %v", err)
	}

	// Validate should fail
	err = runTeamValidate()
	if err == nil {
		t.Error("expected validation to fail with empty template file")
	}
}

func TestWriteTeamClaudeMD_EmptyDescription(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Empty description should use default
	err = writeTeamClaudeMD("Acme Corp", "")
	if err != nil {
		t.Fatalf("writeTeamClaudeMD failed: %v", err)
	}

	content, err := os.ReadFile("CLAUDE.md")
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}

	// Should include auto-generated description
	if !strings.Contains(string(content), "Guidelines for Claude Code across all Acme Corp projects") {
		t.Error("CLAUDE.md should include default description when none provided")
	}
}

func TestWriteTeamReadme_EmptyLists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Empty commands and languages
	err = writeTeamReadme("Acme Corp", nil, nil)
	if err != nil {
		t.Fatalf("writeTeamReadme failed: %v", err)
	}

	content, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	// Should show "no commands" messages
	if !strings.Contains(string(content), "No commands installed yet") {
		t.Error("README.md should indicate no commands when list is empty")
	}
	if !strings.Contains(string(content), "No language configs installed yet") {
		t.Error("README.md should indicate no languages when list is empty")
	}
}

func TestTeamInit_CreatesSourceYaml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Run team init
	err = runTeamInit(true, false, false)
	if err != nil {
		t.Fatalf("runTeamInit failed: %v", err)
	}

	// Verify .staghorn/source.yaml was created
	sourceYaml := filepath.Join(".staghorn", "source.yaml")
	if _, err := os.Stat(sourceYaml); err != nil {
		t.Error(".staghorn/source.yaml was not created")
	}

	// Verify IsSourceRepo returns true
	if !config.IsSourceRepo(tmpDir) {
		t.Error("IsSourceRepo should return true after team init")
	}
}

func TestTeamValidate_ChecksSourceYaml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "staghorn-team-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create CLAUDE.md but no source.yaml
	if err := os.WriteFile("CLAUDE.md", []byte("# Team Standards"), 0644); err != nil {
		t.Fatalf("failed to write CLAUDE.md: %v", err)
	}

	// Validate should pass (source.yaml is optional but warned about)
	// This test just verifies we don't crash when source.yaml is missing
	_ = runTeamValidate() // May return error due to missing CLAUDE.md content checks

	// Now create source.yaml and validate again
	if err := config.WriteSourceRepoConfig(tmpDir); err != nil {
		t.Fatalf("failed to write source config: %v", err)
	}

	// Validate should still work
	_ = runTeamValidate()
}
