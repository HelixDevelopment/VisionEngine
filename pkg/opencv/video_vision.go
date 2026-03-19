// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build vision

package opencv

import (
	"fmt"
	"image"
	"math"
	"time"

	"digital.vasic.visionengine/pkg/analyzer"
	"gocv.io/x/gocv"
)

// realVideoProcessor implements video processing using GoCV.
type realVideoProcessor struct{}

// ExtractFrame extracts a single frame at the given timestamp from a video file.
func (v *realVideoProcessor) ExtractFrame(videoPath string, timestamp time.Duration) ([]byte, error) {
	if videoPath == "" {
		return nil, ErrInvalidVideoPath
	}

	cap, err := gocv.VideoCaptureFile(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open video: %w", err)
	}
	defer cap.Close()

	fps := cap.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 30 // default fallback
	}
	frameNum := int(timestamp.Seconds() * fps)
	cap.Set(gocv.VideoCapturePosFrames, float64(frameNum))

	mat := gocv.NewMat()
	defer mat.Close()

	if ok := cap.Read(&mat); !ok || mat.Empty() {
		return nil, fmt.Errorf("failed to read frame at %v", timestamp)
	}

	buf, err := gocv.IMEncode(gocv.PNGFileExt, mat)
	if err != nil {
		return nil, err
	}
	defer buf.Close()
	return buf.GetBytes(), nil
}

// ExtractKeyFrames extracts key frames from a video by detecting scene changes.
func (v *realVideoProcessor) ExtractKeyFrames(videoPath string) ([]analyzer.KeyFrame, error) {
	if videoPath == "" {
		return nil, ErrInvalidVideoPath
	}

	cap, err := gocv.VideoCaptureFile(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open video: %w", err)
	}
	defer cap.Close()

	fps := cap.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 30
	}

	var keyFrames []analyzer.KeyFrame
	prevGray := gocv.NewMat()
	defer prevGray.Close()

	mat := gocv.NewMat()
	defer mat.Close()

	gray := gocv.NewMat()
	defer gray.Close()

	diff := gocv.NewMat()
	defer diff.Close()

	frameIdx := 0
	keyFrameIdx := 0
	threshold := 30.0 // mean difference threshold for scene change

	for {
		if ok := cap.Read(&mat); !ok || mat.Empty() {
			break
		}

		gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)

		isKeyFrame := false
		if prevGray.Empty() {
			// First frame is always a key frame
			isKeyFrame = true
		} else {
			gocv.AbsDiff(gray, prevGray, &diff)
			meanVal := diff.Mean()
			if meanVal.Val1 > threshold {
				isKeyFrame = true
			}
		}

		if isKeyFrame {
			buf, encErr := gocv.IMEncode(gocv.PNGFileExt, mat)
			if encErr == nil {
				timestamp := time.Duration(float64(frameIdx)/fps*1000) * time.Millisecond
				keyFrames = append(keyFrames, analyzer.KeyFrame{
					Timestamp: timestamp,
					Data:      buf.GetBytes(),
					Index:     keyFrameIdx,
				})
				buf.Close()
				keyFrameIdx++
			}
		}

		gray.CopyTo(&prevGray)
		frameIdx++
	}

	return keyFrames, nil
}

// DetectSceneChanges detects timestamps where scene changes occur in a video.
func (v *realVideoProcessor) DetectSceneChanges(videoPath string) ([]time.Duration, error) {
	if videoPath == "" {
		return nil, ErrInvalidVideoPath
	}

	cap, err := gocv.VideoCaptureFile(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open video: %w", err)
	}
	defer cap.Close()

	fps := cap.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 30
	}

	var changes []time.Duration
	prevGray := gocv.NewMat()
	defer prevGray.Close()

	mat := gocv.NewMat()
	defer mat.Close()

	gray := gocv.NewMat()
	defer gray.Close()

	diff := gocv.NewMat()
	defer diff.Close()

	frameIdx := 0
	threshold := 30.0

	for {
		if ok := cap.Read(&mat); !ok || mat.Empty() {
			break
		}

		gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)

		if !prevGray.Empty() {
			gocv.AbsDiff(gray, prevGray, &diff)
			meanVal := diff.Mean()
			if meanVal.Val1 > threshold {
				timestamp := time.Duration(float64(frameIdx)/fps*1000) * time.Millisecond
				changes = append(changes, timestamp)
			}
		}

		gray.CopyTo(&prevGray)
		frameIdx++
	}

	return changes, nil
}

// GenerateThumbnail generates a thumbnail image at the given timestamp and size.
func (v *realVideoProcessor) GenerateThumbnail(videoPath string, ts time.Duration, size analyzer.Size) ([]byte, error) {
	if videoPath == "" {
		return nil, ErrInvalidVideoPath
	}

	cap, err := gocv.VideoCaptureFile(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open video: %w", err)
	}
	defer cap.Close()

	fps := cap.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 30
	}
	frameNum := int(math.Round(ts.Seconds() * fps))
	cap.Set(gocv.VideoCapturePosFrames, float64(frameNum))

	mat := gocv.NewMat()
	defer mat.Close()

	if ok := cap.Read(&mat); !ok || mat.Empty() {
		return nil, fmt.Errorf("failed to read frame at %v", ts)
	}

	// Resize to requested thumbnail size
	thumb := gocv.NewMat()
	defer thumb.Close()
	gocv.Resize(mat, &thumb, image.Pt(size.Width, size.Height), 0, 0, gocv.InterpolationArea)

	buf, err := gocv.IMEncode(gocv.PNGFileExt, thumb)
	if err != nil {
		return nil, err
	}
	defer buf.Close()
	return buf.GetBytes(), nil
}
