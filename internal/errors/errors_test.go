package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicAuthFailed(t *testing.T) {
	err := AnthropicAuthFailed()

	assert.Equal(t, ErrAnthropicAuthFailed, err.Code)
	assert.Contains(t, err.Error(), "Anthropic API authentication failed")
	assert.Contains(t, err.Hint, "ANTHROPIC_API_KEY")
}

func TestOptimizationFailed(t *testing.T) {
	cause := errors.New("API timeout")
	err := OptimizationFailed("API request timed out", cause)

	assert.Equal(t, ErrOptimizationFailed, err.Code)
	assert.Contains(t, err.Error(), "optimization failed")
	assert.Contains(t, err.Error(), "API request timed out")
	assert.Contains(t, err.Hint, "--deterministic")

	// Test error unwrapping
	unwrapped := err.Unwrap()
	require.NotNil(t, unwrapped)
	assert.Equal(t, cause, unwrapped)
}

func TestOptimizationFailed_NilCause(t *testing.T) {
	err := OptimizationFailed("unknown error", nil)

	assert.Equal(t, ErrOptimizationFailed, err.Code)
	assert.Contains(t, err.Error(), "optimization failed")
	assert.Nil(t, err.Unwrap())
}

func TestValidationFailed(t *testing.T) {
	missing := []string{"pytest", "conftest.py", "ruff"}
	err := ValidationFailed(missing)

	assert.Equal(t, ErrValidationFailed, err.Code)
	assert.Contains(t, err.Error(), "optimization removed critical content")
	assert.Contains(t, err.Error(), "pytest")
	assert.Contains(t, err.Error(), "conftest.py")
	assert.Contains(t, err.Hint, "--force")
}

func TestValidationFailed_EmptyMissing(t *testing.T) {
	err := ValidationFailed([]string{})

	assert.Equal(t, ErrValidationFailed, err.Code)
	assert.Contains(t, err.Error(), "optimization removed critical content")
}

func TestStaghornError_Error(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := &StaghornError{
			Code:    ErrOptimizationFailed,
			Message: "test message",
		}
		assert.Equal(t, "test message", err.Error())
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("root cause")
		err := &StaghornError{
			Code:    ErrOptimizationFailed,
			Message: "test message",
			Cause:   cause,
		}
		assert.Equal(t, "test message: root cause", err.Error())
	})
}

func TestNew(t *testing.T) {
	err := New(ErrOptimizationFailed, "test message", "test hint")

	assert.Equal(t, ErrOptimizationFailed, err.Code)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, "test hint", err.Hint)
	assert.Nil(t, err.Cause)
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(ErrOptimizationFailed, "wrapper message", "wrapper hint", cause)

	assert.Equal(t, ErrOptimizationFailed, err.Code)
	assert.Equal(t, "wrapper message", err.Message)
	assert.Equal(t, "wrapper hint", err.Hint)
	assert.Equal(t, cause, err.Cause)
}
