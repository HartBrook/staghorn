// Package errors provides typed errors for staghorn.
package errors

import "fmt"

// ErrorCode identifies the type of error.
type ErrorCode string

const (
	ErrConfigNotFound    ErrorCode = "CONFIG_NOT_FOUND"
	ErrConfigInvalid     ErrorCode = "CONFIG_INVALID"
	ErrGitHubAuthFailed  ErrorCode = "GITHUB_AUTH_FAILED"
	ErrGitHubFetchFailed ErrorCode = "GITHUB_FETCH_FAILED"
	ErrCacheNotFound     ErrorCode = "CACHE_NOT_FOUND"
	ErrCacheStale        ErrorCode = "CACHE_STALE"
	ErrNoNetwork         ErrorCode = "NO_NETWORK"
	ErrProjectNotFound   ErrorCode = "PROJECT_NOT_FOUND"
	ErrInvalidRepo       ErrorCode = "INVALID_REPO"
)

// StaghornError represents a typed error with user-friendly hints.
type StaghornError struct {
	Code    ErrorCode
	Message string
	Hint    string
	Cause   error
}

func (e *StaghornError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *StaghornError) Unwrap() error {
	return e.Cause
}

// New creates a new StaghornError.
func New(code ErrorCode, message, hint string) *StaghornError {
	return &StaghornError{
		Code:    code,
		Message: message,
		Hint:    hint,
	}
}

// Wrap creates a new StaghornError wrapping an existing error.
func Wrap(code ErrorCode, message, hint string, cause error) *StaghornError {
	return &StaghornError{
		Code:    code,
		Message: message,
		Hint:    hint,
		Cause:   cause,
	}
}

// ConfigNotFound returns an error for missing config file.
func ConfigNotFound(path string) *StaghornError {
	return &StaghornError{
		Code:    ErrConfigNotFound,
		Message: fmt.Sprintf("config file not found: %s", path),
		Hint:    "Run `staghorn init` to create a configuration",
	}
}

// ConfigInvalid returns an error for invalid config.
func ConfigInvalid(reason string) *StaghornError {
	return &StaghornError{
		Code:    ErrConfigInvalid,
		Message: fmt.Sprintf("invalid config: %s", reason),
		Hint:    "Check your config file at ~/.config/staghorn/config.yaml",
	}
}

// GitHubAuthFailed returns an error for authentication failures.
func GitHubAuthFailed(cause error) *StaghornError {
	return &StaghornError{
		Code:    ErrGitHubAuthFailed,
		Message: "GitHub authentication failed",
		Hint:    "Run `gh auth login` or set STAGHORN_GITHUB_TOKEN environment variable",
		Cause:   cause,
	}
}

// GitHubFetchFailed returns an error for fetch failures.
func GitHubFetchFailed(repo string, cause error) *StaghornError {
	return &StaghornError{
		Code:    ErrGitHubFetchFailed,
		Message: fmt.Sprintf("failed to fetch from %s", repo),
		Hint:    "Check that the repository exists and you have access",
		Cause:   cause,
	}
}

// CacheNotFound returns an error when cache doesn't exist.
func CacheNotFound(repo string) *StaghornError {
	return &StaghornError{
		Code:    ErrCacheNotFound,
		Message: fmt.Sprintf("no cached config for %s", repo),
		Hint:    "Run `staghorn sync` to fetch the team config",
	}
}

// InvalidRepo returns an error for malformed repo strings.
func InvalidRepo(repo string) *StaghornError {
	return &StaghornError{
		Code:    ErrInvalidRepo,
		Message: fmt.Sprintf("invalid repository format: %s", repo),
		Hint:    "Use format: github.com/owner/repo or owner/repo",
	}
}
