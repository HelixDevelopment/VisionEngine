// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build vision

package opencv

import (
	"math"

	"digital.vasic.visionengine/pkg/analyzer"
	"gocv.io/x/gocv"
)

// realColorAnalyzer implements color analysis using GoCV.
type realColorAnalyzer struct{}

// DominantColors extracts the top N dominant colors from an image using
// pixel sampling and color quantization.
func (a *realColorAnalyzer) DominantColors(img []byte, count int) ([]Color, error) {
	if len(img) == 0 {
		return nil, ErrInvalidImage
	}

	mat, err := gocv.IMDecode(img, gocv.IMReadColor)
	if err != nil {
		return nil, err
	}
	defer mat.Close()

	rows := mat.Rows()
	cols := mat.Cols()
	totalPixels := rows * cols
	step := int(math.Max(1, float64(totalPixels)/1000))

	// Sample pixels and quantize colors
	colorMap := make(map[Color]int)
	for i := 0; i < totalPixels; i += step {
		r := i / cols
		c := i % cols
		if r < rows && c < cols {
			vec := mat.GetVecbAt(r, c)
			// Quantize to 16 levels per channel to reduce unique colors
			quantized := Color{
				R: vec[2] & 0xF0,
				G: vec[1] & 0xF0,
				B: vec[0] & 0xF0,
			}
			colorMap[quantized]++
		}
	}

	// Sort by frequency using selection sort for top N
	type colorCount struct {
		c Color
		n int
	}
	sorted := make([]colorCount, 0, len(colorMap))
	for c, n := range colorMap {
		sorted = append(sorted, colorCount{c, n})
	}
	for i := 0; i < len(sorted) && i < count; i++ {
		maxIdx := i
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].n > sorted[maxIdx].n {
				maxIdx = j
			}
		}
		sorted[i], sorted[maxIdx] = sorted[maxIdx], sorted[i]
	}

	result := make([]Color, 0, count)
	for i := 0; i < len(sorted) && i < count; i++ {
		result = append(result, sorted[i].c)
	}
	return result, nil
}

// ContrastRatio computes the contrast ratio between the foreground and
// background luminance within a region of an image. It splits pixels into
// light and dark groups and computes the WCAG contrast ratio.
func (a *realColorAnalyzer) ContrastRatio(img []byte, region analyzer.Rect) (float64, error) {
	if len(img) == 0 {
		return 0, ErrInvalidImage
	}

	mat, err := gocv.IMDecode(img, gocv.IMReadColor)
	if err != nil {
		return 0, err
	}
	defer mat.Close()

	rows := mat.Rows()
	cols := mat.Cols()

	// Clamp region to image bounds
	x1 := clampInt(region.X, 0, cols)
	y1 := clampInt(region.Y, 0, rows)
	x2 := clampInt(region.X+region.Width, 0, cols)
	y2 := clampInt(region.Y+region.Height, 0, rows)

	if x2 <= x1 || y2 <= y1 {
		return 0, ErrInvalidImage
	}

	// Collect luminance values for all pixels in the region
	var lightSum, darkSum float64
	var lightCount, darkCount int
	midpoint := 0.5

	for r := y1; r < y2; r++ {
		for c := x1; c < x2; c++ {
			vec := mat.GetVecbAt(r, c)
			lum := relativeLuminance(vec[2], vec[1], vec[0])
			if lum >= midpoint {
				lightSum += lum
				lightCount++
			} else {
				darkSum += lum
				darkCount++
			}
		}
	}

	if lightCount == 0 || darkCount == 0 {
		return 1.0, nil // uniform region
	}

	lightAvg := lightSum / float64(lightCount)
	darkAvg := darkSum / float64(darkCount)

	// WCAG contrast ratio formula
	return (lightAvg + 0.05) / (darkAvg + 0.05), nil
}

// relativeLuminance computes relative luminance per WCAG 2.x from BGR values.
func relativeLuminance(r, g, b uint8) float64 {
	rLin := linearize(float64(r) / 255.0)
	gLin := linearize(float64(g) / 255.0)
	bLin := linearize(float64(b) / 255.0)
	return 0.2126*rLin + 0.7152*gLin + 0.0722*bLin
}

// linearize converts a sRGB channel value to linear light.
func linearize(v float64) float64 {
	if v <= 0.03928 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// clampInt clamps v to [min, max].
func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
