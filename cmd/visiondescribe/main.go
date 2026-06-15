// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Command visiondescribe is a thin, project-agnostic CLI bridge that turns a
// PNG screenshot into a structured, human-like UI description by calling a REAL
// vision-capable LLM through VisionEngine's pkg/llmvision adapters.
//
// It is decoupled from any consuming project (CONST-051(B) / §11.4.28): it reads
// only env vars + a screenshot path and prints structured JSON to stdout. The
// ATMOSphere test harness (or any consumer) calls it via ui_vision.sh.
//
// Provider selection (honest, evidence-driven — §11.4.6): it builds a fallback
// chain from whichever vision-capable keys are present in the environment, in
// priority order anthropic > openai > gemini > openrouter(openai-compatible).
// When NO vision-capable key resolves, it exits 3 (OPERATOR-BLOCKED) and emits a
// structured envelope saying so — it NEVER fabricates a canned description.
//
// Anti-bluff provenance (§11.4.116 unforgeable): every successful call emits the
// resolved provider name, model id, request image byte count, wall-clock latency
// and the raw model-text length — fields an offline stub cannot forge, because
// they come from the live HTTP round-trip.
//
// Usage:
//
//	visiondescribe -image shot.png [-focus "playback screen"] [-provider auto]
//	visiondescribe -probe        # report which vision providers are reachable
//
// Exit codes: 0 = described, 2 = bad input / model error, 3 = no vision provider
// configured (OPERATOR-BLOCKED), 4 = vision call failed (provider unreachable).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"digital.vasic.visionengine/pkg/config"
	"digital.vasic.visionengine/pkg/llmvision"
)

// structuredPrompt instructs the vision model to return STRICT JSON describing
// the screen the way a human looking at the device would. The §11.4.107 /
// §11.4.117 oracle then consumes the structured fields; the full_description is
// the free-text human-like account the operator mandate asks for.
const structuredPrompt = `You are looking at a single screenshot of an Android TV / tablet device screen.
Describe it exactly as a careful human tester would, then return ONLY a strict
JSON object (no markdown fences, no prose outside the JSON) with these fields:
{
  "app_label":      "<the app/launcher name shown, or 'unknown'>",
  "app_pkg_guess":  "<best-guess Android package id, or 'unknown'>",
  "screen_context": "<one of: launcher, app_home, content_list, player, settings, dialog, sign_in, error, loading, unknown>",
  "visible_title":  "<the most prominent on-screen title text, verbatim, or ''>",
  "attributes":     ["<short visible facts: e.g. '5.1 audio badge', '4K', 'progress bar 35%'>"],
  "playback_state": "<one of: playing, paused, buffering, stopped, not_applicable>",
  "overlays":       ["<any dialog/toast/ad/paywall/ANR/sign-in/geo-block overlay text, verbatim>"],
  "full_description": "<2-4 sentences: what app, what screen, what a human sees>"
}
Be literal about visible text. If a field is unknown use 'unknown' or '' or [].`

// envelope is the structured result the shell bridge consumes. It wraps the
// model's own JSON (parsed into the typed fields when possible) with the
// load-bearing real-call provenance.
type envelope struct {
	OK            bool            `json:"ok"`
	Provider      string          `json:"provider"`       // resolved provider name (gemini/openai/anthropic/fallback)
	Model         string          `json:"model"`          // model id used
	ImageBytes    int             `json:"image_bytes"`    // PNG byte count sent (proves a real image went out)
	LatencyMS     int64           `json:"latency_ms"`     // wall-clock of the HTTP round-trip (un-forgeable by a stub)
	RawTextLen    int             `json:"raw_text_len"`   // length of the model's raw reply
	ParsedOK      bool            `json:"parsed_ok"`      // did the model return valid JSON we could type
	Result        *visionResult   `json:"result"`         // typed structured fields (nil if parse failed)
	RawText       string          `json:"raw_text"`       // the model's raw reply (always present for audit)
	Error         string          `json:"error,omitempty"`
	OperatorBlock string          `json:"operator_block,omitempty"` // set when no vision key configured
	ProbeProviders []string       `json:"probe_providers,omitempty"`
}

