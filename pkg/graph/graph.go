// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package graph provides a directed navigation graph for tracking screen transitions.
package graph

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"digital.vasic.visionengine/pkg/analyzer"
)

var (
	// ErrScreenNotFound is returned when a screen ID is not in the graph.
	ErrScreenNotFound = errors.New("screen not found")
	// ErrNoPath is returned when no path exists between two screens.
	ErrNoPath = errors.New("no path found")
	// ErrEmptyGraph is returned when the graph has no screens.
	ErrEmptyGraph = errors.New("graph is empty")
	// ErrSelfTransition is returned when from and to are the same screen.
	ErrSelfTransition = errors.New("self-transition not allowed")
	// ErrDuplicateScreen is returned when adding a screen that already exists.
	ErrDuplicateScreen = errors.New("screen already exists")
)

// NavigationGraph is the directed graph interface for screen navigation.
type NavigationGraph interface {
	// AddScreen adds a screen node and returns its ID.
	AddScreen(screen analyzer.ScreenIdentity) string
	// AddTransition adds a directed edge from one screen to another.
	AddTransition(from, to string, action analyzer.Action)
	// CurrentScreen returns the ID of the current screen.
	CurrentScreen() string
	// SetCurrent sets the current screen by ID.
	SetCurrent(screenID string)
	// PathTo returns the shortest path from the current screen to the target.
	PathTo(targetID string) ([]Transition, error)
	// UnvisitedScreens returns screen IDs that have not been visited.
	UnvisitedScreens() []string
	// Coverage returns the ratio of visited screens to total screens.
	Coverage() float64
	// Export returns a snapshot of the graph.
	Export() GraphSnapshot
	// Screens returns all screen nodes.
	Screens() []ScreenNode
	// Transitions returns all transitions.
	Transitions() []Transition
}

// ScreenNode represents a node in the navigation graph.
type ScreenNode struct {
	ID       string                  `json:"id"`
	Identity analyzer.ScreenIdentity `json:"identity"`
	Visited  bool                    `json:"visited"`
	VisitedAt time.Time              `json:"visited_at,omitempty"`
}

// Transition represents a directed edge in the navigation graph.
type Transition struct {
	From   string          `json:"from"`
	To     string          `json:"to"`
	Action analyzer.Action `json:"action"`
}

// GraphSnapshot is a serializable representation of the graph.
type GraphSnapshot struct {
	Screens     []ScreenNode `json:"screens"`
	Transitions []Transition `json:"transitions"`
	Current     string       `json:"current"`
	Coverage    float64      `json:"coverage"`
	CreatedAt   time.Time    `json:"created_at"`
}

// navGraph is the thread-safe implementation of NavigationGraph.
type navGraph struct {
	screens     map[string]*ScreenNode
	transitions []Transition
	adjacency   map[string][]Transition // from → transitions
	current     string
	order       []string // insertion order of screen IDs
	mu          sync.RWMutex
}

// NewNavigationGraph creates a new empty NavigationGraph.
func NewNavigationGraph() NavigationGraph {
	return &navGraph{
		screens:   make(map[string]*ScreenNode),
		adjacency: make(map[string][]Transition),
		order:     make([]string, 0),
	}
}

// AddScreen adds a screen node. Returns the screen's ID.
// If the screen already exists (by fingerprint), returns the existing ID.
func (g *navGraph) AddScreen(screen analyzer.ScreenIdentity) string {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check for existing screen by fingerprint
	for _, node := range g.screens {
		if node.Identity.Fingerprint != "" && node.Identity.Fingerprint == screen.Fingerprint {
			return node.ID
		}
	}

	// Check for existing screen by ID
	if _, exists := g.screens[screen.ID]; exists {
		return screen.ID
	}

	node := &ScreenNode{
		ID:       screen.ID,
		Identity: screen,
		Visited:  false,
	}
	g.screens[screen.ID] = node
	g.order = append(g.order, screen.ID)
	return screen.ID
}

