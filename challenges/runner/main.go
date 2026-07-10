// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Command runner exercises VisionEngine's public surface end-to-end
// with REAL constructions (no mocks, no stubs of stubs) and prints
// runtime evidence per Article XI §11.9 + CONST-035.
//
// What this proves at runtime (every invocation):
//
//  1. analyzer.StubAnalyzer constructs and executes AnalyzeScreen on a
//     non-empty screenshot buffer, and HONESTLY refuses to fabricate
//     an analysis result — it returns the documented
//     ErrStubAnalyzerNotImplemented sentinel rather than a populated
//     ScreenAnalysis. (Audit round 2026-07-10: this check previously
//     expected AnalyzeScreen to SUCCEED with a populated ScreenID —
//     that was the pre-round-27 bluff behaviour pkg/analyzer/stub.go
//     documents as REMOVED; the runner was never updated to match,
//     so it had been failing this check on every real invocation
//     since round-27 landed. Reconciled here to assert the CURRENT,
//     correct, honest-refusal contract instead of reverting the
//     round-27 anti-bluff fix or silently weakening this check.)
//  2. The same call rejects an empty buffer with ErrEmptyScreenshot
//     (negative-path proof — the analyzer is not vacuously OK on any
//     input).
//  3. graph.NewNavigationGraph + AddScreen + AddTransition +
//     SetCurrent + PathTo wire together; a 3-screen BFS path is
//     discovered and printed.
//  4. graph.ExportMermaid + ExportDOT + ExportJSON each produce
//     non-empty output (3 export back-ends, real serializer paths).
//  5. analyzer.Rect.Contains / Overlaps / Area / Center compute the
//     expected geometry for a sample 100x50 rectangle.
//  6. i18n.NoopTranslator.T returns the msgID verbatim (the
//     CONST-051(B) standalone-default seam), proving the seam exists
//     and behaves per CONST-046.
//
// Bilingual evidence (CONST-046): every printed line carries a
// 5-locale label (en / sr / ja / de / es) demonstrating that the
// captured artefacts are not English-only smoke. The locale strings
// are tagged as runner-side fixtures (NOT inside the module's source
// tree) so the module itself remains language-agnostic per
// CONST-046.
//
// Exit codes:
//
//	0  — every check above produced positive runtime evidence
//	1  — at least one check failed (failure cause printed)
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"digital.vasic.visionengine/pkg/analyzer"
	"digital.vasic.visionengine/pkg/graph"
	"digital.vasic.visionengine/pkg/i18n"
)

// localeLabel returns a 5-locale label for a runner check name so the
// captured output is verifiably bilingual per CONST-046 round-297
// evidence-floor.
func localeLabel(en, sr, ja, de, es string) string {
	return fmt.Sprintf("[en=%s | sr=%s | ja=%s | de=%s | es=%s]", en, sr, ja, de, es)
}

