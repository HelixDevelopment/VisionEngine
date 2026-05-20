// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// ErrStubAnalyzerNotImplemented is returned by every StubAnalyzer
// analysis method. Round-27 §11.4 audit (2026-05-17): the previous
// stub returned hardcoded "Unknown Screen" titles, empty UIElement
// / TextRegion / VisualIssue slices, and byte-equality-based
// "comparison" — all dressed up as successful screen analysis. That
// pattern silently passed any test that only checked
// require.NoError + assertion-against-the-fabricated-value, hiding
// the fact that no real vision analysis had occurred.
//
// Mirrors the opencv package's ErrOpenCVNotAvailable pattern (which
// IS honest — every stub OpenCV method returns the sentinel
// instead of returning a fabricated zero-value).
//
// To unblock real analysis: build with `-tags vision` for the
// OpenCV-backed implementation, OR wire an LLMVisionProvider via
// NewStubAnalyzerWithProvider and use the provider directly (the
// provider, not StubAnalyzer, performs the real work).
//
// Constitutional anchors: CONST-035 (anti-bluff), CONST-050(A)
// (no-fakes-beyond-unit-tests), Article XI §11.9 (forensic anchor).
var ErrStubAnalyzerNotImplemented = errors.New("visionengine: StubAnalyzer is a placeholder — install OpenCV (build tag: vision) or wire an LLM vision provider before invoking analyzer methods (the previous stub returned hardcoded 'Unknown Screen' / empty elements; §11.4 PASS-bluff removed)")

// StubAnalyzer is a constructor-only placeholder. Every analysis
// method returns ErrStubAnalyzerNotImplemented; consumers who need
// real screen analysis must build with `-tags vision` (OpenCV) or
// use an LLMVisionProvider directly.
//
// HISTORICAL BLUFF (resolved round-27, 2026-05-17): earlier
// revisions of this struct's methods returned ScreenAnalysis{Title:
// "Unknown Screen", Elements: []UIElement{}, ...} with err = nil,
// presenting the no-op result as if real analysis had succeeded.
// That pattern is removed; every method now returns the sentinel.
type StubAnalyzer struct {
	// Provider is an optional LLM vision provider for intelligent
	// analysis. When set, callers SHOULD invoke the provider
	// directly rather than going through StubAnalyzer's methods
	// (which always return ErrStubAnalyzerNotImplemented per the
	// round-27 §11.4 audit fix).
	Provider LLMVisionProvider
}

// LLMVisionProvider is a simplified interface for LLM vision
// integration. The full interface is in pkg/llmvision.
type LLMVisionProvider interface {
	AnalyzeImage(ctx context.Context, image []byte, prompt string) (string, error)
	CompareImages(ctx context.Context, img1, img2 []byte, prompt string) (string, error)
}

// NewStubAnalyzer creates a new StubAnalyzer. All analysis methods
// will return ErrStubAnalyzerNotImplemented.
func NewStubAnalyzer() *StubAnalyzer {
	return &StubAnalyzer{}
}

// NewStubAnalyzerWithProvider creates a new StubAnalyzer with an
// LLM vision provider attached. Note: StubAnalyzer's methods still
// return ErrStubAnalyzerNotImplemented even when Provider is set —
// the provider is exposed on the struct so callers can invoke it
// directly. (The previous "stub-passes-through-to-provider"
// behaviour did not exist; this constructor is preserved for API
// stability, but the field is the consumer-facing API.)
func NewStubAnalyzerWithProvider(provider LLMVisionProvider) *StubAnalyzer {
	return &StubAnalyzer{Provider: provider}
}

