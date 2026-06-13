// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package opencv provides OpenCV-based vision capabilities.
// The real implementation requires the "vision" build tag and OpenCV 4.x.
// This file provides stub implementations that work without OpenCV.
package opencv

import (
	"errors"
	"time"

	"digital.vasic.visionengine/pkg/analyzer"
)

// opencv stub sentinel errors. Per CONST-046 the literal text on each
// sentinel is only the bundled English fallback (NoopTranslator path);
// user-facing surfacing routes through the errOpenCV* helpers below,
// which preserve errors.Is matchability while emitting a locale-
// appropriate message when a translator is wired via SetPkgTranslator.
var (
	// ErrOpenCVNotAvailable is returned when OpenCV is not installed.
	ErrOpenCVNotAvailable = errors.New("OpenCV not available: build with -tags vision")
	// ErrInvalidImage is returned when image data is invalid.
	ErrInvalidImage = errors.New("invalid image data")
	// ErrInvalidVideoPath is returned when a video path is invalid.
	ErrInvalidVideoPath = errors.New("invalid video path")
	// ErrNoFeatures is returned when no ORB features can be detected in an image.
	ErrNoFeatures = errors.New("no features detected in image")
)

// errOpenCVNotAvailable returns a translator-routed error that unwraps
// to ErrOpenCVNotAvailable (CONST-046).
func errOpenCVNotAvailable() error {
	return localizedError(ErrOpenCVNotAvailable,
		"visionengine_opencv_not_available", fallbackOpenCVNotAvailable)
}

// errInvalidImage returns a translator-routed error that unwraps to
// ErrInvalidImage (CONST-046).
func errInvalidImage() error {
	return localizedError(ErrInvalidImage,
		"visionengine_opencv_invalid_image", fallbackOpenCVInvalidImage)
}

// errInvalidVideoPath returns a translator-routed error that unwraps to
// ErrInvalidVideoPath (CONST-046).
func errInvalidVideoPath() error {
	return localizedError(ErrInvalidVideoPath,
		"visionengine_opencv_invalid_video_path", fallbackOpenCVInvalidVideoPath)
}

// errNoFeatures returns a translator-routed error that unwraps to
// ErrNoFeatures (CONST-046).
func errNoFeatures() error {
	return localizedError(ErrNoFeatures,
		"visionengine_opencv_no_features", fallbackOpenCVNoFeatures)
}

// StubDiffer provides stub image diffing without OpenCV.
type StubDiffer struct{}

// NewStubDiffer creates a new StubDiffer.
func NewStubDiffer() *StubDiffer {
	return &StubDiffer{}
}

// SSIM returns an error indicating OpenCV is not available.
func (d *StubDiffer) SSIM(img1, img2 []byte) (float64, error) {
	if len(img1) == 0 || len(img2) == 0 {
		return 0, errInvalidImage()
	}
	return 0, errOpenCVNotAvailable()
}

// PixelDiff returns an error indicating OpenCV is not available.
func (d *StubDiffer) PixelDiff(img1, img2 []byte) ([]byte, error) {
	if len(img1) == 0 || len(img2) == 0 {
		return nil, errInvalidImage()
	}
	return nil, errOpenCVNotAvailable()
}

// ChangeMask returns an error indicating OpenCV is not available.
func (d *StubDiffer) ChangeMask(img1, img2 []byte) ([]analyzer.Rect, error) {
	if len(img1) == 0 || len(img2) == 0 {
		return nil, errInvalidImage()
	}
	return nil, errOpenCVNotAvailable()
}

// StubDetector provides stub element detection without OpenCV.
type StubDetector struct{}

// NewStubDetector creates a new StubDetector.
func NewStubDetector() *StubDetector {
	return &StubDetector{}
}

// DetectEdges returns an error indicating OpenCV is not available.
func (d *StubDetector) DetectEdges(img []byte) ([]analyzer.Rect, error) {
	if len(img) == 0 {
		return nil, errInvalidImage()
	}
	return nil, errOpenCVNotAvailable()
}

