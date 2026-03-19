// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmvision

import (
	"context"
	"fmt"
	"sync"
)

// FallbackProvider wraps multiple VisionProviders with a fallback chain.
// If the primary provider fails, subsequent providers are tried in order.
type FallbackProvider struct {
	providers []VisionProvider
	mu        sync.RWMutex
}

// NewFallbackProvider creates a new FallbackProvider with the given providers.
// Providers are tried in the order given.
func NewFallbackProvider(providers ...VisionProvider) (*FallbackProvider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one provider is required")
	}
	return &FallbackProvider{
		providers: providers,
	}, nil
}

// Name returns "fallback".
func (f *FallbackProvider) Name() string {
	return "fallback"
}

// SupportsVision returns true if any provider supports vision.
func (f *FallbackProvider) SupportsVision() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, p := range f.providers {
		if p.SupportsVision() {
			return true
		}
	}
	return false
}

// MaxImageSize returns the minimum max image size across all providers.
func (f *FallbackProvider) MaxImageSize() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	min := 0
	for _, p := range f.providers {
		s := p.MaxImageSize()
		if min == 0 || s < min {
			min = s
		}
	}
	return min
}

// AnalyzeImage tries each provider in order until one succeeds.
func (f *FallbackProvider) AnalyzeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	f.mu.RLock()
	providers := make([]VisionProvider, len(f.providers))
	copy(providers, f.providers)
	f.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		result, err := p.AnalyzeImage(ctx, image, prompt)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("all providers failed, last error: %w", lastErr)
}

// CompareImages tries each provider in order until one succeeds.
func (f *FallbackProvider) CompareImages(ctx context.Context, img1, img2 []byte, prompt string) (string, error) {
	f.mu.RLock()
	providers := make([]VisionProvider, len(f.providers))
	copy(providers, f.providers)
	f.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		result, err := p.CompareImages(ctx, img1, img2, prompt)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("all providers failed, last error: %w", lastErr)
}

// Providers returns a copy of the provider list.
func (f *FallbackProvider) Providers() []VisionProvider {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make([]VisionProvider, len(f.providers))
	copy(result, f.providers)
	return result
}
