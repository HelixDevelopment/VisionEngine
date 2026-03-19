// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package opencv

import (
	"digital.vasic.visionengine/pkg/analyzer"
)

// Differ provides image comparison capabilities.
type Differ interface {
	// SSIM computes the structural similarity index between two images.
	SSIM(img1, img2 []byte) (float64, error)
	// PixelDiff computes a pixel-level diff image between two images.
	PixelDiff(img1, img2 []byte) ([]byte, error)
	// ChangeMask detects changed regions between two images.
	ChangeMask(img1, img2 []byte) ([]analyzer.Rect, error)
}

// ElementDetector provides UI element detection capabilities.
type ElementDetector interface {
	// DetectEdges detects edges in an image and returns bounding rectangles.
	DetectEdges(img []byte) ([]analyzer.Rect, error)
	// DetectContours detects contours in an image and returns bounding rectangles.
	DetectContours(img []byte) ([]analyzer.Rect, error)
	// TemplateMatch finds occurrences of a template image within a source image.
	TemplateMatch(img, template []byte) ([]analyzer.Rect, error)
}

// ColorAnalyzer provides color analysis capabilities.
type ColorAnalyzer interface {
	// DominantColors extracts the top N dominant colors from an image.
	DominantColors(img []byte, count int) ([]Color, error)
	// ContrastRatio computes the contrast ratio within a region of an image.
	ContrastRatio(img []byte, region analyzer.Rect) (float64, error)
}