// AnalyzeScreen returns ErrStubAnalyzerNotImplemented.
//
// HISTORICAL BLUFF (resolved round-27, 2026-05-17): previous
// revisions returned ScreenAnalysis{Title: "Unknown Screen",
// Elements: []UIElement{}, ...} with err = nil. See package-level
// ErrStubAnalyzerNotImplemented documentation.
func (s *StubAnalyzer) AnalyzeScreen(ctx context.Context, screenshot []byte) (ScreenAnalysis, error) {
	if len(screenshot) == 0 {
		return ScreenAnalysis{}, errEmptyScreenshot()
	}
	select {
	case <-ctx.Done():
		return ScreenAnalysis{}, ctx.Err()
	default:
	}
	return ScreenAnalysis{}, ErrStubAnalyzerNotImplemented
}

// CompareScreens returns ErrStubAnalyzerNotImplemented.
//
// HISTORICAL BLUFF (resolved round-27, 2026-05-17): previous
// revisions returned ScreenDiff{Similarity: 1.0 if bytes equal else
// 0.0, ChangedRegions/NewElements/GoneElements all empty} — byte-
// equality is not screen comparison.
func (s *StubAnalyzer) CompareScreens(ctx context.Context, before, after []byte) (ScreenDiff, error) {
	if len(before) == 0 || len(after) == 0 {
		return ScreenDiff{}, errEmptyScreenshot()
	}
	select {
	case <-ctx.Done():
		return ScreenDiff{}, ctx.Err()
	default:
	}
	return ScreenDiff{}, ErrStubAnalyzerNotImplemented
}

// DetectElements returns ErrStubAnalyzerNotImplemented.
//
// HISTORICAL BLUFF (resolved round-27, 2026-05-17): previous
// revisions returned `[]UIElement{}, nil` — an empty slice with no
// error pretending detection had succeeded with zero results.
func (s *StubAnalyzer) DetectElements(screenshot []byte) ([]UIElement, error) {
	if len(screenshot) == 0 {
		return nil, errEmptyScreenshot()
	}
	return nil, ErrStubAnalyzerNotImplemented
}

// DetectText returns ErrStubAnalyzerNotImplemented.
//
// HISTORICAL BLUFF (resolved round-27, 2026-05-17): previous
// revisions returned `[]TextRegion{}, nil`. See ErrStub doc.
func (s *StubAnalyzer) DetectText(screenshot []byte) ([]TextRegion, error) {
	if len(screenshot) == 0 {
		return nil, errEmptyScreenshot()
	}
	return nil, ErrStubAnalyzerNotImplemented
}

// IdentifyScreen returns ErrStubAnalyzerNotImplemented.
//
// HISTORICAL BLUFF (resolved round-27, 2026-05-17): previous
// revisions returned ScreenIdentity{Name: "Unknown", Category:
// "unknown", Fingerprint: sha256(screenshot)} with err = nil —
// SHA-256 of bytes is a deterministic hash, not screen
// identification.
func (s *StubAnalyzer) IdentifyScreen(ctx context.Context, screenshot []byte) (ScreenIdentity, error) {
	if len(screenshot) == 0 {
		return ScreenIdentity{}, errEmptyScreenshot()
	}
	select {
	case <-ctx.Done():
		return ScreenIdentity{}, ctx.Err()
	default:
	}
	return ScreenIdentity{}, ErrStubAnalyzerNotImplemented
}

// DetectIssues returns ErrStubAnalyzerNotImplemented.
//
// HISTORICAL BLUFF (resolved round-27, 2026-05-17): previous
// revisions returned `[]VisualIssue{}, nil`. See ErrStub doc.
func (s *StubAnalyzer) DetectIssues(ctx context.Context, screenshot []byte) ([]VisualIssue, error) {
	if len(screenshot) == 0 {
		return nil, errEmptyScreenshot()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return nil, ErrStubAnalyzerNotImplemented
}

// fingerprint computes a SHA-256 fingerprint of the data. Retained
// as a package-internal helper; no longer exposed by the StubAnalyzer
// analysis surface (IdentifyScreen used to leak this hash as a
// screen-identity bluff per round-27 audit).
func fingerprint(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
