// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"strings"
	"testing"
)

// fakeTranslator is a unit-test-only stub. Per CONST-050(A), mocks are
// permitted inside *_test.go (this file). It records every msgID it
// sees and returns a sentinel-wrapped form so call sites can be proven
// to route through the Translator seam — a hardcoded literal regression
// would NOT contain the "<TRANSLATED:" sentinel and the assertion would
// fail.
type fakeTranslator struct {
	seen []string
}

func (f *fakeTranslator) T(_ context.Context, msgID string, _ ...any) string {
	f.seen = append(f.seen, msgID)
	return "<TRANSLATED:" + msgID + ">"
}

// withFakeTranslator wires the fakeTranslator at the package level for
// the duration of t and restores the NoopTranslator default afterwards.
// Per CONST-046 round 218, every Config.Validate user-facing error path
// MUST route through pkgTranslator.
func withFakeTranslator(t *testing.T) *fakeTranslator {
	t.Helper()
	tr := &fakeTranslator{}
	SetPkgTranslator(tr)
	t.Cleanup(func() { SetPkgTranslator(nil) })
	return tr
}

// TestValidate_UnknownProvider_RoutesThroughTranslator asserts that the
// "unknown vision provider" error path resolves through the Translator
// seam — NOT a hardcoded literal. Anti-bluff: the sentinel wrapper is
// the proof; a regression that inlines the literal will not produce
// "<TRANSLATED:" and the assertion fails.
func TestValidate_UnknownProvider_RoutesThroughTranslator(t *testing.T) {
	tr := withFakeTranslator(t)
	cfg := DefaultConfig()
	cfg.VisionProvider = "not-a-real-provider"

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("Validate() with unknown provider returned nil; expected ErrInvalidConfig")
	}
	want := "<TRANSLATED:visionengine_config_invalid_vision_provider>"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Validate error=%q; expected sentinel %q (call site bypassed Translator → CONST-046 violation)", err.Error(), want)
	}
	if len(tr.seen) == 0 || tr.seen[0] != "visionengine_config_invalid_vision_provider" {
		t.Fatalf("translator was not invoked with expected msgID; seen=%v", tr.seen)
	}
}

// TestValidate_InvalidSSIM_RoutesThroughTranslator asserts that the
// "SSIM threshold must be between 0 and 1" error path resolves through
// the Translator seam.
func TestValidate_InvalidSSIM_RoutesThroughTranslator(t *testing.T) {
	withFakeTranslator(t)
	cfg := DefaultConfig()
	cfg.SSIMThreshold = 1.5

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("Validate() with SSIM=1.5 returned nil; expected ErrInvalidConfig")
	}
	want := "<TRANSLATED:visionengine_config_invalid_ssim_threshold>"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Validate error=%q; expected sentinel %q (call site bypassed Translator → CONST-046 violation)", err.Error(), want)
	}
}

// TestValidate_InvalidMaxImageSize_RoutesThroughTranslator asserts the
// "max image size must be positive" path routes through Translator.
func TestValidate_InvalidMaxImageSize_RoutesThroughTranslator(t *testing.T) {
	withFakeTranslator(t)
	cfg := DefaultConfig()
	cfg.MaxImageSize = -1

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("Validate() with MaxImageSize=-1 returned nil; expected ErrInvalidConfig")
	}
	want := "<TRANSLATED:visionengine_config_invalid_max_image_size>"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Validate error=%q; expected sentinel %q (call site bypassed Translator → CONST-046 violation)", err.Error(), want)
	}
}

// TestValidate_InvalidTimeout_RoutesThroughTranslator asserts the
// "timeout must be positive" path routes through Translator.
func TestValidate_InvalidTimeout_RoutesThroughTranslator(t *testing.T) {
	withFakeTranslator(t)
	cfg := DefaultConfig()
	cfg.TimeoutSecs = -1

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("Validate() with TimeoutSecs=-1 returned nil; expected ErrInvalidConfig")
	}
	want := "<TRANSLATED:visionengine_config_invalid_timeout>"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Validate error=%q; expected sentinel %q (call site bypassed Translator → CONST-046 violation)", err.Error(), want)
	}
}

