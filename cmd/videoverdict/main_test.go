// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// fr is a tiny frameResult builder for the aggregation tests.
func fr(content, active bool, scene, onscreen, sub string, overlays ...string) frameResult {
	return frameResult{
		ImageBytes: 1, ParsedOK: true,
		ContentPresent: content, PlaybackActive: active,
		SceneDesc: scene, OnScreenText: onscreen, SubtitleText: sub, Overlays: overlays,
	}
}

// PASS: real content, advancing (distinct scenes), no overlay.
func TestAggregate_PlayingAdvancing_PASS(t *testing.T) {
	a := buildAggregate(12, []frameResult{
		fr(true, true, "spaceship corridor", "", ""),
		fr(true, true, "astronaut in cockpit", "", ""),
		fr(true, true, "alien creature emerges", "", ""),
	}, "", 0.5)
	if a.Verdict != "PASS" {
		t.Fatalf("want PASS, got %s reasons=%v", a.Verdict, a.Reasons)
	}
	if !a.VideoAdvancing || a.DistinctScenes != 3 {
		t.Fatalf("advancing=%v distinct=%d, want advancing + 3", a.VideoAdvancing, a.DistinctScenes)
	}
	if !a.ContentPresent || a.ContentFraction != 1.0 {
		t.Fatalf("content_present=%v frac=%.2f", a.ContentPresent, a.ContentFraction)
	}
}

// DEGRADED: real content but every frame is the SAME scene (frozen/stale).
func TestAggregate_FrozenSameScene_DEGRADED(t *testing.T) {
	same := "spaceship corridor with blue lights"
	a := buildAggregate(12, []frameResult{
		fr(true, true, same, "", ""),
		fr(true, true, same, "", ""),
		fr(true, true, same, "", ""),
	}, "", 0.5)
	if a.Verdict != "DEGRADED" {
		t.Fatalf("want DEGRADED (frozen), got %s reasons=%v", a.Verdict, a.Reasons)
	}
	if a.VideoAdvancing {
		t.Fatalf("frozen frames must not be advancing")
	}
}

// FAIL: mostly menu/black frames (content_fraction below threshold).
func TestAggregate_MenuNotContent_FAIL(t *testing.T) {
	a := buildAggregate(12, []frameResult{
		fr(false, false, "launcher home screen", "", ""),
		fr(false, false, "app grid", "", ""),
		fr(true, true, "brief content flash", "", ""),
	}, "", 0.5)
	if a.Verdict != "FAIL" {
		t.Fatalf("want FAIL (not content), got %s", a.Verdict)
	}
	if a.ContentPresent {
		t.Fatalf("content_present should be false (frac %.2f)", a.ContentFraction)
	}
}

// FAIL: a hostile overlay outranks otherwise-good content.
func TestAggregate_HostileOverlay_FAIL(t *testing.T) {
	a := buildAggregate(12, []frameResult{
		fr(true, true, "scene a", "", ""),
		fr(true, true, "scene b", "", "", "Application not responding"),
	}, "", 0.5)
	if a.Verdict != "FAIL" {
		t.Fatalf("want FAIL (overlay), got %s", a.Verdict)
	}
	if len(a.Obstructions) != 1 {
		t.Fatalf("want 1 obstruction, got %v", a.Obstructions)
	}
}

// Subtitle union + fuzzy match vs expected.
func TestAggregate_SubtitleMatch(t *testing.T) {
	a := buildAggregate(12, []frameResult{
		fr(true, true, "scene a", "", "I admire its purity"),
		fr(true, true, "scene b", "", ""),
	}, "I admire its purity. A survivor.", 0.5)
	if !a.SubtitleSeen {
		t.Fatalf("subtitle should be seen")
	}
	if !a.SubtitleMatch {
		t.Fatalf("expected subtitle match, ratio=%.2f pair=%v", a.SubtitleBestRatio, a.SubtitleBestPair)
	}
	if len(a.SubtitleLines) != 1 {
		t.Fatalf("want 1 subtitle line, got %v", a.SubtitleLines)
	}
}

// No subtitle read at all → subtitle_seen false, match false.
func TestAggregate_NoSubtitle(t *testing.T) {
	a := buildAggregate(12, []frameResult{
		fr(true, true, "scene a", "", ""),
		fr(true, true, "scene b", "", ""),
	}, "some expected line", 0.5)
	if a.SubtitleSeen || a.SubtitleMatch {
		t.Fatalf("no subtitle text was read; seen=%v match=%v", a.SubtitleSeen, a.SubtitleMatch)
	}
}

