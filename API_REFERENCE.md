# VisionEngine API Reference

## pkg/analyzer

### Interfaces

- `Analyzer` - Primary vision analysis interface
- `VideoProcessor` - Video analysis capabilities

### Types

- `Rect` - Bounding box rectangle
- `Size` - Image dimensions
- `UIElement` - Detected UI element
- `TextRegion` - Detected text region
- `VisualIssue` - Detected visual problem
- `Action` - Navigation/interaction action
- `ScreenIdentity` - Unique screen identifier
- `ScreenAnalysis` - Full screen analysis result
- `ScreenDiff` - Screen comparison result
- `KeyFrame` - Video key frame

### Functions

- `NewStubAnalyzer()` - Create stub analyzer (no OpenCV)
- `NewStubAnalyzerWithProvider(provider)` - Create stub with LLM vision

## pkg/graph

### Interfaces

- `NavigationGraph` - Directed graph of screens and transitions

### Types

- `ScreenNode` - Node in the graph
- `Transition` - Directed edge
- `GraphSnapshot` - Serializable graph state

### Functions

- `NewNavigationGraph()` - Create empty graph
- `ExportDOT(g)` - Export to Graphviz DOT
- `ExportJSON(g)` - Export to JSON
- `ExportMermaid(g)` - Export to Mermaid

## pkg/llmvision

### Interfaces

- `VisionProvider` - LLM vision API adapter

### Types

- `ProviderConfig` - Provider configuration
- `FallbackProvider` - Multi-provider with fallback

### Functions

- `NewOpenAIProvider(config)` - GPT-4o adapter
- `NewAnthropicProvider(config)` - Claude adapter
- `NewGeminiProvider(config)` - Gemini adapter
- `NewQwenProvider(config)` - Qwen-VL adapter
- `NewFallbackProvider(providers...)` - Fallback chain

## pkg/config

### Types

- `Config` - VisionEngine configuration

### Functions

- `DefaultConfig()` - Sensible defaults
- `LoadFromEnv()` - Load from environment variables