// visionResult mirrors the JSON contract in structuredPrompt.
type visionResult struct {
	AppLabel        string   `json:"app_label"`
	AppPkgGuess     string   `json:"app_pkg_guess"`
	ScreenContext   string   `json:"screen_context"`
	VisibleTitle    string   `json:"visible_title"`
	Attributes      []string `json:"attributes"`
	PlaybackState   string   `json:"playback_state"`
	Overlays        []string `json:"overlays"`
	FullDescription string   `json:"full_description"`
}

func main() {
	var (
		imagePath = flag.String("image", "", "path to the PNG screenshot to describe")
		focus     = flag.String("focus", "", "optional extra hint appended to the prompt (e.g. 'is video playing?')")
		probe     = flag.Bool("probe", false, "report which vision providers are configured/reachable and exit")
	)
	flag.Parse()

	cfg := config.LoadFromEnv()

	if *probe {
		emit(envelope{OK: true, ProbeProviders: visionCapableProviders(cfg)})
		os.Exit(0)
	}

	if *imagePath == "" {
		emit(envelope{OK: false, Error: "missing -image <path>"})
		os.Exit(2)
	}

	img, err := os.ReadFile(*imagePath)
	if err != nil {
		emit(envelope{OK: false, Error: fmt.Sprintf("cannot read image: %v", err)})
		os.Exit(2)
	}
	if len(img) == 0 {
		emit(envelope{OK: false, Error: "image file is empty (0 bytes) — screencap likely failed or surface is FLAG_SECURE"})
		os.Exit(2)
	}

	provider, pname, pmodel, blockReason := buildProvider(cfg)
	if provider == nil {
		// §11.4.6 / §11.4.123 honest OPERATOR-BLOCKED — never a canned description.
		emit(envelope{
			OK:            false,
			OperatorBlock: blockReason,
			Error:         "no vision-capable provider configured",
		})
		os.Exit(3)
	}

	prompt := structuredPrompt
	if *focus != "" {
		prompt = prompt + "\n\nFocus especially on: " + *focus
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(maxInt(cfg.TimeoutSecs, 60))*time.Second)
	defer cancel()

	start := time.Now()
	rawText, err := provider.AnalyzeImage(ctx, img, prompt)
	latency := time.Since(start)
	if err != nil {
		emit(envelope{
			OK: false, Provider: pname, Model: pmodel,
			ImageBytes: len(img), LatencyMS: latency.Milliseconds(),
			Error: fmt.Sprintf("vision call failed: %v", err),
		})
		os.Exit(4)
	}

	env := envelope{
		OK:         true,
		Provider:   pname,
		Model:      pmodel,
		ImageBytes: len(img),
		LatencyMS:  latency.Milliseconds(),
		RawText:    rawText,
		RawTextLen: len(rawText),
	}
	if res, ok := parseVisionJSON(rawText); ok {
		env.ParsedOK = true
		env.Result = res
	}
	emit(env)
	os.Exit(0)
}

