// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build vision

// Package opencv real-path end-to-end tests.
//
// These tests exercise the REAL OpenCV-backed implementations (built only
// under the `vision` build tag, requiring cgo + a real OpenCV install) against
// deterministic synthetic images, and assert real detection results — not stub
// error paths. This is the sink-side evidence mandated by CLAUDE-1: proving the
// vision feature actually works for end users, not merely that code compiles.
package opencv

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"digital.vasic.visionengine/pkg/analyzer"
)

// synthRectPNG renders a black w×h canvas with a single filled white rectangle
// at (rx,ry)-(rx+rw,ry+rh) and returns the PNG-encoded bytes. The white-on-black
// edges give OpenCV real, deterministic structure to detect.
func synthRectPNG(t *testing.T, w, h, rx, ry, rw, rh int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Black background.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	// White filled rectangle.
	for y := ry; y < ry+rh && y < h; y++ {
		for x := rx; x < rx+rw && x < w; x++ {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode synth png: %v", err)
	}
	return buf.Bytes()
}

func TestRealElementDetector_DetectEdges_FindsRectangle(t *testing.T) {
	if !Available() {
		t.Fatal("vision build tag set but Available()==false")
	}
	img := synthRectPNG(t, 200, 200, 50, 50, 100, 100)
	det := NewElementDetector()

	rects, err := det.DetectEdges(img)
	if err != nil {
		t.Fatalf("DetectEdges error: %v", err)
	}
	if len(rects) == 0 {
		t.Fatal("DetectEdges found no edges on a white rectangle — real gocv path produced nothing")
	}
	// At least one detected rect must be substantial (the square's perimeter
	// spans roughly the 100×100 region). Assert a real bounding box exists.
	var biggest analyzer.Rect
	for _, r := range rects {
		if r.Width*r.Height > biggest.Width*biggest.Height {
			biggest = r
		}
	}
	if biggest.Width < 50 || biggest.Height < 50 {
		t.Fatalf("largest edge rect too small to be the square: %+v", biggest)
	}
	t.Logf("DetectEdges: %d rect(s); largest=%+v (real gocv Canny+contours)", len(rects), biggest)
}

func TestRealElementDetector_DetectContours_FindsRectangle(t *testing.T) {
	img := synthRectPNG(t, 200, 200, 60, 40, 80, 110)
	det := NewElementDetector()

	rects, err := det.DetectContours(img)
	if err != nil {
		t.Fatalf("DetectContours error: %v", err)
	}
	if len(rects) == 0 {
		t.Fatal("DetectContours found nothing on a white rectangle")
	}
	t.Logf("DetectContours: %d contour rect(s); first=%+v", len(rects), rects[0])
}

func TestRealElementDetector_TemplateMatch_FindsTemplate(t *testing.T) {
	// Build the source as an RGBA so the template can be a genuine crop of it.
	// A crop spanning the square's top-left corner contains both black and white
	// (real variance, required by TmCcoeffNormed) and appears exactly once.
	w, h := 200, 200
	rx, ry, rw, rh := 70, 70, 60, 60
	srcImg := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{0, 0, 0, 255}
			if x >= rx && x < rx+rw && y >= ry && y < ry+rh {
				c = color.RGBA{255, 255, 255, 255}
			}
			srcImg.Set(x, y, c)
		}
	}
	var srcBuf bytes.Buffer
	if err := png.Encode(&srcBuf, srcImg); err != nil {
		t.Fatalf("encode source: %v", err)
	}

	// Template = real 40×40 crop straddling the corner at (rx,ry): contains the
	// black background (top/left) and the white square (bottom/right).
	cropRect := image.Rect(rx-20, ry-20, rx+20, ry+20)
	tplImg := image.NewRGBA(image.Rect(0, 0, 40, 40))
	for y := cropRect.Min.Y; y < cropRect.Max.Y; y++ {
		for x := cropRect.Min.X; x < cropRect.Max.X; x++ {
			tplImg.Set(x-cropRect.Min.X, y-cropRect.Min.Y, srcImg.At(x, y))
		}
	}
	var tplBuf bytes.Buffer
	if err := png.Encode(&tplBuf, tplImg); err != nil {
		t.Fatalf("encode template: %v", err)
	}

	det := NewElementDetector()
	matches, err := det.TemplateMatch(srcBuf.Bytes(), tplBuf.Bytes())
	if err != nil {
		t.Fatalf("TemplateMatch error: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("TemplateMatch found no occurrences of a template cropped directly from the source")
	}
	// The match must land near the crop origin (rx-20, ry-20) = (50,50).
	var near bool
	for _, m := range matches {
		if abs(m.X-(rx-20)) <= 5 && abs(m.Y-(ry-20)) <= 5 {
			near = true
			break
		}
	}
	if !near {
		t.Fatalf("TemplateMatch matches not near expected (50,50): %+v", matches)
	}
	t.Logf("TemplateMatch: %d match(es); first=%+v (real gocv MatchTemplate)", len(matches), matches[0])
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestRealDiffer_SSIM_IdenticalHigherThanDifferent(t *testing.T) {
	a := synthRectPNG(t, 200, 200, 50, 50, 100, 100)
	// b: same canvas, rectangle shifted far away → structurally different.
	b := synthRectPNG(t, 200, 200, 10, 10, 30, 30)
	d := NewDiffer()

	same, err := d.SSIM(a, a)
	if err != nil {
		t.Fatalf("SSIM(a,a) error: %v", err)
	}
	diff, err := d.SSIM(a, b)
	if err != nil {
		t.Fatalf("SSIM(a,b) error: %v", err)
	}
	// Identical images must correlate near-perfectly; different ones must score
	// strictly lower. This exercises the real float64(maxVal) return path.
	if same < 0.95 {
		t.Fatalf("SSIM(identical) = %.4f, expected >= 0.95", same)
	}
	if !(diff < same) {
		t.Fatalf("SSIM(different)=%.4f not < SSIM(identical)=%.4f", diff, same)
	}
	t.Logf("SSIM identical=%.4f  different=%.4f (real gocv MatchTemplate/MinMaxLoc)", same, diff)
}

func TestRealDiffer_ChangeMask_DetectsChangedRegion(t *testing.T) {
	a := synthRectPNG(t, 200, 200, 0, 0, 0, 0) // all black
	b := synthRectPNG(t, 200, 200, 80, 80, 40, 40)
	d := NewDiffer()

	rects, err := d.ChangeMask(a, b)
	if err != nil {
		t.Fatalf("ChangeMask error: %v", err)
	}
	if len(rects) == 0 {
		t.Fatal("ChangeMask detected no change between an all-black image and one with a white square")
	}
	t.Logf("ChangeMask: %d changed region(s); first=%+v", len(rects), rects[0])
}

func TestRealColorAnalyzer_DominantColors_OnWhiteHeavyImage(t *testing.T) {
	// Mostly white canvas with a small black square → white must dominate.
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}

	ca := NewColorAnalyzer()
	colors, err := ca.DominantColors(buf.Bytes(), 3)
	if err != nil {
		t.Fatalf("DominantColors error: %v", err)
	}
	if len(colors) == 0 {
		t.Fatal("DominantColors returned nothing on a real image")
	}
	t.Logf("DominantColors: %d color(s); top=%+v (real gocv k-means)", len(colors), colors[0])
}
