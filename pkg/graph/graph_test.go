// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"digital.vasic.visionengine/pkg/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func makeScreen(id, name, category string) analyzer.ScreenIdentity {
	return analyzer.ScreenIdentity{
		ID:          id,
		Name:        name,
		Category:    category,
		Fingerprint: fmt.Sprintf("fp-%s", id),
		Tags:        []string{category},
	}
}

func makeAction(actionType, target string) analyzer.Action {
	return analyzer.Action{
		Type:       actionType,
		Target:     target,
		Confidence: 1.0,
	}
}

func TestNewNavigationGraph(t *testing.T) {
	g := NewNavigationGraph()
	require.NotNil(t, g)
	assert.Equal(t, "", g.CurrentScreen())
	assert.Empty(t, g.Screens())
	assert.Empty(t, g.Transitions())
	assert.Equal(t, 0.0, g.Coverage())
}

func TestNavigationGraph_AddScreen(t *testing.T) {
	g := NewNavigationGraph()
	screen := makeScreen("home", "Home", "main")

	id := g.AddScreen(screen)
	assert.Equal(t, "home", id)
	assert.Len(t, g.Screens(), 1)
}

func TestNavigationGraph_AddScreen_Duplicate(t *testing.T) {
	g := NewNavigationGraph()
	screen := makeScreen("home", "Home", "main")

	id1 := g.AddScreen(screen)
	id2 := g.AddScreen(screen)
	assert.Equal(t, id1, id2)
	assert.Len(t, g.Screens(), 1, "Duplicate screen should not be added")
}

func TestNavigationGraph_AddScreen_DuplicateFingerprint(t *testing.T) {
	g := NewNavigationGraph()
	s1 := analyzer.ScreenIdentity{ID: "s1", Fingerprint: "same-fp"}
	s2 := analyzer.ScreenIdentity{ID: "s2", Fingerprint: "same-fp"}

	id1 := g.AddScreen(s1)
	id2 := g.AddScreen(s2)
	assert.Equal(t, id1, id2, "Same fingerprint should return existing screen")
	assert.Len(t, g.Screens(), 1)
}

func TestNavigationGraph_AddScreen_MultipleUnique(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddScreen(makeScreen("about", "About", "info"))

	assert.Len(t, g.Screens(), 3)
}

func TestNavigationGraph_AddTransition(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))

	g.AddTransition("home", "settings", makeAction("click", "settings_btn"))
	assert.Len(t, g.Transitions(), 1)
}

func TestNavigationGraph_AddTransition_SelfLoop(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))

	g.AddTransition("home", "home", makeAction("click", "refresh"))
	assert.Empty(t, g.Transitions(), "Self-transition should be silently ignored")
}

func TestNavigationGraph_AddTransition_Duplicate(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))

	action := makeAction("click", "settings_btn")
	g.AddTransition("home", "settings", action)
	g.AddTransition("home", "settings", action)
	assert.Len(t, g.Transitions(), 1, "Duplicate transition should not be added")
}

func TestNavigationGraph_AddTransition_DifferentActions(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))

	g.AddTransition("home", "settings", makeAction("click", "settings_btn"))
	g.AddTransition("home", "settings", makeAction("click", "gear_icon"))
	assert.Len(t, g.Transitions(), 2, "Different actions to same target are distinct transitions")
}

func TestNavigationGraph_CurrentScreen(t *testing.T) {
	g := NewNavigationGraph()
	assert.Equal(t, "", g.CurrentScreen())

	g.AddScreen(makeScreen("home", "Home", "main"))
	g.SetCurrent("home")
	assert.Equal(t, "home", g.CurrentScreen())
}

func TestNavigationGraph_SetCurrent_MarksVisited(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))

	g.SetCurrent("home")
	screens := g.Screens()
	for _, s := range screens {
		if s.ID == "home" {
			assert.True(t, s.Visited)
			assert.NotZero(t, s.VisitedAt)
		} else {
			assert.False(t, s.Visited)
		}
	}
}

func TestNavigationGraph_SetCurrent_NonExistent(t *testing.T) {
	g := NewNavigationGraph()
	g.SetCurrent("nonexistent")
	assert.Equal(t, "nonexistent", g.CurrentScreen())
}

func TestNavigationGraph_PathTo_Simple(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddTransition("home", "settings", makeAction("click", "settings_btn"))
	g.SetCurrent("home")

	path, err := g.PathTo("settings")
	require.NoError(t, err)
	assert.Len(t, path, 1)
	assert.Equal(t, "home", path[0].From)
	assert.Equal(t, "settings", path[0].To)
}