// buildProvider constructs a real vision provider (or a fallback chain) from the
// resolved env config, in priority order. Returns (nil,...,reason) when none is
// configured so the caller can surface an honest OPERATOR-BLOCKED.
func buildProvider(cfg config.Config) (llmvision.VisionProvider, string, string, string) {
	var chain []llmvision.VisionProvider
	primaryName, primaryModel := "", ""

	add := func(p llmvision.VisionProvider, err error, name, model string) {
		if err == nil && p != nil {
			if len(chain) == 0 {
				primaryName, primaryModel = name, model
			}
			chain = append(chain, p)
		}
	}

	// Priority: anthropic > openai > gemini > openrouter(openai-compatible).
	if cfg.AnthropicAPIKey != "" {
		model := orDefault(cfg.AnthropicModel, "claude-sonnet-4-20250514")
		p, err := llmvision.NewAnthropicProvider(llmvision.ProviderConfig{
			APIKey: cfg.AnthropicAPIKey, Model: model, TimeoutSecs: cfg.TimeoutSecs,
		})
		add(p, err, "anthropic", model)
	}
	if cfg.OpenAIAPIKey != "" && os.Getenv("OPENAI_API_KEY") != "" {
		model := orDefault(cfg.OpenAIModel, "gpt-4o")
		p, err := llmvision.NewOpenAIProvider(llmvision.ProviderConfig{
			APIKey: cfg.OpenAIAPIKey, Model: model, TimeoutSecs: cfg.TimeoutSecs,
		})
		add(p, err, "openai", model)
	}
	if cfg.GoogleAPIKey != "" {
		model := orDefault(cfg.GeminiModel, "gemini-2.5-flash")
		p, err := llmvision.NewGeminiProvider(llmvision.ProviderConfig{
			APIKey: cfg.GoogleAPIKey, Model: model, TimeoutSecs: cfg.TimeoutSecs,
		})
		add(p, err, "gemini", model)
	}
	// OpenRouter is OpenAI-API-compatible and serves vision models. It was loaded
	// into cfg.OpenAIAPIKey when OPENAI_API_KEY was absent (config.go fallback),
	// so only use it here if the raw OPENROUTER key is what resolved.
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("OPENROUTER_API_KEY") != "" && cfg.OpenAIAPIKey != "" {
		// Default to a free OpenRouter vision model so a credit-less key still
		// works (§11.4.6: proven reachable). MaxTokens is bounded to fit a free
		// tier — a 4096-token request is rejected 402 on a zero-credit account.
		model := orDefault(os.Getenv("HELIX_VISION_OPENROUTER_MODEL"), "nvidia/nemotron-nano-12b-v2-vl:free")
		maxTok := 1024
		if v := os.Getenv("HELIX_VISION_OPENROUTER_MAXTOK"); v != "" {
			if n, e := strconvAtoi(v); e == nil && n > 0 {
				maxTok = n
			}
		}
		p, err := llmvision.NewOpenAIProvider(llmvision.ProviderConfig{
			APIKey:    cfg.OpenAIAPIKey,
			BaseURL:   "https://openrouter.ai/api/v1",
			Model:     model,
			MaxTokens: maxTok,
			TimeoutSecs: cfg.TimeoutSecs,
		})
		add(p, err, "openrouter", model)
	}

	if len(chain) == 0 {
		return nil, "", "", "no vision-capable API key resolved in environment " +
			"(checked ANTHROPIC_API_KEY, OPENAI_API_KEY, GOOGLE_API_KEY/GEMINI_API_KEY, OPENROUTER_API_KEY). " +
			"Export a vision key via the credential single-source-of-truth, then re-run."
	}
	if len(chain) == 1 {
		return chain[0], primaryName, primaryModel, ""
	}
	fb, err := llmvision.NewFallbackProvider(chain...)
	if err != nil {
		return chain[0], primaryName, primaryModel, ""
	}
	// The fallback chain tries primary first; report the primary's identity so the
	// provenance reflects the most-likely-used provider (the chain still records it).
	return fb, "fallback(" + primaryName + ")", primaryModel, ""
}

// visionCapableProviders lists which configured providers can do vision.
func visionCapableProviders(cfg config.Config) []string {
	var out []string
	if cfg.AnthropicAPIKey != "" {
		out = append(out, "anthropic")
	}
	if cfg.OpenAIAPIKey != "" && os.Getenv("OPENAI_API_KEY") != "" {
		out = append(out, "openai")
	}
	if cfg.GoogleAPIKey != "" {
		out = append(out, "gemini")
	}
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("OPENROUTER_API_KEY") != "" {
		out = append(out, "openrouter")
	}
	return out
}

// parseVisionJSON tolerantly extracts the model's JSON object (models sometimes
// wrap it in ```json fences despite instructions).
func parseVisionJSON(raw string) (*visionResult, bool) {
	s := strings.TrimSpace(raw)
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[i+3:]
		if strings.HasPrefix(strings.ToLower(s), "json") {
			s = s[4:]
		}
		if j := strings.Index(s, "```"); j >= 0 {
			s = s[:j]
		}
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return nil, false
	}
	var res visionResult
	if err := json.Unmarshal([]byte(s[start:end+1]), &res); err != nil {
		return nil, false
	}
	return &res, true
}

func emit(e envelope) {
	b, _ := json.Marshal(e)
	fmt.Println(string(b))
}

func orDefault(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func strconvAtoi(s string) (int, error) { return strconv.Atoi(s) }