// Vision-failed frames are excluded from FramesWithVision but not from total.
func TestAggregate_VisionFailedFramesExcluded(t *testing.T) {
	failed := frameResult{Frame: "f2.png", Error: "vision call failed"}
	a := buildAggregate(3, []frameResult{
		fr(true, true, "scene a", "", ""),
		failed,
		fr(true, true, "scene b", "", ""),
	}, "", 0.5)
	if a.FramesWithVision != 2 {
		t.Fatalf("want 2 frames with vision, got %d", a.FramesWithVision)
	}
	if a.ContentFraction != 1.0 {
		t.Fatalf("content fraction should divide by frames-with-vision, got %.2f", a.ContentFraction)
	}
}

func TestEvenSample(t *testing.T) {
	in := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	out := evenSample(in, 4)
	if len(out) != 4 {
		t.Fatalf("want 4 sampled, got %d: %v", len(out), out)
	}
	if out[0] != "a" || out[len(out)-1] != "j" {
		t.Fatalf("even sample must include first+last: %v", out)
	}
	if got := evenSample(in, 0); len(got) != len(in) {
		t.Fatalf("sample<=0 returns all")
	}
	if got := evenSample(in[:3], 6); len(got) != 3 {
		t.Fatalf("sample>=len returns all")
	}
}

func TestParseFrameJSON_Fenced(t *testing.T) {
	raw := "```json\n{\"content_present\": true, \"scene_desc\": \"x\", \"subtitle_text\": \"hello\"}\n```"
	pf, ok := parseFrameJSON(raw)
	if !ok || !pf.ContentPresent || pf.SubtitleText != "hello" {
		t.Fatalf("fenced JSON parse failed: ok=%v pf=%+v", ok, pf)
	}
	if _, ok := parseFrameJSON("not json at all"); ok {
		t.Fatalf("non-JSON must not parse")
	}
}

func TestResolveFrames_DirSkipsROI(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"f01.png", "f02.png", "f01.png.roi.png", "note.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := resolveFrames(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 frames (roi + txt skipped), got %v", got)
	}
}

// §11.4.117(10) self-validation: a bracketed caption the model mis-files in
// overlays must NOT false-FAIL good playing content; a real ANR MUST FAIL.
func TestAggregate_CaptionNotHostile_PASS(t *testing.T) {
	a := buildAggregate(12, []frameResult{
		fr(true, true, "spaceship corridor", "", "", "[Noises Stop]"),
		fr(true, true, "astronaut in cockpit", "", ""),
		fr(true, true, "alien emerges", "", ""),
	}, "", 0.5)
	if a.Verdict != "PASS" {
		t.Fatalf("caption '[Noises Stop]' must NOT FAIL good content, got %s reasons=%v obstructions=%v", a.Verdict, a.Reasons, a.Obstructions)
	}
	if len(a.Obstructions) != 0 {
		t.Fatalf("a bracketed caption is not an obstruction, got %v", a.Obstructions)
	}
	// the caption is retained as a subtitle line (not discarded)
	found := false
	for _, s := range a.SubtitleLines {
		if s == "[Noises Stop]" {
			found = true
		}
	}
	if !found {
		t.Fatalf("caption should be retained as subtitle line, got %v", a.SubtitleLines)
	}
}

func TestAggregate_RealANR_FAIL(t *testing.T) {
	a := buildAggregate(12, []frameResult{
		fr(true, true, "scene a", "", "", "Application not responding"),
		fr(true, true, "scene b", "", ""),
	}, "", 0.5)
	if a.Verdict != "FAIL" {
		t.Fatalf("a real ANR overlay MUST FAIL, got %s", a.Verdict)
	}
}

func TestIsHostileOverlay(t *testing.T) {
	hostile := []string{"Application not responding", "Please sign in to continue", "Subscribe to watch", "not available in your region", "Playback error", "YouTube has stopped"}
	for _, s := range hostile {
		if !isHostileOverlay(s) {
			t.Fatalf("%q should be hostile", s)
		}
	}
	notHostile := []string{"[Noises Stop]", "[music]", "I admire its purity", "a spaceship corridor", "[gunfire]"}
	for _, s := range notHostile {
		if isHostileOverlay(s) {
			t.Fatalf("%q should NOT be hostile (it is a caption/scene)", s)
		}
	}
}

func TestCleanText(t *testing.T) {
	cases := map[string]string{
		"''":            "",
		"\"\"":          "",
		"none":          "",
		"N/A":           "",
		"  hello  ":     "hello",
		"'I admire it'": "I admire it",
	}
	for in, want := range cases {
		if got := cleanText(in); got != want {
			t.Fatalf("cleanText(%q)=%q want %q", in, got, want)
		}
	}
}

func TestRatio(t *testing.T) {
	if ratio("hello world", "hello world") < 0.99 {
		t.Fatal("identical strings ~1.0")
	}
	if ratio("", "x") != 0 {
		t.Fatal("empty ratio 0")
	}
	if ratio("abc", "xyz") > 0.34 {
		t.Fatal("disjoint low")
	}
}
