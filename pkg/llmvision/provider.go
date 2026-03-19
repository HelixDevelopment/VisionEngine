// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package llmvision provides LLM Vision API adapters for image analysis.
package llmvision

import (
	"context"
	"errors"
)

var (
	// ErrNoAPIKey is returned when no API key is configured.
	ErrNoAPIKey = errors.New("API key not configured")
	// ErrEmptyImage is returned when an empty image is provided.
	ErrEmptyImage = errors.New("empty image data")
	// ErrEmptyPrompt is returned when an empty prompt is provided.
	ErrEmptyPrompt = errors.New("empty prompt")
	// ErrImageTooLarge is returned when the image exceeds the max size.
	ErrImageTooLarge = errors.New("image exceeds maximum size")
	// ErrProviderUnavailable is returned when the provider cannot be reached.
	ErrProviderUnavailable = errors.New("vision provider unavailable")
	// ErrRateLimited is returned when the API rate limit is hit.
	ErrRateLimited = errors.New("API rate limited")
	// ErrInvalidResponse is returned when the API response is invalid.
	ErrInvalidResponse = errors.New("invalid API response")
)

// VisionProvider is the interface for LLM vision API adapters.
type VisionProvider interface {
	// AnalyzeImage sends an image with a prompt to the vision API.
	AnalyzeImage(ctx context.Context, image []byte, prompt string) (string, error)
	// CompareImages sends two images with a prompt for comparison.
	CompareImages(ctx context.Context, img1, img2 []byte, prompt string) (string, error)
	// SupportsVision returns true if the provider supports vision.
	SupportsVision() bool
	// MaxImageSize returns the maximum image size in bytes.
	MaxImageSize() int
	// Name returns the provider name.
	Name() string
}

// ProviderConfig holds common configuration for vision providers.
type ProviderConfig struct {
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url,omitempty"`
	Model        string `json:"model,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	MaxImageSize int    `json:"max_image_size,omitempty"` // bytes
	TimeoutSecs  int    `json:"timeout_secs,omitempty"`
}

// DefaultMaxImageSize is the default maximum image size (20 MB).
const DefaultMaxImageSize = 20 * 1024 * 1024

// validateImage checks that the image data is valid for the provider.
func validateImage(image []byte, maxSize int) error {
	if len(image) == 0 {
		return ErrEmptyImage
	}
	if maxSize > 0 && len(image) > maxSize {
		return ErrImageTooLarge
	}
	return nil
}

// validatePrompt checks that the prompt is non-empty.
func validatePrompt(prompt string) error {
	if prompt == "" {
		return ErrEmptyPrompt
	}
	return nil
}
