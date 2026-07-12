// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Command videoverdict is a thin, project-agnostic CLI bridge that turns a
// DIRECTORY of ordered video frames (PNG/JPG screencaps) into a structured
// video-level "SEE" verdict by calling a REAL vision-capable LLM through
// VisionEngine's pkg/llmvision adapters, once per sampled frame.
//
// Where the sibling `visiondescribe` describes ONE image, videoverdict answers
// the three questions the operator's anti-bluff mandate asks of a video clip
// (§11.4.107 liveness + §11.4.117 pixel-oracle + §11.4.137 subtitle-content):
//
//	(a) Is real content PLAYING and ADVANCING (frames change, not a frozen
//	    stale frame, not a menu/black/error) — the SEE cross-check that
//	    complements a deterministic perceptual-hash oracle.
//	(b) READ the on-screen text, including burned-in SUBTITLES, verbatim.
//	(c) Flag OBSTRUCTIONS (ANR / sign-in / paywall / geo-block / ad / error).
//
// It is fully decoupled from any consuming project (CONST-051(B) / §11.4.28):
// it reads only env vars + a frames path + an optional expected-subtitle string,
// and prints structured JSON to stdout. The consuming project's test harness
// composes it (e.g. a project's av_deep_verdict.sh wrapper) with its own paths
// + .srt — the engine itself stays project-not-aware (§11.4.10 / §11.4.28).
//
// Provider selection is honest + evidence-driven (§11.4.6): it builds a
// fallback chain from whichever vision-capable keys are present in the
// environment, priority anthropic > openai > gemini > openrouter. When NO
// vision key resolves it exits 3 (OPERATOR-BLOCKED) with a structured envelope
// saying so — it NEVER fabricates a canned verdict (that is the caller's cue to
// fall back to the deterministic OCR/perceptual-hash oracle per §11.4.117).
//
// Anti-bluff provenance (§11.4.116 unforgeable): every per-frame result records
// the resolved provider, model, PNG byte count sent, and wall-clock latency of
// the live HTTP round-trip — fields an offline stub cannot forge.
//
// Usage:
//
//	videoverdict -frames DIR [-sample 6] [-expect-subtitle "TEXT"] \
//	    [-focus "extra hint"] [-min-content-frac 0.5]
//	videoverdict -frames a.png,b.png,c.png ...
//	videoverdict -probe                    # which vision providers are reachable
//
// Exit codes: 0 = verdict produced, 2 = bad input, 3 = no vision provider
// (OPERATOR-BLOCKED), 4 = every sampled frame's vision call failed.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"digital.vasic.visionengine/pkg/config"
	"digital.vasic.visionengine/pkg/llmvision"
)

// framePrompt asks the model for STRICT JSON per frame. The fields drive the
// §11.4.107 (content/advancing), §11.4.137 (subtitle) and obstruction oracles.
const framePrompt = `You are looking at ONE still frame from a video played on a TV/tablet screen.
Return ONLY a strict JSON object (no markdown fences, no prose outside JSON):
{
  "content_present": <true if this frame shows real playing video CONTENT (a film/show/scene/gameplay), false if it is a launcher/menu/home-screen/black-or-blank/loading-spinner/error page>,
  "scene_desc":      "<one short sentence describing what is visibly happening in the frame>",
  "on_screen_text":  "<ALL legible text visible anywhere in the frame, verbatim; '' if none>",
  "subtitle_text":   "<ONLY the burned-in subtitle/caption line(s), verbatim (usually lower-centre); '' if no subtitle visible>",
  "playback_active": <true if the frame looks like active playback (a scene), false if paused/stopped/menu>,
  "overlays":        ["<ONLY a HOSTILE app/system UI overlay text verbatim: 'Application not responding'/ANR, sign-in/login dialog, paywall/subscribe prompt, geo-block/'not available in your region', an advertisement, 'has stopped'/force-close, or a playback ERROR dialog. [] if none>"]
}
IMPORTANT: a subtitle or closed-caption line (including bracketed sound descriptions like "[Noises Stop]", "[music]", song lyrics, or dialogue) is NOT a hostile overlay — put such text in "subtitle_text", NEVER in "overlays". "overlays" is exclusively for app/system UI that BLOCKS the user from watching.
Be literal about visible text. Do not invent text that is not there.`

