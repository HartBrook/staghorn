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
			owner, repo, err := ParseRepo(tt.repo)

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
				Source: Source{Simple: "acme/standards"},
			},
			wantErr: false,
		},
		{
			name: "valid full config",
			config: Config{
				Version: 1,
				Source:  Source{Simple: "github.com/acme/standards"},
				Cache: CacheConfig{
					TTL: "24h",
				},
			},
			wantErr: false,
		},
		{
			name: "empty source (local-only mode)",
			config: Config{
				Source: Source{},
			},
			wantErr: false,
		},
		{
			name: "invalid source format",
			config: Config{
				Source: Source{Simple: "invalid"},
			},
			wantErr: true,
		},
		{
			name: "invalid TTL format",
			config: Config{
				Source: Source{Simple: "acme/standards"},
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
		Source: Source{Simple: "acme/standards"},
	}

	cfg.applyDefaults()

	if cfg.Version != DefaultVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, DefaultVersion)
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
		Source:  Source{Simple: "acme/standards"},
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

	if loaded.Source.DefaultRepo() != original.Source.DefaultRepo() {
		t.Errorf("Source = %q, want %q", loaded.Source.DefaultRepo(), original.Source.DefaultRepo())
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
		ttl     string
		wantHrs int
	}{
		{"24h", 24},
		{"1h", 1},
		{"168h", 168},   // 1 week
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

func TestSourceConfig(t *testing.T) {
	t.Run("simple source", func(t *testing.T) {
		s := Source{Simple: "acme/standards"}
		if s.DefaultRepo() != "acme/standards" {
			t.Errorf("DefaultRepo() = %q, want %q", s.DefaultRepo(), "acme/standards")
		}
		if s.RepoForBase() != "acme/standards" {
			t.Errorf("RepoForBase() = %q, want %q", s.RepoForBase(), "acme/standards")
		}
		if s.RepoForLanguage("python") != "acme/standards" {
			t.Errorf("RepoForLanguage(python) = %q, want %q", s.RepoForLanguage("python"), "acme/standards")
		}
	})

	t.Run("multi source", func(t *testing.T) {
		s := Source{
			Multi: &SourceConfig{
				Default: "acme/standards",
				Base:    "acme/base-config",
				Languages: map[string]string{
					"python": "community/python-standards",
				},
				Commands: map[string]string{
					"code-review": "acme/internal-commands",
				},
			},
		}

		if s.DefaultRepo() != "acme/standards" {
			t.Errorf("DefaultRepo() = %q, want %q", s.DefaultRepo(), "acme/standards")
		}
		if s.RepoForBase() != "acme/base-config" {
			t.Errorf("RepoForBase() = %q, want %q", s.RepoForBase(), "acme/base-config")
		}
		if s.RepoForLanguage("python") != "community/python-standards" {
			t.Errorf("RepoForLanguage(python) = %q, want %q", s.RepoForLanguage("python"), "community/python-standards")
		}
		if s.RepoForLanguage("go") != "acme/standards" {
			t.Errorf("RepoForLanguage(go) = %q, want %q (should fall back to default)", s.RepoForLanguage("go"), "acme/standards")
		}
		if s.RepoForCommand("code-review") != "acme/internal-commands" {
			t.Errorf("RepoForCommand(code-review) = %q, want %q", s.RepoForCommand("code-review"), "acme/internal-commands")
		}
	})

	t.Run("all repos", func(t *testing.T) {
		s := Source{
			Multi: &SourceConfig{
				Default: "acme/standards",
				Base:    "acme/base-config",
				Languages: map[string]string{
					"python": "community/python-standards",
				},
			},
		}

		repos := s.AllRepos()
		if len(repos) != 3 {
			t.Errorf("AllRepos() = %d repos, want 3", len(repos))
		}
	})
}

func TestTrust(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		trusted []string
		want    bool
	}{
		{
			name:    "exact match",
			repo:    "acme/standards",
			trusted: []string{"acme/standards"},
			want:    true,
		},
		{
			name:    "org-level trust",
			repo:    "acme/any-repo",
			trusted: []string{"acme"},
			want:    true,
		},
		{
			name:    "no match",
			repo:    "other/repo",
			trusted: []string{"acme"},
			want:    false,
		},
		{
			name:    "empty trusted list",
			repo:    "acme/standards",
			trusted: []string{},
			want:    false,
		},
		{
			name:    "case insensitive",
			repo:    "ACME/Standards",
			trusted: []string{"acme/standards"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTrusted(tt.repo, tt.trusted)
			if got != tt.want {
				t.Errorf("IsTrusted(%q, %v) = %v, want %v", tt.repo, tt.trusted, got, tt.want)
			}
		})
	}
}

func TestConfig_IsTrustedSource_DefaultSources(t *testing.T) {
	// Config with empty trusted list should still trust default sources
	cfg := &Config{
		Trusted: []string{},
	}

	tests := []struct {
		name string
		repo string
		want bool
	}{
		{
			name: "official community repo is trusted by default",
			repo: "HartBrook/staghorn-community",
			want: true,
		},
		{
			name: "official community repo case insensitive",
			repo: "hartbrook/staghorn-community",
			want: true,
		},
		{
			name: "random repo is not trusted",
			repo: "random/repo",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.IsTrustedSource(tt.repo)
			if got != tt.want {
				t.Errorf("IsTrustedSource(%q) = %v, want %v", tt.repo, got, tt.want)
			}
		})
	}
}