// AddTransition adds a directed edge.
func (g *navGraph) AddTransition(from, to string, action analyzer.Action) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if from == to {
		return // silently ignore self-transitions
	}

	// Check for duplicate transition
	for _, t := range g.adjacency[from] {
		if t.To == to && t.Action.Type == action.Type && t.Action.Target == action.Target {
			return // duplicate
		}
	}

	t := Transition{
		From:   from,
		To:     to,
		Action: action,
	}
	g.transitions = append(g.transitions, t)
	g.adjacency[from] = append(g.adjacency[from], t)
}

// CurrentScreen returns the current screen ID.
func (g *navGraph) CurrentScreen() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.current
}

// SetCurrent sets the current screen and marks it as visited.
func (g *navGraph) SetCurrent(screenID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.current = screenID
	if node, ok := g.screens[screenID]; ok {
		if !node.Visited {
			node.Visited = true
			node.VisitedAt = time.Now()
		}
	}
}

// PathTo finds the shortest path from the current screen to the target using BFS.
func (g *navGraph) PathTo(targetID string) ([]Transition, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.screens) == 0 {
		return nil, ErrEmptyGraph
	}
	if g.current == "" {
		return nil, fmt.Errorf("%w: no current screen set", ErrScreenNotFound)
	}
	if _, ok := g.screens[targetID]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrScreenNotFound, targetID)
	}
	if g.current == targetID {
		return []Transition{}, nil
	}

	return g.bfs(g.current, targetID)
}

// bfs performs breadth-first search. Must be called with at least RLock held.
func (g *navGraph) bfs(start, target string) ([]Transition, error) {
	visited := make(map[string]bool)
	parent := make(map[string]Transition)
	queue := []string{start}
	visited[start] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, t := range g.adjacency[current] {
			if visited[t.To] {
				continue
			}
			visited[t.To] = true
			parent[t.To] = t

			if t.To == target {
				// Reconstruct path
				return g.reconstructPath(parent, start, target), nil
			}
			queue = append(queue, t.To)
		}
	}

	return nil, fmt.Errorf("%w: from %s to %s", ErrNoPath, start, target)
}

// reconstructPath builds the path from BFS parent map.
func (g *navGraph) reconstructPath(parent map[string]Transition, start, target string) []Transition {
	path := make([]Transition, 0)
	current := target
	for current != start {
		t := parent[current]
		path = append(path, t)
		current = t.From
	}
	// Reverse path
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// UnvisitedScreens returns IDs of screens that haven't been visited.
func (g *navGraph) UnvisitedScreens() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]string, 0)
	for _, id := range g.order {
		if !g.screens[id].Visited {
			result = append(result, id)
		}
	}
	return result
}

// Coverage returns the ratio of visited to total screens.
func (g *navGraph) Coverage() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.screens) == 0 {
		return 0.0
	}

	visited := 0
	for _, node := range g.screens {
		if node.Visited {
			visited++
		}
	}
	return float64(visited) / float64(len(g.screens))
}

// Export returns a snapshot of the graph.
func (g *navGraph) Export() GraphSnapshot {
	g.mu.RLock()
	defer g.mu.RUnlock()

	screens := make([]ScreenNode, 0, len(g.screens))
	for _, id := range g.order {
		screens = append(screens, *g.screens[id])
	}

	transitions := make([]Transition, len(g.transitions))
	copy(transitions, g.transitions)

	return GraphSnapshot{
		Screens:     screens,
		Transitions: transitions,
		Current:     g.current,
		Coverage:    g.coverageUnsafe(),
		CreatedAt:   time.Now(),
	}
}

// Screens returns all screen nodes in insertion order.
func (g *navGraph) Screens() []ScreenNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]ScreenNode, 0, len(g.screens))
	for _, id := range g.order {
		result = append(result, *g.screens[id])
	}
	return result
}

// Transitions returns all transitions.
func (g *navGraph) Transitions() []Transition {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]Transition, len(g.transitions))
	copy(result, g.transitions)
	return result
}

// coverageUnsafe computes coverage without locking. Must hold at least RLock.
func (g *navGraph) coverageUnsafe() float64 {
	if len(g.screens) == 0 {
		return 0.0
	}
	visited := 0
	for _, node := range g.screens {
		if node.Visited {
			visited++
		}
	}
	return float64(visited) / float64(len(g.screens))
}
