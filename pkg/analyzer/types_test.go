// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRect_Contains(t *testing.T) {
	tests := []struct {
		name   string
		rect   Rect
		px, py int
		want   bool
	}{
		{"inside", Rect{10, 10, 100, 100}, 50, 50, true},
		{"top_left_corner", Rect{10, 10, 100, 100}, 10, 10, true},
		{"outside_left", Rect{10, 10, 100, 100}, 5, 50, false},
		{"outside_right", Rect{10, 10, 100, 100}, 110, 50, false},
		{"outside_top", Rect{10, 10, 100, 100}, 50, 5, false},
		{"outside_bottom", Rect{10, 10, 100, 100}, 50, 110, false},
		{"on_right_edge", Rect{10, 10, 100, 100}, 110, 50, false},
		{"on_bottom_edge", Rect{10, 10, 100, 100}, 50, 110, false},
		{"zero_rect", Rect{0, 0, 0, 0}, 0, 0, false},
		{"negative_point", Rect{0, 0, 10, 10}, -1, -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rect.Contains(tt.px, tt.py))
		})
	}
}

func TestRect_Overlaps(t *testing.T) {
	tests := []struct {
		name  string
		r1    Rect
		r2    Rect
		want  bool
	}{
		{"overlapping", Rect{0, 0, 100, 100}, Rect{50, 50, 100, 100}, true},
		{"not_overlapping", Rect{0, 0, 10, 10}, Rect{20, 20, 10, 10}, false},
		{"touching_edges", Rect{0, 0, 10, 10}, Rect{10, 0, 10, 10}, false},
		{"contained", Rect{0, 0, 100, 100}, Rect{25, 25, 50, 50}, true},
		{"same_rect", Rect{0, 0, 10, 10}, Rect{0, 0, 10, 10}, true},
		{"one_pixel_overlap", Rect{0, 0, 10, 10}, Rect{9, 9, 10, 10}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.r1.Overlaps(tt.r2))
		})
	}
}

func TestRect_Area(t *testing.T) {
	tests := []struct {
		name string
		rect Rect
		want int
	}{
		{"normal", Rect{0, 0, 10, 20}, 200},
		{"zero_width", Rect{0, 0, 0, 10}, 0},
		{"zero_height", Rect{0, 0, 10, 0}, 0},
		{"negative_width", Rect{0, 0, -5, 10}, 0},
		{"single_pixel", Rect{0, 0, 1, 1}, 1},
		{"large", Rect{0, 0, 1920, 1080}, 2073600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rect.Area())
		})
	}
}

func TestRect_Center(t *testing.T) {
	tests := []struct {
		name       string
		rect       Rect
		wantX      int
		wantY      int
	}{
		{"origin", Rect{0, 0, 100, 100}, 50, 50},
		{"offset", Rect{10, 20, 100, 100}, 60, 70},
		{"odd_size", Rect{0, 0, 11, 11}, 5, 5},
		{"single_pixel", Rect{5, 5, 1, 1}, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := tt.rect.Center()
			assert.Equal(t, tt.wantX, x)
			assert.Equal(t, tt.wantY, y)
		})
	}
}

func TestUIElement_Fields(t *testing.T) {
	elem := UIElement{
		Type:        "button",
		Label:       "Submit",
		BoundingBox: Rect{100, 200, 80, 40},
		Clickable:   true,
		Confidence:  0.95,
	}

	assert.Equal(t, "button", elem.Type)
	assert.Equal(t, "Submit", elem.Label)
	assert.True(t, elem.Clickable)
	assert.Equal(t, 0.95, elem.Confidence)
	assert.Equal(t, 3200, elem.BoundingBox.Area())
}

func TestTextRegion_Fields(t *testing.T) {
	region := TextRegion{
		Text:        "Hello World",
		BoundingBox: Rect{10, 10, 200, 30},
		Confidence:  0.99,
		Language:    "en",
		FontSize:    14.0,
	}

	assert.Equal(t, "Hello World", region.Text)
	assert.Equal(t, "en", region.Language)
	assert.Equal(t, 14.0, region.FontSize)
}

func TestVisualIssue_Fields(t *testing.T) {
	issue := VisualIssue{
		Type:        "truncation",
		Severity:    "medium",
		Description: "Button text is truncated",
		BoundingBox: Rect{100, 200, 80, 40},
		Confidence:  0.85,
		Suggestion:  "Use wrap_content",
	}

	assert.Equal(t, "truncation", issue.Type)
	assert.Equal(t, "medium", issue.Severity)
	assert.Equal(t, "Use wrap_content", issue.Suggestion)
}

func TestAction_Fields(t *testing.T) {
	action := Action{
		Type:       "click",
		Target:     "login_button",
		Value:      "",
		Confidence: 0.9,
	}

	assert.Equal(t, "click", action.Type)
	assert.Equal(t, "login_button", action.Target)
}

func TestScreenIdentity_Fields(t *testing.T) {
	identity := ScreenIdentity{
		ID:          "screen-abc123",
		Name:        "Login Screen",
		Category:    "auth",
		Fingerprint: "sha256:abc123",
		Tags:        []string{"login", "auth", "initial"},
	}

	assert.Equal(t, "screen-abc123", identity.ID)
	assert.Equal(t, "Login Screen", identity.Name)
	assert.Equal(t, "auth", identity.Category)
	assert.Len(t, identity.Tags, 3)
}

func TestScreenAnalysis_Fields(t *testing.T) {
	analysis := ScreenAnalysis{
		ScreenID:    "screen-001",
		Title:       "Settings",
		Description: "App settings screen",
		Elements:    []UIElement{{Type: "button", Label: "Save"}},
		TextRegions: []TextRegion{{Text: "Settings"}},
		Issues:      []VisualIssue{},
		Navigable:   []Action{{Type: "click", Target: "back_button"}},
	}

	assert.Equal(t, "screen-001", analysis.ScreenID)
	assert.Len(t, analysis.Elements, 1)
	assert.Len(t, analysis.TextRegions, 1)
	assert.Len(t, analysis.Issues, 0)
	assert.Len(t, analysis.Navigable, 1)
}

func TestScreenDiff_Fields(t *testing.T) {
	diff := ScreenDiff{
		Similarity:     0.85,
		ChangedRegions: []Rect{{10, 10, 50, 50}},
		NewElements:    []UIElement{{Type: "button", Label: "New Button"}},
		GoneElements:   []UIElement{},
		IsNewScreen:    false,
	}

	assert.Equal(t, 0.85, diff.Similarity)
	assert.Len(t, diff.ChangedRegions, 1)
	assert.Len(t, diff.NewElements, 1)
	assert.Len(t, diff.GoneElements, 0)
	assert.False(t, diff.IsNewScreen)
}

func TestKeyFrame_Fields(t *testing.T) {
	kf := KeyFrame{
		Timestamp: 5000000000, // 5 seconds in nanoseconds
		Data:      []byte{0xFF, 0xD8, 0xFF},
		Index:     3,
	}

	assert.Equal(t, 3, kf.Index)
	assert.Len(t, kf.Data, 3)
}

func TestSize_Fields(t *testing.T) {
	size := Size{Width: 1920, Height: 1080}
	assert.Equal(t, 1920, size.Width)
	assert.Equal(t, 1080, size.Height)
}

func TestRect_Overlaps_Symmetric(t *testing.T) {
	r1 := Rect{0, 0, 50, 50}
	r2 := Rect{25, 25, 50, 50}
	assert.Equal(t, r1.Overlaps(r2), r2.Overlaps(r1), "Overlap should be symmetric")
}
