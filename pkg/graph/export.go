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

	for _, s := range screens {
		label := sanitizeMermaidLabel(s.Identity.Name)
		if label == "" {
			label = sanitizeMermaidLabel(s.ID)
		}
		id := sanitizeMermaidID(s.ID)
		sb.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", id, label))
	}

	transitions := g.Transitions()
	for _, t := range transitions {
		fromID := sanitizeMermaidID(t.From)
		toID := sanitizeMermaidID(t.To)
		label := sanitizeMermaidLabel(t.Action.Type)
		if t.Action.Target != "" {
			label += ": " + sanitizeMermaidLabel(t.Action.Target)
		}
		sb.WriteString(fmt.Sprintf("  %s -->|\"%s\"| %s\n", fromID, label, toID))
	}

	// Style visited and current nodes
	for _, s := range screens {
		id := sanitizeMermaidID(s.ID)
		if s.Visited {
			sb.WriteString(fmt.Sprintf("  style %s fill:#90EE90\n", id))
		}
		if s.ID == current {
			sb.WriteString(fmt.Sprintf("  style %s stroke-width:3px\n", id))
		}
	}

	return sb.String()
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
