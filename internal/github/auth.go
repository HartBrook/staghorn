// Package github provides GitHub API integration.
package github

import (
	"os"
	"os/exec"
	"strings"

	"github.com/HartBrook/staghorn/internal/errors"
)

const (
	// EnvGitHubToken is the environment variable for fallback token auth.
	EnvGitHubToken = "STAGHORN_GITHUB_TOKEN"
)

// GetToken resolves a GitHub token using the auth chain.
// Priority: 1) gh auth token, 2) STAGHORN_GITHUB_TOKEN env
func GetToken() (string, error) {
	// Try gh CLI first
	token, err := GetTokenFromGHCLI()
	if err == nil && token != "" {
		return token, nil
	}

	// Fall back to environment variable
	token = GetTokenFromEnv()
	if token != "" {
		return token, nil
	}

	return "", errors.GitHubAuthFailed(err)
}

// GetTokenFromGHCLI executes `gh auth token` to get token.
func GetTokenFromGHCLI() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetTokenFromEnv reads STAGHORN_GITHUB_TOKEN.
func GetTokenFromEnv() string {
	return os.Getenv(EnvGitHubToken)
}

// IsGHCLIAvailable checks if the gh CLI is installed and authenticated.
func IsGHCLIAvailable() bool {
	_, err := GetTokenFromGHCLI()
	return err == nil
}

// IsGHCLIInstalled checks if the gh CLI binary is installed (but not necessarily authenticated).
func IsGHCLIInstalled() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// AuthMethod returns a string describing the current auth method.
func AuthMethod() string {
	if _, err := GetTokenFromGHCLI(); err == nil {
		return "gh CLI"
	}
	if GetTokenFromEnv() != "" {
		return "STAGHORN_GITHUB_TOKEN"
	}
	return "none"
}
