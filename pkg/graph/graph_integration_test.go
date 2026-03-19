// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"digital.vasic.visionengine/pkg/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_AnalyzerToGraphPipeline(t *testing.T) {
	// Simulate the full pipeline: analyzer identifies screens -> graph tracks navigation
	a := analyzer.NewStubAnalyzer()
	g := NewNavigationGraph()

	// Simulate navigating through 5 screens
	screenshots := [][]byte{
		[]byte("home-screen-data"),
		[]byte("settings-screen-data"),
		[]byte("about-screen-data"),
		[]byte("editor-screen-data"),
		[]byte("file-list-screen-data"),
	}

	var prevID string
	for i, img := range screenshots {
		identity, err := a.IdentifyScreen(context.Background(), img)
		require.NoError(t, err)

		screenID := g.AddScreen(identity)
		g.SetCurrent(screenID)

		if prevID != "" {
			g.AddTransition(prevID, screenID, analyzer.Action{
				Type:       "navigate",
				Target:     fmt.Sprintf("step-%d", i),
				Confidence: 0.95,
			})
		}
		prevID = screenID
	}

	assert.Len(t, g.Screens(), 5)
	assert.Len(t, g.Transitions(), 4)
	assert.Equal(t, 1.0, g.Coverage())
	assert.Empty(t, g.UnvisitedScreens())
}

func TestIntegration_GraphExportRoundTrip(t *testing.T) {
	// Create a graph, export to JSON, verify the snapshot
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddTransition("home", "settings", makeAction("click", "gear"))
	g.SetCurrent("home")

	// Export to JSON
	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)

	// Parse back
	var snapshot GraphSnapshot
	err = json.Unmarshal([]byte(jsonStr), &snapshot)
	require.NoError(t, err)

	// Verify
	assert.Len(t, snapshot.Screens, 2)
	assert.Len(t, snapshot.Transitions, 1)
	assert.Equal(t, "home", snapshot.Current)
	assert.Equal(t, 0.5, snapshot.Coverage)
}

func TestIntegration_ScreenComparisonToGraphUpdate(t *testing.T) {
	// Simulate: compare screens -> determine if new screen -> update graph
	a := analyzer.NewStubAnalyzer()
	g := NewNavigationGraph()

	before := []byte("before-screenshot")
	after := []byte("after-screenshot")

	// Identify and add "before" screen
	beforeID, err := a.IdentifyScreen(context.Background(), before)
	require.NoError(t, err)
	g.AddScreen(beforeID)
	g.SetCurrent(beforeID.ID)

	// Compare screens
	diff, err := a.CompareScreens(context.Background(), before, after)
	require.NoError(t, err)

	if diff.IsNewScreen {
		// Identify "after" screen
		afterID, err := a.IdentifyScreen(context.Background(), after)
		require.NoError(t, err)
		newScreenID := g.AddScreen(afterID)
		g.AddTransition(beforeID.ID, newScreenID, analyzer.Action{
			Type:       "click",
			Target:     "some-element",
			Confidence: 0.8,
		})
		g.SetCurrent(newScreenID)
	}

	assert.Len(t, g.Screens(), 2)
	assert.Equal(t, 1.0, g.Coverage())
}

