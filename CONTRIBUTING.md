# Contributing to VisionEngine

## Development Setup

1. Clone the repository
2. Install Go 1.24+
3. Run `go mod tidy`
4. Run `make test-race`

## Code Standards

- All code must pass `go vet ./...`
- All tests must pass with `go test ./... -race -count=1`
- No test may be removed, disabled, or skipped
- Use table-driven tests where appropriate
- Thread safety: use sync.RWMutex for shared state

## Build Tags

- Default: No OpenCV dependency, stubs return errors
- `vision`: Full OpenCV/GoCV support

## Test Requirements

- Unit tests for all public functions
- Integration tests for cross-package flows
- Stress tests for concurrent operations
- Security tests for input validation