// frameResult mirrors framePrompt's JSON contract plus per-frame provenance.
type frameResult struct {
	Frame          string   `json:"frame"`
	ImageBytes     int      `json:"image_bytes"`
	LatencyMS      int64    `json:"latency_ms"`
	ParsedOK       bool     `json:"parsed_ok"`
	ContentPresent bool     `json:"content_present"`
	SceneDesc      string   `json:"scene_desc"`
	OnScreenText   string   `json:"on_screen_text"`
	SubtitleText   string   `json:"subtitle_text"`
	PlaybackActive bool     `json:"playback_active"`
	Overlays       []string `json:"overlays"`
	RawText        string   `json:"raw_text"`
	Error          string   `json:"error,omitempty"`
}

// aggregate is the video-level SEE verdict distilled from the per-frame results.
type aggregate struct {
	FramesTotal       int      `json:"frames_total"`
	FramesAnalyzed    int      `json:"frames_analyzed"`
	FramesWithVision  int      `json:"frames_with_vision"` // frames whose LLM call succeeded
	ContentFrames     int      `json:"content_frames"`     // frames with content_present==true
	ContentFraction   float64  `json:"content_fraction"`
	DistinctScenes    int      `json:"distinct_scenes"` // count of unique scene_desc (advancing signal)
	VideoAdvancing    bool     `json:"video_advancing"` // distinct_scenes >= 2 (content changes across frames)
	ContentPresent    bool     `json:"content_present"` // content_fraction >= min-content-frac
	SubtitleLines     []string `json:"subtitle_lines"`  // union of non-empty subtitle_text
	OnScreenTextUnion []string `json:"on_screen_text_union"`
	Obstructions      []string `json:"obstructions"` // union of overlays
	// Subtitle-vs-expected (only when -expect-subtitle given).
	ExpectSubtitle    string    `json:"expect_subtitle,omitempty"`
	SubtitleSeen      bool      `json:"subtitle_seen"`  // any subtitle line read at all
	SubtitleMatch     bool      `json:"subtitle_match"` // best fuzzy ratio >= 0.55 vs expected
	SubtitleBestRatio float64   `json:"subtitle_best_ratio"`
	SubtitleBestPair  [2]string `json:"subtitle_best_pair,omitempty"`
	// Overall.
	Verdict string   `json:"verdict"` // PASS | DEGRADED | FAIL | OPERATOR_BLOCKED
	Reasons []string `json:"reasons"`
}

// envelope is the top-level structured output the shell bridge consumes.
type envelope struct {
	OK             bool          `json:"ok"`
	Provider       string        `json:"provider"`
	Model          string        `json:"model"`
	Aggregate      *aggregate    `json:"aggregate"`
	Frames         []frameResult `json:"frames"`
	Error          string        `json:"error,omitempty"`
	OperatorBlock  string        `json:"operator_block,omitempty"`
	ProbeProviders []string      `json:"probe_providers,omitempty"`
}

