// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build vision

package opencv

import (
	"digital.vasic.visionengine/pkg/analyzer"
	"gocv.io/x/gocv"
)

// realDiffer implements screenshot diffing using GoCV.
type realDiffer struct{}

// SSIM computes the structural similarity index between two images using
// template matching as a correlation approximation.
func (d *realDiffer) SSIM(img1, img2 []byte) (float64, error) {
	if len(img1) == 0 || len(img2) == 0 {
		return 0, ErrInvalidImage
	}

	mat1, err := gocv.IMDecode(img1, gocv.IMReadColor)
	if err != nil {
		return 0, err
	}
	defer mat1.Close()

	mat2, err := gocv.IMDecode(img2, gocv.IMReadColor)
	if err != nil {
		return 0, err
	}
	defer mat2.Close()

	// Convert to grayscale for SSIM computation
	gray1 := gocv.NewMat()
	defer gray1.Close()
	gocv.CvtColor(mat1, &gray1, gocv.ColorBGRToGray)

	gray2 := gocv.NewMat()
	defer gray2.Close()
	gocv.CvtColor(mat2, &gray2, gocv.ColorBGRToGray)

	// Compute SSIM using normalized cross-correlation as approximation
	result := gocv.NewMat()
	defer result.Close()
	mask := gocv.NewMat()
	defer mask.Close()
	gocv.MatchTemplate(gray1, gray2, &result, gocv.TmCcoeffNormed, mask)

	_, maxVal, _, _ := gocv.MinMaxLoc(result)
	return maxVal, nil
}

// PixelDiff computes a pixel-level diff image between two images.
func (d *realDiffer) PixelDiff(img1, img2 []byte) ([]byte, error) {
	if len(img1) == 0 || len(img2) == 0 {
		return nil, ErrInvalidImage
	}

	mat1, err := gocv.IMDecode(img1, gocv.IMReadColor)
	if err != nil {
		return nil, err
	}
	defer mat1.Close()

	mat2, err := gocv.IMDecode(img2, gocv.IMReadColor)
	if err != nil {
		return nil, err
	}
	defer mat2.Close()

	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(mat1, mat2, &diff)

	buf, err := gocv.IMEncode(gocv.PNGFileExt, diff)
	if err != nil {
		return nil, err
	}
	defer buf.Close()
	return buf.GetBytes(), nil
}

// ChangeMask detects changed regions between two images using absolute
// difference, thresholding, and contour detection.
func (d *realDiffer) ChangeMask(img1, img2 []byte) ([]analyzer.Rect, error) {
	if len(img1) == 0 || len(img2) == 0 {
		return nil, ErrInvalidImage
	}

	mat1, err := gocv.IMDecode(img1, gocv.IMReadColor)
	if err != nil {
		return nil, err
	}
	defer mat1.Close()

	mat2, err := gocv.IMDecode(img2, gocv.IMReadColor)
	if err != nil {
		return nil, err
	}
	defer mat2.Close()

	// Compute absolute difference
	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(mat1, mat2, &diff)

	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(diff, &gray, gocv.ColorBGRToGray)

	// Threshold to get binary mask
	binary := gocv.NewMat()
	defer binary.Close()
	gocv.Threshold(gray, &binary, 30, 255, gocv.ThresholdBinary)

	// Find contours of changed regions
	contours := gocv.FindContours(binary, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	var rects []analyzer.Rect
	for i := 0; i < contours.Size(); i++ {
		r := gocv.BoundingRect(contours.At(i))
		if r.Dx() > 5 && r.Dy() > 5 { // filter noise
			rects = append(rects, analyzer.Rect{
				X:      r.Min.X,
				Y:      r.Min.Y,
				Width:  r.Dx(),
				Height: r.Dy(),
			})
		}
	}
	return rects, nil
}