// DetectContours returns an error indicating OpenCV is not available.
func (d *StubDetector) DetectContours(img []byte) ([]analyzer.Rect, error) {
	if len(img) == 0 {
		return nil, errInvalidImage()
	}
	return nil, errOpenCVNotAvailable()
}

// TemplateMatch returns an error indicating OpenCV is not available.
func (d *StubDetector) TemplateMatch(img, template []byte) ([]analyzer.Rect, error) {
	if len(img) == 0 || len(template) == 0 {
		return nil, errInvalidImage()
	}
	return nil, errOpenCVNotAvailable()
}

// StubColorAnalyzer provides stub color analysis without OpenCV.
type StubColorAnalyzer struct{}

// NewStubColorAnalyzer creates a new StubColorAnalyzer.
func NewStubColorAnalyzer() *StubColorAnalyzer {
	return &StubColorAnalyzer{}
}

// DominantColors returns an error indicating OpenCV is not available.
func (c *StubColorAnalyzer) DominantColors(img []byte, count int) ([]Color, error) {
	if len(img) == 0 {
		return nil, errInvalidImage()
	}
	return nil, errOpenCVNotAvailable()
}

// ContrastRatio returns an error indicating OpenCV is not available.
func (c *StubColorAnalyzer) ContrastRatio(img []byte, region analyzer.Rect) (float64, error) {
	if len(img) == 0 {
		return 0, errInvalidImage()
	}
	return 0, errOpenCVNotAvailable()
}

// StubFeatureDetector provides stub ORB feature detection without OpenCV.
type StubFeatureDetector struct{}

// NewStubFeatureDetector creates a new StubFeatureDetector.
func NewStubFeatureDetector() *StubFeatureDetector {
	return &StubFeatureDetector{}
}

// DetectKeypoints returns an error indicating OpenCV is not available.
func (f *StubFeatureDetector) DetectKeypoints(img []byte) ([]Keypoint, error) {
	if len(img) == 0 {
		return nil, errInvalidImage()
	}
	return nil, errOpenCVNotAvailable()
}

// MatchFeatures returns an error indicating OpenCV is not available.
func (f *StubFeatureDetector) MatchFeatures(img, template []byte) (int, error) {
	if len(img) == 0 || len(template) == 0 {
		return 0, errInvalidImage()
	}
	return 0, errOpenCVNotAvailable()
}

// Color represents an RGB color.
type Color struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// StubVideoProcessor provides stub video processing without OpenCV.
type StubVideoProcessor struct{}

// NewStubVideoProcessor creates a new StubVideoProcessor.
func NewStubVideoProcessor() *StubVideoProcessor {
	return &StubVideoProcessor{}
}

// ExtractFrame returns an error indicating OpenCV is not available.
func (v *StubVideoProcessor) ExtractFrame(videoPath string, timestamp time.Duration) ([]byte, error) {
	if videoPath == "" {
		return nil, errInvalidVideoPath()
	}
	return nil, errOpenCVNotAvailable()
}

// ExtractKeyFrames returns an error indicating OpenCV is not available.
func (v *StubVideoProcessor) ExtractKeyFrames(videoPath string) ([]analyzer.KeyFrame, error) {
	if videoPath == "" {
		return nil, errInvalidVideoPath()
	}
	return nil, errOpenCVNotAvailable()
}

// DetectSceneChanges returns an error indicating OpenCV is not available.
func (v *StubVideoProcessor) DetectSceneChanges(videoPath string) ([]time.Duration, error) {
	if videoPath == "" {
		return nil, errInvalidVideoPath()
	}
	return nil, errOpenCVNotAvailable()
}

// GenerateThumbnail returns an error indicating OpenCV is not available.
func (v *StubVideoProcessor) GenerateThumbnail(videoPath string, ts time.Duration, size analyzer.Size) ([]byte, error) {
	if videoPath == "" {
		return nil, errInvalidVideoPath()
	}
	return nil, errOpenCVNotAvailable()
}
