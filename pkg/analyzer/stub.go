// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// StubAnalyzer provides a stub implementation of Analyzer that works without OpenCV.
// It uses basic heuristics and returns placeholder data. For real vision analysis,
// use the OpenCV-based implementation (build tag: vision) or LLM vision providers.
type StubAnalyzer struct {
	// Provider is an optional LLM vision provider for intelligent analysis.
	Provider LLMVisionProvider
}

// LLMVisionProvider is a simplified interface for LLM vision integration.
// The full interface is in pkg/llmvision.
type LLMVisionProvider interface {
	AnalyzeImage(ctx context.Context, image []byte, prompt string) (string, error)
	CompareImages(ctx context.Context, img1, img2 []byte, prompt string) (string, error)
}

// NewStubAnalyzer creates a new StubAnalyzer.
func NewStubAnalyzer() *StubAnalyzer {
	return &StubAnalyzer{}
}

// NewStubAnalyzerWithProvider creates a new StubAnalyzer with an LLM vision provider.
func NewStubAnalyzerWithProvider(provider LLMVisionProvider) *StubAnalyzer {
	return &StubAnalyzer{Provider: provider}
}

// AnalyzeScreen performs stub screen analysis.
func (s *StubAnalyzer) AnalyzeScreen(ctx context.Context, screenshot []byte) (ScreenAnalysis, error) {
	if len(screenshot) == 0 {
		return ScreenAnalysis{}, ErrEmptyScreenshot
	}
	select {
	case <-ctx.Done():
		return ScreenAnalysis{}, ctx.Err()
	default:
	}

	id := fingerprint(screenshot)
	return ScreenAnalysis{
		ScreenID:    id,
		Title:       "Unknown Screen",
		Description: "Stub analysis - install OpenCV or use LLM vision for detailed analysis",
		Elements:    []UIElement{},
		TextRegions: []TextRegion{},
		Issues:      []VisualIssue{},
		Navigable:   []Action{},
		Timestamp:   time.Now(),
	}, nil
}

// CompareScreens performs stub screen comparison.
func (s *StubAnalyzer) CompareScreens(ctx context.Context, before, after []byte) (ScreenDiff, error) {
	if len(before) == 0 || len(after) == 0 {
		return ScreenDiff{}, ErrEmptyScreenshot
	}
	select {
	case <-ctx.Done():
		return ScreenDiff{}, ctx.Err()
	default:
	}

	// Simple byte comparison
	same := len(before) == len(after)
	if same {
		for i := range before {
			if before[i] != after[i] {
				same = false
				break
			}
		}
	}

	similarity := 0.0
	if same {
		similarity = 1.0
	}

	return ScreenDiff{
		Similarity:     similarity,
		ChangedRegions: []Rect{},
		NewElements:    []UIElement{},
		GoneElements:   []UIElement{},
		IsNewScreen:    !same,
	}, nil
}

// DetectElements returns an empty element list (stub).
func (s *StubAnalyzer) DetectElements(screenshot []byte) ([]UIElement, error) {
	if len(screenshot) == 0 {
		return nil, ErrEmptyScreenshot
	}
	return []UIElement{}, nil
}

// DetectText returns an empty text region list (stub).
func (s *StubAnalyzer) DetectText(screenshot []byte) ([]TextRegion, error) {
	if len(screenshot) == 0 {
		return nil, ErrEmptyScreenshot
	}
	return []TextRegion{}, nil
}

// IdentifyScreen returns a basic screen identity based on image hash.
func (s *StubAnalyzer) IdentifyScreen(ctx context.Context, screenshot []byte) (ScreenIdentity, error) {
	if len(screenshot) == 0 {
		return ScreenIdentity{}, ErrEmptyScreenshot
	}
	select {
	case <-ctx.Done():
		return ScreenIdentity{}, ctx.Err()
	default:
	}

	fp := fingerprint(screenshot)
	return ScreenIdentity{
		ID:          fmt.Sprintf("screen-%s", fp[:8]),
		Name:        "Unknown",
		Category:    "unknown",
		Fingerprint: fp,
		Tags:        []string{},
	}, nil
}

// DetectIssues returns an empty issue list (stub).
func (s *StubAnalyzer) DetectIssues(ctx context.Context, screenshot []byte) ([]VisualIssue, error) {
	if len(screenshot) == 0 {
		return nil, ErrEmptyScreenshot
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return []VisualIssue{}, nil
}

// fingerprint computes a SHA-256 fingerprint of the data.
func fingerprint(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
