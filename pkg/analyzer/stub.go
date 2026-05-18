// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"digital.vasic.visionengine/pkg/i18n"
)

// StubAnalyzer provides a stub implementation of Analyzer that works without OpenCV.
// It uses basic heuristics and returns placeholder data. For real vision analysis,
// use the OpenCV-based implementation (build tag: vision) or LLM vision providers.
type StubAnalyzer struct {
	// Provider is an optional LLM vision provider for intelligent analysis.
	Provider LLMVisionProvider
	// translator is the CONST-046 user-facing-string seam. nil falls
	// back to i18n.NoopTranslator + bundled English fallback so the
	// module stays standalone-buildable per CONST-051(B).
	translator i18n.Translator
}

// SetTranslator wires a consuming-project Translator. nil resets to
// NoopTranslator default. This is the CONST-046 seam — consumers that
// want localized stub output wire here; standalone callers get the
// bundled English fallback.
func (s *StubAnalyzer) SetTranslator(tr i18n.Translator) {
	s.translator = tr
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
	// CONST-046: user-facing Title + Description routed through Translator.
	// NoopTranslator path yields the bundled English fallback.
	return ScreenAnalysis{
		ScreenID:    id,
		Title:       resolveOrFallback(ctx, s.translator, "visionengine_stub_screen_title", fallbackStubScreenTitle),
		Description: resolveOrFallback(ctx, s.translator, "visionengine_stub_screen_description", fallbackStubScreenDescription),
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
	// CONST-046: user-facing Name routed through Translator. NoopTranslator
	// path yields the bundled English "Unknown" fallback.
	return ScreenIdentity{
		ID:          fmt.Sprintf("screen-%s", fp[:8]),
		Name:        resolveOrFallback(ctx, s.translator, "visionengine_stub_screen_unknown_category", fallbackStubScreenUnknownCategory),
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
