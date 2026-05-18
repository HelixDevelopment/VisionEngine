// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"context"
	"fmt"

	"digital.vasic.visionengine/pkg/i18n"
)

// English fallbacks used when no real i18n.Translator is wired
// (NoopTranslator path). Per CONST-046 these are NOT the canonical
// strings — the canonical source is pkg/i18n/bundles/active.en.yaml.
// They are duplicated here so that the standalone NoopTranslator path
// produces a sensible default without forcing the consuming project to
// ship a translator.
const (
	fallbackStubScreenTitle           = "Unknown Screen"
	fallbackStubScreenDescription     = "Stub analysis - install OpenCV or use LLM vision for detailed analysis"
	fallbackStubScreenUnknownCategory = "Unknown"
)

// resolveOrFallback routes a user-facing string through tr.T. When the
// translator is the noop (msgID-verbatim path), the call site receives
// the msgID back and we substitute the bundled English fallback. When
// a real translator is wired, its result is used directly.
//
// This is the single seam every CONST-046 migration in the analyzer
// package passes through.
func resolveOrFallback(ctx context.Context, tr i18n.Translator, msgID, fallback string, args ...any) string {
	if tr == nil {
		tr = i18n.Default()
	}
	got := tr.T(ctx, msgID, args...)
	if got == msgID {
		if len(args) == 0 {
			return fallback
		}
		return fmt.Sprintf(fallback, args...)
	}
	return got
}