func TestIntegration_NavigationWithBacktracking(t *testing.T) {
	g := NewNavigationGraph()

	// Build a simple app navigation graph
	screens := []struct{ id, name, cat string }{
		{"home", "Home", "main"},
		{"menu", "Menu", "nav"},
		{"settings", "Settings", "settings"},
		{"profile", "Profile", "user"},
		{"help", "Help", "info"},
	}
	for _, s := range screens {
		g.AddScreen(makeScreen(s.id, s.name, s.cat))
	}

	// Forward navigation
	g.AddTransition("home", "menu", makeAction("click", "hamburger"))
	g.AddTransition("menu", "settings", makeAction("click", "settings"))
	g.AddTransition("menu", "profile", makeAction("click", "profile"))
	g.AddTransition("menu", "help", makeAction("click", "help"))

	// Back navigation
	g.AddTransition("settings", "menu", makeAction("back", ""))
	g.AddTransition("profile", "menu", makeAction("back", ""))
	g.AddTransition("help", "menu", makeAction("back", ""))
	g.AddTransition("menu", "home", makeAction("back", ""))

	// Navigate forward
	g.SetCurrent("home")
	g.SetCurrent("menu")
	g.SetCurrent("settings")

	// Use pathfinding to go to profile
	path, err := g.PathTo("profile")
	require.NoError(t, err)
	assert.Len(t, path, 2) // settings->menu->profile

	// Navigate there
	for _, step := range path {
		g.SetCurrent(step.To)
	}
	assert.Equal(t, "profile", g.CurrentScreen())

	// Check unvisited
	unvisited := g.UnvisitedScreens()
	assert.Len(t, unvisited, 1)
	assert.Contains(t, unvisited, "help")
}

func TestIntegration_AllExportFormats(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("login", "Login", "auth"))
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddTransition("login", "home", makeAction("click", "login_btn"))
	g.AddTransition("home", "settings", makeAction("click", "gear"))
	g.AddTransition("settings", "home", makeAction("back", ""))
	g.SetCurrent("home")

	// DOT
	dot := ExportDOT(g)
	assert.Contains(t, dot, "digraph")
	assert.Contains(t, dot, "Login")
	assert.Contains(t, dot, "->")

	// JSON
	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)
	var snapshot GraphSnapshot
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &snapshot))
	assert.Len(t, snapshot.Screens, 3)

	// Mermaid
	mermaid := ExportMermaid(g)
	assert.Contains(t, mermaid, "graph LR")
	assert.Contains(t, mermaid, "-->")
}

func TestIntegration_CyclicNavigation(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("a", "A", "test"))
	g.AddScreen(makeScreen("b", "B", "test"))
	g.AddScreen(makeScreen("c", "C", "test"))

	// Create a cycle: a -> b -> c -> a
	g.AddTransition("a", "b", makeAction("click", "next"))
	g.AddTransition("b", "c", makeAction("click", "next"))
	g.AddTransition("c", "a", makeAction("click", "next"))

	g.SetCurrent("a")

	// BFS should find direct paths despite cycle
	path, err := g.PathTo("c")
	require.NoError(t, err)
	assert.Len(t, path, 2) // a->b->c
}

func TestIntegration_DisconnectedComponents(t *testing.T) {
	g := NewNavigationGraph()

	// Component 1
	g.AddScreen(makeScreen("a1", "A1", "comp1"))
	g.AddScreen(makeScreen("a2", "A2", "comp1"))
	g.AddTransition("a1", "a2", makeAction("click", "next"))

	// Component 2 (disconnected)
	g.AddScreen(makeScreen("b1", "B1", "comp2"))
	g.AddScreen(makeScreen("b2", "B2", "comp2"))
	g.AddTransition("b1", "b2", makeAction("click", "next"))

	g.SetCurrent("a1")

	// Should find path within component
	path, err := g.PathTo("a2")
	require.NoError(t, err)
	assert.Len(t, path, 1)

	// Should not find path to other component
	_, err = g.PathTo("b1")
	assert.ErrorIs(t, err, ErrNoPath)
}

func TestIntegration_ExportAfterManyOperations(t *testing.T) {
	g := NewNavigationGraph()

	// Simulate a long QA session
	for i := 0; i < 20; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("Screen %d", i), "qa"))
		if i > 0 {
			g.AddTransition(fmt.Sprintf("s-%d", i-1), fmt.Sprintf("s-%d", i), makeAction("navigate", ""))
		}
		g.SetCurrent(fmt.Sprintf("s-%d", i))
	}

	// Export all formats
	dot := ExportDOT(g)
	assert.NotEmpty(t, dot)

	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonStr)

	mermaid := ExportMermaid(g)
	assert.NotEmpty(t, mermaid)

	// Verify final state
	assert.Equal(t, 1.0, g.Coverage())
	assert.Equal(t, "s-19", g.CurrentScreen())
}