func main() {
	var (
		framesArg      = flag.String("frames", "", "directory of frame images, a glob, or a comma-separated list of image paths")
		sample         = flag.Int("sample", 6, "max number of frames to send to the vision model (evenly spaced across the ordered set)")
		expectSubtitle = flag.String("expect-subtitle", "", "optional expected subtitle text to fuzzy-match against what the model reads on screen")
		focus          = flag.String("focus", "", "optional extra hint appended to every frame prompt")
		minContent     = flag.Float64("min-content-frac", 0.5, "fraction of analyzed frames that must show real content for content_present")
		delayMS        = flag.Int("delay-ms", 1200, "sleep between per-frame vision calls to avoid provider throttling (a burst of back-to-back calls gets rate-limited)")
		retry          = flag.Int("retry", 1, "retry count on a transient per-frame failure (timeout / rate-limit) with linear backoff")
		timeoutSecs    = flag.Int("timeout-secs", 0, "override per-call vision timeout in seconds (0 = provider config default)")
		fromJSON       = flag.String("from-frames-json", "", "re-aggregate an existing videoverdict envelope JSON WITHOUT re-calling the LLM (deterministic §11.4.50 — proves the aggregation on already-captured evidence)")
		probe          = flag.Bool("probe", false, "report which vision providers are configured and exit")
	)
	flag.Parse()

	cfg := config.LoadFromEnv()

	if *probe {
		emit(envelope{OK: true, ProbeProviders: visionCapableProviders(cfg)})
		os.Exit(0)
	}

	// Offline re-aggregation: recompute the video-level verdict from a prior
	// run's captured per-frame model output. No network, fully deterministic.
	if *fromJSON != "" {
		raw, err := os.ReadFile(*fromJSON)
		if err != nil {
			emit(envelope{OK: false, Error: fmt.Sprintf("cannot read -from-frames-json: %v", err)})
			os.Exit(2)
		}
		var prev envelope
		if err := json.Unmarshal(raw, &prev); err != nil {
			emit(envelope{OK: false, Error: fmt.Sprintf("bad envelope JSON: %v", err)})
			os.Exit(2)
		}
		total := len(prev.Frames)
		if prev.Aggregate != nil && prev.Aggregate.FramesTotal > total {
			total = prev.Aggregate.FramesTotal
		}
		agg := buildAggregate(total, prev.Frames, *expectSubtitle, *minContent)
		emit(envelope{OK: true, Provider: prev.Provider, Model: prev.Model, Aggregate: agg, Frames: prev.Frames})
		os.Exit(0)
	}

	if *framesArg == "" {
		emit(envelope{OK: false, Error: "missing -frames <dir|glob|comma-list>"})
		os.Exit(2)
	}

	frames, err := resolveFrames(*framesArg)
	if err != nil {
		emit(envelope{OK: false, Error: err.Error()})
		os.Exit(2)
	}
	if len(frames) == 0 {
		emit(envelope{OK: false, Error: "no frame images resolved from -frames"})
		os.Exit(2)
	}

	provider, pname, pmodel, blockReason := buildProvider(cfg)
	if provider == nil {
		emit(envelope{OK: false, OperatorBlock: blockReason, Error: "no vision-capable provider configured"})
		os.Exit(3)
	}

	sampled := evenSample(frames, *sample)
	prompt := framePrompt
	if *focus != "" {
		prompt = prompt + "\n\nAlso focus on: " + *focus
	}
	perCallTimeout := maxInt(cfg.TimeoutSecs, 60)
	if *timeoutSecs > 0 {
		perCallTimeout = *timeoutSecs
	}

	results := make([]frameResult, 0, len(sampled))
	visionOK := 0
	for i, fp := range sampled {
		if i > 0 && *delayMS > 0 {
			time.Sleep(time.Duration(*delayMS) * time.Millisecond)
		}
		fr := analyzeFrameWithRetry(provider, fp, prompt, perCallTimeout, *retry)
		if fr.Error == "" {
			visionOK++
		}
		results = append(results, fr)
	}

	if visionOK == 0 {
		emit(envelope{
			OK: false, Provider: pname, Model: pmodel, Frames: results,
			Error: "every sampled frame's vision call failed (provider unreachable/rate-limited)",
		})
		os.Exit(4)
	}

	agg := buildAggregate(len(frames), results, *expectSubtitle, *minContent)
	emit(envelope{OK: true, Provider: pname, Model: pmodel, Aggregate: agg, Frames: results})
	os.Exit(0)
}

// analyzeFrameWithRetry retries a transient per-frame failure (timeout /
// rate-limit / provider-unavailable) with linear backoff. A read/empty-image
// error is NOT retried (it will never succeed). This makes a burst of frames
// robust to provider throttling (§11.4.50 deterministic-under-load, §11.4.6 —
// the last real error is surfaced verbatim, never masked as a fake success).
func analyzeFrameWithRetry(p llmvision.VisionProvider, fp, prompt string, timeoutSecs, retry int) frameResult {
	var fr frameResult
	for attempt := 0; attempt <= retry; attempt++ {
		fr = analyzeFrame(p, fp, prompt, timeoutSecs)
		if fr.Error == "" {
			return fr
		}
		// Do not retry unrecoverable input errors.
		if strings.HasPrefix(fr.Error, "read:") || strings.HasPrefix(fr.Error, "empty image") {
			return fr
		}
		if attempt < retry {
			time.Sleep(time.Duration(attempt+1) * 1500 * time.Millisecond)
		}
	}
	return fr
}

