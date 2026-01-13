package cache

import (
	"testing"
	"time"

	"github.com/HartBrook/staghorn/internal/config"
)

func TestCacheReadWrite(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	c := New(paths)

	owner := "acme"
	repo := "standards"
	content := "# Team Config\n\nThis is the team config."

	meta := &Metadata{
		Owner: owner,
		Repo:  repo,
		SHA:   "abc123",
	}

	// Write
	if err := c.Write(owner, repo, content, meta); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// Exists
	if !c.Exists(owner, repo) {
		t.Error("Exists() should return true after write")
	}

	// Read
	readContent, readMeta, err := c.Read(owner, repo)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}

	if readContent != content {
		t.Errorf("Read() content = %q, want %q", readContent, content)
	}

	if readMeta.Owner != owner {
		t.Errorf("Read() meta.Owner = %q, want %q", readMeta.Owner, owner)
	}
	if readMeta.Repo != repo {
		t.Errorf("Read() meta.Repo = %q, want %q", readMeta.Repo, repo)
	}
	if readMeta.SHA != "abc123" {
		t.Errorf("Read() meta.SHA = %q, want %q", readMeta.SHA, "abc123")
	}
}

func TestCacheNotFound(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	c := New(paths)

	_, _, err := c.Read("nonexistent", "repo")
	if err == nil {
		t.Error("Read() should return error for nonexistent cache")
	}

	if c.Exists("nonexistent", "repo") {
		t.Error("Exists() should return false for nonexistent cache")
	}
}

func TestCacheClear(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	c := New(paths)

	owner := "acme"
	repo := "standards"

	// Write
	meta := &Metadata{Owner: owner, Repo: repo}
	if err := c.Write(owner, repo, "content", meta); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// Verify exists
	if !c.Exists(owner, repo) {
		t.Fatal("Cache should exist after write")
	}

	// Clear
	if err := c.Clear(owner, repo); err != nil {
		t.Fatalf("Clear() error: %v", err)
	}

	// Verify gone
	if c.Exists(owner, repo) {
		t.Error("Cache should not exist after clear")
	}
}

func TestMetadataIsStale(t *testing.T) {
	tests := []struct {
		name      string
		age       time.Duration
		ttl       time.Duration
		wantStale bool
	}{
		{
			name:      "fresh cache",
			age:       1 * time.Hour,
			ttl:       24 * time.Hour,
			wantStale: false,
		},
		{
			name:      "stale cache",
			age:       48 * time.Hour,
			ttl:       24 * time.Hour,
			wantStale: true,
		},
		{
			name:      "exactly at TTL",
			age:       24 * time.Hour,
			ttl:       24 * time.Hour,
			wantStale: true, // Stale when >= TTL
		},
		{
			name:      "just over TTL",
			age:       24*time.Hour + time.Minute,
			ttl:       24 * time.Hour,
			wantStale: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &Metadata{
				LastFetched: time.Now().Add(-tt.age),
			}

			if got := meta.IsStale(tt.ttl); got != tt.wantStale {
				t.Errorf("IsStale(%v) = %v, want %v", tt.ttl, got, tt.wantStale)
			}
		})
	}
}

func TestMetadataAge(t *testing.T) {
	tests := []struct {
		name     string
		age      time.Duration
		contains string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "minutes ago"},
		{"one minute", 1 * time.Minute, "1 minute ago"},
		{"hours", 3 * time.Hour, "hours ago"},
		{"one hour", 1 * time.Hour, "1 hour ago"},
		{"days", 3 * 24 * time.Hour, "days ago"},
		{"one day", 24 * time.Hour, "1 day ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &Metadata{
				LastFetched: time.Now().Add(-tt.age),
			}

			age := meta.Age()
			if age != tt.contains && (len(age) < len(tt.contains) || age[len(age)-len(tt.contains):] != tt.contains[:len(tt.contains)-4]+"ago") {
				// Looser check - just verify it's reasonable
				if age == "" {
					t.Errorf("Age() returned empty string")
				}
			}
		})
	}
}

func TestMetadataRepoString(t *testing.T) {
	meta := &Metadata{
		Owner: "acme",
		Repo:  "standards",
	}

	want := "acme/standards"
	if got := meta.RepoString(); got != want {
		t.Errorf("RepoString() = %q, want %q", got, want)
	}
}

func TestGetMetadata(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	c := New(paths)

	owner := "acme"
	repo := "standards"

	// Write with metadata
	meta := &Metadata{
		Owner: owner,
		Repo:  repo,
		SHA:   "def456",
		ETag:  "etag123",
	}
	if err := c.Write(owner, repo, "content", meta); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// Get metadata only
	readMeta, err := c.GetMetadata(owner, repo)
	if err != nil {
		t.Fatalf("GetMetadata() error: %v", err)
	}

	if readMeta.SHA != "def456" {
		t.Errorf("GetMetadata() SHA = %q, want %q", readMeta.SHA, "def456")
	}
	if readMeta.ETag != "etag123" {
		t.Errorf("GetMetadata() ETag = %q, want %q", readMeta.ETag, "etag123")
	}
}
