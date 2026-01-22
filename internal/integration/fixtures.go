package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HartBrook/staghorn/internal/config"
	"gopkg.in/yaml.v3"
)

// Fixture represents a test scenario loaded from YAML.
type Fixture struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Setup       FixtureSetup      `yaml:"setup"`
	Assertions  FixtureAssertions `yaml:"assertions"`
}

// FixtureSetup defines the test environment setup.
type FixtureSetup struct {
	Team     *TeamSetup     `yaml:"team"`
	Personal *PersonalSetup `yaml:"personal"`
	Config   *ConfigSetup   `yaml:"config"`
}

// TeamSetup simulates the team repo content.
type TeamSetup struct {
	Source    string            `yaml:"source"`
	ClaudeMD  string            `yaml:"claude_md"`
	Languages map[string]string `yaml:"languages"`
}

// PersonalSetup defines personal config files.
type PersonalSetup struct {
	PersonalMD string            `yaml:"personal_md"`
	Languages  map[string]string `yaml:"languages"`
}

// ConfigSetup defines the staghorn config.yaml content.
type ConfigSetup struct {
	Version   int             `yaml:"version"`
	Source    string          `yaml:"source"`
	Languages *LanguagesSetup `yaml:"languages"`
}

// LanguagesSetup defines language configuration.
type LanguagesSetup struct {
	Enabled  []string `yaml:"enabled"`
	Disabled []string `yaml:"disabled"`
}

// FixtureAssertions defines what to verify.
type FixtureAssertions struct {
	OutputExists bool             `yaml:"output_exists"`
	Header       *HeaderAssertion `yaml:"header"`
	Provenance   *ProvenanceCheck `yaml:"provenance"`
	Contains     []string         `yaml:"contains"`
	NotContains  []string         `yaml:"not_contains"`
	Sections     []string         `yaml:"sections"`
	Languages    []LanguageCheck  `yaml:"languages"`
}

// HeaderAssertion checks the staghorn header.
type HeaderAssertion struct {
	ManagedBy  bool   `yaml:"managed_by"`
	SourceRepo string `yaml:"source_repo"`
}

// ProvenanceCheck verifies provenance markers.
type ProvenanceCheck struct {
	HasTeam     bool     `yaml:"has_team"`
	HasPersonal bool     `yaml:"has_personal"`
	Order       []string `yaml:"order"`
}

// LanguageCheck verifies language section content.
type LanguageCheck struct {
	Name               string   `yaml:"name"`
	HasTeamContent     bool     `yaml:"has_team_content"`
	HasPersonalContent bool     `yaml:"has_personal_content"`
	Contains           []string `yaml:"contains"`
}

// LoadFixture loads a fixture from a YAML file.
func LoadFixture(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var fixture Fixture
	if err := yaml.Unmarshal(data, &fixture); err != nil {
		return nil, err
	}

	if err := fixture.Validate(); err != nil {
		return nil, fmt.Errorf("invalid fixture %s: %w", path, err)
	}

	return &fixture, nil
}

// Validate checks that the fixture has all required fields.
func (f *Fixture) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if f.Setup.Team == nil {
		return fmt.Errorf("missing required field: setup.team")
	}
	if f.Setup.Team.Source == "" {
		return fmt.Errorf("missing required field: setup.team.source")
	}
	if f.Setup.Config == nil {
		return fmt.Errorf("missing required field: setup.config")
	}
	return nil
}

// LoadAllFixtures loads all fixtures from a directory.
func LoadAllFixtures(dir string) ([]*Fixture, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var fixtures []*Fixture
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".yaml" && filepath.Ext(name) != ".yml" {
			continue
		}

		fixture, err := LoadFixture(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		fixtures = append(fixtures, fixture)
	}

	return fixtures, nil
}

// ToConfig converts fixture config setup to a config.Config.
func (c *ConfigSetup) ToConfig() *config.Config {
	cfg := &config.Config{
		Version: c.Version,
		Source:  config.Source{Simple: c.Source},
	}

	if c.Languages != nil {
		cfg.Languages.Enabled = c.Languages.Enabled
		cfg.Languages.Disabled = c.Languages.Disabled
	}

	return cfg
}

// ApplySetup applies the fixture setup to a test environment.
func ApplySetup(env *TestEnv, setup FixtureSetup) error {
	if setup.Team == nil {
		return nil
	}

	// Parse owner/repo from source
	owner, repo := parseOwnerRepo(setup.Team.Source)

	// Setup team config
	if setup.Team.ClaudeMD != "" {
		if err := env.SetupTeamConfig(owner, repo, setup.Team.ClaudeMD); err != nil {
			return err
		}
	}

	// Setup team languages
	for lang, content := range setup.Team.Languages {
		if err := env.SetupTeamLanguage(owner, repo, lang, content); err != nil {
			return err
		}
	}

	// Setup personal config
	if setup.Personal != nil {
		if setup.Personal.PersonalMD != "" {
			if err := env.SetupPersonalConfig(setup.Personal.PersonalMD); err != nil {
				return err
			}
		}

		// Setup personal languages
		for lang, content := range setup.Personal.Languages {
			if err := env.SetupPersonalLanguage(lang, content); err != nil {
				return err
			}
		}
	}

	// Setup config.yaml
	if setup.Config != nil {
		cfg := setup.Config.ToConfig()
		if err := env.SetupConfig(cfg); err != nil {
			return err
		}
	}

	return nil
}

// parseOwnerRepo splits "owner/repo" into owner and repo.
func parseOwnerRepo(source string) (string, string) {
	parts := strings.SplitN(source, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return source, ""
}
