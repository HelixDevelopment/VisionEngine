## INHERITED FROM Helix Constitution

This module is a submodule of the consuming project that
includes the Helix Constitution submodule at the parent's
`constitution/` path. All rules in `constitution/CLAUDE.md` and the
`constitution/Constitution.md` it references (universal anti-bluff
covenant §11.4, no-guessing mandate §11.4.6, credentials-handling
mandate §11.4.10, host-session safety §12, data safety §9, mutation-
paired gates §1.1) apply unconditionally to every change landed here.
The module-specific rules below extend them — they never weaken any
universal clause.

When this file disagrees with the constitution submodule, the
constitution wins. Locate the constitution submodule from any
arbitrary nested depth using its `find_constitution.sh` helper.

Canonical reference: <https://github.com/HelixDevelopment/HelixConstitution>

---

# CLAUDE.md — VisionEngine AI Agent Manual

## INHERITED FROM constitution/CLAUDE.md

All rules in `constitution/CLAUDE.md` (and the `constitution/Constitution.md` it references) apply unconditionally. This file's rules below extend them — they MUST NOT weaken any inherited rule. Use `constitution/find_constitution.sh` from the parent project root to resolve the absolute path of the submodule from any nested location.

## VisionEngine — AI Agent Operating Manual

**Version**: 0.1.0
**Date**: 2026-07-05 (last revision of this document)
**Scope**: This document guides AI agents working on this module's own codebase.
**Authority**: Cascaded from the parent project's root `CLAUDE.md`, with module-specific addenda below.

> **Versioning note**: this module ships no `VERSION` file and `go.mod` carries
> no semantic-version directive. The only concrete version marker found in
> the repository is `CHANGELOG.md`'s `[0.1.0] - 2026-03-19` entry, so `0.1.0`
> is used here (and in `AGENTS.md`) for consistency. Treat it as the best
> available signal, not an authoritative release tag, until a canonical
> `VERSION` file is introduced.

---

## 1. Module Identity & Purpose

VisionEngine (Go module `digital.vasic.visionengine`) is a standalone,
project-not-aware computer-vision + LLM-vision toolkit for UI analysis and
navigation-graph construction. It is decoupled from any consuming project by
design: it imports nothing from a consumer and is meant to be added as an
equal-codebase submodule by any project that needs screenshot/UI analysis.

**Your mandate as an agent here**: write real, working, tested code. No
simulations, no hardcoded fixture responses standing in for a real
implementation. `pkg/analyzer/stub.go`'s `StubAnalyzer` intentionally returns
an explicit error when no real backend (the `vision` build tag or a wired LLM
vision provider) is configured, instead of a fabricated screen description —
do not "fix" that by reintroducing a hardcoded placeholder result.

## 2. Technology Stack

- **Language**: Go. Module directive `go 1.25.3` per `go.mod` — this is the
  single authoritative Go version for this module; every reference to a Go
  version in this document and in `AGENTS.md` MUST match it.
- **Module path**: `digital.vasic.visionengine` (`go.mod` line 1 — the
  authoritative module path; there is no other module path anywhere in this
  repository).
- **Direct dependencies** (`go.mod`):
  - `github.com/stretchr/testify v1.11.1` — test assertions.
  - `gocv.io/x/gocv v0.43.0` — OpenCV bindings, compiled only under the
    `vision` build tag.
  - `golang.org/x/crypto v0.51.0` — used by `pkg/remote`'s SSH client.
  - Indirect: `davecgh/go-spew`, `kr/pretty`, `pmezard/go-difflib`,
    `rogpeppe/go-internal`, `golang.org/x/sys`, `gopkg.in/check.v1`,
    `gopkg.in/yaml.v3`.
- This module has **no** database, cache, web-framework, or GUI-toolkit
  dependency of any kind.

## 3. Package Structure

Real, verified package tree (`find pkg -maxdepth 2 -type d`):

