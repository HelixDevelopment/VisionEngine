// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStubAnalyzer(t *testing.T) {
	a := NewStubAnalyzer()
	require.NotNil(t, a)
	assert.Nil(t, a.Provider)
}

func TestNewStubAnalyzerWithProvider(t *testing.T) {
	a := NewStubAnalyzerWithProvider(nil)
	require.NotNil(t, a)
	assert.Nil(t, a.Provider)
}

// TestStubAnalyzer_AnalyzeScreen — round-27 §11.4 audit (2026-05-17):
// AnalyzeScreen now returns ErrStubAnalyzerNotImplemented for valid
// non-empty input. The previous "successful no-op with Unknown
// Screen title" was a PASS-bluff and has been removed.
func TestStubAnalyzer_AnalyzeScreen(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	_, err := a.AnalyzeScreen(ctx, []byte("fake screenshot data"))
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrStubAnalyzerNotImplemented),
		"expected errors.Is(err, ErrStubAnalyzerNotImplemented), got: %v", err)
}

func TestStubAnalyzer_AnalyzeScreen_Empty(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	_, err := a.AnalyzeScreen(ctx, []byte{})
	assert.ErrorIs(t, err, ErrEmptyScreenshot)
}

func TestStubAnalyzer_AnalyzeScreen_Cancelled(t *testing.T) {
	a := NewStubAnalyzer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := a.AnalyzeScreen(ctx, []byte("data"))
	assert.Error(t, err)
	// Either ctx.Err() (preferred — cancel-before-method-body) or
	// the sentinel is acceptable; assert NOT-nil and either of them.
	if !errors.Is(err, context.Canceled) && !errors.Is(err, ErrStubAnalyzerNotImplemented) {
		t.Fatalf("expected context.Canceled or ErrStubAnalyzerNotImplemented, got: %v", err)
	}
}

// TestStubAnalyzer_CompareScreens — round-27 §11.4 audit: byte-equality
// "comparison" replaced with sentinel.
func TestStubAnalyzer_CompareScreens(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	img1 := []byte("image one")
	img2 := []byte("image two")

	_, err := a.CompareScreens(ctx, img1, img2)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrStubAnalyzerNotImplemented),
		"expected errors.Is(err, ErrStubAnalyzerNotImplemented), got: %v", err)
}

// TestStubAnalyzer_CompareScreens_Identical — even when bytes match
// (the case the old bluff handled with Similarity: 1.0), the honest
// stub still returns the sentinel: it cannot prove the screens are
// visually identical from byte equality.
func TestStubAnalyzer_CompareScreens_Identical(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	img := []byte("same image data")

	_, err := a.CompareScreens(ctx, img, img)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrStubAnalyzerNotImplemented),
		"expected errors.Is(err, ErrStubAnalyzerNotImplemented), got: %v", err)
}

func TestStubAnalyzer_CompareScreens_EmptyBefore(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	_, err := a.CompareScreens(ctx, []byte{}, []byte("after"))
	assert.ErrorIs(t, err, ErrEmptyScreenshot)
}

func TestStubAnalyzer_CompareScreens_EmptyAfter(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	_, err := a.CompareScreens(ctx, []byte("before"), []byte{})
	assert.ErrorIs(t, err, ErrEmptyScreenshot)
}

func TestStubAnalyzer_CompareScreens_Cancelled(t *testing.T) {
	a := NewStubAnalyzer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := a.CompareScreens(ctx, []byte("a"), []byte("b"))
	assert.Error(t, err)
}

// TestStubAnalyzer_DetectElements — round-27 §11.4 audit: empty-slice
// success replaced with sentinel.
func TestStubAnalyzer_DetectElements(t *testing.T) {
	a := NewStubAnalyzer()

	_, err := a.DetectElements([]byte("screenshot"))
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrStubAnalyzerNotImplemented),
		"expected errors.Is(err, ErrStubAnalyzerNotImplemented), got: %v", err)
}

func TestStubAnalyzer_DetectElements_Empty(t *testing.T) {
	a := NewStubAnalyzer()

	_, err := a.DetectElements([]byte{})
	assert.ErrorIs(t, err, ErrEmptyScreenshot)
}

// TestStubAnalyzer_DetectText — round-27 §11.4 audit.
func TestStubAnalyzer_DetectText(t *testing.T) {
	a := NewStubAnalyzer()

	_, err := a.DetectText([]byte("screenshot"))
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrStubAnalyzerNotImplemented),
		"expected errors.Is(err, ErrStubAnalyzerNotImplemented), got: %v", err)
}

func TestStubAnalyzer_DetectText_Empty(t *testing.T) {
	a := NewStubAnalyzer()

	_, err := a.DetectText([]byte{})
	assert.ErrorIs(t, err, ErrEmptyScreenshot)
}

// TestStubAnalyzer_IdentifyScreen — round-27 §11.4 audit: SHA-256
// hash leak as "screen identity" replaced with sentinel.
func TestStubAnalyzer_IdentifyScreen(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	_, err := a.IdentifyScreen(ctx, []byte("screenshot"))
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrStubAnalyzerNotImplemented),
		"expected errors.Is(err, ErrStubAnalyzerNotImplemented), got: %v", err)
}

func TestStubAnalyzer_IdentifyScreen_Empty(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	_, err := a.IdentifyScreen(ctx, []byte{})
	assert.ErrorIs(t, err, ErrEmptyScreenshot)
}

func TestStubAnalyzer_IdentifyScreen_Cancelled(t *testing.T) {
	a := NewStubAnalyzer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := a.IdentifyScreen(ctx, []byte("data"))
	assert.Error(t, err)
}

// TestStubAnalyzer_DetectIssues — round-27 §11.4 audit.
func TestStubAnalyzer_DetectIssues(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	_, err := a.DetectIssues(ctx, []byte("screenshot"))
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrStubAnalyzerNotImplemented),
		"expected errors.Is(err, ErrStubAnalyzerNotImplemented), got: %v", err)
}

func TestStubAnalyzer_DetectIssues_Empty(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	_, err := a.DetectIssues(ctx, []byte{})
	assert.ErrorIs(t, err, ErrEmptyScreenshot)
}

func TestStubAnalyzer_DetectIssues_Cancelled(t *testing.T) {
	a := NewStubAnalyzer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := a.DetectIssues(ctx, []byte("data"))
	assert.Error(t, err)
}

func TestStubAnalyzer_ImplementsInterface(t *testing.T) {
	var _ Analyzer = (*StubAnalyzer)(nil)
}

// TestFingerprint_Deterministic — fingerprint() remains an internal
// helper; tests retained to confirm SHA-256 determinism (used by the
// real OpenCV implementation downstream).
func TestFingerprint_Deterministic(t *testing.T) {
	data := []byte("test data for fingerprinting")
	fp1 := fingerprint(data)
	fp2 := fingerprint(data)
	assert.Equal(t, fp1, fp2)
	assert.Len(t, fp1, 64) // SHA-256 hex string length
}

func TestFingerprint_Unique(t *testing.T) {
	fp1 := fingerprint([]byte("data1"))
	fp2 := fingerprint([]byte("data2"))
	assert.NotEqual(t, fp1, fp2)
}
