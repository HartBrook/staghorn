// Package cache manages local cached team configs.
package cache

import (
	"fmt"
	"time"
)

// Metadata stores cache state for conditional fetching.
type Metadata struct {
	Owner       string    `json:"owner"`
	Repo        string    `json:"repo"`
	ETag        string    `json:"etag,omitempty"`
	SHA         string    `json:"sha,omitempty"`
	LastFetched time.Time `json:"last_fetched"`
}

// IsStale returns true if cache is at or older than the TTL.
func (m *Metadata) IsStale(ttl time.Duration) bool {
	return time.Since(m.LastFetched) >= ttl
}

// Age returns human-readable age string.
func (m *Metadata) Age() string {
	duration := time.Since(m.LastFetched)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// RepoString returns "owner/repo" format.
func (m *Metadata) RepoString() string {
	return fmt.Sprintf("%s/%s", m.Owner, m.Repo)
}
