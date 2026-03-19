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

var (
	// ErrOpenCVNotAvailable is returned when OpenCV is not installed.
	ErrOpenCVNotAvailable = errors.New("OpenCV not available: build with -tags vision")
	// ErrInvalidImage is returned when image data is invalid.
	ErrInvalidImage = errors.New("invalid image data")
	// ErrInvalidVideoPath is returned when a video path is invalid.
	ErrInvalidVideoPath = errors.New("invalid video path")
)

// StubDiffer provides stub image diffing without OpenCV.
type StubDiffer struct{}

// NewStubDiffer creates a new StubDiffer.
func NewStubDiffer() *StubDiffer {
	return &StubDiffer{}
}

// SSIM returns an error indicating OpenCV is not available.
func (d *StubDiffer) SSIM(img1, img2 []byte) (float64, error) {
	if len(img1) == 0 || len(img2) == 0 {
		return 0, ErrInvalidImage
	}
	return 0, ErrOpenCVNotAvailable
}

// PixelDiff returns an error indicating OpenCV is not available.
func (d *StubDiffer) PixelDiff(img1, img2 []byte) ([]byte, error) {
	if len(img1) == 0 || len(img2) == 0 {
		return nil, ErrInvalidImage
	}
	return nil, ErrOpenCVNotAvailable
}

// ChangeMask returns an error indicating OpenCV is not available.
func (d *StubDiffer) ChangeMask(img1, img2 []byte) ([]analyzer.Rect, error) {
	if len(img1) == 0 || len(img2) == 0 {
		return nil, ErrInvalidImage
	}
	return nil, ErrOpenCVNotAvailable
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
		return nil, ErrInvalidImage
	}
	return nil, ErrOpenCVNotAvailable
}

// DetectContours returns an error indicating OpenCV is not available.
func (d *StubDetector) DetectContours(img []byte) ([]analyzer.Rect, error) {
	if len(img) == 0 {
		return nil, ErrInvalidImage
	}
	return nil, ErrOpenCVNotAvailable
}

// TemplateMatch returns an error indicating OpenCV is not available.
func (d *StubDetector) TemplateMatch(img, template []byte) ([]analyzer.Rect, error) {
	if len(img) == 0 || len(template) == 0 {
		return nil, ErrInvalidImage
	}
	return nil, ErrOpenCVNotAvailable
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
		return nil, ErrInvalidImage
	}
	return nil, ErrOpenCVNotAvailable
}

// ContrastRatio returns an error indicating OpenCV is not available.
func (c *StubColorAnalyzer) ContrastRatio(img []byte, region analyzer.Rect) (float64, error) {
	if len(img) == 0 {
		return 0, ErrInvalidImage
	}
	return 0, ErrOpenCVNotAvailable
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
		return nil, ErrInvalidVideoPath
	}
	return nil, ErrOpenCVNotAvailable
}

// ExtractKeyFrames returns an error indicating OpenCV is not available.
func (v *StubVideoProcessor) ExtractKeyFrames(videoPath string) ([]analyzer.KeyFrame, error) {
	if videoPath == "" {
		return nil, ErrInvalidVideoPath
	}
	return nil, ErrOpenCVNotAvailable
}

// DetectSceneChanges returns an error indicating OpenCV is not available.
func (v *StubVideoProcessor) DetectSceneChanges(videoPath string) ([]time.Duration, error) {
	if videoPath == "" {
		return nil, ErrInvalidVideoPath
	}
	return nil, ErrOpenCVNotAvailable
}

// GenerateThumbnail returns an error indicating OpenCV is not available.
func (v *StubVideoProcessor) GenerateThumbnail(videoPath string, ts time.Duration, size analyzer.Size) ([]byte, error) {
	if videoPath == "" {
		return nil, ErrInvalidVideoPath
	}
	return nil, ErrOpenCVNotAvailable
}
