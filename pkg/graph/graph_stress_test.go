// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"digital.vasic.visionengine/pkg/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStress_LargeGraphCreation(t *testing.T) {
	g := NewNavigationGraph()
	nodeCount := 500
	start := time.Now()

	for i := 0; i < nodeCount; i++ {
		g.AddScreen(makeScreen(
			fmt.Sprintf("screen-%d", i),
			fmt.Sprintf("Screen %d", i),
			fmt.Sprintf("category-%d", i%10),
		))
	}

	elapsed := time.Since(start)
	assert.Len(t, g.Screens(), nodeCount)
	assert.Less(t, elapsed, 2*time.Second, "500 nodes should be created in under 2 seconds")
}

func TestStress_LargeGraphWithTransitions(t *testing.T) {
	g := NewNavigationGraph()
	nodeCount := 500

	for i := 0; i < nodeCount; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("S%d", i), "test"))
	}

	// Create random transitions
	rng := rand.New(rand.NewSource(42))
	transitionCount := 0
	for i := 0; i < 2000; i++ {
		from := fmt.Sprintf("s-%d", rng.Intn(nodeCount))
		to := fmt.Sprintf("s-%d", rng.Intn(nodeCount))
		if from != to {
			g.AddTransition(from, to, makeAction("click", fmt.Sprintf("btn-%d", i)))
			transitionCount++
		}
	}

	assert.Greater(t, len(g.Transitions()), 0)
}

func TestStress_ConcurrentReadsWrites(t *testing.T) {
	g := NewNavigationGraph()

	// Pre-populate
	for i := 0; i < 50; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("S%d", i), "test"))
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	// 10 writers adding transitions
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				select {
				case <-done:
					return
				default:
					from := fmt.Sprintf("s-%d", (w*10+i)%50)
					to := fmt.Sprintf("s-%d", (w*10+i+1)%50)
					g.AddTransition(from, to, makeAction("click", fmt.Sprintf("w%d-btn-%d", w, i)))
				}
			}
		}(w)
	}

	// 10 readers querying
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				select {
				case <-done:
					return
				default:
					_ = g.Coverage()
					_ = g.Screens()
					_ = g.Transitions()
					_ = g.UnvisitedScreens()
				}
			}
		}()
	}

	// 5 navigators
	for n := 0; n < 5; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				g.SetCurrent(fmt.Sprintf("s-%d", (n*10+i)%50))
			}
		}(n)
	}

	wg.Wait()
	close(done)

	// Graph should still be consistent
	screens := g.Screens()
	assert.Len(t, screens, 50)
}

func TestStress_RapidAddRemoveScreens(t *testing.T) {
	g := NewNavigationGraph()
	var wg sync.WaitGroup

	// Rapidly add screens from multiple goroutines
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				id := fmt.Sprintf("g%d-s%d", i, j)
				g.AddScreen(makeScreen(id, fmt.Sprintf("Screen %s", id), "stress"))
			}
		}(i)
	}

	wg.Wait()
	assert.Len(t, g.Screens(), 500)
}

func TestStress_PathfindingLargeGraph(t *testing.T) {
	g := NewNavigationGraph()

	// Create a chain of 200 nodes
	for i := 0; i < 200; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("chain-%d", i), fmt.Sprintf("Chain %d", i), "chain"))
	}
	for i := 0; i < 199; i++ {
		g.AddTransition(fmt.Sprintf("chain-%d", i), fmt.Sprintf("chain-%d", i+1), makeAction("next", ""))
	}

	g.SetCurrent("chain-0")
	start := time.Now()
	path, err := g.PathTo("chain-199")
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, path, 199)
	assert.Less(t, elapsed, time.Second, "BFS on 200 nodes should be under 1 second")
}

func TestStress_ConcurrentExport(t *testing.T) {
	g := NewNavigationGraph()
	for i := 0; i < 100; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("S%d", i), "test"))
	}
	for i := 0; i < 99; i++ {
		g.AddTransition(fmt.Sprintf("s-%d", i), fmt.Sprintf("s-%d", i+1), makeAction("click", "next"))
	}
	g.SetCurrent("s-0")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = ExportDOT(g)
		}()
		go func() {
			defer wg.Done()
			_, _ = ExportJSON(g)
		}()
		go func() {
			defer wg.Done()
			_ = ExportMermaid(g)
		}()
	}
	wg.Wait()
}

func TestStress_500Nodes_FullCoverage(t *testing.T) {
	g := NewNavigationGraph()

	for i := 0; i < 500; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("s-%d", i), fmt.Sprintf("S%d", i), "test"))
	}

	assert.Equal(t, 0.0, g.Coverage())
	assert.Len(t, g.UnvisitedScreens(), 500)

	for i := 0; i < 500; i++ {
		g.SetCurrent(fmt.Sprintf("s-%d", i))
	}

	assert.Equal(t, 1.0, g.Coverage())
	assert.Empty(t, g.UnvisitedScreens())
}

func TestStress_BFSOnDenseGraph(t *testing.T) {
	g := NewNavigationGraph()

	// Create a dense graph (30 nodes, fully connected)
	n := 30
	for i := 0; i < n; i++ {
		g.AddScreen(makeScreen(fmt.Sprintf("d-%d", i), fmt.Sprintf("D%d", i), "dense"))
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i != j {
				g.AddTransition(fmt.Sprintf("d-%d", i), fmt.Sprintf("d-%d", j),
					analyzer.Action{Type: "direct", Target: fmt.Sprintf("d%d", j)})
			}
		}
	}

	g.SetCurrent("d-0")

	// All nodes should be reachable in 1 hop
	for i := 1; i < n; i++ {
		path, err := g.PathTo(fmt.Sprintf("d-%d", i))
		require.NoError(t, err)
		assert.Len(t, path, 1, "Direct connection should give 1-hop path")
	}
}

func TestStress_ConcurrentPathfinding(t *testing.T) {
	g := NewNavigationGraph()

	// Create a grid-like graph
	size := 20
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			id := fmt.Sprintf("grid-%d-%d", r, c)
			g.AddScreen(makeScreen(id, id, "grid"))
			if c > 0 {
				g.AddTransition(fmt.Sprintf("grid-%d-%d", r, c-1), id, makeAction("right", ""))
			}
			if r > 0 {
				g.AddTransition(fmt.Sprintf("grid-%d-%d", r-1, c), id, makeAction("down", ""))
			}
		}
	}
	g.SetCurrent("grid-0-0")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := (i%size + 1) % size // avoid 0,0 (current screen) which gives empty path
			c := ((i*3)%size + 1) % size
			if r == 0 && c == 0 {
				r = 1
			}
			target := fmt.Sprintf("grid-%d-%d", r, c)
			path, err := g.PathTo(target)
			assert.NoError(t, err)
			assert.NotEmpty(t, path, "Path to %s from grid-0-0 should not be empty", target)
		}(i)
	}
	wg.Wait()
}
