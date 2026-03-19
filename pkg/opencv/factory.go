// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build !vision

package opencv

import (
	"digital.vasic.visionengine/pkg/analyzer"
)

// NewDiffer returns a stub Differ when OpenCV is not available.
func NewDiffer() Differ {
	return NewStubDiffer()
}

// NewElementDetector returns a stub ElementDetector when OpenCV is not available.
func NewElementDetector() ElementDetector {
	return NewStubDetector()
}

// NewColorAnalyzer returns a stub ColorAnalyzer when OpenCV is not available.
func NewColorAnalyzer() ColorAnalyzer {
	return NewStubColorAnalyzer()
}

// NewVideoProcessor returns a stub VideoProcessor when OpenCV is not available.
func NewVideoProcessor() analyzer.VideoProcessor {
	return NewStubVideoProcessor()
}

// Available reports whether real OpenCV support is compiled in.
func Available() bool {
	return false
}
