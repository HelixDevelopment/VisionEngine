// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build vision

// Package opencv ORB feature-detection end-to-end tests.
//
// These tests exercise the REAL gocv ORB + brute-force Hamming matcher (built
// only under the `vision` build tag, requiring cgo + a real OpenCV install)
// against deterministic synthetic images. They assert real keypoint counts and
// a MEANINGFUL match-count ordering (self-crop strictly beats an unrelated
// image) — the sink-side evidence mandated by CLAUDE-1 that ORB really locates a
// template, not that it returns a constant.
package opencv

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// synthTexturedPNG renders a w×h canvas tiled with a deterministic checker-plus-
// diagonal pattern that produces many strong corners/edges (the structure ORB's
// FAST detector keys on). The `seed` shifts the pattern so two different seeds
// yield visually unrelated textures with disjoint corner layouts.
func synthTexturedPNG(t *testing.T, w, h, block, seed int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Checkerboard of `block`-sized cells.
			cell := ((x/block)+(y/block)+seed)%2 == 0
			// Diagonal stripes overlaid to add oriented corners, seed-shifted.
			stripe := ((x + y + seed*7) / (block / 2)) % 2 == 0
			var v uint8
			switch {
			case cell && stripe:
				v = 255
			case cell && !stripe:
				v = 190
			case !cell && stripe:
				v = 60
			default:
				v = 0
			}
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode textured png: %v", err)
	}
	return buf.Bytes()
}