```
pkg/
├── analyzer/     # Analyzer interface, VideoProcessor interface, value types, StubAnalyzer
├── config/       # env-var configuration loader + validator
├── graph/        # NavigationGraph: BFS pathfinding + DOT/JSON/Mermaid export
├── i18n/         # Translator interface + NoopTranslator + bundles/
│   └── bundles/  # active.en.yaml message bundle
├── llmvision/    # VisionProvider interface + cloud/local adapters + FallbackChain
├── opencv/       # OpenCV stubs (default) / real GoCV bindings (-tags vision)
└── remote/       # SSH-driven remote/distributed Ollama + llama.cpp worker management
```

| Package | Purpose |
|---|---|
| `pkg/analyzer` | `Analyzer` interface, value types (`UIElement`, `ScreenAnalysis`, `ScreenDiff`, `Rect`, `Size`, `TextRegion`, `VisualIssue`, `ScreenIdentity`, `Action`, `KeyFrame`), `StubAnalyzer` reference implementation |
| `pkg/graph` | `NavigationGraph` — BFS `PathTo`, `ExportDOT`/`ExportJSON`/`ExportMermaid`, coverage tracking |
| `pkg/llmvision` | `VisionProvider` interface + adapters: OpenAI, Anthropic, Gemini, Qwen, Kimi, StepFun/StepGUI, Astica, Ollama; `FallbackProvider`/`FallbackChain` composer |
| `pkg/opencv` | OpenCV stubs (default build) + real GoCV bindings behind `-tags vision` |
| `pkg/config` | Env-driven configuration loader + i18n-routed validation |
| `pkg/i18n` | Minimal, dependency-free `Translator` interface + `NoopTranslator` default |
| `pkg/remote` | SSH-driven remote/distributed Ollama + llama.cpp-RPC worker management |

Outside `pkg/`:

| Path | Purpose |
|---|---|
| `cmd/visiondescribe/` | CLI: turns a screenshot into a structured JSON UI description via `pkg/llmvision` |
| `internal/archdoc/` | Internal helper package that generates architecture documentation from source |
| `challenges/runner/` | Standalone runner exercising the public API end-to-end with captured evidence |
| `challenges/scripts/` | Shell Challenge scripts (chaos-failure-injection, DDoS/health-flood, host-no-auto-suspend, no-suspend-calls, scaling, stress-sustained-load, terminal-UI interaction, UX end-to-end flow, and a paired-mutation "describe" Challenge) |

## 4. Build & Test — verified in this session

Run from the module root:

```bash
go build ./...                # verified: succeeds, no OpenCV required
go test -count=1 ./...        # verified: all packages pass
go test -race -count=1 ./...  # verified: passes with the race detector
go vet ./...                  # verified: clean
```

`go test -count=1 ./...` reports `[no test files]` for `challenges/runner` and
`cmd/visiondescribe` (both are `main` entry points, not libraries) and `ok`
for every package under `pkg/` plus `internal/archdoc`.

With the `vision` build tag (real OpenCV bindings):

```bash
go build -tags vision ./...
```

This requires OpenCV4 development headers discoverable via `pkg-config`
(`opencv4.pc`). It was **not** confirmed to succeed in this remediation
session — the host used had no OpenCV4 dev package installed, and the build
failed with the expected `pkg-config … opencv4 … not found` error. That is a
host-environment prerequisite, not a defect in this module; confirm local
OpenCV availability before relying on `-tags vision`.

`Makefile` also wires: `make build`, `make build-vision`, `make test`,
`make test-race`, `make test-vision`, `make test-coverage`, `make vet`,
`make lint`, `make fmt`, `make tidy`, `make clean`, `make check`, `make all` —
plus two portable Definition-of-Done gates: `make no-silent-skips` (fails on
an unannotated test skip) and `make demo-all` / `make demo-one MOD=<name>`
(runs each module's acceptance demo, discovered from any `CLAUDE.md` in the
tree).

## 5. Anti-bluff / Definition of Done