// TestValidate_MissingAPIKeys_RouteThroughTranslator drives each
// provider-key-required branch through the Translator seam in a table
// to keep the unit-test surface tight while still proving every call
// site individually.
func TestValidate_MissingAPIKeys_RouteThroughTranslator(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantSentinel string
	}{
		{"openai", "openai", "<TRANSLATED:visionengine_config_openai_key_required>"},
		{"anthropic", "anthropic", "<TRANSLATED:visionengine_config_anthropic_key_required>"},
		{"gemini", "gemini", "<TRANSLATED:visionengine_config_gemini_key_required>"},
		{"qwen", "qwen", "<TRANSLATED:visionengine_config_qwen_key_required>"},
		{"kimi", "kimi", "<TRANSLATED:visionengine_config_kimi_key_required>"},
		{"stepgui", "stepgui", "<TRANSLATED:visionengine_config_stepgui_key_required>"},
		{"astica", "astica", "<TRANSLATED:visionengine_config_astica_key_required>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withFakeTranslator(t)
			cfg := DefaultConfig()
			cfg.VisionProvider = tt.provider
			// All API key fields zero-initialized via DefaultConfig.

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() for provider=%s with empty key returned nil; expected ErrInvalidConfig", tt.provider)
			}
			if !strings.Contains(err.Error(), tt.wantSentinel) {
				t.Fatalf("Validate error=%q; expected sentinel %q (call site bypassed Translator → CONST-046 violation)", err.Error(), tt.wantSentinel)
			}
		})
	}
}

// TestValidate_NoTranslator_UsesEnglishFallback documents the
// standalone path: with the NoopTranslator default (no consumer-side
// translator wired), each error path falls back to the bundled English
// literal via resolveOrFallback. The assertion checks the formatted
// argument propagates correctly (verifying fmt.Sprintf is invoked).
func TestValidate_NoTranslator_UsesEnglishFallback(t *testing.T) {
	SetPkgTranslator(nil) // explicit noop reset.

	t.Run("unknown_provider_formatted_arg", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.VisionProvider = "bogus"
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("Validate() returned nil; expected ErrInvalidConfig")
		}
		// resolveOrFallback uses fmt.Sprintf(fallback, args...) → the
		// %q format verb wraps "bogus" in quotes. Anti-bluff: if the
		// fallback wiring is wrong, the %q substitution wouldn't fire.
		if !strings.Contains(err.Error(), `"bogus"`) {
			t.Fatalf("standalone fallback err=%q; expected formatted arg `\"bogus\"`", err.Error())
		}
	})

	t.Run("missing_openai_key_literal", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.VisionProvider = "openai"
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("Validate() returned nil; expected ErrInvalidConfig")
		}
		if !strings.Contains(err.Error(), "OPENAI_API_KEY required for openai provider") {
			t.Fatalf("standalone fallback err=%q; expected English literal", err.Error())
		}
	})

	// round-414 §11.4 CONST-046 Phase 4: the astica branch — previously
	// the lone hardcoded literal in Validate — now routes through the
	// resolveOrFallback seam. Standalone path falls back to the bundled
	// English literal.
	t.Run("missing_astica_key_literal", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.VisionProvider = "astica"
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("Validate() returned nil; expected ErrInvalidConfig")
		}
		if !strings.Contains(err.Error(), "ASTICA_API_KEY required for astica provider") {
			t.Fatalf("standalone fallback err=%q; expected English literal", err.Error())
		}
	})
}

// TestValidate_AsticaKey_RoutesThroughTranslator is the round-414
// paired-mutation guard for the astica migration. Anti-bluff: if a
// regression re-inlines the literal `ASTICA_API_KEY required for astica
// provider`, the "<TRANSLATED:" sentinel will be absent and this test
// FAILs — proving the call site genuinely routes through pkgTranslator.
func TestValidate_AsticaKey_RoutesThroughTranslator(t *testing.T) {
	tr := withFakeTranslator(t)
	cfg := DefaultConfig()
	cfg.VisionProvider = "astica"

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("Validate() for astica with empty key returned nil; expected ErrInvalidConfig")
	}
	want := "<TRANSLATED:visionengine_config_astica_key_required>"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Validate error=%q; expected sentinel %q (astica call site bypassed Translator → CONST-046 violation)", err.Error(), want)
	}
	found := false
	for _, id := range tr.seen {
		if id == "visionengine_config_astica_key_required" {
			found = true
		}
	}
	if !found {
		t.Fatalf("translator not invoked with astica msgID; seen=%v", tr.seen)
	}
}

// TestSetPkgTranslator_NilResetsToDefault asserts the SetPkgTranslator
// nil-reset contract — used by t.Cleanup to restore the noop default.
// Paired with the §1.1 meta-test invariant: if SetPkgTranslator silently
// dropped the nil case, subsequent tests would see leaked sentinel
// translator state and produce confused failures.
func TestSetPkgTranslator_NilResetsToDefault(t *testing.T) {
	tr := &fakeTranslator{}
	SetPkgTranslator(tr)
	if PkgTranslator() != tr {
		t.Fatalf("SetPkgTranslator(tr) did not store tr")
	}
	SetPkgTranslator(nil)
	if PkgTranslator() == tr {
		t.Fatalf("SetPkgTranslator(nil) did not reset; still references fake")
	}
	// Sanity: post-reset translator behaves as noop (returns msgID).
	got := PkgTranslator().T(context.Background(), "probe-msg")
	if got != "probe-msg" {
		t.Fatalf("post-reset translator T(probe-msg)=%q; expected msgID verbatim", got)
	}
}
