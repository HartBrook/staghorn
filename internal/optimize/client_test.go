package optimize

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_NoAPIKey(t *testing.T) {
	// Unset the API key
	original := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	_, err := NewClient()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Anthropic API authentication failed")
}

func TestNewClient_WithAPIKey(t *testing.T) {
	// Set a test API key
	original := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	client, err := NewClient()

	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, defaultModel, client.model)
	assert.Equal(t, defaultBaseURL, client.baseURL)
}

func TestNewClient_WithOptions(t *testing.T) {
	original := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	customClient := &http.Client{}
	client, err := NewClient(
		WithModel("claude-opus-4-20250514"),
		WithBaseURL("https://custom.api.com"),
		WithHTTPClient(customClient),
	)

	require.NoError(t, err)
	assert.Equal(t, "claude-opus-4-20250514", client.model)
	assert.Equal(t, "https://custom.api.com", client.baseURL)
	assert.Equal(t, customClient, client.httpClient)
}

func TestClient_Optimize(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/messages", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
		assert.Equal(t, apiVersion, r.Header.Get("anthropic-version"))

		// Return mock response
		resp := messagesResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "## Optimized\n\n- Use pytest\n- Use ruff\n"},
			},
			StopReason: "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Set API key
	original := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	client, err := NewClient(WithBaseURL(server.URL))
	require.NoError(t, err)

	result, err := client.Optimize(
		context.Background(),
		"## Testing\n\nYou should use pytest for all tests. Make sure to use ruff for linting.",
		100,
		[]string{"pytest", "ruff"},
	)

	require.NoError(t, err)
	assert.Contains(t, result, "pytest")
	assert.Contains(t, result, "ruff")
}

func TestClient_Optimize_RateLimited(t *testing.T) {
	// Create a mock server that returns 429
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(apiError{
			Type: "error",
			Error: struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{
				Type:    "rate_limit_error",
				Message: "Rate limit exceeded",
			},
		})
	}))
	defer server.Close()

	original := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	client, err := NewClient(WithBaseURL(server.URL))
	require.NoError(t, err)

	_, err = client.Optimize(context.Background(), "content", 100, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rate limit exceeded")
}

func TestClient_Optimize_APIError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(apiError{
			Type: "error",
			Error: struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{
				Type:    "invalid_request_error",
				Message: "Invalid request: missing required field",
			},
		})
	}))
	defer server.Close()

	original := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	client, err := NewClient(WithBaseURL(server.URL))
	require.NoError(t, err)

	_, err = client.Optimize(context.Background(), "content", 100, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid request")
}

func TestClient_Optimize_NetworkError(t *testing.T) {
	original := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	// Use an invalid URL to simulate network error
	client, err := NewClient(WithBaseURL("http://localhost:1"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = client.Optimize(ctx, "content", 100, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "optimization failed")
}

func TestClient_Optimize_InvalidJSON(t *testing.T) {
	// Create a mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	original := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	client, err := NewClient(WithBaseURL(server.URL))
	require.NoError(t, err)

	_, err = client.Optimize(context.Background(), "content", 100, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}
