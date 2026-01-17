// Package config handles staghorn configuration.
package config

import "strings"

// DefaultTrustedSources contains sources that are trusted by default.
// These are official staghorn repositories maintained by HartBrook.
var DefaultTrustedSources = []string{
	"HartBrook/staghorn-community",
}

// IsTrusted checks if a repository is in the trusted list.
// The trusted list can contain:
//   - Full repo references: "owner/repo"
//   - Org-level trust: "owner" (trusts all repos from that owner)
func IsTrusted(repo string, trusted []string) bool {
	if len(trusted) == 0 {
		return false
	}

	// Parse the repo once upfront
	repoOwner, repoName, err := ParseRepo(repo)
	if err != nil {
		return false
	}

	for _, t := range trusted {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}

		// Check for org-level trust (just "owner" without slash)
		if !strings.Contains(t, "/") {
			if strings.EqualFold(repoOwner, t) {
				return true
			}
			continue
		}

		// Parse the trusted entry and compare owner/repo
		tOwner, tRepo, err := ParseRepo(t)
		if err == nil && strings.EqualFold(tOwner, repoOwner) && strings.EqualFold(tRepo, repoName) {
			return true
		}
	}

	return false
}

// TrustWarning returns a warning message for untrusted sources.
func TrustWarning(repo string) string {
	return `⚠️  You're installing from an untrusted source: ` + repo + `

    This source is not in your trusted list.
    Review the source before proceeding: https://github.com/` + repo + `

    To trust this source, add it to your config:
      trusted:
        - ` + repo + `
`
}