// analyzeFrame runs one real vision call for a single frame.
func analyzeFrame(p llmvision.VisionProvider, fp, prompt string, timeoutSecs int) frameResult {
	fr := frameResult{Frame: fp}
	img, err := os.ReadFile(fp)
	if err != nil {
		fr.Error = fmt.Sprintf("read: %v", err)
		return fr
	}
	if len(img) == 0 {
		fr.Error = "empty image (0 bytes) — screencap failed or FLAG_SECURE surface"
		return fr
	}
	fr.ImageBytes = len(img)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)
	defer cancel()
	start := time.Now()
	raw, err := p.AnalyzeImage(ctx, img, prompt)
	fr.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		fr.Error = fmt.Sprintf("vision call failed: %v", err)
		return fr
	}
	fr.RawText = raw
	if parsed, ok := parseFrameJSON(raw); ok {
		fr.ParsedOK = true
		fr.ContentPresent = parsed.ContentPresent
		fr.SceneDesc = strings.TrimSpace(parsed.SceneDesc)
		fr.OnScreenText = strings.TrimSpace(parsed.OnScreenText)
		fr.SubtitleText = strings.TrimSpace(parsed.SubtitleText)
		fr.PlaybackActive = parsed.PlaybackActive
		fr.Overlays = nonEmpty(parsed.Overlays)
	}
	return fr
}

// buildAggregate distills the per-frame results into the video-level verdict.
func buildAggregate(total int, results []frameResult, expect string, minContent float64) *aggregate {
	a := &aggregate{FramesTotal: total, FramesAnalyzed: len(results)}
	sceneSet := map[string]struct{}{}
	subLineSet := map[string]struct{}{}
	osTextSet := map[string]struct{}{}
	obsSet := map[string]struct{}{}
	for _, r := range results {
		if r.Error != "" {
			continue
		}
		a.FramesWithVision++
		if r.ContentPresent {
			a.ContentFrames++
		}
		if s := normScene(r.SceneDesc); s != "" {
			sceneSet[s] = struct{}{}
		}
		if sub := cleanText(r.SubtitleText); sub != "" {
			subLineSet[sub] = struct{}{}
		}
		if os := cleanText(r.OnScreenText); os != "" {
			osTextSet[os] = struct{}{}
		}
		for _, o := range r.Overlays {
			// §11.4.117(10) self-validation: only a genuinely HOSTILE overlay
			// counts as an obstruction. A subtitle / caption / bracketed sound
			// description the model mis-filed here must NOT false-FAIL good
			// content (it is captured under subtitle instead).
			oc := cleanText(o)
			if oc == "" {
				continue
			}
			if isHostileOverlay(oc) {
				obsSet[oc] = struct{}{}
			} else {
				// A non-hostile string the model put in overlays is still
				// on-screen text worth keeping (often a caption).
				subLineSet[oc] = struct{}{}
			}
		}
	}
	a.DistinctScenes = len(sceneSet)
	a.VideoAdvancing = a.DistinctScenes >= 2
	if a.FramesWithVision > 0 {
		a.ContentFraction = float64(a.ContentFrames) / float64(a.FramesWithVision)
	}
	a.ContentPresent = a.ContentFraction >= minContent
	a.SubtitleLines = sortedKeys(subLineSet)
	a.OnScreenTextUnion = sortedKeys(osTextSet)
	a.Obstructions = sortedKeys(obsSet)
	a.SubtitleSeen = len(a.SubtitleLines) > 0

	if strings.TrimSpace(expect) != "" {
		a.ExpectSubtitle = expect
		en := normText(expect)
		best := 0.0
		var bestPair [2]string
		// Compare the expected text against every subtitle line AND every
		// on-screen-text blob the model read (subtitles sometimes land in
		// on_screen_text). Substring containment gets a floor of 0.85.
		cands := append([]string{}, a.SubtitleLines...)
		cands = append(cands, a.OnScreenTextUnion...)
		for _, c := range cands {
			cn := normText(c)
			if cn == "" {
				continue
			}
			r := ratio(cn, en)
			if en != "" && (strings.Contains(cn, en) || strings.Contains(en, cn)) {
				if r < 0.85 {
					r = 0.85
				}
			}
			if r > best {
				best = r
				bestPair = [2]string{trunc(c, 80), trunc(expect, 80)}
			}
		}
		a.SubtitleBestRatio = round3(best)
		a.SubtitleBestPair = bestPair
		a.SubtitleMatch = best >= 0.55
	}

	// Verdict.
	if len(a.Obstructions) > 0 {
		a.Verdict = "FAIL"
		a.Reasons = append(a.Reasons, "hostile overlay(s) detected: "+strings.Join(a.Obstructions, " | "))
	} else if !a.ContentPresent {
		a.Verdict = "FAIL"
		a.Reasons = append(a.Reasons, fmt.Sprintf("content_fraction %.2f < %.2f (frames look like menu/black/error, not playing content)", a.ContentFraction, minContent))
	} else if !a.VideoAdvancing {
		a.Verdict = "DEGRADED"
		a.Reasons = append(a.Reasons, fmt.Sprintf("only %d distinct scene(s) across %d analyzed frames (possible frozen/stale frame — cross-check the perceptual-hash liveness oracle)", a.DistinctScenes, a.FramesWithVision))
	} else {
		a.Verdict = "PASS"
		a.Reasons = append(a.Reasons, fmt.Sprintf("real content playing + advancing (%d distinct scenes over %d frames, content_fraction %.2f)", a.DistinctScenes, a.FramesWithVision, a.ContentFraction))
	}
	if strings.TrimSpace(expect) != "" {
		if a.SubtitleMatch {
			a.Reasons = append(a.Reasons, fmt.Sprintf("subtitle read matches expected (ratio %.2f)", a.SubtitleBestRatio))
		} else if a.SubtitleSeen {
			a.Reasons = append(a.Reasons, fmt.Sprintf("subtitle text read on screen but did NOT match expected (best ratio %.2f)", a.SubtitleBestRatio))
		} else {
			a.Reasons = append(a.Reasons, "no burned-in subtitle text read in any analyzed frame")
		}
	}
	return a
}

