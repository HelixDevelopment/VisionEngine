// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"digital.vasic.visionengine/pkg/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestGraph() NavigationGraph {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home Screen", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddScreen(makeScreen("about", "About", "info"))
	g.AddTransition("home", "settings", makeAction("click", "settings_btn"))
	g.AddTransition("settings", "about", makeAction("click", "about_btn"))
	g.AddTransition("about", "home", makeAction("click", "back"))
	g.SetCurrent("settings")
	return g
}

// --- DOT Export Tests ---

func TestExportDOT_BasicStructure(t *testing.T) {
	g := buildTestGraph()
	dot := ExportDOT(g)

	assert.Contains(t, dot, "digraph navigation")
	assert.Contains(t, dot, "rankdir=LR")
	assert.Contains(t, dot, "node [shape=box")
}

func TestExportDOT_ContainsScreenNodes(t *testing.T) {
	g := buildTestGraph()
	dot := ExportDOT(g)

	assert.Contains(t, dot, "Home Screen")
	assert.Contains(t, dot, "Settings")
	assert.Contains(t, dot, "About")
}

func TestExportDOT_ContainsTransitions(t *testing.T) {
	g := buildTestGraph()
	dot := ExportDOT(g)

	assert.Contains(t, dot, "\"home\" -> \"settings\"")
	assert.Contains(t, dot, "\"settings\" -> \"about\"")
	assert.Contains(t, dot, "\"about\" -> \"home\"")
}

func TestExportDOT_VisitedNodesStyling(t *testing.T) {
	g := buildTestGraph()
	dot := ExportDOT(g)

	// settings is current and visited, should have special styling
	assert.Contains(t, dot, "penwidth=3")
	assert.Contains(t, dot, "fillcolor")
}

func TestExportDOT_EmptyGraph(t *testing.T) {
	g := NewNavigationGraph()
	dot := ExportDOT(g)

	assert.Contains(t, dot, "digraph navigation")
	assert.Contains(t, dot, "}")
}

func TestExportDOT_SpecialCharacters(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "special",
		Name:        `Screen with "quotes" and \slashes`,
		Fingerprint: "fp-special",
	})

	dot := ExportDOT(g)
	// Quotes are escaped to \" in DOT format
	assert.Contains(t, dot, `\"quotes\"`)
	assert.Contains(t, dot, `\\slashes`)
}

func TestExportDOT_TransitionLabels(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("a", "A", "cat"))
	g.AddScreen(makeScreen("b", "B", "cat"))
	g.AddTransition("a", "b", analyzer.Action{Type: "click", Target: "button"})

	dot := ExportDOT(g)
	assert.Contains(t, dot, "click: button")
}

func TestExportDOT_TransitionWithoutTarget(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("a", "A", "cat"))
	g.AddScreen(makeScreen("b", "B", "cat"))
	g.AddTransition("a", "b", analyzer.Action{Type: "back"})

	dot := ExportDOT(g)
	assert.Contains(t, dot, `label="back"`)
}

// --- JSON Export Tests ---

func TestExportJSON_ValidJSON(t *testing.T) {
	g := buildTestGraph()
	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)

	var snapshot GraphSnapshot
	err = json.Unmarshal([]byte(jsonStr), &snapshot)
	require.NoError(t, err)

	assert.Len(t, snapshot.Screens, 3)
	assert.Len(t, snapshot.Transitions, 3)
	assert.Equal(t, "settings", snapshot.Current)
}

func TestExportJSON_ScreenFields(t *testing.T) {
	g := buildTestGraph()
	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)

	var snapshot GraphSnapshot
	err = json.Unmarshal([]byte(jsonStr), &snapshot)
	require.NoError(t, err)

	found := false
	for _, s := range snapshot.Screens {
		if s.ID == "home" {
			found = true
			assert.Equal(t, "Home Screen", s.Identity.Name)
			assert.Equal(t, "main", s.Identity.Category)
		}
	}
	assert.True(t, found, "home screen should be in export")
}

func TestExportJSON_TransitionFields(t *testing.T) {
	g := buildTestGraph()
	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)

	var snapshot GraphSnapshot
	err = json.Unmarshal([]byte(jsonStr), &snapshot)
	require.NoError(t, err)

	found := false
	for _, tr := range snapshot.Transitions {
		if tr.From == "home" && tr.To == "settings" {
			found = true
			assert.Equal(t, "click", tr.Action.Type)
			assert.Equal(t, "settings_btn", tr.Action.Target)
		}
	}
	assert.True(t, found)
}

func TestExportJSON_EmptyGraph(t *testing.T) {
	g := NewNavigationGraph()
	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)

	var snapshot GraphSnapshot
	err = json.Unmarshal([]byte(jsonStr), &snapshot)
	require.NoError(t, err)

	assert.Empty(t, snapshot.Screens)
	assert.Empty(t, snapshot.Transitions)
}

func TestExportJSON_Coverage(t *testing.T) {
	g := buildTestGraph()
	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)

	var snapshot GraphSnapshot
	err = json.Unmarshal([]byte(jsonStr), &snapshot)
	require.NoError(t, err)

	// settings was set as current, so 1/3 visited
	assert.InDelta(t, 1.0/3.0, snapshot.Coverage, 0.01)
}

func TestExportJSON_Indented(t *testing.T) {
	g := buildTestGraph()
	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)

	assert.Contains(t, jsonStr, "\n")
	assert.Contains(t, jsonStr, "  ")
}

// --- Mermaid Export Tests ---

