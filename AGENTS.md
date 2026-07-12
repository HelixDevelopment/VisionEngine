## INHERITED FROM Helix Constitution

> Base agent rules live in the Helix Constitution submodule at the
> parent project's `constitution/AGENTS.md` and the universal
> `constitution/Constitution.md` it references. **READ THOSE FIRST.**
> The base file is authoritative for any topic not covered here.
> Module-specific rules below extend them; they never weaken them.

Critical universal rules every CLI agent (Claude Code, Cursor, Aider,
Codex, Gemini CLI) MUST honour while working in this module:

- **No bluffing.** Every PASS carries positive evidence. Constitution §11.4.
- **Mutation-paired gates.** Every new gate has a paired mutation
  proving it catches regressions. Constitution §1.1.
- **No guessing language** (`likely`, `probably`, `maybe`, `seems`).
  Constitution §11.4.6.
- **Credentials never tracked.** `.env` patterns git-ignored; runtime-load
  only. Constitution §11.4.10.
- **Never force-push.** Force-push requires explicit per-session
  authorization AND a green §9.1.5 post-op gate. Constitution §9.
- **CONTINUATION.md kept in sync** in every non-trivial commit.
  Constitution §12.10.
- **60% RAM cap.** Heavy work wrapped in bounded execution scope.
  Constitution §12.6.

Canonical reference: <https://github.com/HelixDevelopment/HelixConstitution>

---

# AGENTS.md — VisionEngine Agent Guide

## INHERITED FROM constitution/AGENTS.md

All rules in `constitution/AGENTS.md` (and the `constitution/Constitution.md` it references) apply unconditionally. This file's rules below extend them — they MUST NOT weaken any inherited rule. Use `constitution/find_constitution.sh` from the parent project root to resolve the absolute path of the submodule from any nested location.

## VisionEngine Agent Guidelines

**Version**: 0.1.0
**Date**: 2026-07-05 (last revision of this document)
**Scope**: All AI agents, human contributors, and automated processes working on this module.
**Authority**: Derived from the parent project's `AGENTS.md`, with module-specific enhancements below.

> **Versioning note**: this module ships no `VERSION` file and `go.mod`
> carries no semantic-version directive. The only concrete version marker
> found in the repository is `CHANGELOG.md`'s `[0.1.0] - 2026-03-19` entry,
> so `0.1.0` is used here (consistently with `CLAUDE.md`) rather than the
> previously self-contradicting `1.0.0` / `3.0.0` pair.

---

## Module Overview

VisionEngine is a standalone, project-not-aware Go module providing
computer-vision and LLM-vision building blocks for UI screenshot analysis and
navigation-graph construction. It provides four cooperating layers:

- **Analyzer** (`pkg/analyzer`) — `Analyzer` interface, `VideoProcessor`
  interface, value types, and a `StubAnalyzer` reference implementation.
- **NavigationGraph** (`pkg/graph`) — directed graph tracking app screen
  transitions, with BFS pathfinding and DOT/JSON/Mermaid export.
- **LLM Vision providers** (`pkg/llmvision`) — a `VisionProvider` interface
  and adapters for OpenAI, Anthropic, Gemini, Qwen, Kimi, StepFun/StepGUI,
  Astica, and Ollama, plus a `FallbackProvider` composer.
- **Configuration** (`pkg/config`) — env-var loader/validator, with every
  user-facing error routed through `pkg/i18n.Translator`.

It imports nothing from any consuming project. Project-specific behaviour
(hostnames, credentials, consumer identity) is injected at runtime via
environment variables and interfaces — never hardcoded into `pkg/` or
`cmd/` source.

---

## Technology Stack

**Core**:
- **Language**: Go, module directive `go 1.25.3` (`go.mod` — authoritative;
  every Go-version mention in this file and `CLAUDE.md` must agree with it).
- **Module path**: `digital.vasic.visionengine` (`go.mod` — authoritative;
  there is no other module path used anywhere in this repository).
- **Testing**: `stretchr/testify v1.11.1`.
- **Vision**: `gocv.io/x/gocv v0.43.0` (compiled only under the `vision`
  build tag).
- **Crypto/SSH**: `golang.org/x/crypto v0.51.0` (used by `pkg/remote`).

This module has no HTTP framework, no database, no cache, and no GUI
toolkit — it is a library plus one thin CLI (`cmd/visiondescribe`) and a
Challenge runner (`challenges/runner`).

---

## Working Directory & Build System

All build and test commands run from this module's own root — there is no
nested inner module and no separate subdirectory to `cd` into first.

### Build & Test Commands (verified in this remediation session)

| Command | Result |
|---|---|
| `go build ./...` | Verified: succeeds, no OpenCV required |
| `go test -count=1 ./...` | Verified: all packages `ok`; `challenges/runner` and `cmd/visiondescribe` report `[no test files]` (both are `main` entry points) |
| `go test -race -count=1 ./...` | Verified: passes with the race detector |
| `go vet ./...` | Verified: clean |
| `go build -tags vision ./...` | Requires OpenCV4 dev headers reachable via `pkg-config`; **not** confirmed to succeed in this session (host had no `opencv4.pc` — expected `pkg-config` failure, not a module defect) |

