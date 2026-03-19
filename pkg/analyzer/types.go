// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"time"
)

// Rect represents a bounding box rectangle.
type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Contains returns true if the point (px, py) is inside the rectangle.
func (r Rect) Contains(px, py int) bool {
	return px >= r.X && px < r.X+r.Width && py >= r.Y && py < r.Y+r.Height
}

// Overlaps returns true if the two rectangles overlap.
func (r Rect) Overlaps(other Rect) bool {
	return r.X < other.X+other.Width &&
		r.X+r.Width > other.X &&
		r.Y < other.Y+other.Height &&
		r.Y+r.Height > other.Y
}

// Area returns the area of the rectangle.
func (r Rect) Area() int {
	if r.Width <= 0 || r.Height <= 0 {
		return 0
	}
	return r.Width * r.Height
}

// Center returns the center point of the rectangle.
func (r Rect) Center() (int, int) {
	return r.X + r.Width/2, r.Y + r.Height/2
}

// Size represents image dimensions.
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// UIElement represents a detected UI element on a screen.
type UIElement struct {
	Type        string  `json:"type"`         // button, input, link, menu, tab, etc.
	Label       string  `json:"label"`        // display text or accessibility label
	BoundingBox Rect    `json:"bounding_box"` // position and size
	Clickable   bool    `json:"clickable"`    // whether the element can be clicked
	Confidence  float64 `json:"confidence"`   // detection confidence 0-1
}

// TextRegion represents a region of detected text.
type TextRegion struct {
	Text        string  `json:"text"`
	BoundingBox Rect    `json:"bounding_box"`
	Confidence  float64 `json:"confidence"`
	Language    string  `json:"language,omitempty"`
	FontSize    float64 `json:"font_size,omitempty"`
}

// VisualIssue represents a detected visual or UX problem.
type VisualIssue struct {
	Type        string  `json:"type"`        // overlap, truncation, contrast, alignment, spacing
	Severity    string  `json:"severity"`    // critical, high, medium, low
	Description string  `json:"description"` // human-readable description
	BoundingBox Rect    `json:"bounding_box"`
	Confidence  float64 `json:"confidence"`
	Suggestion  string  `json:"suggestion,omitempty"` // recommended fix
}

// Action represents a navigation or interaction action.
type Action struct {
	Type       string  `json:"type"`       // click, type, scroll, navigate, back, home, long_press, swipe, key_press
	Target     string  `json:"target"`     // element label or coordinates
	Value      string  `json:"value"`      // text to type, scroll amount, etc.
	Confidence float64 `json:"confidence"` // action confidence 0-1
}

// ScreenIdentity represents a uniquely identified screen.
type ScreenIdentity struct {
	ID          string   `json:"id"`          // unique screen identifier
	Name        string   `json:"name"`        // human-readable screen name
	Category    string   `json:"category"`    // settings, main, dialog, etc.
	Fingerprint string   `json:"fingerprint"` // visual fingerprint hash
	Tags        []string `json:"tags"`        // descriptive tags
}

// ScreenAnalysis is the result of analyzing a screenshot.
type ScreenAnalysis struct {
	ScreenID    string        `json:"screen_id"`
	Title       string        `json:"title"`       // LLM-identified screen name
	Description string        `json:"description"` // screen description
	Elements    []UIElement   `json:"elements"`
	TextRegions []TextRegion  `json:"text_regions"`
	Issues      []VisualIssue `json:"issues"`
	Navigable   []Action      `json:"navigable"` // possible navigation actions
	Timestamp   time.Time     `json:"timestamp"`
}

// ScreenDiff represents the differences between two screenshots.
type ScreenDiff struct {
	Similarity     float64     `json:"similarity"`      // SSIM score 0-1
	ChangedRegions []Rect      `json:"changed_regions"`
	NewElements    []UIElement `json:"new_elements"`
	GoneElements   []UIElement `json:"gone_elements"`
	DiffImage      []byte      `json:"diff_image,omitempty"`
	IsNewScreen    bool        `json:"is_new_screen"`
}

// KeyFrame represents an extracted video key frame.
type KeyFrame struct {
	Timestamp time.Duration `json:"timestamp"`
	Data      []byte        `json:"data"`
	Index     int           `json:"index"`
}
