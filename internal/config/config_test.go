package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRepo(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "full github.com URL",
			repo:      "github.com/acme/standards",
			wantOwner: "acme",
			wantRepo:  "standards",
		},
		{
			name:      "https URL",
			repo:      "https://github.com/acme/standards",
			wantOwner: "acme",
			wantRepo:  "standards",
		},
		{
			name:      "short format",
			repo:      "acme/standards",
			wantOwner: "acme",
			wantRepo:  "standards",
		},
		{
			name:      "with hyphens and underscores",
			repo:      "my-org/my_repo-name",
			wantOwner: "my-org",
			wantRepo:  "my_repo-name",
		},
		{
			name:      "with dots",
			repo:      "org.name/repo.name",
			wantOwner: "org.name",
			wantRepo:  "repo.name",
		},
		{
			name:      "https URL with .git suffix",
			repo:      "https://github.com/acme/standards.git",
			wantOwner: "acme",
			wantRepo:  "standards",
		},
		{
			name:      "URL with trailing slash",
			repo:      "https://github.com/acme/standards/",
			wantOwner: "acme",
			wantRepo:  "standards",
		},
		{
			name:      "http URL",
			repo:      "http://github.com/acme/standards",
			wantOwner: "acme",
			wantRepo:  "standards",
		},
		{
			name:      "URL with /tree/main",
			repo:      "https://github.com/acme/standards/tree/main",
			wantOwner: "acme",
			wantRepo:  "standards",
		},
		{
			name:      "URL with /tree/feature-branch",
			repo:      "https://github.com/acme/standards/tree/feature-branch",
			wantOwner: "acme",
			wantRepo:  "standards",
		},
		{
			name:      "URL with /blob/main/file",
			repo:      "https://github.com/acme/standards/blob/main/CLAUDE.md",
			wantOwner: "acme",
			wantRepo:  "standards",
		},
		{
			name:    "missing repo",
			repo:    "acme",
			wantErr: true,
		},
		{
			name:    "empty string",
			repo:    "",
			wantErr: true,
		},
		{
			name:      "extra path segments are ignored",
			repo:      "github.com/acme/standards/extra",
			wantOwner: "acme",
			wantRepo:  "standards",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &TeamConfig{Repo: tt.repo}
			owner, repo, err := cfg.ParseRepo()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRepo() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRepo() unexpected error: %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("ParseRepo() owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("ParseRepo() repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid minimal config",
			config: Config{
				Team: TeamConfig{
					Repo: "acme/standards",
				},
			},
			wantErr: false,
		},
		{
			name: "valid full config",
			config: Config{
				Version: 1,
				Team: TeamConfig{
					Repo:   "github.com/acme/standards",
					Branch: "main",
					Path:   "CLAUDE.md",
				},
				Cache: CacheConfig{
					TTL: "24h",
				},
			},
			wantErr: false,
		},
		{
			name: "missing repo",
			config: Config{
				Team: TeamConfig{},
			},
			wantErr: true,
		},
		{
			name: "invalid repo format",
			config: Config{
				Team: TeamConfig{
					Repo: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid TTL format",
			config: Config{
				Team: TeamConfig{
					Repo: "acme/standards",
				},
				Cache: CacheConfig{
					TTL: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.applyDefaults()
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := &Config{
		Team: TeamConfig{
			Repo: "acme/standards",
		},
	}

	cfg.applyDefaults()

	if cfg.Version != DefaultVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, DefaultVersion)
	}
	if cfg.Team.Path != DefaultPath {
		t.Errorf("Path = %q, want %q", cfg.Team.Path, DefaultPath)
	}
	if cfg.Cache.TTL != DefaultCacheTTL {
		t.Errorf("TTL = %q, want %q", cfg.Cache.TTL, DefaultCacheTTL)
	}
}

func TestLoadAndSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	original := &Config{
		Version: 1,
		Team: TeamConfig{
			Repo:   "acme/standards",
			Branch: "main",
		},
		Cache: CacheConfig{
			TTL: "12h",
		},
	}

	// Save
	if err := SaveTo(original, configPath); err != nil {
		t.Fatalf("SaveTo() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Config file not created: %v", err)
	}

	// Load
	loaded, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom() error: %v", err)
	}

	if loaded.Team.Repo != original.Team.Repo {
		t.Errorf("Repo = %q, want %q", loaded.Team.Repo, original.Team.Repo)
	}
	if loaded.Team.Branch != original.Team.Branch {
		t.Errorf("Branch = %q, want %q", loaded.Team.Branch, original.Team.Branch)
	}
	if loaded.Cache.TTL != original.Cache.TTL {
		t.Errorf("TTL = %q, want %q", loaded.Cache.TTL, original.Cache.TTL)
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := LoadFrom("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadFrom() expected error for nonexistent file")
	}
}

func TestTTLDuration(t *testing.T) {
	tests := []struct {
		ttl      string
		wantHrs  int
	}{
		{"24h", 24},
		{"1h", 1},
		{"168h", 168}, // 1 week
		{"invalid", 24}, // Falls back to default
		{"", 24},        // Falls back to default
	}

	for _, tt := range tests {
		t.Run(tt.ttl, func(t *testing.T) {
			cfg := CacheConfig{TTL: tt.ttl}
			d := cfg.TTLDuration()
			gotHrs := int(d.Hours())
			if gotHrs != tt.wantHrs {
				t.Errorf("TTLDuration() = %d hours, want %d", gotHrs, tt.wantHrs)
			}
		})
	}
}
