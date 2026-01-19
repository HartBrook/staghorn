package optimize

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/HartBrook/staghorn/internal/errors"
)

const (
	defaultBaseURL   = "https://api.anthropic.com/v1"
	defaultModel     = "claude-sonnet-4-20250514"
	defaultMaxTokens = 8192
	apiVersion       = "2023-06-01"
)

// Client handles communication with the Claude API.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithModel sets the model to use.
func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// NewClient creates a new Claude API client.
// It reads the API key from the ANTHROPIC_API_KEY environment variable.
func NewClient(opts ...ClientOption) (*Client, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, errors.AnthropicAuthFailed()
	}

	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		model:   defaultModel,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Message represents a message in the Claude API.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// messagesRequest represents a request to the messages API.
type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
}

// contentBlock represents a content block in the response.
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// messagesResponse represents a response from the messages API.
type messagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []contentBlock `json:"content"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// apiError represents an error from the Claude API.
type apiError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Optimize sends content to Claude for optimization.
func (c *Client) Optimize(ctx context.Context, content string, targetTokens int, anchors []string) (string, error) {
	systemPrompt := buildSystemPrompt()
	userPrompt := buildUserPrompt(content, targetTokens, anchors)

	req := messagesRequest{
		Model:     c.model,
		MaxTokens: defaultMaxTokens,
		System:    systemPrompt,
		Messages: []Message{
			{Role: "user", Content: userPrompt},
		},
	}

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return "", err
	}

	// Extract text from response
	var result string
	for _, block := range resp.Content {
		if block.Type == "text" {
			result += block.Text
		}
	}

	return result, nil
}

// sendRequest sends a request to the Claude API.
func (c *Client) sendRequest(ctx context.Context, req messagesRequest) (*messagesResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.OptimizationFailed("failed to encode request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, errors.OptimizationFailed("failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", apiVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.OptimizationFailed("API request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.OptimizationFailed("failed to read response", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var apiErr apiError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			return nil, errors.OptimizationFailed(
				fmt.Sprintf("API error (%d): %s", resp.StatusCode, apiErr.Error.Message),
				nil,
			)
		}
		return nil, errors.OptimizationFailed(
			fmt.Sprintf("API returned status %d", resp.StatusCode),
			nil,
		)
	}

	var result messagesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errors.OptimizationFailed("failed to decode response", err)
	}

	return &result, nil
}
