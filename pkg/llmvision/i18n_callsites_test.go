// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmvision

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeTranslator is a unit-test-only stub. Per CONST-050(A), mocks are
// permitted inside *_test.go (this file).
type fakeTranslator struct {
	seen []string
}

func (f *fakeTranslator) T(_ context.Context, msgID string, _ ...any) string {
	f.seen = append(f.seen, msgID)
	return "<TRANSLATED:" + msgID + ">"
}

// TestNewFallbackProvider_EmptyProviders_RoutesErrorThroughTranslator
// asserts that NewFallbackProvider's "at least one provider is required"
// error is routed through the package-level Translator — NOT a
// hardcoded literal. If a regression inlines the literal, this test
// fails (CONST-046 anti-bluff guarantee).
func TestNewFallbackProvider_EmptyProviders_RoutesErrorThroughTranslator(t *testing.T) {
	tr := &fakeTranslator{}
	SetPkgTranslator(tr)
	t.Cleanup(func() { SetPkgTranslator(nil) }) // reset to noop default.

	_, err := NewFallbackProvider() // zero providers — triggers the error path.
	if err == nil {
		t.Fatalf("NewFallbackProvider() with no providers returned nil error; expected requires-one error")
	}
	want := "<TRANSLATED:visionengine_provider_fallback_requires_one>"
	if err.Error() != want {
		t.Fatalf("NewFallbackProvider error=%q; expected sentinel %q (call site bypassed Translator → CONST-046 violation)", err.Error(), want)
	}
}

// TestNewFallbackProvider_NoTranslator_UsesEnglishFallback documents the
// standalone path: when the package Translator is the noop default, the
// bundled English fallback drives the error message.
func TestNewFallbackProvider_NoTranslator_UsesEnglishFallback(t *testing.T) {
	SetPkgTranslator(nil) // reset to noop default explicitly.
	_, err := NewFallbackProvider()
	if err == nil {
		t.Fatalf("NewFallbackProvider() with no providers returned nil error; expected requires-one error")
	}
	if err.Error() != "at least one provider is required" {
		t.Fatalf("standalone fallback error=%q; expected English %q", err.Error(), "at least one provider is required")
	}
}

// alwaysFailProvider is a unit-test-only VisionProvider whose
// AnalyzeImage / CompareImages always fail — used to drive the
// FallbackProvider aggregate-failure path. Per CONST-050(A) mocks are
// permitted in *_test.go.
type alwaysFailProvider struct{}

func (alwaysFailProvider) AnalyzeImage(context.Context, []byte, string) (string, error) {
	return "", errors.New("provider boom")
}
func (alwaysFailProvider) CompareImages(context.Context, []byte, []byte, string) (string, error) {
	return "", errors.New("provider boom")
}
func (alwaysFailProvider) SupportsVision() bool { return true }
func (alwaysFailProvider) MaxImageSize() int    { return 1 << 20 }
func (alwaysFailProvider) Name() string         { return "always-fail" }

// TestFallbackProvider_AllFailed_RoutesThroughTranslator is the
// round-414 §11.4 CONST-046 Phase 4 guard. The FallbackProvider
// aggregate-failure message ("all providers failed, last error") was
// previously a hardcoded literal at two call sites; it now routes
// through the translator. Anti-bluff: the "<TRANSLATED:" sentinel is
// the proof — a regression re-inlining the literal will not produce it.
func TestFallbackProvider_AllFailed_RoutesThroughTranslator(t *testing.T) {
	tr := &fakeTranslator{}
	SetPkgTranslator(tr)
	t.Cleanup(func() { SetPkgTranslator(nil) })

	fp, err := NewFallbackProvider(alwaysFailProvider{})
	if err != nil {
		t.Fatalf("NewFallbackProvider returned unexpected error: %v", err)
	}
	want := "<TRANSLATED:visionengine_provider_fallback_all_failed_prefix>"

	_, aErr := fp.AnalyzeImage(context.Background(), []byte{1}, "prompt")
	if aErr == nil || !strings.Contains(aErr.Error(), want) {
		t.Fatalf("AnalyzeImage all-failed err=%v; expected sentinel %q (CONST-046 violation)", aErr, want)
	}
	_, cErr := fp.CompareImages(context.Background(), []byte{1}, []byte{1}, "prompt")
	if cErr == nil || !strings.Contains(cErr.Error(), want) {
		t.Fatalf("CompareImages all-failed err=%v; expected sentinel %q (CONST-046 violation)", cErr, want)
	}
}

