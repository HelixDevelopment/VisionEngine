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
	openAIDefaultBaseURL = "https://api.openai.com/v1"
	openAIDefaultModel   = "gpt-4o"
	openAIMaxImageSize   = 20 * 1024 * 1024 // 20 MB
)

// OpenAIProvider implements VisionProvider for GPT-4o.
type OpenAIProvider struct {
	config     ProviderConfig
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI vision provider.
func NewOpenAIProvider(config ProviderConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, ErrNoAPIKey
	}
	if config.BaseURL == "" {
		config.BaseURL = openAIDefaultBaseURL
	}
	if config.Model == "" {
		config.Model = openAIDefaultModel
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.MaxImageSize == 0 {
		config.MaxImageSize = openAIMaxImageSize
	}
	timeout := 60 * time.Second
	if config.TimeoutSecs > 0 {
		timeout = time.Duration(config.TimeoutSecs) * time.Second
	}
	return &OpenAIProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Name returns "openai".
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// SupportsVision returns true.
func (p *OpenAIProvider) SupportsVision() bool {
	return true
}

// MaxImageSize returns the max image size.
func (p *OpenAIProvider) MaxImageSize() int {
	return p.config.MaxImageSize
}

// AnalyzeImage sends an image to GPT-4o for analysis.
func (p *OpenAIProvider) AnalyzeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	if err := validateImage(image, p.config.MaxImageSize); err != nil {
		return "", err
	}
	if err := validatePrompt(prompt); err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(image)
	body := map[string]any{
		"model": p.config.Model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": prompt},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:image/png;base64,%s", encoded),
						},
					},
				},
			},
		},
		"max_tokens": p.config.MaxTokens,
	}

	return p.sendRequest(ctx, body)
}

// CompareImages sends two images to GPT-4o for comparison.
func (p *OpenAIProvider) CompareImages(ctx context.Context, img1, img2 []byte, prompt string) (string, error) {
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
		"model": p.config.Model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": prompt},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:image/png;base64,%s", enc1),
						},
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:image/png;base64,%s", enc2),
						},
					},
				},
			},
		},
		"max_tokens": p.config.MaxTokens,
	}

	return p.sendRequest(ctx, body)
}

func (p *OpenAIProvider) sendRequest(ctx context.Context, body map[string]any) (string, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.config.BaseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

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
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("%w: no choices in response", ErrInvalidResponse)
	}

	return result.Choices[0].Message.Content, nil
}