### Makefile targets

| Command | Purpose |
|---------|---------|
| `make build` | `go build ./...` |
| `make build-vision` | `go build -tags vision ./...` |
| `make test` | `go test ./... -count=1` |
| `make test-race` | `go test ./... -race -count=1` |
| `make test-vision` | `go test -tags vision ./... -race -count=1` |
| `make test-coverage` | Coverage report → `coverage.html` |
| `make vet` | `go vet ./...` |
| `make lint` | `golangci-lint run` (falls back to `go vet` if not installed) |
| `make fmt` | `gofmt -s -w .` |
| `make tidy` | `go mod tidy` |
| `make clean` | Removes coverage artifacts + Go build/test cache |
| `make check` | `vet` + `test-race` |
| `make all` | `tidy` + `fmt` + `vet` + `test-race` |
| `make no-silent-skips` | Definition-of-Done gate: fails on an unannotated test skip |
| `make demo-all` / `make demo-one MOD=<name>` | Runs each discovered module's acceptance demo |

### Single-test invocation

```bash
go test -v -run TestName ./pkg/<package>
go test -tags vision -run TestName ./pkg/opencv
```

### Challenge runner / Challenge scripts

```bash
go run ./challenges/runner/                                        # exercises the public API end-to-end
./challenges/scripts/visionengine_describe_challenge.sh             # normal mode
./challenges/scripts/visionengine_describe_challenge.sh --mutate    # paired-mutation proof — verified exit 99 in this session
```

Both the runner and the describe-Challenge's normal mode legitimately exit
non-zero when no vision backend (OpenCV build or an LLM vision provider key)
is configured — `StubAnalyzer` refuses to fabricate a result rather than
returning a hardcoded placeholder. That is intended anti-bluff behaviour, not
a broken build; it was observed in this session's verification run.

---

## Architecture & Code Organization

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

cmd/
└── visiondescribe/   # CLI: screenshot -> structured JSON UI description via pkg/llmvision

internal/
└── archdoc/          # Generates architecture documentation from source (not part of the public API)

challenges/
├── runner/            # End-to-end evidence runner exercising the public API
└── scripts/           # Shell Challenges: chaos, DDoS/health-flood, host-no-auto-suspend,
                        #   no-suspend-calls, scaling, stress, terminal-UI, UX-flow,
                        #   and the paired-mutation "describe" Challenge
