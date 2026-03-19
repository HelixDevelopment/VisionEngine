// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build vision

package opencv

import (
	"digital.vasic.visionengine/pkg/analyzer"
)

// NewDiffer returns a real OpenCV-backed Differ.
func NewDiffer() Differ {
	return &realDiffer{}
}

// NewElementDetector returns a real OpenCV-backed ElementDetector.
func NewElementDetector() ElementDetector {
	return &realElementDetector{}
}

// NewColorAnalyzer returns a real OpenCV-backed ColorAnalyzer.
func NewColorAnalyzer() ColorAnalyzer {
	return &realColorAnalyzer{}
}

// NewVideoProcessor returns a real OpenCV-backed VideoProcessor.
func NewVideoProcessor() analyzer.VideoProcessor {
	return &realVideoProcessor{}
}

// Available reports whether real OpenCV support is compiled in.
func Available() bool {
	return true
}
