// Package openrouter is a minimal client for OpenRouter's chat completions
// API (https://openrouter.ai/docs), which is OpenAI-compatible. This keeps
// the model swappable (Claude, GPT, or anything else OpenRouter proxies)
// purely through OPENROUTER_MODEL — no code change needed to switch models.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const chatCompletionsURL = "https://openrouter.ai/api/v1/chat/completions"

// Message is a single entry in a chat completion conversation, covering
// the "system", "user", "assistant", and "tool" roles.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall is an assistant message's request to invoke a tool.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall names the tool being called and its arguments, JSON-encoded
// as a string (matching OpenAI's function-calling wire format).
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool describes a callable tool to offer the model, in OpenAI's
// function-calling format.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient builds a client for the given OPENROUTER_API_KEY and
// OPENROUTER_MODEL (e.g. "anthropic/claude-3.5-sonnet", "openai/gpt-4o").
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ChatCompletion sends the conversation so far (plus the available tools)
// to OpenRouter and returns the model's next message — which may carry
// ToolCalls instead of (or alongside) Content, for the caller to execute
// and feed back in a follow-up call.
func (c *Client) ChatCompletion(ctx context.Context, messages []Message, tools []Tool) (Message, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
	})
	if err != nil {
		return Message{}, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionsURL, bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("request to OpenRouter failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, fmt.Errorf("failed to read OpenRouter response: %w", err)
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Message{}, fmt.Errorf("failed to parse OpenRouter response (status %d): %w", resp.StatusCode, err)
	}

	if parsed.Error != nil {
		return Message{}, fmt.Errorf("OpenRouter error: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return Message{}, fmt.Errorf("OpenRouter returned status %d: %s", resp.StatusCode, string(respBody))
	}
	if len(parsed.Choices) == 0 {
		return Message{}, fmt.Errorf("OpenRouter returned no choices")
	}

	return parsed.Choices[0].Message, nil
}
