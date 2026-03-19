// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"context"
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

func TestStubAnalyzer_AnalyzeScreen(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	result, err := a.AnalyzeScreen(ctx, []byte("fake screenshot data"))
	require.NoError(t, err)
	assert.NotEmpty(t, result.ScreenID)
	assert.Equal(t, "Unknown Screen", result.Title)
	assert.NotZero(t, result.Timestamp)
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
}

func TestStubAnalyzer_CompareScreens(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	img1 := []byte("image one")
	img2 := []byte("image two")

	diff, err := a.CompareScreens(ctx, img1, img2)
	require.NoError(t, err)
	assert.Equal(t, 0.0, diff.Similarity)
	assert.True(t, diff.IsNewScreen)
}

func TestStubAnalyzer_CompareScreens_Identical(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	img := []byte("same image data")

	diff, err := a.CompareScreens(ctx, img, img)
	require.NoError(t, err)
	assert.Equal(t, 1.0, diff.Similarity)
	assert.False(t, diff.IsNewScreen)
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

func TestStubAnalyzer_DetectElements(t *testing.T) {
	a := NewStubAnalyzer()

	elements, err := a.DetectElements([]byte("screenshot"))
	require.NoError(t, err)
	assert.Empty(t, elements)
}

func TestStubAnalyzer_DetectElements_Empty(t *testing.T) {
	a := NewStubAnalyzer()

	_, err := a.DetectElements([]byte{})
	assert.ErrorIs(t, err, ErrEmptyScreenshot)
}

func TestStubAnalyzer_DetectText(t *testing.T) {
	a := NewStubAnalyzer()

	regions, err := a.DetectText([]byte("screenshot"))
	require.NoError(t, err)
	assert.Empty(t, regions)
}

func TestStubAnalyzer_DetectText_Empty(t *testing.T) {
	a := NewStubAnalyzer()

	_, err := a.DetectText([]byte{})
	assert.ErrorIs(t, err, ErrEmptyScreenshot)
}

func TestStubAnalyzer_IdentifyScreen(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	identity, err := a.IdentifyScreen(ctx, []byte("screenshot"))
	require.NoError(t, err)
	assert.Contains(t, identity.ID, "screen-")
	assert.Equal(t, "Unknown", identity.Name)
	assert.Equal(t, "unknown", identity.Category)
	assert.NotEmpty(t, identity.Fingerprint)
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

func TestStubAnalyzer_IdentifyScreen_Deterministic(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()
	data := []byte("same screenshot data")

	id1, err := a.IdentifyScreen(ctx, data)
	require.NoError(t, err)
	id2, err := a.IdentifyScreen(ctx, data)
	require.NoError(t, err)
	assert.Equal(t, id1.Fingerprint, id2.Fingerprint, "Same data should produce same fingerprint")
}

func TestStubAnalyzer_DetectIssues(t *testing.T) {
	a := NewStubAnalyzer()
	ctx := context.Background()

	issues, err := a.DetectIssues(ctx, []byte("screenshot"))
	require.NoError(t, err)
	assert.Empty(t, issues)
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
