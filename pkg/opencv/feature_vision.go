// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build vision

package opencv

import (
	"gocv.io/x/gocv"
)

// realFeatureDetector implements ORB (Oriented FAST + Rotated BRIEF) keypoint
// detection and descriptor matching using GoCV. ORB descriptors are binary, so
// matching uses a brute-force matcher with Hamming distance plus Lowe's ratio
// test to keep only discriminative matches.
type realFeatureDetector struct{}

// loweRatio is David Lowe's classic ratio-test threshold: a candidate match is
// "good" only if its best descriptor distance is < loweRatio × the second-best.
// This rejects ambiguous matches that pair equally well with multiple keypoints.
const loweRatio = 0.75

// strongHammingDistance is a tight absolute Hamming bound (out of 256 bits) below
// which a best-match is accepted even if the Lowe ratio test rejects it. On
// self-similar / repetitive imagery the second-best neighbor can be almost as
// close as the best (defeating the ratio test) while the best is still a genuine
// near-exact descriptor match — a documented ratio-test failure mode. Accepting
// near-exact best-matches (distance well under a third of the descriptor width)
// recovers those true positives without admitting noise, since random ORB
// descriptors sit near 128 bits of Hamming distance apart.
const strongHammingDistance = 24

// DetectKeypoints extracts ORB keypoints from an image. The returned slice is
// ordered by detector response (strongest first).
func (d *realFeatureDetector) DetectKeypoints(img []byte) ([]Keypoint, error) {
	if len(img) == 0 {
		return nil, ErrInvalidImage
	}

	mat, err := gocv.IMDecode(img, gocv.IMReadGrayScale)
	if err != nil {
		return nil, err
	}
	defer mat.Close()
	if mat.Empty() {
		return nil, ErrInvalidImage
	}

	orb := gocv.NewORB()
	defer orb.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	kps, desc := orb.DetectAndCompute(mat, mask)
	defer desc.Close()

	out := make([]Keypoint, 0, len(kps))
	for _, kp := range kps {
		out = append(out, Keypoint{
			X:        kp.X,
			Y:        kp.Y,
			Size:     kp.Size,
			Angle:    kp.Angle,
			Response: kp.Response,
			Octave:   kp.Octave,
		})
	}
	// Strongest-response first so callers can take the top-N most reliable points.
	sortKeypointsByResponseDesc(out)
	return out, nil
}

// MatchFeatures detects ORB keypoints + descriptors in both the source and the
// template, brute-force KNN-matches the template's descriptors against the
// source's (k=2), and returns the count of matches surviving Lowe's ratio test.
// A self-crop of the source yields many good matches; an unrelated image yields
// far fewer — which is what makes this a real locator rather than a constant.
func (d *realFeatureDetector) MatchFeatures(img, template []byte) (int, error) {
	if len(img) == 0 || len(template) == 0 {
		return 0, ErrInvalidImage
	}

	srcMat, err := gocv.IMDecode(img, gocv.IMReadGrayScale)
	if err != nil {
		return 0, err
	}
	defer srcMat.Close()
	if srcMat.Empty() {
		return 0, ErrInvalidImage
	}

	tplMat, err := gocv.IMDecode(template, gocv.IMReadGrayScale)
	if err != nil {
		return 0, err
	}
	defer tplMat.Close()
	if tplMat.Empty() {
		return 0, ErrInvalidImage
	}

	orb := gocv.NewORB()
	defer orb.Close()

	srcMask := gocv.NewMat()
	defer srcMask.Close()
	tplMask := gocv.NewMat()
	defer tplMask.Close()

	srcKps, srcDesc := orb.DetectAndCompute(srcMat, srcMask)
	defer srcDesc.Close()
	tplKps, tplDesc := orb.DetectAndCompute(tplMat, tplMask)
	defer tplDesc.Close()

	// No descriptors on either side → no possible feature match. A featureless
	// (flat) template/source is a real, honest "no features" condition.
	if srcDesc.Empty() || tplDesc.Empty() || len(srcKps) == 0 || len(tplKps) == 0 {
		return 0, errNoFeatures()
	}

	// Hamming distance is the correct metric for ORB's binary descriptors.
	matcher := gocv.NewBFMatcherWithParams(gocv.NormHamming, false)
	defer matcher.Close()

	// Query = template descriptors, Train = source descriptors. For each template
	// descriptor get its 2 nearest source descriptors, then apply the ratio test.
	knn := matcher.KnnMatch(tplDesc, srcDesc, 2)

	good := 0
	for _, m := range knn {
		if len(m) == 0 {
			continue
		}
		best := m[0]
		switch {
		case len(m) >= 2:
			// Primary discriminator: Lowe's ratio test. Fallback: accept a
			// near-exact best-match the ratio test rejects only because the
			// second-best is also close (repetitive/self-similar texture).
			if best.Distance < loweRatio*m[1].Distance || best.Distance <= strongHammingDistance {
				good++
			}
		default: // exactly one neighbor (tiny train set)
			if best.Distance <= strongHammingDistance {
				good++
			}
		}
	}
	return good, nil
}

// sortKeypointsByResponseDesc orders keypoints strongest-response first using a
// simple insertion sort (keypoint counts are small; avoids a sort import for
// this single call site).
func sortKeypointsByResponseDesc(kps []Keypoint) {
	for i := 1; i < len(kps); i++ {
		j := i
		for j > 0 && kps[j-1].Response < kps[j].Response {
			kps[j-1], kps[j] = kps[j], kps[j-1]
			j--
		}
	}
}
