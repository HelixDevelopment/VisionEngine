// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package i18n provides a minimal, dependency-free Translator interface
// for the VisionEngine module. Per CONST-051(B), this submodule MUST
// NOT import any consuming-project package (e.g. helix_code) — it
// defines its own interface and ships a NoopTranslator default so the
// module remains standalone-testable and project-not-aware.
//
// Per CONST-046, every user-facing string in this module that
// previously lived as a hardcoded literal (configuration error
// messages, screen-analysis Title/Description fields, provider
// validation errors surfaced to end users) MUST resolve through this
// Translator at the call site. Consumers wire a real translator at
// construction time; the NoopTranslator returns the message ID so
// the call site falls back to bundled English (see translator helpers
// in the pkg consuming the migration).
package i18n

import "context"

// Translator is the minimal i18n contract this submodule depends on.
// A consuming project supplies a concrete implementation that loads
// per-locale bundles (YAML, JSON, etc.). When no translator is wired,
// NoopTranslator below resolves to the English fallback supplied by
// the call site.
type Translator interface {
	// T resolves a message ID to its locale-appropriate text.
	// args are positional substitution values for templated entries.
	// Implementations MUST never return an empty string; on miss they
	// MUST return the fallback (typically the message ID itself) so
	// callers can detect misses without panicking.
	T(ctx context.Context, msgID string, args ...any) string
}

// NoopTranslator is the standalone-default Translator. It returns the
// message ID verbatim, so call-site fallbacks (passed alongside) drive
// the user-visible output. This keeps the submodule fully decoupled
// per CONST-051(B) while still routing every user-facing string through
// the Translator seam per CONST-046.
type NoopTranslator struct{}

// T returns the msgID itself. The caller is expected to compose the
// final user-visible string from the fallback bundled alongside the
// migration (see bundles/active.en.yaml). When a real Translator is
// wired by the consuming project, T resolves the msgID against the
// locale bundle and substitutes args.
func (NoopTranslator) T(_ context.Context, msgID string, _ ...any) string {
	return msgID
}

// Default returns the standalone-default Translator. Use this when no
// consuming-project translator is wired (tests, demos, standalone runs).
func Default() Translator { return NoopTranslator{} }
