package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Client wraps the GitHub API for staghorn's needs.
type Client struct {
	rest *api.RESTClient
}

// FetchResult contains the result of a fetch operation.
type FetchResult struct {
	Content     string
	ETag        string
	SHA         string
	NotModified bool // True if server returned 304
}

// NewClient creates a GitHub client using go-gh (automatic auth).
func NewClient() (*Client, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}
	return &Client{rest: client}, nil
}

// NewClientWithToken creates a GitHub client with explicit token.
func NewClientWithToken(token string) (*Client, error) {
	client, err := api.NewRESTClient(api.ClientOptions{
		AuthToken: token,
	})
	if err != nil {
		return nil, err
	}
	return &Client{rest: client}, nil
}

// NewUnauthenticatedClient creates a GitHub client without authentication.
// This works for public repositories only and has lower rate limits (60/hour).
// Use this when accessing public configs without requiring user auth.
func NewUnauthenticatedClient() (*Client, error) {
	client, err := api.NewRESTClient(api.ClientOptions{})
	if err != nil {
		return nil, err
	}
	return &Client{rest: client}, nil
}

// fileContentsResponse represents GitHub's contents API response.
type fileContentsResponse struct {
	Type        string `json:"type"`
	Encoding    string `json:"encoding"`
	Size        int    `json:"size"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	SHA         string `json:"sha"`
	URL         string `json:"url"`
	GitURL      string `json:"git_url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
}

// FetchFile fetches a file from a repo.
// If etag is provided and content hasn't changed, returns NotModified=true.
func (c *Client) FetchFile(ctx context.Context, owner, repo, path, branch string) (*FetchResult, error) {
	if owner == "" || repo == "" || path == "" {
		return nil, fmt.Errorf("owner, repo, and path are required")
	}

	endpoint := fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, url.PathEscape(path))
	if branch != "" {
		endpoint += "?ref=" + url.QueryEscape(branch)
	}

	var response fileContentsResponse
	err := c.rest.Get(endpoint, &response)
	if err != nil {
		return nil, err
	}

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(response.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode content: %w", err)
	}

	return &FetchResult{
		Content: string(content),
		SHA:     response.SHA,
	}, nil
}

// GetDefaultBranch returns the repo's default branch.
func (c *Client) GetDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	endpoint := fmt.Sprintf("repos/%s/%s", owner, repo)

	var response struct {
		DefaultBranch string `json:"default_branch"`
	}

	err := c.rest.Get(endpoint, &response)
	if err != nil {
		return "", err
	}

	return response.DefaultBranch, nil
}

// RepoExists checks if a repository exists and is accessible.
func (c *Client) RepoExists(ctx context.Context, owner, repo string) (bool, error) {
	endpoint := fmt.Sprintf("repos/%s/%s", owner, repo)

	var response struct {
		ID int `json:"id"`
	}

	err := c.rest.Get(endpoint, &response)
	if err != nil {
		// Check if it's a 404
		if httpErr, ok := err.(*api.HTTPError); ok {
			if httpErr.StatusCode == http.StatusNotFound {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

// FileExists checks if a file exists in a repo.
func (c *Client) FileExists(ctx context.Context, owner, repo, path, branch string) (bool, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, url.PathEscape(path))
	if branch != "" {
		endpoint += "?ref=" + url.QueryEscape(branch)
	}

	var response fileContentsResponse
	err := c.rest.Get(endpoint, &response)
	if err != nil {
		if httpErr, ok := err.(*api.HTTPError); ok {
			if httpErr.StatusCode == http.StatusNotFound {
				return false, nil
			}
		}
		return false, err
	}

	return response.Type == "file", nil
}

// DirectoryEntry represents an item in a directory listing.
type DirectoryEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "file" or "dir"
	SHA  string `json:"sha"`
}

// ListDirectory lists contents of a directory in a repo.
// Returns nil, nil if the directory doesn't exist.
func (c *Client) ListDirectory(ctx context.Context, owner, repo, path, branch string) ([]DirectoryEntry, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, url.PathEscape(path))
	if branch != "" {
		endpoint += "?ref=" + url.QueryEscape(branch)
	}

	var response []DirectoryEntry
	err := c.rest.Get(endpoint, &response)
	if err != nil {
		if httpErr, ok := err.(*api.HTTPError); ok {
			if httpErr.StatusCode == http.StatusNotFound {
				return nil, nil // Directory doesn't exist
			}
		}
		return nil, err
	}

	return response, nil
}
