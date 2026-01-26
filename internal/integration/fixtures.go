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
	Team        *TeamSetup        `yaml:"team"`
	MultiSource []MultiSourceRepo `yaml:"multi_source,omitempty"`
	Personal    *PersonalSetup    `yaml:"personal"`
	Config      *ConfigSetup      `yaml:"config"`
}

// MultiSourceRepo defines content from a specific repository in multi-source setup.
type MultiSourceRepo struct {
	Source    string            `yaml:"source"`
	ClaudeMD  string            `yaml:"claude_md,omitempty"`
	Languages map[string]string `yaml:"languages,omitempty"`
	Commands  map[string]string `yaml:"commands,omitempty"`
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
	Source    interface{}     `yaml:"source"` // Can be string or SourceConfigSetup
	Languages *LanguagesSetup `yaml:"languages"`
}

// SourceConfigSetup defines multi-source configuration in fixtures.
type SourceConfigSetup struct {
	Default   string            `yaml:"default"`
	Base      string            `yaml:"base,omitempty"`
	Languages map[string]string `yaml:"languages,omitempty"`
	Commands  map[string]string `yaml:"commands,omitempty"`
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
	// Either Team or MultiSource must be present
	if f.Setup.Team == nil && len(f.Setup.MultiSource) == 0 {
		return fmt.Errorf("missing required field: setup.team or setup.multi_source")
	}
	if f.Setup.Team != nil && f.Setup.Team.Source == "" {
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
// It logs warnings for malformed or missing fields in the source configuration.
func (c *ConfigSetup) ToConfig() *config.Config {
	cfg := &config.Config{
		Version: c.Version,
	}

	// Handle source - can be string or SourceConfigSetup
	switch src := c.Source.(type) {
	case string:
		cfg.Source = config.Source{Simple: src}
	case map[string]interface{}:
		// YAML unmarshals objects as map[string]interface{}
		multi := &config.SourceConfig{}
		if def, ok := src["default"].(string); ok {
			multi.Default = def
		}
		if base, ok := src["base"].(string); ok {
			multi.Base = base
		}
		if langs, ok := src["languages"].(map[string]interface{}); ok {
			multi.Languages = make(map[string]string)
			for k, v := range langs {
				if vs, ok := v.(string); ok {
					multi.Languages[k] = vs
				}
			}
		}
		if cmds, ok := src["commands"].(map[string]interface{}); ok {
			multi.Commands = make(map[string]string)
			for k, v := range cmds {
				if vs, ok := v.(string); ok {
					multi.Commands[k] = vs
				}
			}
		}
		// Validate that default is set for multi-source configs
		if multi.Default == "" {
			// Fall back to simple source if default is missing
			cfg.Source = config.Source{}
		} else {
			cfg.Source = config.Source{Multi: multi}
		}
	case nil:
		// Source is nil - this will be caught by validation
	default:
		// Unknown type - leave source empty
	}

	if c.Languages != nil {
		cfg.Languages.Enabled = c.Languages.Enabled
		cfg.Languages.Disabled = c.Languages.Disabled
	}

	return cfg
}

// ApplySetup applies the fixture setup to a test environment.
func ApplySetup(env *TestEnv, setup FixtureSetup) error {
	// Handle multi-source setup
	if len(setup.MultiSource) > 0 {
		for _, src := range setup.MultiSource {
			owner, repo := parseOwnerRepo(src.Source)

			// Setup base config for this source
			if src.ClaudeMD != "" {
				if err := env.SetupTeamConfig(owner, repo, src.ClaudeMD); err != nil {
					return err
				}
			}

			// Setup languages for this source
			for lang, content := range src.Languages {
				if err := env.SetupTeamLanguage(owner, repo, lang, content); err != nil {
					return err
				}
			}

			// Setup commands for this source
			for cmd, content := range src.Commands {
				if err := env.SetupTeamCommand(owner, repo, cmd, content); err != nil {
					return err
				}
			}
		}
	}

	// Handle single-source team setup (backwards compatible)
	if setup.Team != nil {
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
