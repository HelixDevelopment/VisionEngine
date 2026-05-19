// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

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
	fallbackConfigInvalidVisionProvider = "unknown vision provider %q"
	fallbackConfigInvalidSSIMThreshold  = "SSIM threshold must be between 0 and 1, got %f"
	fallbackConfigInvalidMaxImageSize   = "max image size must be positive, got %d"
	fallbackConfigInvalidTimeout        = "timeout must be positive, got %d"
	fallbackConfigOpenAIKeyRequired     = "OPENAI_API_KEY required for openai provider"
	fallbackConfigAnthropicKeyRequired  = "ANTHROPIC_API_KEY required for anthropic provider"
	fallbackConfigGeminiKeyRequired     = "GOOGLE_API_KEY required for gemini provider"
	fallbackConfigQwenKeyRequired       = "QWEN_API_KEY required for qwen provider"
	fallbackConfigKimiKeyRequired       = "KIMI_API_KEY or MOONSHOT_API_KEY required for kimi provider"
	fallbackConfigStepGUIKeyRequired    = "STEPFUN_API_KEY required for stepgui provider"
)

// resolveOrFallback routes a user-facing string through tr.T. When the
// translator is the noop (msgID-verbatim path), the call site receives
// the msgID back and we substitute the bundled English fallback. When
// a real translator is wired, its result is used directly.
//
// This is the single seam every CONST-046 migration in the config
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

// pkgTranslator is the package-level Translator used by Config.Validate
// — a value-receiver method on a plain struct without a translator
// field. Tests override via SetPkgTranslator; production uses the
// NoopTranslator default. Consuming projects wire a real Translator at
// init time. nil reset reverts to noop default. This is the minimal
// seam that lets value-receiver methods route through i18n without
// changing the public Config struct shape (CONST-051(B) decoupling).
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
