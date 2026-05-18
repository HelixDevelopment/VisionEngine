// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmvision

import (
	"context"
	"fmt"

	"digital.vasic.visionengine/pkg/i18n"
)

// English fallbacks used when no real i18n.Translator is wired
// (NoopTranslator path). Per CONST-046 these are NOT the canonical
// strings — the canonical source is pkg/i18n/bundles/active.en.yaml.
const (
	fallbackProviderFallbackRequiresOne = "at least one provider is required"
	fallbackProviderFallbackAllFailed   = "all providers failed, last error: %v"
)

// resolveOrFallback routes a user-facing string through tr.T. When the
// translator is the noop (msgID-verbatim path), the call site receives
// the msgID back and we substitute the bundled English fallback. When
// a real translator is wired, its result is used directly.
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

// pkgTranslator is the package-level Translator used by call sites that
// don't accept a translator via parameter (e.g. NewFallbackProvider —
// returning an error before any struct exists). Tests override via
// SetPkgTranslator; production uses NoopTranslator default. This is
// the minimal seam that lets a free function route through i18n
// without an enclosing struct.
var pkgTranslator i18n.Translator = i18n.Default()

// SetPkgTranslator wires a package-level Translator. Used by consuming
// projects at init time; tests use it to inject the fakeTranslator.
// nil resets to NoopTranslator default.
func SetPkgTranslator(tr i18n.Translator) {
	if tr == nil {
		pkgTranslator = i18n.Default()
		return
	}
	pkgTranslator = tr
}

// PkgTranslator returns the current package-level Translator.
func PkgTranslator() i18n.Translator { return pkgTranslator }