// cropPNG returns a PNG of the (rx,ry,rw,rh) sub-region of a source RGBA image.
func cropPNG(t *testing.T, src *image.RGBA, rx, ry, rw, rh int) []byte {
	t.Helper()
	out := image.NewRGBA(image.Rect(0, 0, rw, rh))
	for y := 0; y < rh; y++ {
		for x := 0; x < rw; x++ {
			out.Set(x, y, src.At(rx+x, ry+y))
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		t.Fatalf("encode crop png: %v", err)
	}
	return buf.Bytes()
}

// synthTexturedRGBA is the *image.RGBA twin of synthTexturedPNG, so a template
// can be a genuine pixel crop of the exact source the matcher sees.
func synthTexturedRGBA(w, h, block, seed int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cell := ((x/block)+(y/block)+seed)%2 == 0
			stripe := ((x + y + seed*7) / (block / 2)) % 2 == 0
			var v uint8
			switch {
			case cell && stripe:
				v = 255
			case cell && !stripe:
				v = 190
			case !cell && stripe:
				v = 60
			default:
				v = 0
			}
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	return img
}

func encodePNG(t *testing.T, img *image.RGBA) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestRealFeatureDetector_DetectKeypoints_FindsManyOnTexture(t *testing.T) {
	if !Available() {
		t.Fatal("vision build tag set but Available()==false")
	}
	img := synthTexturedPNG(t, 240, 240, 24, 0)
	fd := NewFeatureDetector()

	kps, err := fd.DetectKeypoints(img)
	if err != nil {
		t.Fatalf("DetectKeypoints error: %v", err)
	}
	if len(kps) == 0 {
		t.Fatal("DetectKeypoints found no keypoints on a richly textured image — real gocv ORB produced nothing")
	}
	// A 240×240 checker/diagonal texture has many corners; require a non-trivial
	// count so a degenerate "returns one point" impl would fail.
	if len(kps) < 10 {
		t.Fatalf("DetectKeypoints found only %d keypoints; expected several on a textured image", len(kps))
	}
	// Keypoints must be ordered strongest-response first and carry real geometry.
	for i := 1; i < len(kps); i++ {
		if kps[i-1].Response < kps[i].Response {
			t.Fatalf("keypoints not sorted by response desc at %d: %.4f < %.4f",
				i, kps[i-1].Response, kps[i].Response)
		}
	}
	top := kps[0]
	if top.X < 0 || top.X >= 240 || top.Y < 0 || top.Y >= 240 || top.Size <= 0 {
		t.Fatalf("top keypoint has implausible geometry: %+v", top)
	}
	t.Logf("DetectKeypoints: %d keypoints (real gocv ORB); strongest=%+v", len(kps), top)
}

func TestRealFeatureDetector_DetectKeypoints_FlatImageHasFewOrNone(t *testing.T) {
	// A flat (uniform) image has no corners → ORB should find essentially nothing.
	// This proves keypoint count is data-driven, not a constant.
	flat := image.NewRGBA(image.Rect(0, 0, 120, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 120; x++ {
			flat.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}
	fd := NewFeatureDetector()
	kps, err := fd.DetectKeypoints(encodePNG(t, flat))
	if err != nil {
		t.Fatalf("DetectKeypoints(flat) error: %v", err)
	}
	textured, err := fd.DetectKeypoints(synthTexturedPNG(t, 120, 120, 12, 0))
	if err != nil {
		t.Fatalf("DetectKeypoints(textured) error: %v", err)
	}
	if !(len(kps) < len(textured)) {
		t.Fatalf("flat image keypoints (%d) not fewer than textured (%d) — count is not data-driven",
			len(kps), len(textured))
	}
	t.Logf("DetectKeypoints flat=%d  textured=%d (data-driven, real gocv ORB)", len(kps), len(textured))
}

func TestRealFeatureDetector_MatchFeatures_SelfCropBeatsUnrelated(t *testing.T) {
	// Source: large textured image (seed 0). Template: a genuine, generously
	// sized crop of it (small blocks relative to the crop → many template
	// keypoints). Unrelated: a different-seed texture with a disjoint layout.
	w, h, block := 320, 320, 16
	srcRGBA := synthTexturedRGBA(w, h, block, 0)
	srcPNG := encodePNG(t, srcRGBA)

	// Template is an exact 160×160 crop spanning ~10×10 pattern cells → rich,
	// distinctive feature set that ORB can extract many keypoints from.
	tplPNG := cropPNG(t, srcRGBA, 80, 80, 160, 160)

	// Unrelated source: different seed, same size/texture family but a different
	// pixel layout, so ORB descriptors should mostly NOT match the seed-0 crop.
	unrelatedPNG := synthTexturedPNG(t, w, h, block, 1)

	fd := NewFeatureDetector()

	self, err := fd.MatchFeatures(srcPNG, tplPNG)
	if err != nil {
		t.Fatalf("MatchFeatures(self-crop) error: %v", err)
	}
	unrelated, err := fd.MatchFeatures(unrelatedPNG, tplPNG)
	if err != nil {
		// errNoFeatures is an acceptable "zero match" outcome for the unrelated
		// case, but a self-crop must always have features; treat any non-nil
		// unrelated error as zero good matches for the ordering assertion.
		t.Logf("MatchFeatures(unrelated) returned err=%v (treated as 0 good matches)", err)
		unrelated = 0
	}

	// A self-crop MUST produce a meaningful number of good matches.
	if self < 3 {
		t.Fatalf("MatchFeatures(self-crop)=%d good matches; expected several (real descriptor match broken?)", self)
	}
	// The load-bearing assertion: self-crop STRICTLY beats unrelated. If matching
	// were a constant or broken, this ordering would not hold.
	if !(self > unrelated) {
		t.Fatalf("self-crop good matches (%d) not strictly greater than unrelated (%d) — matching is not discriminative",
			self, unrelated)
	}
	t.Logf("MatchFeatures: self-crop=%d  unrelated=%d (real gocv ORB + BFMatcher + Lowe ratio)", self, unrelated)
}

func TestRealFeatureDetector_MatchFeatures_InvalidInputs(t *testing.T) {
	fd := NewFeatureDetector()
	if _, err := fd.MatchFeatures(nil, []byte{1}); err == nil {
		t.Fatal("MatchFeatures(nil src) expected error")
	}
	if _, err := fd.MatchFeatures([]byte{1}, nil); err == nil {
		t.Fatal("MatchFeatures(nil template) expected error")
	}
	if _, err := fd.DetectKeypoints(nil); err == nil {
		t.Fatal("DetectKeypoints(nil) expected error")
	}
}
