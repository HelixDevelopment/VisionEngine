// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmvision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	anthropicDefaultBaseURL = "https://api.anthropic.com/v1"
	anthropicDefaultModel   = "claude-sonnet-4-20250514"
	anthropicMaxImageSize   = 20 * 1024 * 1024 // 20 MB
	anthropicAPIVersion     = "2023-06-01"
)

// AnthropicProvider implements VisionProvider for Claude.
type AnthropicProvider struct {
	config     ProviderConfig
	httpClient *http.Client
}

// NewAnthropicProvider creates a new Anthropic vision provider.
func NewAnthropicProvider(config ProviderConfig) (*AnthropicProvider, error) {
	if config.APIKey == "" {
		return nil, ErrNoAPIKey
	}
	if config.BaseURL == "" {
		config.BaseURL = anthropicDefaultBaseURL
	}
	if config.Model == "" {
		config.Model = anthropicDefaultModel
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.MaxImageSize == 0 {
		config.MaxImageSize = anthropicMaxImageSize
	}
	timeout := 60 * time.Second
	if config.TimeoutSecs > 0 {
		timeout = time.Duration(config.TimeoutSecs) * time.Second
	}
	return &AnthropicProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Name returns "anthropic".
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// SupportsVision returns true.
func (p *AnthropicProvider) SupportsVision() bool {
	return true
}

// MaxImageSize returns the max image size.
func (p *AnthropicProvider) MaxImageSize() int {
	return p.config.MaxImageSize
}

// AnalyzeImage sends an image to Claude for analysis.
func (p *AnthropicProvider) AnalyzeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	if err := validateImage(image, p.config.MaxImageSize); err != nil {
		return "", err
	}
	if err := validatePrompt(prompt); err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(image)
	body := map[string]any{
		"model":      p.config.Model,
		"max_tokens": p.config.MaxTokens,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "image",
						"source": map[string]string{
							"type":         "base64",
							"media_type":   "image/png",
							"data":         encoded,
						},
					},
					{"type": "text", "text": prompt},
				},
			},
		},
	}

	return p.sendRequest(ctx, body)
}

// CompareImages sends two images to Claude for comparison.
func (p *AnthropicProvider) CompareImages(ctx context.Context, img1, img2 []byte, prompt string) (string, error) {
	if err := validateImage(img1, p.config.MaxImageSize); err != nil {
		return "", err
	}
	if err := validateImage(img2, p.config.MaxImageSize); err != nil {
		return "", err
	}
	if err := validatePrompt(prompt); err != nil {
		return "", err
	}

	enc1 := base64.StdEncoding.EncodeToString(img1)
	enc2 := base64.StdEncoding.EncodeToString(img2)
	body := map[string]any{
		"model":      p.config.Model,
		"max_tokens": p.config.MaxTokens,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "image",
						"source": map[string]string{
							"type":       "base64",
							"media_type": "image/png",
							"data":       enc1,
						},
					},
					{
						"type": "image",
						"source": map[string]string{
							"type":       "base64",
							"media_type": "image/png",
							"data":       enc2,
						},
					},
					{"type": "text", "text": prompt},
				},
			},
		},
	}

	return p.sendRequest(ctx, body)
}

func (p *AnthropicProvider) sendRequest(ctx context.Context, body map[string]any) (string, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.config.BaseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d: %s", ErrProviderUnavailable, resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	for _, c := range result.Content {
		if c.Type == "text" {
			return c.Text, nil
		}
	}

	return "", fmt.Errorf("%w: no text content in response", ErrInvalidResponse)
}
