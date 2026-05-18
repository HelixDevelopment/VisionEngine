// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"context"
	"testing"
)

// fakeTranslator is a unit-test-only stub. Per CONST-050(A), mocks are
// permitted inside *_test.go (this file). It returns a sentinel
// "<TRANSLATED:msg_id>" so call-site tests assert against that exact
// marker — NEVER the original English literal. Regression that inlines
// the literal fails the assertion (CONST-046 anti-bluff guarantee).
type fakeTranslator struct {
	seen []string
}

func (f *fakeTranslator) T(_ context.Context, msgID string, _ ...any) string {
	f.seen = append(f.seen, msgID)
	return "<TRANSLATED:" + msgID + ">"
}

// TestStubAnalyzer_AnalyzeScreen_RoutesTitleThroughTranslator asserts the
// Title field in the returned ScreenAnalysis is the Translator sentinel
// when a Translator is wired — NOT the hardcoded "Unknown Screen"
// literal. If a regression reintroduces the literal, this test fails.
func TestStubAnalyzer_AnalyzeScreen_RoutesTitleThroughTranslator(t *testing.T) {
	tr := &fakeTranslator{}
	s := NewStubAnalyzer()
	s.SetTranslator(tr)

	got, err := s.AnalyzeScreen(context.Background(), []byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Fatalf("AnalyzeScreen returned error: %v", err)
	}
	wantTitle := "<TRANSLATED:visionengine_stub_screen_title>"
	if got.Title != wantTitle {
		t.Fatalf("ScreenAnalysis.Title=%q; expected sentinel %q (call site bypassed Translator → CONST-046 violation)", got.Title, wantTitle)
	}
	wantDesc := "<TRANSLATED:visionengine_stub_screen_description>"
	if got.Description != wantDesc {
		t.Fatalf("ScreenAnalysis.Description=%q; expected sentinel %q", got.Description, wantDesc)
	}
}

// TestStubAnalyzer_IdentifyScreen_RoutesNameThroughTranslator is the
// sibling assertion for ScreenIdentity.Name.
func TestStubAnalyzer_IdentifyScreen_RoutesNameThroughTranslator(t *testing.T) {
	tr := &fakeTranslator{}
	s := NewStubAnalyzer()
	s.SetTranslator(tr)

	got, err := s.IdentifyScreen(context.Background(), []byte{0x10, 0x20})
	if err != nil {
		t.Fatalf("IdentifyScreen returned error: %v", err)
	}
	want := "<TRANSLATED:visionengine_stub_screen_unknown_category>"
	if got.Name != want {
		t.Fatalf("ScreenIdentity.Name=%q; expected sentinel %q (call site bypassed Translator → CONST-046 violation)", got.Name, want)
	}
}

// TestStubAnalyzer_NoTranslator_UsesEnglishFallback documents the
// standalone path: when no real translator is wired, the bundled
// English fallback drives Title/Description. This keeps the module
// standalone-buildable per CONST-051(B).
func TestStubAnalyzer_NoTranslator_UsesEnglishFallback(t *testing.T) {
	s := NewStubAnalyzer() // no SetTranslator call — translator field is nil.
	got, err := s.AnalyzeScreen(context.Background(), []byte{0xAA})
	if err != nil {
		t.Fatalf("AnalyzeScreen returned error: %v", err)
	}
	if got.Title != "Unknown Screen" {
		t.Fatalf("standalone fallback Title=%q; expected %q", got.Title, "Unknown Screen")
	}
	if got.Description == "" {
		t.Fatalf("standalone fallback Description is empty; expected English text from bundle")
	}
}
