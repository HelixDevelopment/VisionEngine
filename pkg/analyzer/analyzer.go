// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrEmptyScreenshot is returned when an empty screenshot is provided.
	ErrEmptyScreenshot = errors.New("empty screenshot data")
	// ErrAnalysisFailed is returned when screen analysis fails.
	ErrAnalysisFailed = errors.New("screen analysis failed")
	// ErrComparisonFailed is returned when screen comparison fails.
	ErrComparisonFailed = errors.New("screen comparison failed")
	// ErrDetectionFailed is returned when element detection fails.
	ErrDetectionFailed = errors.New("element detection failed")
	// ErrIdentificationFailed is returned when screen identification fails.
	ErrIdentificationFailed = errors.New("screen identification failed")
)

// Analyzer is the primary vision analysis interface.
type Analyzer interface {
	// AnalyzeScreen performs comprehensive analysis of a screenshot.
	AnalyzeScreen(ctx context.Context, screenshot []byte) (ScreenAnalysis, error)
	// CompareScreens compares two screenshots and returns the differences.
	CompareScreens(ctx context.Context, before, after []byte) (ScreenDiff, error)
	// DetectElements detects UI elements in a screenshot.
	DetectElements(screenshot []byte) ([]UIElement, error)
	// DetectText detects text regions in a screenshot.
	DetectText(screenshot []byte) ([]TextRegion, error)
	// IdentifyScreen identifies the screen from a screenshot.
	IdentifyScreen(ctx context.Context, screenshot []byte) (ScreenIdentity, error)
	// DetectIssues detects visual issues in a screenshot.
	DetectIssues(ctx context.Context, screenshot []byte) ([]VisualIssue, error)
}

// VideoProcessor provides video analysis capabilities.
type VideoProcessor interface {
	// ExtractFrame extracts a single frame at the given timestamp.
	ExtractFrame(videoPath string, timestamp time.Duration) ([]byte, error)
	// ExtractKeyFrames extracts key frames from a video.
	ExtractKeyFrames(videoPath string) ([]KeyFrame, error)
	// DetectSceneChanges detects scene change timestamps in a video.
	DetectSceneChanges(videoPath string) ([]time.Duration, error)
	// GenerateThumbnail generates a thumbnail at the given timestamp and size.
	GenerateThumbnail(videoPath string, ts time.Duration, size Size) ([]byte, error)
}
