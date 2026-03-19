# Changelog

## [0.1.0] - 2026-03-19

### Added
- Core `Analyzer` interface with `StubAnalyzer` implementation
- `NavigationGraph` with BFS pathfinding, coverage tracking, DOT/JSON/Mermaid export
- LLM Vision providers: OpenAI (GPT-4o), Anthropic (Claude), Google (Gemini), Qwen-VL
- `FallbackProvider` for multi-provider resilience
- OpenCV stub implementations with build-tag gating
- Configuration via environment variables
- Comprehensive test suite (~190 tests)
- Documentation: README, ARCHITECTURE, API_REFERENCE, CONTRIBUTING
