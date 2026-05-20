// Copyright 2026 HelixDevelopment. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package opencv

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeTranslator is a unit-test-only stub. Per CONST-050(A), mocks are
// permitted inside *_test.go (this file). It returns a sentinel-wrapped
// form so call sites can be proven to route through the Translator seam
// — a hardcoded literal regression would NOT contain the "<TRANSLATED:"
// sentinel and the assertions below would fail.
type fakeTranslator struct {
	seen []string
}

func (f *fakeTranslator) T(_ context.Context, msgID string, _ ...any) string {
	f.seen = append(f.seen, msgID)
	return "<TRANSLATED:" + msgID + ">"
}

// withFakeTranslator wires the fakeTranslator at the package level for
// the duration of t and restores the NoopTranslator default afterwards.
func withFakeTranslator(t *testing.T) *fakeTranslator {
	t.Helper()
	tr := &fakeTranslator{}
	SetPkgTranslator(tr)
	t.Cleanup(func() { SetPkgTranslator(nil) })
	return tr
}

// TestStub_OpenCVErrors_RouteThroughTranslator drives every opencv stub
// error path through the Translator seam. round-414 §11.4 CONST-046
// Phase 4: ErrOpenCVNotAvailable / ErrInvalidImage / ErrInvalidVideoPath
// were previously surfaced as raw sentinel literals; they now route via
// the err* helpers. Anti-bluff: the "<TRANSLATED:" sentinel is the
// proof — a regression that returns the bare sentinel will not produce
// it and the assertion fails.
func TestStub_OpenCVErrors_RouteThroughTranslator(t *testing.T) {
	withFakeTranslator(t)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			"differ_invalid_image",
			func() error { _, e := NewStubDiffer().SSIM(nil, nil); return e },
			"<TRANSLATED:visionengine_opencv_invalid_image>",
		},
		{
			"differ_not_available",
			func() error { _, e := NewStubDiffer().SSIM([]byte{1}, []byte{1}); return e },
			"<TRANSLATED:visionengine_opencv_not_available>",
		},
		{
			"detector_not_available",
			func() error { _, e := NewStubDetector().DetectEdges([]byte{1}); return e },
			"<TRANSLATED:visionengine_opencv_not_available>",
		},
		{
			"video_invalid_path",
			func() error { _, e := NewStubVideoProcessor().ExtractFrame("", 0); return e },
			"<TRANSLATED:visionengine_opencv_invalid_video_path>",
		},
		{
			"video_not_available",
			func() error { _, e := NewStubVideoProcessor().ExtractFrame("/x.mp4", 0); return e },
			"<TRANSLATED:visionengine_opencv_not_available>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil {
				t.Fatalf("%s returned nil; expected an error", tt.name)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("%s err=%q; expected sentinel %q (call site bypassed Translator → CONST-046 violation)", tt.name, err.Error(), tt.want)
			}
		})
	}
}

// TestStub_OpenCVErrors_PreserveErrorsIs proves the localized errors
// remain errors.Is-compatible with the underlying sentinels, so
// consumers' existing match logic keeps working after the migration.
func TestStub_OpenCVErrors_PreserveErrorsIs(t *testing.T) {
	withFakeTranslator(t)

	_, errImg := NewStubDiffer().SSIM(nil, nil)
	if !errors.Is(errImg, ErrInvalidImage) {
		t.Fatalf("localized invalid-image error does not unwrap to ErrInvalidImage")
	}
	_, errCV := NewStubDiffer().SSIM([]byte{1}, []byte{1})
	if !errors.Is(errCV, ErrOpenCVNotAvailable) {
		t.Fatalf("localized not-available error does not unwrap to ErrOpenCVNotAvailable")
	}
	_, errVid := NewStubVideoProcessor().ExtractFrame("", 0)
	if !errors.Is(errVid, ErrInvalidVideoPath) {
		t.Fatalf("localized invalid-video-path error does not unwrap to ErrInvalidVideoPath")
	}
}

// TestStub_OpenCVErrors_NoTranslator_EnglishFallback documents the
// standalone path: with the NoopTranslator default, each error path
// falls back to the bundled English literal.
func TestStub_OpenCVErrors_NoTranslator_EnglishFallback(t *testing.T) {
	SetPkgTranslator(nil)

	_, err := NewStubDiffer().SSIM([]byte{1}, []byte{1})
	if err == nil || !strings.Contains(err.Error(), "OpenCV not available") {
		t.Fatalf("standalone fallback err=%v; expected English literal", err)
	}
}

// TestSetPkgTranslator_OpenCV_NilResetsToDefault is the §1.1 paired
// meta-test guard for the opencv translator seam.
func TestSetPkgTranslator_OpenCV_NilResetsToDefault(t *testing.T) {
	tr := &fakeTranslator{}
	SetPkgTranslator(tr)
	if PkgTranslator() != tr {
		t.Fatalf("SetPkgTranslator(tr) did not store tr")
	}
	SetPkgTranslator(nil)
	if PkgTranslator() == tr {
		t.Fatalf("SetPkgTranslator(nil) did not reset; still references fake")
	}
	if got := PkgTranslator().T(context.Background(), "probe"); got != "probe" {
		t.Fatalf("post-reset translator T(probe)=%q; expected msgID verbatim", got)
	}
}
