// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration for VisionEngine.
package config

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

var (
	// ErrInvalidConfig is returned when configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")
)

// Config holds the VisionEngine configuration.
type Config struct {
	// VisionProvider selects the primary vision provider: "openai", "anthropic", "gemini", "qwen", "astica", "auto".
	VisionProvider string `json:"vision_provider"`
	// OpenCVEnabled indicates if OpenCV features are enabled.
	OpenCVEnabled bool `json:"opencv_enabled"`
	// SSIMThreshold is the SSIM threshold for screen comparison.
	SSIMThreshold float64 `json:"ssim_threshold"`
	// MaxImageSize is the maximum image size in bytes.
	MaxImageSize int `json:"max_image_size"`

	// API keys for vision providers.
	OpenAIAPIKey    string `json:"-"`
	AnthropicAPIKey string `json:"-"`
	GoogleAPIKey    string `json:"-"`
	QwenAPIKey      string `json:"-"`
	DeepSeekAPIKey  string `json:"-"`
	GroqAPIKey      string `json:"-"`
	KimiAPIKey      string `json:"-"`
	StepfunAPIKey   string `json:"-"`
	AsticaAPIKey    string `json:"-"`

	// Provider-specific models.
	OpenAIModel    string `json:"openai_model,omitempty"`
	AnthropicModel string `json:"anthropic_model,omitempty"`
	GeminiModel    string `json:"gemini_model,omitempty"`
	QwenModel      string `json:"qwen_model,omitempty"`
	KimiModel      string `json:"kimi_model,omitempty"`
	StepGUIModel   string `json:"stepgui_model,omitempty"`
	AsticaModel    string `json:"astica_model,omitempty"`

	// Timeouts in seconds.
	TimeoutSecs int `json:"timeout_secs"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		VisionProvider: "auto",
		OpenCVEnabled:  true,
		SSIMThreshold:  0.95,
		MaxImageSize:   4096 * 4096 * 4, // ~64 MB
		TimeoutSecs:    60,
	}
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("HELIX_VISION_PROVIDER"); v != "" {
		cfg.VisionProvider = v
	}
	if v := os.Getenv("HELIX_VISION_OPENCV_ENABLED"); v != "" {
		cfg.OpenCVEnabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("HELIX_VISION_SSIM_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.SSIMThreshold = f
		}
	}
	if v := os.Getenv("HELIX_VISION_MAX_IMAGE_SIZE"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.MaxImageSize = i
		}
	}

	cfg.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	if cfg.OpenAIAPIKey == "" {
		cfg.OpenAIAPIKey = os.Getenv("OPENROUTER_API_KEY")
	}
	cfg.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
	cfg.GoogleAPIKey = os.Getenv("GOOGLE_API_KEY")
	if cfg.GoogleAPIKey == "" {
		cfg.GoogleAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	cfg.QwenAPIKey = os.Getenv("QWEN_API_KEY")
	cfg.DeepSeekAPIKey = os.Getenv("DEEPSEEK_API_KEY")
	cfg.GroqAPIKey = os.Getenv("GROQ_API_KEY")
	cfg.KimiAPIKey = os.Getenv("KIMI_API_KEY")
	if cfg.KimiAPIKey == "" {
		cfg.KimiAPIKey = os.Getenv("MOONSHOT_API_KEY")
	}
	cfg.StepfunAPIKey = os.Getenv("STEPFUN_API_KEY")
	cfg.AsticaAPIKey = os.Getenv("ASTICA_API_KEY")

	if v := os.Getenv("HELIX_VISION_OPENAI_MODEL"); v != "" {
		cfg.OpenAIModel = v
	}
	if v := os.Getenv("HELIX_VISION_ANTHROPIC_MODEL"); v != "" {
		cfg.AnthropicModel = v
	}
	if v := os.Getenv("HELIX_VISION_GEMINI_MODEL"); v != "" {
		cfg.GeminiModel = v
	}
	if v := os.Getenv("HELIX_VISION_QWEN_MODEL"); v != "" {
		cfg.QwenModel = v
	}
	if v := os.Getenv("HELIX_VISION_KIMI_MODEL"); v != "" {
		cfg.KimiModel = v
	}
	if v := os.Getenv("HELIX_VISION_STEPGUI_MODEL"); v != "" {
		cfg.StepGUIModel = v
	}
	if v := os.Getenv("HELIX_VISION_ASTICA_MODEL"); v != "" {
		cfg.AsticaModel = v
	}
	if v := os.Getenv("HELIX_VISION_TIMEOUT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.TimeoutSecs = i
		}
	}

	return cfg
}

// Validate checks that the configuration is valid.
//
// CONST-046 migration (round 218): every user-facing validation error
// surfaced from this method is routed through the package-level
// translator (see i18n_defaults.go) so consuming projects can localize
// without forking the submodule. Standalone (NoopTranslator default)
// path falls back to the bundled English literal via resolveOrFallback.
// Every provider branch — including astica — routes through the seam
// (round-414 §11.4 CONST-046 Phase 4 closed the astica residual).
func (c Config) Validate() error {
	ctx := context.Background()
	validProviders := map[string]bool{
		"auto": true, "openai": true, "anthropic": true, "gemini": true, "qwen": true,
		"kimi": true, "stepgui": true, "astica": true,
	}
	if !validProviders[c.VisionProvider] {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
			ctx, pkgTranslator,
			"visionengine_config_invalid_vision_provider",
			fallbackConfigInvalidVisionProvider,
			c.VisionProvider,
		))
	}
	// math.IsNaN is required in addition to the range comparisons below:
	// in Go (IEEE 754), every ordered comparison against NaN — including
	// `NaN < 0` and `NaN > 1` — evaluates to false, so a bare range check
	// silently accepts NaN as "in range". A NaN threshold reaches this
	// point when `HELIX_VISION_SSIM_THRESHOLD=NaN` (or `nan`, `NaN`, all
	// accepted by strconv.ParseFloat) is set, or when a caller assigns
	// math.NaN() directly. Downstream comparisons such as
	// `similarity >= cfg.SSIMThreshold` then ALWAYS evaluate false,
	// silently breaking screen-comparison logic while Validate() reports
	// the configuration as valid — a §11.4 input-validation defect found
	// during the 2026-07-10 adversarial audit
	// (qa-results/audit_20260710/RED_config_nan_ssim.txt).
	if math.IsNaN(c.SSIMThreshold) || c.SSIMThreshold < 0 || c.SSIMThreshold > 1 {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
			ctx, pkgTranslator,
			"visionengine_config_invalid_ssim_threshold",
			fallbackConfigInvalidSSIMThreshold,
			c.SSIMThreshold,
		))
	}
	if c.MaxImageSize <= 0 {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
			ctx, pkgTranslator,
			"visionengine_config_invalid_max_image_size",
			fallbackConfigInvalidMaxImageSize,
			c.MaxImageSize,
		))
	}
	if c.TimeoutSecs <= 0 {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
			ctx, pkgTranslator,
			"visionengine_config_invalid_timeout",
			fallbackConfigInvalidTimeout,
			c.TimeoutSecs,
		))
	}

	// If a specific provider is selected, check API key
	switch c.VisionProvider {
	case "openai":
		if c.OpenAIAPIKey == "" {
			return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
				ctx, pkgTranslator,
				"visionengine_config_openai_key_required",
				fallbackConfigOpenAIKeyRequired,
			))
		}
	case "anthropic":
		if c.AnthropicAPIKey == "" {
			return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
				ctx, pkgTranslator,
				"visionengine_config_anthropic_key_required",
				fallbackConfigAnthropicKeyRequired,
			))
		}
	case "gemini":
		if c.GoogleAPIKey == "" {
			return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
				ctx, pkgTranslator,
				"visionengine_config_gemini_key_required",
				fallbackConfigGeminiKeyRequired,
			))
		}
	case "qwen":
		if c.QwenAPIKey == "" {
			return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
				ctx, pkgTranslator,
				"visionengine_config_qwen_key_required",
				fallbackConfigQwenKeyRequired,
			))
		}
	case "kimi":
		if c.KimiAPIKey == "" {
			return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
				ctx, pkgTranslator,
				"visionengine_config_kimi_key_required",
				fallbackConfigKimiKeyRequired,
			))
		}
	case "stepgui":
		if c.StepfunAPIKey == "" {
			return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
				ctx, pkgTranslator,
				"visionengine_config_stepgui_key_required",
				fallbackConfigStepGUIKeyRequired,
			))
		}
	case "astica":
		if c.AsticaAPIKey == "" {
			return fmt.Errorf("%w: %s", ErrInvalidConfig, resolveOrFallback(
				ctx, pkgTranslator,
				"visionengine_config_astica_key_required",
				fallbackConfigAsticaKeyRequired,
			))
		}
	}

	return nil
}

// AvailableProviders returns a list of provider names that have API keys configured.
func (c Config) AvailableProviders() []string {
	var providers []string
	if c.OpenAIAPIKey != "" {
		providers = append(providers, "openai")
	}
	if c.AnthropicAPIKey != "" {
		providers = append(providers, "anthropic")
	}
	if c.GoogleAPIKey != "" {
		providers = append(providers, "gemini")
	}
	if c.QwenAPIKey != "" {
		providers = append(providers, "qwen")
	}
	if c.DeepSeekAPIKey != "" {
		providers = append(providers, "deepseek")
	}
	if c.GroqAPIKey != "" {
		providers = append(providers, "groq")
	}
	if c.KimiAPIKey != "" {
		providers = append(providers, "kimi")
	}
	if c.StepfunAPIKey != "" {
		providers = append(providers, "stepgui")
	}
	if c.AsticaAPIKey != "" {
		providers = append(providers, "astica")
	}
	return providers
}
