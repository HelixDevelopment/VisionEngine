// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "auto", cfg.VisionProvider)
	assert.True(t, cfg.OpenCVEnabled)
	assert.Equal(t, 0.95, cfg.SSIMThreshold)
	assert.Greater(t, cfg.MaxImageSize, 0)
	assert.Equal(t, 60, cfg.TimeoutSecs)
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := DefaultConfig()
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_InvalidProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VisionProvider = "invalid"
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestConfig_Validate_InvalidSSIM(t *testing.T) {
	cfg := DefaultConfig()

	cfg.SSIMThreshold = -0.1
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)

	cfg.SSIMThreshold = 1.1
	err = cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestConfig_Validate_InvalidMaxImageSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxImageSize = 0
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestConfig_Validate_InvalidTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TimeoutSecs = 0
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestConfig_Validate_OpenAIRequiresKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VisionProvider = "openai"
	cfg.OpenAIAPIKey = ""
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)

	cfg.OpenAIAPIKey = "sk-test"
	err = cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_AnthropicRequiresKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VisionProvider = "anthropic"
	cfg.AnthropicAPIKey = ""
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)

	cfg.AnthropicAPIKey = "sk-ant-test"
	err = cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_GeminiRequiresKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VisionProvider = "gemini"
	cfg.GoogleAPIKey = ""
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)

	cfg.GoogleAPIKey = "AItest"
	err = cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_QwenRequiresKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VisionProvider = "qwen"
	cfg.QwenAPIKey = ""
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)

	cfg.QwenAPIKey = "qwen-test"
	err = cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_AllProviders(t *testing.T) {
	providers := []string{"auto", "openai", "anthropic", "gemini", "qwen"}
	for _, p := range providers {
		cfg := DefaultConfig()
		cfg.VisionProvider = p
		cfg.OpenAIAPIKey = "sk-test"
		cfg.AnthropicAPIKey = "sk-ant-test"
		cfg.GoogleAPIKey = "AItest"
		cfg.QwenAPIKey = "qwen-test"
		err := cfg.Validate()
		assert.NoError(t, err, "Provider %s should be valid", p)
	}
}

func TestConfig_AvailableProviders_None(t *testing.T) {
	cfg := DefaultConfig()
	providers := cfg.AvailableProviders()
	assert.Empty(t, providers)
}

func TestConfig_AvailableProviders_All(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OpenAIAPIKey = "sk-test"
	cfg.AnthropicAPIKey = "sk-ant-test"
	cfg.GoogleAPIKey = "AItest"
	cfg.QwenAPIKey = "qwen-test"

	providers := cfg.AvailableProviders()
	assert.Len(t, providers, 4)
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "anthropic")
	assert.Contains(t, providers, "gemini")
	assert.Contains(t, providers, "qwen")
}

func TestConfig_AvailableProviders_Partial(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OpenAIAPIKey = "sk-test"
	cfg.GoogleAPIKey = "AItest"

	providers := cfg.AvailableProviders()
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "gemini")
}

