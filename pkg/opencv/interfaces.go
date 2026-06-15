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

// Keypoint is a single ORB (Oriented FAST + Rotated BRIEF) salient point
// detected in an image. It carries the sub-pixel location and the descriptor
// metadata OpenCV reports for that point. It is a build-tag-independent value
// type so callers (and the !vision stub) need not import gocv.
type Keypoint struct {
	// X, Y are the sub-pixel coordinates of the keypoint in the source image.
	X float64 `json:"x"`
	Y float64 `json:"y"`
	// Size is the diameter of the meaningful keypoint neighborhood.
	Size float64 `json:"size"`
	// Angle is the keypoint orientation in degrees ([0,360), -1 if N/A).
	Angle float64 `json:"angle"`
	// Response is the strength of the keypoint (used to rank/select the best).
	Response float64 `json:"response"`
	// Octave is the pyramid octave the keypoint was extracted from.
	Octave int `json:"octave"`
}

// FeatureDetector provides ORB feature-detection capabilities: keypoint
// extraction and descriptor matching. Unlike TemplateMatch (which only finds
// pixel-exact occurrences under translation), ORB matching locates a template
// within a source under translation plus mild scale and rotation, which is the
// robust way to locate a UI element/template.
type FeatureDetector interface {
	// DetectKeypoints extracts ORB keypoints from an image. The returned slice
	// is ordered by detector response (strongest first) and may be empty for a
	// featureless image (e.g. a flat fill).
	DetectKeypoints(img []byte) ([]Keypoint, error)
	// MatchFeatures detects ORB keypoints + descriptors in both images and
	// returns the number of "good" matches surviving Lowe's ratio test. A higher
	// count means the template is more likely present in the source.
	MatchFeatures(img, template []byte) (int, error)
}