// --- provider construction (self-contained, project-agnostic; mirrors visiondescribe) ---

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
	if cfg.AnthropicAPIKey != "" {
		model := orDefault(cfg.AnthropicModel, "claude-sonnet-4-20250514")
		p, err := llmvision.NewAnthropicProvider(llmvision.ProviderConfig{APIKey: cfg.AnthropicAPIKey, Model: model, TimeoutSecs: cfg.TimeoutSecs})
		add(p, err, "anthropic", model)
	}
	if cfg.OpenAIAPIKey != "" && os.Getenv("OPENAI_API_KEY") != "" {
		model := orDefault(cfg.OpenAIModel, "gpt-4o")
		p, err := llmvision.NewOpenAIProvider(llmvision.ProviderConfig{APIKey: cfg.OpenAIAPIKey, Model: model, TimeoutSecs: cfg.TimeoutSecs})
		add(p, err, "openai", model)
	}
	if cfg.GoogleAPIKey != "" {
		model := orDefault(cfg.GeminiModel, "gemini-2.5-flash")
		p, err := llmvision.NewGeminiProvider(llmvision.ProviderConfig{APIKey: cfg.GoogleAPIKey, Model: model, TimeoutSecs: cfg.TimeoutSecs})
		add(p, err, "gemini", model)
	}
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("OPENROUTER_API_KEY") != "" && cfg.OpenAIAPIKey != "" {
		model := orDefault(os.Getenv("HELIX_VISION_OPENROUTER_MODEL"), "nvidia/nemotron-nano-12b-v2-vl:free")
		p, err := llmvision.NewOpenAIProvider(llmvision.ProviderConfig{APIKey: cfg.OpenAIAPIKey, BaseURL: "https://openrouter.ai/api/v1", Model: model, MaxTokens: 1024, TimeoutSecs: cfg.TimeoutSecs})
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
	return fb, "fallback(" + primaryName + ")", primaryModel, ""
}

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

// --- helpers ---

type parsedFrame struct {
	ContentPresent bool     `json:"content_present"`
	SceneDesc      string   `json:"scene_desc"`
	OnScreenText   string   `json:"on_screen_text"`
	SubtitleText   string   `json:"subtitle_text"`
	PlaybackActive bool     `json:"playback_active"`
	Overlays       []string `json:"overlays"`
}

func parseFrameJSON(raw string) (*parsedFrame, bool) {
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
	var pf parsedFrame
	if err := json.Unmarshal([]byte(s[start:end+1]), &pf); err != nil {
		return nil, false
	}
	return &pf, true
}

