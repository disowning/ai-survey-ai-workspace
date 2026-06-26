package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	apiKey         string
	baseURL        string
	model          string
	embeddingModel string
	http           *http.Client
}

func NewClient(apiKey, baseURL, model, embeddingModel string) *Client {
	return &Client{
		apiKey:         strings.TrimSpace(apiKey),
		baseURL:        strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		model:          strings.TrimSpace(model),
		embeddingModel: strings.TrimSpace(embeddingModel),
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) Configured() bool {
	return c.apiKey != "" && c.baseURL != "" && c.model != ""
}

func (c *Client) EmbeddingsConfigured() bool {
	return c.apiKey != "" && c.baseURL != "" && c.embeddingModel != ""
}

func (c *Client) ChatModel() string {
	return c.model
}

func (c *Client) EmbeddingModel() string {
	return c.embeddingModel
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) Chat(ctx context.Context, messages []ChatMessage, temperature float64) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("AI provider is not configured")
	}

	body, err := json.Marshal(chatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: temperature,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var parsed chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return "", fmt.Errorf("AI provider error: %s", parsed.Error.Message)
		}
		return "", fmt.Errorf("AI provider error: HTTP %d", resp.StatusCode)
	}

	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("AI provider returned empty response")
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func (c *Client) Embedding(ctx context.Context, input string) ([]float64, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("embedding input is empty")
	}
	if !c.EmbeddingsConfigured() {
		return nil, fmt.Errorf("AI embedding provider is not configured")
	}

	body, err := json.Marshal(embeddingRequest{
		Model: c.embeddingModel,
		Input: input,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return nil, fmt.Errorf("AI embedding provider error: %s", parsed.Error.Message)
		}
		return nil, fmt.Errorf("AI embedding provider error: HTTP %d", resp.StatusCode)
	}
	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("AI embedding provider returned empty response")
	}

	return parsed.Data[0].Embedding, nil
}
