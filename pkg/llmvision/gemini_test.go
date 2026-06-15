// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmvision

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeminiProvider_AnalyzeImage_MultiPartTextConcatenated reproduces a
// data-loss defect: the Gemini generateContent response splits a single
// model turn across multiple `candidates[0].content.parts[]` entries
// (this is the documented API behaviour — the official google-genai SDK's
// `response.text` accessor joins the text of EVERY part). The previous
// sendRequest returned only `parts[0].text`, silently DROPPING every
// subsequent part. End users received a TRUNCATED analysis presented as a
// complete success — a §11.4 PASS-bluff at the response-parsing layer.
//
// RED on the pre-fix code: the assertion below fails because only the
// first part ("The screen shows a ") is returned, dropping
// "login dialog with two input fields." GREEN after the fix concatenates
// all parts.
func TestGeminiProvider_AnalyzeImage_MultiPartTextConcatenated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// A real two-part Gemini text response. Both parts together form
		// the complete answer; neither alone is the full answer.
		_, _ = w.Write([]byte(`{
			"candidates": [
				{
					"content": {
						"parts": [
							{"text": "The screen shows a "},
							{"text": "login dialog with two input fields."}
						]
					}
				}
			]
		}`))
	}))
	defer server.Close()

	p, err := NewGeminiProvider(ProviderConfig{APIKey: "key", BaseURL: server.URL})
	require.NoError(t, err)

	got, err := p.AnalyzeImage(context.Background(), []byte("img-bytes"), "describe")
	require.NoError(t, err)

	want := "The screen shows a login dialog with two input fields."
	assert.Equal(t, want, got,
		"Gemini multi-part text MUST be concatenated across all parts; "+
			"returning only parts[0].text silently truncates the response (data loss)")
}

// TestGeminiProvider_AnalyzeImage_SinglePartUnchanged guards the common
// single-part case so the multi-part fix does not regress it.
func TestGeminiProvider_AnalyzeImage_SinglePartUnchanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"only part"}]}}]}`))
	}))
	defer server.Close()

	p, err := NewGeminiProvider(ProviderConfig{APIKey: "key", BaseURL: server.URL})
	require.NoError(t, err)

	got, err := p.AnalyzeImage(context.Background(), []byte("img"), "describe")
	require.NoError(t, err)
	assert.Equal(t, "only part", got)
}

// TestGeminiProvider_AnalyzeImage_EmptyPartsRejected guards the
// no-content-in-response error path (must remain after the fix).
func TestGeminiProvider_AnalyzeImage_EmptyPartsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[]}}]}`))
	}))
	defer server.Close()

	p, err := NewGeminiProvider(ProviderConfig{APIKey: "key", BaseURL: server.URL})
	require.NoError(t, err)

	_, err = p.AnalyzeImage(context.Background(), []byte("img"), "describe")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidResponse)
}