// resolveFrames turns a dir / glob / comma-list into an ordered list of image
// paths (sorted lexicographically so f01,f02,... stay in capture order).
func resolveFrames(arg string) ([]string, error) {
	if strings.Contains(arg, ",") {
		var out []string
		for _, p := range strings.Split(arg, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		sort.Strings(out)
		return out, nil
	}
	info, err := os.Stat(arg)
	if err == nil && info.IsDir() {
		var out []string
		for _, ext := range []string{"*.png", "*.jpg", "*.jpeg"} {
			m, _ := filepath.Glob(filepath.Join(arg, ext))
			for _, f := range m {
				// Skip derived ROI crops (e.g. f01.png.roi.png).
				if strings.Contains(filepath.Base(f), ".roi.") {
					continue
				}
				out = append(out, f)
			}
		}
		sort.Strings(out)
		return out, nil
	}
	// Treat as a glob.
	m, gerr := filepath.Glob(arg)
	if gerr != nil {
		return nil, fmt.Errorf("bad -frames glob %q: %v", arg, gerr)
	}
	var out []string
	for _, f := range m {
		if strings.Contains(filepath.Base(f), ".roi.") {
			continue
		}
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}

// evenSample picks up to n evenly-spaced frames preserving order.
func evenSample(frames []string, n int) []string {
	if n <= 0 || len(frames) <= n {
		return frames
	}
	out := make([]string, 0, n)
	step := float64(len(frames)-1) / float64(n-1)
	seen := map[int]bool{}
	for i := 0; i < n; i++ {
		idx := int(float64(i)*step + 0.5)
		if idx >= len(frames) {
			idx = len(frames) - 1
		}
		if !seen[idx] {
			out = append(out, frames[idx])
			seen[idx] = true
		}
	}
	return out
}

func emit(e envelope) {
	b, _ := json.Marshal(e)
	fmt.Println(string(b))
}

func nonEmpty(in []string) []string {
	var out []string
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normScene(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// cleanText strips the model's empty-value artifacts (” / "" / none / n/a) and
// surrounding quote characters, returning "" for anything that is effectively
// empty so it never leaks into the subtitle/on-screen unions.
func cleanText(s string) string {
	s = strings.TrimSpace(s)
	// Strip matching leading/trailing single or double quotes the model
	// sometimes echoes literally (e.g. the two-char string ' ' ).
	for len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[len(s)-1] == s[0] {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	switch strings.ToLower(s) {
	case "", "none", "n/a", "na", "null", "unknown", "-":
		return ""
	}
	return s
}

// hostileOverlayNeedles are the substrings that mark an overlay as genuinely
// BLOCKING the user from watching (§11.4.5 obstruction census). A caption /
// subtitle / bracketed sound description is NOT here — so it never false-FAILs.
var hostileOverlayNeedles = []string{
	"not responding", "isn't responding", "anr",
	"sign in", "sign-in", "signin", "log in", "login", "please sign",
	"paywall", "subscribe", "subscription", "premium required", "upgrade to",
	"not available in your", "unavailable in your", "geo", "not available in this region", "restricted in your",
	"has stopped", "keeps stopping", "force close", "force stop",
	"error", "can't play", "cannot play", "unable to play", "playback error", "something went wrong",
	"advertisement", "skip ad", "your ad", "ad will",
	"not certified", "play protect",
}

// isHostileOverlay classifies an overlay string. A wholly-bracketed token like
// "[Noises Stop]" / "[music]" is a caption, never hostile.
func isHostileOverlay(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
		return false
	}
	for _, n := range hostileOverlayNeedles {
		if strings.Contains(t, n) {
			return true
		}
	}
	return false
}

func normText(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// ratio is a lightweight normalized similarity (longest-common-subsequence /
// max length) — no external deps, deterministic (§11.4.50).
func ratio(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	la, lb := len(a), len(b)
	dp := make([]int, lb+1)
	for i := 1; i <= la; i++ {
		prev := 0
		for j := 1; j <= lb; j++ {
			tmp := dp[j]
			if a[i-1] == b[j-1] {
				dp[j] = prev + 1
			} else if dp[j-1] > dp[j] {
				dp[j] = dp[j-1]
			}
			prev = tmp
		}
	}
	lcs := dp[lb]
	m := la
	if lb > m {
		m = lb
	}
	return float64(lcs) / float64(m)
}

func round3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
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