- No mocks/fakes/placeholders outside unit tests.
- `StubAnalyzer` (`pkg/analyzer/stub.go`) is a real reference implementation,
  not a placeholder: with no OpenCV build or LLM vision provider wired, it
  returns an explicit error rather than a fabricated result.
- `challenges/scripts/visionengine_describe_challenge.sh` is a
  paired-mutation Challenge: normal mode runs the runner unmodified;
  `--mutate` plants a deliberate regression in a scratch copy of
  `pkg/graph/graph.go` and asserts the runner detects it. Verified in this
  session: `--mutate` exits `99` as documented. Note: in an environment with
  no vision provider/API key configured, both `go run ./challenges/runner/`
  and the script's normal mode legitimately exit non-zero (the `StubAnalyzer`
  refuses to fabricate a result) — that is the intended anti-bluff behaviour,
  not a broken build.
- `docs/test-coverage.md` maps exported symbols to the test/Challenge that
  covers them; any gap is listed honestly rather than pretended-covered.
- Decoupling self-check: grepping `pkg/` and `go.mod` for any consuming
  project's name should return nothing — this module has zero own-org
  submodule dependencies (`helix-deps.yaml`: `deps: []`).

## 6. Submodule Decoupling

This module MUST NOT import, hardcode, or otherwise depend on the identity,
directory layout, or assumptions of whatever project consumes it. It is
designed to be added as a submodule by any project that needs computer-vision
or LLM-vision UI analysis. Consumer-specific values (hostnames, individual
usernames, API keys) belong in that consumer's own `.env`, never committed
into this module's source or documentation — `docs/USAGE.md` and
`.env.example` use a generic placeholder for the example SSH/remote username
for this reason.

## 7. Configuration

Configuration is entirely env-var-driven (`pkg/config`, `.env.example`).
Representative variables — see `.env.example` for the full, current list:
`HELIX_VISION_PROVIDER`, `ASTICA_API_KEY`, `OPENAI_API_KEY`,
`ANTHROPIC_API_KEY`, `GOOGLE_API_KEY`, `QWEN_API_KEY`, `KIMI_API_KEY`,
`STEPFUN_API_KEY`, `HELIX_VISION_OPENCV_ENABLED`, `HELIX_VISION_TIMEOUT`,
`HELIX_OLLAMA_URL`, `HELIX_OLLAMA_MODEL`, `HELIX_VISION_HOSTS`,
`HELIX_VISION_USER` (a placeholder value — never a real individual's account
name), `HELIX_LLAMACPP_RPC_*`.

## 8. Integration Seams

| Direction | Notes |
|---|---|
| Upstream (this module imports) | None — zero own-org submodule dependencies |
| Downstream (consumers) | Any project needing screenshot/UI analysis or navigation-graph tracking can import this module's public `pkg/*` API |

## 9. Peer Documents in This Module

`README.md` is the primary source of truth for usage, build commands, and
architecture — keep this file consistent with it. Other governance/reference
docs present here: `CONSTITUTION.md`, `AGENTS.md`, `QWEN.md`,
`ARCHITECTURE.md` (root and `docs/`), `API_REFERENCE.md`, `CHANGELOG.md`,
`CONTRIBUTING.md`, `docs/USAGE.md`, `docs/test-coverage.md`,
`docs/HOST_POWER_MANAGEMENT.md`. There is no `CRUSH.md`, `setup.sh`,
`scripts/init-submodules.sh`, or `docs/issues/` tree in this module — do not
reference them.

## 10. Known Real Implementations vs. Gaps

- `pkg/analyzer`, `pkg/graph`, `pkg/llmvision`, `pkg/config`, `pkg/i18n`,
  `pkg/remote` are real, tested implementations (see `go test` output above).
- `pkg/opencv` ships working stubs by default; the real GoCV-backed path
  requires the `vision` build tag plus a host OpenCV4 installation.
- The `Analyzer` interface's `VideoProcessor` counterpart has no shipped
  implementation yet — see `docs/test-coverage.md` for the current,
  honestly-tracked list of gaps.