func TestNavigationGraph_PathTo_MultiHop(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddScreen(makeScreen("about", "About", "info"))
	g.AddTransition("home", "settings", makeAction("click", "settings_btn"))
	g.AddTransition("settings", "about", makeAction("click", "about_btn"))
	g.SetCurrent("home")

	path, err := g.PathTo("about")
	require.NoError(t, err)
	assert.Len(t, path, 2)
	assert.Equal(t, "home", path[0].From)
	assert.Equal(t, "settings", path[0].To)
	assert.Equal(t, "settings", path[1].From)
	assert.Equal(t, "about", path[1].To)
}

func TestNavigationGraph_PathTo_SameScreen(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.SetCurrent("home")

	path, err := g.PathTo("home")
	require.NoError(t, err)
	assert.Empty(t, path)
}

func TestNavigationGraph_PathTo_NoPath(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	// No transition from home to settings
	g.SetCurrent("home")

	_, err := g.PathTo("settings")
	assert.ErrorIs(t, err, ErrNoPath)
}

func TestNavigationGraph_PathTo_ScreenNotFound(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.SetCurrent("home")

	_, err := g.PathTo("nonexistent")
	assert.ErrorIs(t, err, ErrScreenNotFound)
}

func TestNavigationGraph_PathTo_NoCurrent(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))

	_, err := g.PathTo("home")
	assert.ErrorIs(t, err, ErrScreenNotFound)
}

func TestNavigationGraph_PathTo_EmptyGraph(t *testing.T) {
	g := NewNavigationGraph()
	_, err := g.PathTo("anything")
	assert.ErrorIs(t, err, ErrEmptyGraph)
}

func TestNavigationGraph_PathTo_ShortestPath(t *testing.T) {
	// Create a graph with a direct and indirect path
	// home -> settings -> about (indirect, 2 hops)
	// home -> about (direct, 1 hop)
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddScreen(makeScreen("about", "About", "info"))
	g.AddTransition("home", "settings", makeAction("click", "settings"))
	g.AddTransition("settings", "about", makeAction("click", "about"))
	g.AddTransition("home", "about", makeAction("click", "about_shortcut"))
	g.SetCurrent("home")

	path, err := g.PathTo("about")
	require.NoError(t, err)
	assert.Len(t, path, 1, "BFS should find shortest path")
	assert.Equal(t, "home", path[0].From)
	assert.Equal(t, "about", path[0].To)
}

func TestNavigationGraph_UnvisitedScreens(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddScreen(makeScreen("about", "About", "info"))

	assert.Len(t, g.UnvisitedScreens(), 3)

	g.SetCurrent("home")
	assert.Len(t, g.UnvisitedScreens(), 2)

	g.SetCurrent("settings")
	assert.Len(t, g.UnvisitedScreens(), 1)

	g.SetCurrent("about")
	assert.Len(t, g.UnvisitedScreens(), 0)
}

func TestNavigationGraph_UnvisitedScreens_Empty(t *testing.T) {
	g := NewNavigationGraph()
	assert.Empty(t, g.UnvisitedScreens())
}

func TestNavigationGraph_Coverage(t *testing.T) {
	g := NewNavigationGraph()
	assert.Equal(t, 0.0, g.Coverage())

	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	assert.Equal(t, 0.0, g.Coverage())

	g.SetCurrent("home")
	assert.Equal(t, 0.5, g.Coverage())

	g.SetCurrent("settings")
	assert.Equal(t, 1.0, g.Coverage())
}

func TestNavigationGraph_Coverage_DoubleVisit(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))

	g.SetCurrent("home")
	g.SetCurrent("home") // visit again
	assert.Equal(t, 0.5, g.Coverage(), "Double visit should not inflate coverage")
}

func TestNavigationGraph_Export(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddTransition("home", "settings", makeAction("click", "settings_btn"))
	g.SetCurrent("home")

	snapshot := g.Export()
	assert.Len(t, snapshot.Screens, 2)
	assert.Len(t, snapshot.Transitions, 1)
	assert.Equal(t, "home", snapshot.Current)
	assert.Equal(t, 0.5, snapshot.Coverage)
	assert.NotZero(t, snapshot.CreatedAt)
}

func TestNavigationGraph_Screens_Order(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("c", "C", "cat"))
	g.AddScreen(makeScreen("a", "A", "cat"))
	g.AddScreen(makeScreen("b", "B", "cat"))

	screens := g.Screens()
	assert.Equal(t, "c", screens[0].ID)
	assert.Equal(t, "a", screens[1].ID)
	assert.Equal(t, "b", screens[2].ID)
}

