# VisionEngine User Guide

## Installation

```bash
go get digital.vasic.visionengine
```

## Configuration

Set environment variables in `.env` or export them:

```bash
export HELIX_VISION_PROVIDER=auto
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...
```

See `.env.example` for all configuration options.

## Using NavigationGraph

```go
import (
    "digital.vasic.visionengine/pkg/analyzer"
    "digital.vasic.visionengine/pkg/graph"
)

// Create graph
g := graph.NewNavigationGraph()

// Add screens
g.AddScreen(analyzer.ScreenIdentity{ID: "home", Name: "Home Screen"})
g.AddScreen(analyzer.ScreenIdentity{ID: "settings", Name: "Settings"})

// Add transitions
g.AddTransition("home", "settings", analyzer.Action{Type: "click", Target: "gear"})

// Navigate
g.SetCurrent("home")
path, err := g.PathTo("settings")

// Export
dot := graph.ExportDOT(g)
jsonStr, _ := graph.ExportJSON(g)
mermaid := graph.ExportMermaid(g)
```

## Using LLM Vision Providers

```go
import "digital.vasic.visionengine/pkg/llmvision"

provider, _ := llmvision.NewOpenAIProvider(llmvision.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

result, err := provider.AnalyzeImage(ctx, screenshotBytes, "What do you see?")
```

## Fallback Chain

```go
openai, _ := llmvision.NewOpenAIProvider(openaiConfig)
anthropic, _ := llmvision.NewAnthropicProvider(anthropicConfig)
fallback, _ := llmvision.NewFallbackProvider(openai, anthropic)

// Automatically falls back to Anthropic if OpenAI fails
result, err := fallback.AnalyzeImage(ctx, img, "Describe this screen")
```
