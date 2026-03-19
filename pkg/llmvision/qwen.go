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
	qwenDefaultBaseURL = "https://dashscope.aliyuncs.com/api/v1"
	qwenDefaultModel   = "qwen-vl-max"
	qwenMaxImageSize   = 20 * 1024 * 1024 // 20 MB
)

// QwenProvider implements VisionProvider for Qwen-VL.
type QwenProvider struct {
	config     ProviderConfig
	httpClient *http.Client
}

// NewQwenProvider creates a new Qwen-VL vision provider.
func NewQwenProvider(config ProviderConfig) (*QwenProvider, error) {
	if config.APIKey == "" {
		return nil, ErrNoAPIKey
	}
	if config.BaseURL == "" {
		config.BaseURL = qwenDefaultBaseURL
	}
	if config.Model == "" {
		config.Model = qwenDefaultModel
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.MaxImageSize == 0 {
		config.MaxImageSize = qwenMaxImageSize
	}
	timeout := 60 * time.Second
	if config.TimeoutSecs > 0 {
		timeout = time.Duration(config.TimeoutSecs) * time.Second
	}
	return &QwenProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Name returns "qwen".
func (p *QwenProvider) Name() string {
	return "qwen"
}

// SupportsVision returns true.
func (p *QwenProvider) SupportsVision() bool {
	return true
}

// MaxImageSize returns the max image size.
func (p *QwenProvider) MaxImageSize() int {
	return p.config.MaxImageSize
}

// AnalyzeImage sends an image to Qwen-VL for analysis.
func (p *QwenProvider) AnalyzeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	if err := validateImage(image, p.config.MaxImageSize); err != nil {
		return "", err
	}
	if err := validatePrompt(prompt); err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(image)
	body := map[string]any{
		"model": p.config.Model,
		"input": map[string]any{
			"messages": []map[string]any{
				{
					"role": "user",
					"content": []map[string]any{
						{"text": prompt},
						{"image": fmt.Sprintf("data:image/png;base64,%s", encoded)},
					},
				},
			},
		},
		"parameters": map[string]any{
			"max_tokens": p.config.MaxTokens,
		},
	}

	return p.sendRequest(ctx, body)
}

// CompareImages sends two images to Qwen-VL for comparison.
func (p *QwenProvider) CompareImages(ctx context.Context, img1, img2 []byte, prompt string) (string, error) {
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
		"input": map[string]any{
			"messages": []map[string]any{
				{
					"role": "user",
					"content": []map[string]any{
						{"text": prompt},
						{"image": fmt.Sprintf("data:image/png;base64,%s", enc1)},
						{"image": fmt.Sprintf("data:image/png;base64,%s", enc2)},
					},
				},
			},
		},
		"parameters": map[string]any{
			"max_tokens": p.config.MaxTokens,
		},
	}

	return p.sendRequest(ctx, body)
}

func (p *QwenProvider) sendRequest(ctx context.Context, body map[string]any) (string, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.config.BaseURL+"/services/aigc/multimodal-generation/generation",
		bytes.NewReader(jsonBody))
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
		Output struct {
			Choices []struct {
				Message struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if len(result.Output.Choices) == 0 || len(result.Output.Choices[0].Message.Content) == 0 {
		return "", fmt.Errorf("%w: no content in response", ErrInvalidResponse)
	}

	return result.Output.Choices[0].Message.Content[0].Text, nil
}
