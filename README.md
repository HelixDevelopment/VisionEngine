# VisionEngine

Computer vision and LLM Vision for UI analysis and navigation graph building.

## Overview

VisionEngine provides:

- **Analyzer interface** for screen analysis, element detection, text detection, and visual issue detection
- **NavigationGraph** for tracking app screen navigation with BFS pathfinding and DOT/JSON/Mermaid export
- **LLM Vision providers** for GPT-4o, Claude, Gemini, and Qwen-VL
- **OpenCV stubs** with build-tag gating for optional GoCV integration

## Quick Start

```bash
# Run tests (no OpenCV required)
go test ./... -race -count=1

# Build
go build ./...

# With OpenCV support
go build -tags vision ./...
```

## Packages

| Package | Description |
|---------|-------------|
| `pkg/analyzer` | Analyzer interface, types (UIElement, ScreenAnalysis, ScreenDiff, etc.) |
| `pkg/graph` | NavigationGraph with BFS pathfinding and DOT/JSON/Mermaid export |
| `pkg/llmvision` | VisionProvider interface with OpenAI, Anthropic, Gemini, Qwen adapters |
| `pkg/opencv` | OpenCV stub implementations (real impl behind `vision` build tag) |
| `pkg/config` | Configuration management via environment variables |

## NavigationGraph

The most important package -- imported by HelixQA for autonomous QA sessions.

```go
g := graph.NewNavigationGraph()
g.AddScreen(analyzer.ScreenIdentity{ID: "home", Name: "Home"})
g.AddScreen(analyzer.ScreenIdentity{ID: "settings", Name: "Settings"})
g.AddTransition("home", "settings", analyzer.Action{Type: "click", Target: "gear"})
g.SetCurrent("home")

path, _ := g.PathTo("settings")
fmt.Println(graph.ExportMermaid(g))
```

## License

Apache License 2.0
