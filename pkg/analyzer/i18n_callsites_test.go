// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeTranslator is a unit-test-only stub. Per CONST-050(A), mocks are
// permitted inside *_test.go (this file). It returns a sentinel-wrapped
// form so call sites can be proven to route through the Translator seam.
type fakeTranslator struct {
	seen []string
}

func (f *fakeTranslator) T(_ context.Context, msgID string, _ ...any) string {
	f.seen = append(f.seen, msgID)
	return "<TRANSLATED:" + msgID + ">"
}

// withFakeTranslator wires the fakeTranslator at the package level for
// the duration of t and restores the NoopTranslator default afterwards.
func withFakeTranslator(t *testing.T) *fakeTranslator {
	t.Helper()
	tr := &fakeTranslator{}
	SetPkgTranslator(tr)
	t.Cleanup(func() { SetPkgTranslator(nil) })
	return tr
}

// TestStub_EmptyScreenshot_RoutesThroughTranslator drives every
// StubAnalyzer empty-screenshot error path through the Translator seam.
// round-414 §11.4 CONST-046 Phase 4: ErrEmptyScreenshot was previously
// surfaced as a raw sentinel literal across six call sites; they now
// route via errEmptyScreenshot. Anti-bluff: the "<TRANSLATED:" sentinel
// is the proof — a regression returning the bare sentinel will not
// produce it and the assertion fails.
func TestStub_EmptyScreenshot_RoutesThroughTranslator(t *testing.T) {
	withFakeTranslator(t)
	s := NewStubAnalyzer()
	ctx := context.Background()
	want := "<TRANSLATED:visionengine_analyzer_empty_screenshot>"

	tests := []struct {
		name string
		run  func() error
	}{
		{"analyze_screen", func() error { _, e := s.AnalyzeScreen(ctx, nil); return e }},
		{"compare_screens", func() error { _, e := s.CompareScreens(ctx, nil, nil); return e }},
		{"detect_elements", func() error { _, e := s.DetectElements(nil); return e }},
		{"detect_text", func() error { _, e := s.DetectText(nil); return e }},
		{"identify_screen", func() error { _, e := s.IdentifyScreen(ctx, nil); return e }},
		{"detect_issues", func() error { _, e := s.DetectIssues(ctx, nil); return e }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil {
				t.Fatalf("%s with nil screenshot returned nil; expected an error", tt.name)
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("%s err=%q; expected sentinel %q (call site bypassed Translator → CONST-046 violation)", tt.name, err.Error(), want)
			}
			if !errors.Is(err, ErrEmptyScreenshot) {
				t.Fatalf("%s error does not unwrap to ErrEmptyScreenshot — errors.Is broken", tt.name)
			}
		})
	}
}

// TestLocalizedSentinel_RoutesAllPublicSentinels proves the exported
// LocalizedSentinel helper routes every public analyzer sentinel error
// through the Translator seam while preserving errors.Is matchability.
func TestLocalizedSentinel_RoutesAllPublicSentinels(t *testing.T) {
	withFakeTranslator(t)

	tests := []struct {
		sentinel error
		want     string
	}{
		{ErrEmptyScreenshot, "<TRANSLATED:visionengine_analyzer_empty_screenshot>"},
		{ErrAnalysisFailed, "<TRANSLATED:visionengine_analyzer_analysis_failed>"},
		{ErrComparisonFailed, "<TRANSLATED:visionengine_analyzer_comparison_failed>"},
		{ErrDetectionFailed, "<TRANSLATED:visionengine_analyzer_detection_failed>"},
		{ErrIdentificationFailed, "<TRANSLATED:visionengine_analyzer_identification_failed>"},
	}
	for _, tt := range tests {
		got := LocalizedSentinel(tt.sentinel)
		if !strings.Contains(got.Error(), tt.want) {
			t.Fatalf("LocalizedSentinel(%v)=%q; expected %q", tt.sentinel, got.Error(), tt.want)
		}
		if !errors.Is(got, tt.sentinel) {
			t.Fatalf("LocalizedSentinel(%v) does not unwrap to its sentinel", tt.sentinel)
		}
	}

	// Unknown sentinel passes through unchanged.
	unknown := errors.New("some other error")
	if LocalizedSentinel(unknown) != unknown {
		t.Fatalf("LocalizedSentinel(unknown) did not pass through unchanged")
	}
}

// TestStub_EmptyScreenshot_NoTranslator_EnglishFallback documents the
// standalone path: with the NoopTranslator default, the error path
// falls back to the bundled English literal.
func TestStub_EmptyScreenshot_NoTranslator_EnglishFallback(t *testing.T) {
	SetPkgTranslator(nil)

	_, err := NewStubAnalyzer().AnalyzeScreen(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "empty screenshot data") {
		t.Fatalf("standalone fallback err=%v; expected English literal", err)
	}
}

// TestSetPkgTranslator_Analyzer_NilResetsToDefault is the §1.1 paired
// meta-test guard for the analyzer translator seam.
func TestSetPkgTranslator_Analyzer_NilResetsToDefault(t *testing.T) {
	tr := &fakeTranslator{}
	SetPkgTranslator(tr)
	if PkgTranslator() != tr {
		t.Fatalf("SetPkgTranslator(tr) did not store tr")
	}
	SetPkgTranslator(nil)
	if PkgTranslator() == tr {
		t.Fatalf("SetPkgTranslator(nil) did not reset; still references fake")
	}
	if got := PkgTranslator().T(context.Background(), "probe"); got != "probe" {
		t.Fatalf("post-reset translator T(probe)=%q; expected msgID verbatim", got)
	}
}
