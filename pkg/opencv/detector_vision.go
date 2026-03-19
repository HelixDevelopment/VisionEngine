// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build vision

package opencv

import (
	"image"

	"digital.vasic.visionengine/pkg/analyzer"
	"gocv.io/x/gocv"
)

// realElementDetector implements element detection using GoCV edge and contour analysis.
type realElementDetector struct{}

// DetectEdges detects edges in an image using Canny edge detection and returns
// bounding rectangles of edge clusters.
func (d *realElementDetector) DetectEdges(img []byte) ([]analyzer.Rect, error) {
	if len(img) == 0 {
		return nil, ErrInvalidImage
	}

	mat, err := gocv.IMDecode(img, gocv.IMReadGrayscale)
	if err != nil {
		return nil, err
	}
	defer mat.Close()

	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(mat, &edges, 50, 150)

	// Find contours of edge groups
	contours := gocv.FindContours(edges, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	var rects []analyzer.Rect
	for i := 0; i < contours.Size(); i++ {
		r := gocv.BoundingRect(contours.At(i))
		if r.Dx() > 3 && r.Dy() > 3 { // filter noise
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

// DetectContours detects contours in an image and returns bounding rectangles.
func (d *realElementDetector) DetectContours(img []byte) ([]analyzer.Rect, error) {
	if len(img) == 0 {
		return nil, ErrInvalidImage
	}

	mat, err := gocv.IMDecode(img, gocv.IMReadGrayscale)
	if err != nil {
		return nil, err
	}
	defer mat.Close()

	// Apply Gaussian blur to reduce noise
	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(mat, &blurred, image.Pt(5, 5), 0, 0, gocv.BorderDefault)

	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(blurred, &edges, 50, 150)

	contours := gocv.FindContours(edges, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	var rects []analyzer.Rect
	for i := 0; i < contours.Size(); i++ {
		r := gocv.BoundingRect(contours.At(i))
		if r.Dx() > 5 && r.Dy() > 5 { // filter small noise contours
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

// TemplateMatch finds occurrences of a template image within a source image.
func (d *realElementDetector) TemplateMatch(img, template []byte) ([]analyzer.Rect, error) {
	if len(img) == 0 || len(template) == 0 {
		return nil, ErrInvalidImage
	}

	mat, err := gocv.IMDecode(img, gocv.IMReadGrayscale)
	if err != nil {
		return nil, err
	}
	defer mat.Close()

	tpl, err := gocv.IMDecode(template, gocv.IMReadGrayscale)
	if err != nil {
		return nil, err
	}
	defer tpl.Close()

	result := gocv.NewMat()
	defer result.Close()
	mask := gocv.NewMat()
	defer mask.Close()
	gocv.MatchTemplate(mat, tpl, &result, gocv.TmCcoeffNormed, mask)

	// Find locations above threshold
	threshold := 0.8
	var rects []analyzer.Rect
	rows := result.Rows()
	cols := result.Cols()
	tplW := tpl.Cols()
	tplH := tpl.Rows()

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			val := result.GetFloatAt(r, c)
			if float64(val) >= threshold {
				rects = append(rects, analyzer.Rect{
					X:      c,
					Y:      r,
					Width:  tplW,
					Height: tplH,
				})
			}
		}
	}
	return rects, nil
}