func TestLoadFromEnv(t *testing.T) {
	// Save and restore env vars
	envVars := map[string]string{
		"HELIX_VISION_PROVIDER":       "anthropic",
		"HELIX_VISION_OPENCV_ENABLED": "false",
		"HELIX_VISION_SSIM_THRESHOLD": "0.90",
		"HELIX_VISION_MAX_IMAGE_SIZE": "8192",
		"OPENAI_API_KEY":              "sk-env-test",
		"ANTHROPIC_API_KEY":           "sk-ant-env-test",
		"GOOGLE_API_KEY":              "AI-env-test",
		"QWEN_API_KEY":                "qwen-env-test",
		"HELIX_VISION_TIMEOUT":        "120",
	}

	// Save originals
	originals := make(map[string]string)
	for k := range envVars {
		originals[k] = os.Getenv(k)
	}

	// Set test values
	for k, v := range envVars {
		os.Setenv(k, v)
	}

	// Restore on cleanup
	defer func() {
		for k, v := range originals {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	cfg := LoadFromEnv()
	assert.Equal(t, "anthropic", cfg.VisionProvider)
	assert.False(t, cfg.OpenCVEnabled)
	assert.Equal(t, 0.90, cfg.SSIMThreshold)
	assert.Equal(t, 8192, cfg.MaxImageSize)
	assert.Equal(t, "sk-env-test", cfg.OpenAIAPIKey)
	assert.Equal(t, "sk-ant-env-test", cfg.AnthropicAPIKey)
	assert.Equal(t, "AI-env-test", cfg.GoogleAPIKey)
	assert.Equal(t, "qwen-env-test", cfg.QwenAPIKey)
	assert.Equal(t, 120, cfg.TimeoutSecs)
}

func TestLoadFromEnv_Defaults(t *testing.T) {
	// Clear all relevant env vars
	vars := []string{
		"HELIX_VISION_PROVIDER", "HELIX_VISION_OPENCV_ENABLED",
		"HELIX_VISION_SSIM_THRESHOLD", "HELIX_VISION_MAX_IMAGE_SIZE",
		"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GOOGLE_API_KEY", "QWEN_API_KEY",
		"HELIX_VISION_TIMEOUT",
	}
	originals := make(map[string]string)
	for _, v := range vars {
		originals[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range originals {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	cfg := LoadFromEnv()
	assert.Equal(t, "auto", cfg.VisionProvider)
	assert.True(t, cfg.OpenCVEnabled)
	assert.Equal(t, 0.95, cfg.SSIMThreshold)
}

func TestLoadFromEnv_OpenCVEnabled_Variations(t *testing.T) {
	orig := os.Getenv("HELIX_VISION_OPENCV_ENABLED")
	defer func() {
		if orig == "" {
			os.Unsetenv("HELIX_VISION_OPENCV_ENABLED")
		} else {
			os.Setenv("HELIX_VISION_OPENCV_ENABLED", orig)
		}
	}()

	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"false", false},
		{"0", false},
		{"anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			os.Setenv("HELIX_VISION_OPENCV_ENABLED", tt.value)
			cfg := LoadFromEnv()
			assert.Equal(t, tt.expected, cfg.OpenCVEnabled)
		})
	}
}

func TestLoadFromEnv_InvalidNumbers(t *testing.T) {
	orig := os.Getenv("HELIX_VISION_SSIM_THRESHOLD")
	defer func() {
		if orig == "" {
			os.Unsetenv("HELIX_VISION_SSIM_THRESHOLD")
		} else {
			os.Setenv("HELIX_VISION_SSIM_THRESHOLD", orig)
		}
	}()

	os.Setenv("HELIX_VISION_SSIM_THRESHOLD", "not-a-number")
	cfg := LoadFromEnv()
	assert.Equal(t, 0.95, cfg.SSIMThreshold, "Invalid number should keep default")
}

func TestLoadFromEnv_CustomModels(t *testing.T) {
	envVars := map[string]string{
		"HELIX_VISION_OPENAI_MODEL":    "gpt-4o-mini",
		"HELIX_VISION_ANTHROPIC_MODEL": "claude-3-haiku",
		"HELIX_VISION_GEMINI_MODEL":    "gemini-pro-vision",
		"HELIX_VISION_QWEN_MODEL":      "qwen-vl-plus",
	}

	originals := make(map[string]string)
	for k := range envVars {
		originals[k] = os.Getenv(k)
	}
	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k, v := range originals {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	cfg := LoadFromEnv()
	assert.Equal(t, "gpt-4o-mini", cfg.OpenAIModel)
	assert.Equal(t, "claude-3-haiku", cfg.AnthropicModel)
	assert.Equal(t, "gemini-pro-vision", cfg.GeminiModel)
	assert.Equal(t, "qwen-vl-plus", cfg.QwenModel)
}

// --- Security Tests ---

func TestConfig_APIKeysNotInJSON(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OpenAIAPIKey = "sk-secret-key-12345"
	cfg.AnthropicAPIKey = "sk-ant-secret-key"

	// The json:"-" tag should prevent API keys from being serialized
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "sk-secret-key-12345")
	assert.NotContains(t, jsonStr, "sk-ant-secret-key")
}

func TestConfig_Validate_NegativeMaxImageSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxImageSize = -1
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestConfig_Validate_NegativeTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TimeoutSecs = -1
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestConfig_Validate_SSIMBoundary(t *testing.T) {
	cfg := DefaultConfig()

	cfg.SSIMThreshold = 0.0
	assert.NoError(t, cfg.Validate())

	cfg.SSIMThreshold = 1.0
	assert.NoError(t, cfg.Validate())
}