// die prints an error line and exits 1.
func die(stage string, err error) {
	fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", stage, err)
	os.Exit(1)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("VisionEngine challenge runner — round-297 deep-doc evidence")
	fmt.Println(localeLabel(
		"VisionEngine challenge runner",
		"VisionEngine pokretač izazova",
		"VisionEngineチャレンジランナー",
		"VisionEngine Challenge-Runner",
		"Ejecutor de desafíos VisionEngine",
	))
	fmt.Println(strings.Repeat("=", 72))

	// ── Check 1: AnalyzeScreen on non-empty buffer honestly refuses ──
	// (round-27 anti-bluff fix, reconciled here 2026-07-10: see the
	// package doc comment above for the full defect narrative.)
	stub := analyzer.NewStubAnalyzer()
	screenshot := []byte("PNG-pretend-bytes-for-runner-deterministic-fingerprint")
	_, err := stub.AnalyzeScreen(ctx, screenshot)
	if !errors.Is(err, analyzer.ErrStubAnalyzerNotImplemented) {
		die("AnalyzeScreen(non-empty)", fmt.Errorf("want ErrStubAnalyzerNotImplemented (honest no-bluff refusal), got %v", err))
	}
	fmt.Println(localeLabel(
		"AnalyzeScreen OK — honestly refuses to fabricate analysis (no OpenCV/LLM wired)",
		"AnalyzeScreen radi",
		"AnalyzeScreen成功",
		"AnalyzeScreen erfolgreich",
		"AnalyzeScreen exitoso",
	))

	// ── Check 2: AnalyzeScreen rejects empty (negative path) ──────
	_, err = stub.AnalyzeScreen(ctx, nil)
	if !errors.Is(err, analyzer.ErrEmptyScreenshot) {
		die("AnalyzeScreen(empty)", fmt.Errorf("want ErrEmptyScreenshot, got %v", err))
	}
	fmt.Println(localeLabel(
		"AnalyzeScreen rejects empty buffer",
		"AnalyzeScreen odbija prazan bafer",
		"AnalyzeScreenは空のバッファを拒否",
		"AnalyzeScreen lehnt leeren Puffer ab",
		"AnalyzeScreen rechaza búfer vacío",
	))

	// ── Check 3: NavigationGraph end-to-end ───────────────────────
	g := graph.NewNavigationGraph()
	g.AddScreen(analyzer.ScreenIdentity{ID: "home", Name: "Home"})
	g.AddScreen(analyzer.ScreenIdentity{ID: "settings", Name: "Settings"})
	g.AddScreen(analyzer.ScreenIdentity{ID: "account", Name: "Account"})
	g.AddTransition("home", "settings", analyzer.Action{Type: "click", Target: "gear"})
	g.AddTransition("settings", "account", analyzer.Action{Type: "click", Target: "account-row"})
	g.SetCurrent("home")
	path, err := g.PathTo("account")
	if err != nil {
		die("PathTo(account)", err)
	}
	if len(path) != 2 {
		die("PathTo(account)", fmt.Errorf("want 2 transitions, got %d", len(path)))
	}
	fmt.Println(localeLabel(
		fmt.Sprintf("NavigationGraph BFS found %d-hop path home→settings→account", len(path)),
		"NavigationGraph pronašao putanju",
		"NavigationGraphが経路を発見",
		"NavigationGraph fand Pfad",
		"NavigationGraph encontró ruta",
	))

	// ── Check 4: Three export back-ends produce output ────────────
	mermaid := graph.ExportMermaid(g)
	dot := graph.ExportDOT(g)
	jsonOut, err := graph.ExportJSON(g)
	if err != nil {
		die("ExportJSON", err)
	}
	if len(mermaid) == 0 || len(dot) == 0 || len(jsonOut) == 0 {
		die("Export*", fmt.Errorf("empty output mermaid=%d dot=%d json=%d",
			len(mermaid), len(dot), len(jsonOut)))
	}
	fmt.Println(localeLabel(
		fmt.Sprintf("Exports OK mermaid=%dB dot=%dB json=%dB",
			len(mermaid), len(dot), len(jsonOut)),
		"Izvozi rade",
		"エクスポート成功",
		"Exporte erfolgreich",
		"Exportaciones exitosas",
	))

	// ── Check 5: Rect geometry sanity ─────────────────────────────
	r := analyzer.Rect{X: 10, Y: 20, Width: 100, Height: 50}
	if !r.Contains(50, 40) {
		die("Rect.Contains", errors.New("(50,40) should be inside (10,20,100,50)"))
	}
	if r.Contains(5, 5) {
		die("Rect.Contains", errors.New("(5,5) should be outside (10,20,100,50)"))
	}
	if r.Area() != 5000 {
		die("Rect.Area", fmt.Errorf("want 5000, got %d", r.Area()))
	}
	cx, cy := r.Center()
	if cx != 60 || cy != 45 {
		die("Rect.Center", fmt.Errorf("want (60,45), got (%d,%d)", cx, cy))
	}
	other := analyzer.Rect{X: 50, Y: 30, Width: 100, Height: 50}
	if !r.Overlaps(other) {
		die("Rect.Overlaps", errors.New("rects should overlap"))
	}
	fmt.Println(localeLabel(
		fmt.Sprintf("Rect geometry: area=%d center=(%d,%d) contains+overlaps OK", r.Area(), cx, cy),
		"Geometrija pravougaonika OK",
		"矩形ジオメトリOK",
		"Rechteck-Geometrie OK",
		"Geometría de rectángulo OK",
	))

	// ── Check 6: i18n.NoopTranslator seam ─────────────────────────
	tr := i18n.NoopTranslator{}
	if got := tr.T(ctx, "visionengine_stub_screen_title"); got != "visionengine_stub_screen_title" {
		die("NoopTranslator.T", fmt.Errorf("want passthrough, got %q", got))
	}
	fmt.Println(localeLabel(
		"NoopTranslator passthrough OK (CONST-046 seam present)",
		"NoopTranslator radi (CONST-046)",
		"NoopTranslator確認 (CONST-046)",
		"NoopTranslator OK (CONST-046)",
		"NoopTranslator OK (CONST-046)",
	))

	fmt.Println(strings.Repeat("=", 72))
	fmt.Println("ALL CHECKS PASSED — runtime evidence captured per Article XI §11.9")
}
