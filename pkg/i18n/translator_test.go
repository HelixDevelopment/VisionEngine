// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package i18n

import (
	"context"
	"testing"
)

// TestNoopTranslator_ReturnsMsgIDVerbatim asserts the standalone-default
// translator behavior: on T(ctx, "visionengine_foo"), it MUST return
// "visionengine_foo" exactly. Real production translators wired by the
// consuming project will resolve the ID to localized text; the noop
// guarantees the call site never crashes when no translator is wired.
// This is the contract that lets the submodule stay decoupled per
// CONST-051(B) while still routing user-facing strings through the
// Translator seam per CONST-046.
func TestNoopTranslator_ReturnsMsgIDVerbatim(t *testing.T) {
	tr := Default()
	got := tr.T(context.Background(), "visionengine_stub_screen_title")
	if got != "visionengine_stub_screen_title" {
		t.Fatalf("NoopTranslator.T returned %q; expected msgID verbatim", got)
	}
}

// TestNoopTranslator_IgnoresArgs confirms positional substitution args
// are accepted but unused by the noop. Real translators substitute;
// the noop is a pure pass-through so unit tests can assert "translator
// was called with this msgID" without coupling to a bundle.
func TestNoopTranslator_IgnoresArgs(t *testing.T) {
	tr := NoopTranslator{}
	got := tr.T(context.Background(), "visionengine_config_invalid_provider", "openai", 42)
	if got != "visionengine_config_invalid_provider" {
		t.Fatalf("NoopTranslator.T with args returned %q; expected msgID verbatim", got)
	}
}
