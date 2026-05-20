// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package opencv

import (
	"context"

	"digital.vasic.visionengine/pkg/i18n"
)

// English fallbacks used when no real i18n.Translator is wired
// (NoopTranslator path). Per CONST-046 these are NOT the canonical
// strings — the canonical source is pkg/i18n/bundles/active.en.yaml.
// They are duplicated here so the standalone NoopTranslator path
// produces a sensible default without forcing the consuming project
// to ship a translator.
const (
	fallbackOpenCVNotAvailable = "OpenCV not available: build with -tags vision"
	fallbackOpenCVInvalidImage = "invalid image data"
	fallbackOpenCVInvalidVideoPath = "invalid video path"
)

// resolveOrFallback routes a user-facing string through tr.T. When the
// translator is the noop (msgID-verbatim path), the call site receives
// the msgID back and we substitute the bundled English fallback. When
// a real translator is wired, its result is used directly.
//
// This is the single seam every CONST-046 migration in the opencv
// package passes through.
func resolveOrFallback(ctx context.Context, tr i18n.Translator, msgID, fallback string) string {
	if tr == nil {
		tr = i18n.Default()
	}
	got := tr.T(ctx, msgID)
	if got == msgID {
		return fallback
	}
	return got
}

// pkgTranslator is the package-level Translator used by call sites that
// surface user-facing text. Tests override via SetPkgTranslator;
// production uses the NoopTranslator default. Consuming projects wire a
// real Translator at init time; nil reset reverts to the noop default.
var pkgTranslator i18n.Translator = i18n.Default()

// SetPkgTranslator wires a package-level Translator. Used by consuming
// projects at init time; tests use it to inject a fake translator.
// nil resets to the NoopTranslator default.
func SetPkgTranslator(tr i18n.Translator) {
	if tr == nil {
		pkgTranslator = i18n.Default()
		return
	}
	pkgTranslator = tr
}

// PkgTranslator returns the current package-level Translator.
func PkgTranslator() i18n.Translator { return pkgTranslator }

// localizedError resolves the user-facing message for an opencv
// sentinel error through the package-level translator. The returned
// error wraps the sentinel so errors.Is(returned, sentinel) still
// holds — callers keep their existing match logic while end users see
// a locale-appropriate message.
func localizedError(sentinel error, msgID, fallback string) error {
	return &localizedSentinelError{
		sentinel: sentinel,
		message:  resolveOrFallback(context.Background(), pkgTranslator, msgID, fallback),
	}
}

// localizedSentinelError carries a localized user-facing message while
// remaining errors.Is-compatible with the underlying sentinel.
type localizedSentinelError struct {
	sentinel error
	message  string
}

func (e *localizedSentinelError) Error() string { return e.message }
func (e *localizedSentinelError) Unwrap() error  { return e.sentinel }
