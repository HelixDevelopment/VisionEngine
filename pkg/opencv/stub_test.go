// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package opencv

import (
	"testing"
	"time"

	"digital.vasic.visionengine/pkg/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStubDiffer_SSIM_ReturnsOpenCVError(t *testing.T) {
	d := NewStubDiffer()
	_, err := d.SSIM([]byte("img1"), []byte("img2"))
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubDiffer_SSIM_EmptyImage(t *testing.T) {
	d := NewStubDiffer()
	_, err := d.SSIM([]byte{}, []byte("img2"))
	assert.ErrorIs(t, err, ErrInvalidImage)

	_, err = d.SSIM([]byte("img1"), []byte{})
	assert.ErrorIs(t, err, ErrInvalidImage)
}

func TestStubDiffer_PixelDiff_ReturnsOpenCVError(t *testing.T) {
	d := NewStubDiffer()
	_, err := d.PixelDiff([]byte("img1"), []byte("img2"))
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubDiffer_PixelDiff_EmptyImage(t *testing.T) {
	d := NewStubDiffer()
	_, err := d.PixelDiff([]byte{}, []byte("img2"))
	assert.ErrorIs(t, err, ErrInvalidImage)
}

func TestStubDiffer_ChangeMask_ReturnsOpenCVError(t *testing.T) {
	d := NewStubDiffer()
	_, err := d.ChangeMask([]byte("img1"), []byte("img2"))
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubDiffer_ChangeMask_EmptyImage(t *testing.T) {
	d := NewStubDiffer()
	_, err := d.ChangeMask([]byte{}, []byte("img2"))
	assert.ErrorIs(t, err, ErrInvalidImage)
}

func TestStubDetector_DetectEdges_ReturnsOpenCVError(t *testing.T) {
	d := NewStubDetector()
	_, err := d.DetectEdges([]byte("img"))
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubDetector_DetectEdges_EmptyImage(t *testing.T) {
	d := NewStubDetector()
	_, err := d.DetectEdges([]byte{})
	assert.ErrorIs(t, err, ErrInvalidImage)
}

func TestStubDetector_DetectContours_ReturnsOpenCVError(t *testing.T) {
	d := NewStubDetector()
	_, err := d.DetectContours([]byte("img"))
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubDetector_DetectContours_EmptyImage(t *testing.T) {
	d := NewStubDetector()
	_, err := d.DetectContours([]byte{})
	assert.ErrorIs(t, err, ErrInvalidImage)
}

func TestStubDetector_TemplateMatch_ReturnsOpenCVError(t *testing.T) {
	d := NewStubDetector()
	_, err := d.TemplateMatch([]byte("img"), []byte("tpl"))
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubDetector_TemplateMatch_EmptyImage(t *testing.T) {
	d := NewStubDetector()
	_, err := d.TemplateMatch([]byte{}, []byte("tpl"))
	assert.ErrorIs(t, err, ErrInvalidImage)

	_, err = d.TemplateMatch([]byte("img"), []byte{})
	assert.ErrorIs(t, err, ErrInvalidImage)
}

func TestStubColorAnalyzer_DominantColors_ReturnsOpenCVError(t *testing.T) {
	c := NewStubColorAnalyzer()
	_, err := c.DominantColors([]byte("img"), 5)
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubColorAnalyzer_DominantColors_EmptyImage(t *testing.T) {
	c := NewStubColorAnalyzer()
	_, err := c.DominantColors([]byte{}, 5)
	assert.ErrorIs(t, err, ErrInvalidImage)
}

func TestStubColorAnalyzer_ContrastRatio_ReturnsOpenCVError(t *testing.T) {
	c := NewStubColorAnalyzer()
	_, err := c.ContrastRatio([]byte("img"), analyzer.Rect{X: 0, Y: 0, Width: 10, Height: 10})
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubColorAnalyzer_ContrastRatio_EmptyImage(t *testing.T) {
	c := NewStubColorAnalyzer()
	_, err := c.ContrastRatio([]byte{}, analyzer.Rect{X: 0, Y: 0, Width: 10, Height: 10})
	assert.ErrorIs(t, err, ErrInvalidImage)
}

func TestStubVideoProcessor_ExtractFrame_ReturnsOpenCVError(t *testing.T) {
	v := NewStubVideoProcessor()
	_, err := v.ExtractFrame("/path/to/video.mp4", 5*time.Second)
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubVideoProcessor_ExtractFrame_EmptyPath(t *testing.T) {
	v := NewStubVideoProcessor()
	_, err := v.ExtractFrame("", 5*time.Second)
	assert.ErrorIs(t, err, ErrInvalidVideoPath)
}

func TestStubVideoProcessor_ExtractKeyFrames_ReturnsOpenCVError(t *testing.T) {
	v := NewStubVideoProcessor()
	_, err := v.ExtractKeyFrames("/path/to/video.mp4")
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubVideoProcessor_ExtractKeyFrames_EmptyPath(t *testing.T) {
	v := NewStubVideoProcessor()
	_, err := v.ExtractKeyFrames("")
	assert.ErrorIs(t, err, ErrInvalidVideoPath)
}

func TestStubVideoProcessor_DetectSceneChanges_ReturnsOpenCVError(t *testing.T) {
	v := NewStubVideoProcessor()
	_, err := v.DetectSceneChanges("/path/to/video.mp4")
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubVideoProcessor_DetectSceneChanges_EmptyPath(t *testing.T) {
	v := NewStubVideoProcessor()
	_, err := v.DetectSceneChanges("")
	assert.ErrorIs(t, err, ErrInvalidVideoPath)
}

func TestStubVideoProcessor_GenerateThumbnail_ReturnsOpenCVError(t *testing.T) {
	v := NewStubVideoProcessor()
	_, err := v.GenerateThumbnail("/path/to/video.mp4", 5*time.Second, analyzer.Size{Width: 320, Height: 240})
	assert.ErrorIs(t, err, ErrOpenCVNotAvailable)
}

func TestStubVideoProcessor_GenerateThumbnail_EmptyPath(t *testing.T) {
	v := NewStubVideoProcessor()
	_, err := v.GenerateThumbnail("", 5*time.Second, analyzer.Size{Width: 320, Height: 240})
	assert.ErrorIs(t, err, ErrInvalidVideoPath)
}

func TestColor_Fields(t *testing.T) {
	c := Color{R: 255, G: 128, B: 0}
	assert.Equal(t, uint8(255), c.R)
	assert.Equal(t, uint8(128), c.G)
	assert.Equal(t, uint8(0), c.B)
}

func TestNewStubDiffer_NotNil(t *testing.T) {
	d := NewStubDiffer()
	require.NotNil(t, d)
}

func TestNewStubDetector_NotNil(t *testing.T) {
	d := NewStubDetector()
	require.NotNil(t, d)
}

func TestNewStubColorAnalyzer_NotNil(t *testing.T) {
	c := NewStubColorAnalyzer()
	require.NotNil(t, c)
}

func TestNewStubVideoProcessor_NotNil(t *testing.T) {
	v := NewStubVideoProcessor()
	require.NotNil(t, v)
}
