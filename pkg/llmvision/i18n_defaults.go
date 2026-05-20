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
	fallbackProviderFallbackRequiresOne     = "at least one provider is required"
	fallbackProviderFallbackAllFailed       = "all providers failed, last error: %v"
	fallbackProviderFallbackAllFailedPrefix = "all providers failed, last error"
	fallbackProviderNoAPIKey                = "API key not configured"
	fallbackProviderEmptyImage              = "empty image data"
	fallbackProviderEmptyPrompt             = "empty prompt"
	fallbackProviderImageTooLarge           = "image exceeds maximum size"
	fallbackProviderUnavailable             = "vision provider unavailable"
	fallbackProviderRateLimited             = "API rate limited"
	fallbackProviderInvalidResponse         = "invalid API response"
)

// LocalizedError resolves the user-facing message for a VisionProvider
// sentinel error through the package-level translator. The returned
// error wraps the sentinel so errors.Is(returned, sentinel) still
// holds — callers keep their existing match logic while end users see
// a locale-appropriate message. Per CONST-046 the sentinel's own text
// is only the bundled English fallback (NoopTranslator path); a wired
// translator supplies the localized string.
func LocalizedError(ctx context.Context, sentinel error) error {
	msgID, fallback := sentinelMessage(sentinel)
	if msgID == "" {
		return sentinel
	}
	return &localizedSentinelError{
		sentinel: sentinel,
		message:  resolvePlain(ctx, pkgTranslator, msgID, fallback),
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

// sentinelMessage maps each VisionProvider sentinel to its bundle
// message ID + English fallback. Unknown sentinels return empty IDs so
// LocalizedError passes them through unchanged.
func sentinelMessage(sentinel error) (msgID, fallback string) {
	switch sentinel {
	case ErrNoAPIKey:
		return "visionengine_provider_no_api_key", fallbackProviderNoAPIKey
	case ErrEmptyImage:
		return "visionengine_provider_empty_image", fallbackProviderEmptyImage
	case ErrEmptyPrompt:
		return "visionengine_provider_empty_prompt", fallbackProviderEmptyPrompt
	case ErrImageTooLarge:
		return "visionengine_provider_image_too_large", fallbackProviderImageTooLarge
	case ErrProviderUnavailable:
		return "visionengine_provider_unavailable", fallbackProviderUnavailable
	case ErrRateLimited:
		return "visionengine_provider_rate_limited", fallbackProviderRateLimited
	case ErrInvalidResponse:
		return "visionengine_provider_invalid_response", fallbackProviderInvalidResponse
	default:
		return "", ""
	}
}

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
		//nolint:govet // fallback is an i18n bundle template, not a literal.
		return fmt.Sprintf(fallback, args...)
	}
	return got
}

// resolvePlain is the no-arguments resolver. It is identical to
// resolveOrFallback with zero args but takes no variadic, so `go vet`'s
// printf analysis does not treat the fallback as a format string. Used
// by call sites (e.g. LocalizedError) that never substitute arguments.
func resolvePlain(ctx context.Context, tr i18n.Translator, msgID, fallback string) string {
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
