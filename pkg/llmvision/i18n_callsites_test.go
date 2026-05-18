// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmvision

import (
	"context"
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