```

### `pkg/llmvision` providers

| Provider | File |
|---|---|
| OpenAI | `openai.go` |
| Anthropic | `anthropic.go` |
| Gemini | `gemini.go` |
| Qwen | `qwen.go` |
| Kimi | `kimi.go` |
| StepFun/StepGUI | `stepgui.go` |
| Astica | `astica.go` |
| Ollama | `ollama.go` |
| Fallback composer | `fallback.go` |

`pkg/remote` additionally manages SSH-driven deployment (`deployer.go`,
`ssh.go`) and distributed/llama.cpp-RPC inference across multiple hosts
(`distributed.go`, `remote.go`).

---

## Verified Real Implementations

Confirmed in this session via `go build ./...`, `go test -count=1 ./...`,
`go test -race -count=1 ./...`, and `go vet ./...` — all clean:

- `pkg/analyzer` — `StubAnalyzer` reference implementation + value types.
- `pkg/graph` — `NavigationGraph` with unit, integration, security, stress,
  and automation test files (`graph_*_test.go`).
- `pkg/llmvision` — provider adapters + `FallbackProvider`.
- `pkg/config` — env-var loader + validator.
- `pkg/i18n` — `Translator` + `NoopTranslator`.
- `pkg/remote` — deployer + SSH + distributed-worker management.
- `internal/archdoc` — architecture-doc generator (internal helper, not part
  of the public API).

## Known Gaps (honestly tracked, not hidden)

- `pkg/opencv`'s real GoCV-backed path (`-tags vision`) requires a host
  OpenCV4 installation reachable via `pkg-config`; this was not available on
  the host used for this remediation, so `-tags vision` builds/tests were
  **not** verified to pass here.
- The `Analyzer` interface's `VideoProcessor` counterpart has no shipped
  implementation yet.
- See `docs/test-coverage.md` for the current symbol-to-test coverage ledger.

---

## Configuration

Configuration is entirely env-var-driven (`pkg/config`, `.env.example`).
Representative variables (see `.env.example` for the authoritative, current
list):

```bash
HELIX_VISION_PROVIDER=auto        # auto, astica, openai, anthropic, gemini, qwen, kimi, stepgui, ollama
ASTICA_API_KEY=...
OPENAI_API_KEY=...
ANTHROPIC_API_KEY=...
GOOGLE_API_KEY=...
QWEN_API_KEY=...
KIMI_API_KEY=...
STEPFUN_API_KEY=...
HELIX_VISION_OPENCV_ENABLED=true
HELIX_VISION_TIMEOUT=60
HELIX_OLLAMA_URL=http://<remote-host>:11434
HELIX_OLLAMA_MODEL=minicpm-v:8b
HELIX_VISION_HOSTS=<host1>,<host2>
HELIX_VISION_USER=<your-username>   # placeholder — never a real individual's account name
HELIX_LLAMACPP_RPC_ENABLED=false
```

`.env` is git-ignored (see `.gitignore`); only `.env.example` is committed.

---

## Testing Strategy

### Test categories present in this module

1. **Unit tests** — `*_test.go` beside each package; mocks/fakes permitted
   only here.
2. **Integration / security / stress / automation tests** — present under
   `pkg/graph/` (`graph_integration_test.go`, `graph_security_test.go`,
   `graph_stress_test.go`, `graph_automation_test.go`).
3. **End-to-end evidence runner** — `challenges/runner/main.go`, invoked via
   `go run ./challenges/runner/`.
4. **Shell Challenges** — `challenges/scripts/*.sh`, covering chaos-failure
   injection, DDoS/health-flood, host-suspend guard, scaling, sustained-load
   stress, terminal-UI interaction, UX end-to-end flow, and the
   paired-mutation describe-Challenge.

### Anti-bluff testing rules

- Unit tests: mocks/fakes OK.
- All other test types: exercise real code paths — no fakes.
- No bare `t.Skip()` without an accompanying ticket annotation (enforced by
  `make no-silent-skips`).
- The describe-Challenge's `--mutate` mode proves its own gate is not a
  bluff: it plants a regression and asserts the runner catches it.

---

## Code Style & Development Conventions

- Standard Go formatting: `gofmt -s -w .` (wired as `make fmt`).
- Linting: `golangci-lint run` where available, else `go vet ./...`
  (`make lint`).
- Table-driven tests with `t.Run()` subtests, consistent with the existing
  `*_test.go` files.
- Build tags gate OpenCV: default build ships stubs (`pkg/opencv/stub.go`);
  `-tags vision` compiles the real GoCV-backed files.
- Every user-facing string is routed through `pkg/i18n.Translator` rather
  than hardcoded — see `i18n_defaults.go` / `i18n_callsites_test.go` in each
  package that has user-facing output.

---

## Submodule Decoupling

This module MUST NOT hardcode the identity, directory layout, or
assumptions of whatever project consumes it — it is designed to be added as
an equal-codebase submodule by any project needing computer-vision or
LLM-vision UI analysis. `helix-deps.yaml` declares zero own-org submodule
dependencies for this module itself. Consumer-specific values (hostnames,
individual usernames, API keys) belong in the consumer's own `.env`; example
values in this module's own docs use generic placeholders, never a real
individual's account name.

---

## Common Issues

1. **`go build -tags vision` fails with `pkg-config … opencv4 … not found`**:
   install OpenCV4 development headers + `pkg-config` for your platform, or
   use the default (non-`vision`-tagged) build, which requires neither.
2. **`go run ./challenges/runner/` / the describe-Challenge's normal mode
   exit non-zero**: expected when no vision provider API key and no
   `-tags vision` OpenCV build are configured — this is the intended
   anti-bluff refusal, not a broken build.
3. **Test skips**: any `t.Skip()` needs its ticket annotation, or
   `make no-silent-skips` will flag it.

---

## Resources & References

- **Primary source of truth**: `README.md`.
- **Constitution**: `CONSTITUTION.md` (inheritance pointer to the parent
  project's constitution submodule).
- **CLAUDE.md**: `CLAUDE.md` (sibling agent manual — keep in sync with this
  file).
- **QWEN.md**: `QWEN.md` (Qwen Code CLI entry point — keep in sync).
- **Architecture**: `ARCHITECTURE.md` (root) and `docs/ARCHITECTURE.md`.
- **API reference**: `API_REFERENCE.md`.
- **Usage guide**: `docs/USAGE.md`.
- **Test-coverage ledger**: `docs/test-coverage.md`.
- **Changelog**: `CHANGELOG.md`.

There is no `CRUSH.md`, `setup.sh`, `scripts/init-submodules.sh`, or
`docs/issues/` tree in this module — do not reference them.


---

## Constitutional Anti-Bluff Forensic Anchor (CONST-035 / §11.9, inherited)

> Verbatim user mandate: *"We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used! This MUST NOT be the case and execution of tests and Challenges MUST guarantee the quality, the completion and full usability by end users of the product!"*
>
> Operative rule: **The bar for shipping is not "tests pass" but "users can use the feature."** Every PASS in this codebase MUST carry positive runtime evidence captured during execution. Metadata-only / configuration-only / absence-of-error / grep-based PASS without runtime evidence are critical defects regardless of how green the summary line looks. No false-success results are tolerable.

This anchor is inherited from the Helix Constitution (`constitution/Constitution.md` §11.9 / CONST-035); resolve it via `constitution/find_constitution.sh` from the parent project root. This submodule stays fully decoupled and project-not-aware (§11.4.28) — this is generic governance inheritance only, never project-specific context.