func TestNavigationGraph_Transitions_Copy(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddTransition("home", "settings", makeAction("click", "btn"))

	transitions1 := g.Transitions()
	transitions2 := g.Transitions()
	assert.Equal(t, transitions1, transitions2)

	// Modifying returned slice should not affect the graph
	if len(transitions1) > 0 {
		transitions1[0].From = "modified"
		transitions3 := g.Transitions()
		assert.Equal(t, "home", transitions3[0].From)
	}
}

func TestNavigationGraph_PathTo_ComplexGraph(t *testing.T) {
	//    home -> menu -> settings -> profile
	//       \              |
	//        +-> search -> help
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("home", "Home", "main"))
	g.AddScreen(makeScreen("menu", "Menu", "nav"))
	g.AddScreen(makeScreen("settings", "Settings", "settings"))
	g.AddScreen(makeScreen("profile", "Profile", "user"))
	g.AddScreen(makeScreen("search", "Search", "util"))
	g.AddScreen(makeScreen("help", "Help", "info"))

	g.AddTransition("home", "menu", makeAction("click", "menu_btn"))
	g.AddTransition("menu", "settings", makeAction("click", "settings"))
	g.AddTransition("settings", "profile", makeAction("click", "profile"))
	g.AddTransition("home", "search", makeAction("click", "search"))
	g.AddTransition("settings", "help", makeAction("click", "help_link"))
	g.AddTransition("search", "help", makeAction("click", "help"))
	g.SetCurrent("home")

	// Path to help: home->search->help (2 hops) vs home->menu->settings->help (3 hops)
	path, err := g.PathTo("help")
	require.NoError(t, err)
	assert.Len(t, path, 2, "Should find shortest path")
	assert.Equal(t, "home", path[0].From)
	assert.Equal(t, "search", path[0].To)
}

func TestNavigationGraph_EmptyFingerprint(t *testing.T) {
	g := NewNavigationGraph()
	s1 := analyzer.ScreenIdentity{ID: "s1", Fingerprint: ""}
	s2 := analyzer.ScreenIdentity{ID: "s2", Fingerprint: ""}

	g.AddScreen(s1)
	g.AddScreen(s2)
	assert.Len(t, g.Screens(), 2, "Empty fingerprints should not cause dedup")
}

// --- Stress Tests ---

func TestNavigationGraph_Stress_ManyNodes(t *testing.T) {
	g := NewNavigationGraph()

	// Add 500 screens
	for i := 0; i < 500; i++ {
		id := fmt.Sprintf("screen-%d", i)
		g.AddScreen(makeScreen(id, fmt.Sprintf("Screen %d", i), "test"))
	}
	assert.Len(t, g.Screens(), 500)

	// Add transitions creating a chain
	for i := 0; i < 499; i++ {
		from := fmt.Sprintf("screen-%d", i)
		to := fmt.Sprintf("screen-%d", i+1)
		g.AddTransition(from, to, makeAction("click", "next"))
	}
	assert.Len(t, g.Transitions(), 499)

	// Path from first to last
	g.SetCurrent("screen-0")
	path, err := g.PathTo("screen-499")
	require.NoError(t, err)
	assert.Len(t, path, 499)

	// Coverage
	assert.InDelta(t, 1.0/500.0, g.Coverage(), 0.001)
}

func TestNavigationGraph_Stress_ConcurrentAccess(t *testing.T) {
	g := NewNavigationGraph()

	// Pre-add some screens
	for i := 0; i < 100; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("Screen %d", i), "test"))
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 1000)

	// Concurrent writers: add transitions
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			from := fmt.Sprintf("s-%d", i)
			to := fmt.Sprintf("s-%d", i+1)
			g.AddTransition(from, to, makeAction("click", fmt.Sprintf("btn-%d", i)))
		}(i)
	}

	// Concurrent readers: read screens and coverage
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = g.Screens()
			_ = g.Coverage()
			_ = g.Transitions()
			_ = g.UnvisitedScreens()
			_ = g.Export()
		}()
	}

	// Concurrent set-current
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			g.SetCurrent(fmt.Sprintf("s-%d", i))
		}(i)
	}

	// Concurrent add-screen
	for i := 100; i < 150; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("Screen %d", i), "test"))
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent error: %v", err)
	}

	// Verify graph is still consistent
	screens := g.Screens()
	assert.GreaterOrEqual(t, len(screens), 100)
}