// TestLocalizedError_RoutesProviderSentinels is the round-414 guard for
// the seven VisionProvider sentinel errors. Each must route through the
// Translator seam via LocalizedError while staying errors.Is-compatible
// with its underlying sentinel.
func TestLocalizedError_RoutesProviderSentinels(t *testing.T) {
	tr := &fakeTranslator{}
	SetPkgTranslator(tr)
	t.Cleanup(func() { SetPkgTranslator(nil) })

	tests := []struct {
		sentinel error
		want     string
	}{
		{ErrNoAPIKey, "<TRANSLATED:visionengine_provider_no_api_key>"},
		{ErrEmptyImage, "<TRANSLATED:visionengine_provider_empty_image>"},
		{ErrEmptyPrompt, "<TRANSLATED:visionengine_provider_empty_prompt>"},
		{ErrImageTooLarge, "<TRANSLATED:visionengine_provider_image_too_large>"},
		{ErrProviderUnavailable, "<TRANSLATED:visionengine_provider_unavailable>"},
		{ErrRateLimited, "<TRANSLATED:visionengine_provider_rate_limited>"},
		{ErrInvalidResponse, "<TRANSLATED:visionengine_provider_invalid_response>"},
	}
	for _, tt := range tests {
		got := LocalizedError(context.Background(), tt.sentinel)
		if !strings.Contains(got.Error(), tt.want) {
			t.Fatalf("LocalizedError(%v)=%q; expected %q", tt.sentinel, got.Error(), tt.want)
		}
		if !errors.Is(got, tt.sentinel) {
			t.Fatalf("LocalizedError(%v) does not unwrap to its sentinel — errors.Is broken", tt.sentinel)
		}
	}

	// Unknown sentinel passes through unchanged.
	unknown := errors.New("unrelated")
	if LocalizedError(context.Background(), unknown) != unknown {
		t.Fatalf("LocalizedError(unknown) did not pass through unchanged")
	}
}

// TestValidateImagePrompt_RouteThroughTranslator proves the
// validateImage / validatePrompt helpers surface their errors through
// the Translator seam (round-414 CONST-046 Phase 4).
func TestValidateImagePrompt_RouteThroughTranslator(t *testing.T) {
	tr := &fakeTranslator{}
	SetPkgTranslator(tr)
	t.Cleanup(func() { SetPkgTranslator(nil) })

	if err := validateImage(nil, 1024); err == nil ||
		!strings.Contains(err.Error(), "<TRANSLATED:visionengine_provider_empty_image>") {
		t.Fatalf("validateImage(empty) err=%v; expected translated empty-image sentinel", err)
	}
	if err := validateImage([]byte{1, 2, 3}, 1); err == nil ||
		!strings.Contains(err.Error(), "<TRANSLATED:visionengine_provider_image_too_large>") {
		t.Fatalf("validateImage(too-large) err=%v; expected translated too-large sentinel", err)
	}
	if err := validatePrompt(""); err == nil ||
		!strings.Contains(err.Error(), "<TRANSLATED:visionengine_provider_empty_prompt>") {
		t.Fatalf("validatePrompt(empty) err=%v; expected translated empty-prompt sentinel", err)
	}
}

// TestLocalizedError_NoTranslator_EnglishFallback documents the
// standalone path for the provider sentinels.
func TestLocalizedError_NoTranslator_EnglishFallback(t *testing.T) {
	SetPkgTranslator(nil)
	got := LocalizedError(context.Background(), ErrNoAPIKey)
	if !strings.Contains(got.Error(), "API key not configured") {
		t.Fatalf("standalone LocalizedError(ErrNoAPIKey)=%q; expected English literal", got.Error())
	}
}
