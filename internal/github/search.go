package github

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// SearchResult represents a repository found via search.
type SearchResult struct {
	Owner       string
	Repo        string
	Description string
	Stars       int
	Topics      []string
	URL         string
}

// FullName returns the owner/repo string.
func (r *SearchResult) FullName() string {
	return r.Owner + "/" + r.Repo
}

// searchResponse represents GitHub's search API response.
type searchResponse struct {
	TotalCount int `json:"total_count"`
	Items      []struct {
		FullName    string   `json:"full_name"`
		Description string   `json:"description"`
		Stars       int      `json:"stargazers_count"`
		Topics      []string `json:"topics"`
		HTMLURL     string   `json:"html_url"`
		Owner       struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	} `json:"items"`
}

// SearchByTopic searches for repositories with a specific topic.
// This is used to discover public staghorn configs tagged with "staghorn-config".
func (c *Client) SearchByTopic(ctx context.Context, topic string, query string) ([]SearchResult, error) {
	// Build search query
	q := fmt.Sprintf("topic:%s", topic)
	if query != "" {
		q += " " + query
	}

	endpoint := fmt.Sprintf("search/repositories?q=%s&sort=stars&order=desc&per_page=30",
		url.QueryEscape(q))

	var response searchResponse
	err := c.rest.Get(endpoint, &response)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := make([]SearchResult, 0, len(response.Items))
	for _, item := range response.Items {
		results = append(results, SearchResult{
			Owner:       item.Owner.Login,
			Repo:        item.Name,
			Description: item.Description,
			Stars:       item.Stars,
			Topics:      item.Topics,
			URL:         item.HTMLURL,
		})
	}

	return results, nil
}

// SearchConfigs searches for staghorn config repositories.
// It searches for repos with the "staghorn-config" topic and optionally filters by query.
func (c *Client) SearchConfigs(ctx context.Context, query string) ([]SearchResult, error) {
	return c.SearchByTopic(ctx, "staghorn-config", query)
}

// languageAliases maps common language variations to their canonical topic names.
// This allows users to search with variations like "golang" or "python3" and still
// match repos tagged with "go" or "python".
var languageAliases = map[string]string{
	// Go
	"golang": "go",
	// Python
	"python3": "python",
	"python2": "python",
	"py":      "python",
	"py3":     "python",
	// JavaScript
	"js":         "javascript",
	"ecmascript": "javascript",
	"node":       "javascript",
	"nodejs":     "javascript",
	// TypeScript
	"ts": "typescript",
	// Ruby
	"rb": "ruby",
	// Rust
	"rs": "rust",
	// C++
	"cpp":       "c++",
	"cplusplus": "c++",
	// C#
	"csharp": "c#",
	"cs":     "c#",
	"dotnet": "c#",
	// Kotlin
	"kt": "kotlin",
	// Shell/Bash
	"shell": "bash",
	"sh":    "bash",
	"zsh":   "bash",
	// YAML
	"yml": "yaml",
	// Swift (no common aliases)
	// Java (no common aliases)
}

// normalizeLanguage converts common language aliases to their canonical form.
// For example, "golang" becomes "go", "python3" becomes "python".
func normalizeLanguage(lang string) string {
	lang = strings.ToLower(lang)
	if canonical, ok := languageAliases[lang]; ok {
		return canonical
	}
	return lang
}

// FilterByLanguage filters search results to only those supporting a specific language.
// Languages are determined by the repo's topics (e.g., "python", "typescript").
// Common aliases are normalized (e.g., "golang" -> "go", "py" -> "python").
func FilterByLanguage(results []SearchResult, lang string) []SearchResult {
	if lang == "" {
		return results
	}

	lang = normalizeLanguage(lang)
	filtered := make([]SearchResult, 0)

	for _, r := range results {
		for _, topic := range r.Topics {
			if normalizeLanguage(topic) == lang {
				filtered = append(filtered, r)
				break
			}
		}
	}

	return filtered
}

// FilterByTag filters search results to only those with a specific tag/topic.
func FilterByTag(results []SearchResult, tag string) []SearchResult {
	if tag == "" {
		return results
	}

	tag = strings.ToLower(tag)
	filtered := make([]SearchResult, 0)

	for _, r := range results {
		for _, topic := range r.Topics {
			if strings.ToLower(topic) == tag {
				filtered = append(filtered, r)
				break
			}
		}
	}

	return filtered
}

// SortByStars sorts results by star count (descending).
func SortByStars(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Stars > results[j].Stars
	})
}
