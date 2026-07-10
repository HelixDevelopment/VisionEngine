// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExportDOT exports the graph in Graphviz DOT format.
func ExportDOT(g NavigationGraph) string {
	var sb strings.Builder
	sb.WriteString("digraph navigation {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=rounded];\n\n")

	screens := g.Screens()
	current := g.CurrentScreen()

	for _, s := range screens {
		label := sanitizeDOTLabel(s.Identity.Name)
		if label == "" {
			label = sanitizeDOTLabel(s.ID)
		}
		attrs := fmt.Sprintf("label=\"%s\"", label)
		if s.Visited {
			attrs += ", style=\"rounded,filled\", fillcolor=\"#90EE90\""
		}
		if s.ID == current {
			attrs += ", penwidth=3"
		}
		sb.WriteString(fmt.Sprintf("  \"%s\" [%s];\n", sanitizeDOTLabel(s.ID), attrs))
	}

	sb.WriteString("\n")

	transitions := g.Transitions()
	for _, t := range transitions {
		label := sanitizeDOTLabel(t.Action.Type)
		if t.Action.Target != "" {
			label += ": " + sanitizeDOTLabel(t.Action.Target)
		}
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\"];\n",
			sanitizeDOTLabel(t.From), sanitizeDOTLabel(t.To), label))
	}

	sb.WriteString("}\n")
	return sb.String()
}

// ExportJSON exports the graph as a JSON string.
func ExportJSON(g NavigationGraph) (string, error) {
	snapshot := g.Export()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal graph: %w", err)
	}
	return string(data), nil
}

// ExportMermaid exports the graph in Mermaid flowchart format.
func ExportMermaid(g NavigationGraph) string {
	var sb strings.Builder
	sb.WriteString("graph LR\n")

	screens := g.Screens()
	current := g.CurrentScreen()

	// mermaidIDs deduplicates sanitizeMermaidID's output within this
	// export call. sanitizeMermaidID alone is lossy — e.g. "a.b" and
	// "a-b" both sanitize to "a_b" — which would silently merge two
	// DISTINCT navigation-graph screens into a single node in the
	// rendered diagram (any edge to the "lost" screen would then point
	// at the wrong node). ids() below assigns a numeric disambiguating
	// suffix on collision so every distinct input ID maps to a distinct
	// Mermaid node ID; screens are processed first so their natural
	// sanitized ID wins ties over IDs only ever seen in a transition.
	ids := newMermaidIDMapper()

	for _, s := range screens {
		label := sanitizeMermaidLabel(s.Identity.Name)
		if label == "" {
			label = sanitizeMermaidLabel(s.ID)
		}
		id := ids.get(s.ID)
		sb.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", id, label))
	}

	transitions := g.Transitions()
	for _, t := range transitions {
		fromID := ids.get(t.From)
		toID := ids.get(t.To)
		label := sanitizeMermaidLabel(t.Action.Type)
		if t.Action.Target != "" {
			label += ": " + sanitizeMermaidLabel(t.Action.Target)
		}
		sb.WriteString(fmt.Sprintf("  %s -->|\"%s\"| %s\n", fromID, label, toID))
	}

	// Style visited and current nodes
	for _, s := range screens {
		id := ids.get(s.ID)
		if s.Visited {
			sb.WriteString(fmt.Sprintf("  style %s fill:#90EE90\n", id))
		}
		if s.ID == current {
			sb.WriteString(fmt.Sprintf("  style %s stroke-width:3px\n", id))
		}
	}

	return sb.String()
}

// mermaidIDMapper assigns a stable, collision-free Mermaid node ID to
// each distinct raw screen ID seen during one ExportMermaid call. See
// the "audit round 2026-07-10" comment in ExportMermaid for the defect
// this closes.
type mermaidIDMapper struct {
	used map[string]bool
	ids  map[string]string
}

func newMermaidIDMapper() *mermaidIDMapper {
	return &mermaidIDMapper{used: make(map[string]bool), ids: make(map[string]string)}
}

// get returns the Mermaid-safe node ID for raw, assigning + caching a
// disambiguated one on first sight so repeat calls for the SAME raw ID
// always return the SAME node ID, while two DIFFERENT raw IDs never
// collide even if sanitizeMermaidID maps them to the same base string.
func (m *mermaidIDMapper) get(raw string) string {
	if id, ok := m.ids[raw]; ok {
		return id
	}
	base := sanitizeMermaidID(raw)
	candidate := base
	for n := 2; m.used[candidate]; n++ {
		candidate = fmt.Sprintf("%s_%d", base, n)
	}
	m.used[candidate] = true
	m.ids[raw] = candidate
	return candidate
}

// sanitizeDOTLabel escapes quotes and backslashes for DOT format.
func sanitizeDOTLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// sanitizeMermaidLabel escapes characters that break Mermaid syntax.
func sanitizeMermaidLabel(s string) string {
	s = strings.ReplaceAll(s, "\"", "'")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "[", "(")
	s = strings.ReplaceAll(s, "]", ")")
	return s
}

// sanitizeMermaidID creates a valid Mermaid node ID.
func sanitizeMermaidID(s string) string {
	replacer := strings.NewReplacer(
		" ", "_",
		"-", "_",
		".", "_",
		"/", "_",
		":", "_",
		"(", "_",
		")", "_",
		"[", "_",
		"]", "_",
		"\"", "_",
		"'", "_",
	)
	id := replacer.Replace(s)
	if id == "" {
		return "node"
	}
	return id
}