func TestExportMermaid_BasicStructure(t *testing.T) {
	g := buildTestGraph()
	mermaid := ExportMermaid(g)

	assert.Contains(t, mermaid, "graph LR")
}

func TestExportMermaid_ContainsNodes(t *testing.T) {
	g := buildTestGraph()
	mermaid := ExportMermaid(g)

	assert.Contains(t, mermaid, "Home Screen")
	assert.Contains(t, mermaid, "Settings")
	assert.Contains(t, mermaid, "About")
}

func TestExportMermaid_ContainsTransitions(t *testing.T) {
	g := buildTestGraph()
	mermaid := ExportMermaid(g)

	assert.Contains(t, mermaid, "-->")
	assert.Contains(t, mermaid, "click")
}

func TestExportMermaid_VisitedStyle(t *testing.T) {
	g := buildTestGraph()
	mermaid := ExportMermaid(g)

	assert.Contains(t, mermaid, "fill:#90EE90")
}

func TestExportMermaid_CurrentStyle(t *testing.T) {
	g := buildTestGraph()
	mermaid := ExportMermaid(g)

	assert.Contains(t, mermaid, "stroke-width:3px")
}

func TestExportMermaid_EmptyGraph(t *testing.T) {
	g := NewNavigationGraph()
	mermaid := ExportMermaid(g)

	assert.Contains(t, mermaid, "graph LR")
}

func TestExportMermaid_SpecialCharacters(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "screen-with-[brackets]",
		Name:        `Screen [with] "quotes"`,
		Fingerprint: "fp-special",
	})

	mermaid := ExportMermaid(g)
	assert.NotContains(t, mermaid, "[with]")
	assert.Contains(t, mermaid, "(with)")
}

func TestExportMermaid_TransitionLabels(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("a", "A", "cat"))
	g.AddScreen(makeScreen("b", "B", "cat"))
	g.AddTransition("a", "b", analyzer.Action{Type: "click", Target: "button"})

	mermaid := ExportMermaid(g)
	assert.Contains(t, mermaid, "click: button")
}

// --- Sanitization Tests ---

func TestSanitizeDOTLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`hello`, `hello`},
		{`say "hello"`, `say \"hello\"`},
		{`path\to\file`, `path\\to\\file`},
		{"line1\nline2", `line1\nline2`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeDOTLabel(tt.input))
		})
	}
}

func TestSanitizeMermaidLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`hello`, `hello`},
		{`[brackets]`, `(brackets)`},
		{`"quotes"`, `'quotes'`},
		{"line1\nline2", "line1 line2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeMermaidLabel(tt.input))
		})
	}
}

func TestSanitizeMermaidID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with spaces", "with_spaces"},
		{"with-dashes", "with_dashes"},
		{"with.dots", "with_dots"},
		{"", "node"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeMermaidID(tt.input))
		})
	}
}

// --- Large Graph Export ---

func TestExportDOT_LargeGraph(t *testing.T) {
	g := NewNavigationGraph()
	for i := 0; i < 100; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("Screen %d", i), "test"))
	}
	for i := 0; i < 99; i++ {
		g.AddTransition(fmt.Sprintf("s-%d", i), fmt.Sprintf("s-%d", i+1), makeAction("click", "next"))
	}

	dot := ExportDOT(g)
	assert.True(t, len(dot) > 0)
	// 100 node labels + 99 transition labels = 199
	assert.Equal(t, 199, strings.Count(dot, "label="))
}

func TestExportMermaid_LargeGraph(t *testing.T) {
	g := NewNavigationGraph()
	for i := 0; i < 100; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("Screen %d", i), "test"))
	}
	for i := 0; i < 99; i++ {
		g.AddTransition(fmt.Sprintf("s-%d", i), fmt.Sprintf("s-%d", i+1), makeAction("click", "next"))
	}

	mermaid := ExportMermaid(g)
	assert.True(t, len(mermaid) > 0)
	assert.Equal(t, 99, strings.Count(mermaid, "-->"))
}

func TestExportJSON_LargeGraph(t *testing.T) {
	g := NewNavigationGraph()
	for i := 0; i < 100; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("Screen %d", i), "test"))
	}

	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)

	var snapshot GraphSnapshot
	err = json.Unmarshal([]byte(jsonStr), &snapshot)
	require.NoError(t, err)
	assert.Len(t, snapshot.Screens, 100)
}

// --- Security Tests ---

func TestExportDOT_PathTraversal(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "../../../etc/passwd",
		Name:        "Malicious Screen",
		Fingerprint: "fp-malicious",
	})

	dot := ExportDOT(g)
	// The DOT output should be safe -- it's a string format, not a file path
	assert.Contains(t, dot, "Malicious Screen")
}

func TestExportJSON_PathTraversal(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "../../../etc/passwd",
		Name:        `<script>alert("xss")</script>`,
		Fingerprint: "fp-malicious",
	})

	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)

	// JSON encoding should escape angle brackets
	var snapshot GraphSnapshot
	err = json.Unmarshal([]byte(jsonStr), &snapshot)
	require.NoError(t, err)
	assert.Equal(t, `<script>alert("xss")</script>`, snapshot.Screens[0].Identity.Name)
}

func TestExportMermaid_Injection(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "screen-1",
		Name:        `"; rm -rf / "`,
		Fingerprint: "fp-1",
	})

	mermaid := ExportMermaid(g)
	// Quotes should be replaced
	assert.NotContains(t, mermaid, `";"`)
}