func TestNavigationGraph_Stress_ConcurrentPathfinding(t *testing.T) {
	g := NewNavigationGraph()

	// Create a star graph: center -> spoke1, center -> spoke2, ...
	g.AddScreen(makeScreen("center", "Center", "main"))
	for i := 0; i < 50; i++ {
		id := fmt.Sprintf("spoke-%d", i)
		g.AddScreen(makeScreen(id, fmt.Sprintf("Spoke %d", i), "leaf"))
		g.AddTransition("center", id, makeAction("click", fmt.Sprintf("btn-%d", i)))
	}
	g.SetCurrent("center")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			target := fmt.Sprintf("spoke-%d", i)
			path, err := g.PathTo(target)
			assert.NoError(t, err)
			assert.Len(t, path, 1)
		}(i)
	}
	wg.Wait()
}

func TestNavigationGraph_Stress_500Nodes_Coverage(t *testing.T) {
	g := NewNavigationGraph()

	for i := 0; i < 500; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("S%d", i), "test"))
	}

	// Visit half
	for i := 0; i < 250; i++ {
		g.SetCurrent(fmt.Sprintf("s-%d", i))
	}

	assert.InDelta(t, 0.5, g.Coverage(), 0.01)
	assert.Len(t, g.UnvisitedScreens(), 250)
}

// --- Integration Tests ---

func TestNavigationGraph_Integration_FullWorkflow(t *testing.T) {
	g := NewNavigationGraph()

	// Simulate a QA session navigating through an app
	home := makeScreen("home", "Home Screen", "main")
	settings := makeScreen("settings", "Settings", "settings")
	about := makeScreen("about", "About", "info")
	editor := makeScreen("editor", "Text Editor", "editor")
	fileList := makeScreen("file-list", "File List", "files")

	g.AddScreen(home)
	g.AddScreen(settings)
	g.AddScreen(about)
	g.AddScreen(editor)
	g.AddScreen(fileList)

	g.AddTransition("home", "settings", makeAction("click", "settings_btn"))
	g.AddTransition("home", "editor", makeAction("click", "new_doc"))
	g.AddTransition("home", "file-list", makeAction("click", "open_doc"))
	g.AddTransition("settings", "about", makeAction("click", "about_btn"))
	g.AddTransition("settings", "home", makeAction("click", "back"))
	g.AddTransition("editor", "home", makeAction("click", "back"))
	g.AddTransition("file-list", "editor", makeAction("click", "file_item"))
	g.AddTransition("file-list", "home", makeAction("click", "back"))

	// Start at home
	g.SetCurrent("home")
	assert.Equal(t, 0.2, g.Coverage())

	// Navigate to settings
	g.SetCurrent("settings")
	assert.Equal(t, 0.4, g.Coverage())

	// Navigate to about
	g.SetCurrent("about")
	assert.Equal(t, 0.6, g.Coverage())

	// Go back to home
	g.SetCurrent("home")
	assert.Equal(t, 0.6, g.Coverage())

	// Check unvisited
	unvisited := g.UnvisitedScreens()
	assert.Len(t, unvisited, 2)
	assert.Contains(t, unvisited, "editor")
	assert.Contains(t, unvisited, "file-list")

	// Navigate to all remaining
	g.SetCurrent("editor")
	g.SetCurrent("file-list")
	assert.Equal(t, 1.0, g.Coverage())
	assert.Empty(t, g.UnvisitedScreens())

	// Export and verify
	snapshot := g.Export()
	assert.Len(t, snapshot.Screens, 5)
	assert.Len(t, snapshot.Transitions, 8)
	assert.Equal(t, 1.0, snapshot.Coverage)
}

func TestNavigationGraph_Integration_AnalyzerToGraph(t *testing.T) {
	// Simulate: analyzer identifies screen -> add to graph
	a := analyzer.NewStubAnalyzer()
	g := NewNavigationGraph()

	screens := [][]byte{
		[]byte("screenshot1"),
		[]byte("screenshot2"),
		[]byte("screenshot3"),
	}

	prevID := ""
	for _, img := range screens {
		identity, err := a.IdentifyScreen(context.Background(), img)
		require.NoError(t, err)

		id := g.AddScreen(identity)
		g.SetCurrent(id)

		if prevID != "" {
			g.AddTransition(prevID, id, makeAction("navigate", "next"))
		}
		prevID = id
	}

	assert.Len(t, g.Screens(), 3)
	assert.Len(t, g.Transitions(), 2)
	assert.Equal(t, 1.0, g.Coverage())
}

