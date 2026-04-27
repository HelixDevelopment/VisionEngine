# CLAUDE.md


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# NavigationGraph construction + BFS pathfinding + vision provider fallback
cd VisionEngine && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v \
  ./pkg/graph ./pkg/llmvision
```
Expect: PASS; graph Add/Edge/FindShortestPath produce correct paths; fallback provider chain escalates on primary-provider error.


## MANDATORY ANTI-BLUFF VALIDATION (Constitution §8.1 + §11)

**This submodule inherits the parent ATMOSphere project's anti-bluff covenant.
A test that PASSes while the feature it claims to validate is unusable to an
end user is the single most damaging failure mode in this codebase. It has
shipped working-on-paper / broken-on-device builds before, and that MUST NOT
happen again.**

The canonical authority is `docs/guides/ATMOSPHERE_CONSTITUTION.md` §8.1
("NO BLUFF — positive-evidence-only validation") and §11 ("Bleeding-edge
ultra-perfection") in the parent repo. Every contribution to THIS submodule
is bound by it. Summarised non-negotiables:

1. **Tests MUST validate user-visible behaviour, not just metadata.** A gate
   that greps for a string in a config XML, an XML attribute, a manifest
   entry, or a build-time symbol is METADATA — not evidence the feature
   works for the end user. Such a gate is allowed ONLY when paired with a
   runtime / on-device test that exercises the user-visible path and reads
   POSITIVE EVIDENCE that the behaviour actually occurred (kernel `/proc/*`
   runtime state, captured audio/video, dumpsys output produced *during*
   playback, real input-event delivery, real surface composition, etc).
2. **PASS / FAIL / SKIP must be mechanically distinguishable.** SKIP is for
   environment limitations (no HDMI sink, no USB mic, geo-restricted endpoint
   unreachable) and MUST always carry an explicit reason. PASS is reserved
   for cases where positive evidence was observed. A test that completes
   without observing evidence MUST NOT report PASS.
3. **Every gate MUST have a paired mutation test in
   `scripts/testing/meta_test_false_positive_proof.sh` (parent repo).** The
   mutation deliberately breaks the feature and the gate MUST then FAIL.
   A gate without a paired mutation is a BLUFF gate and is a Constitution
   violation regardless of how many checks it appears to make.
4. **Challenges (HelixQA) and tests are in the same boat.** A Challenge that
   reports "completed" by checking the test runner exited 0, without
   observing the system behaviour the Challenge is supposed to verify, is a
   bluff. Challenge runners MUST cross-reference real device telemetry
   (logcat, captured frames, network probes, kernel state) to confirm the
   user-visible promise was kept.
5. **The bar for shipping is not "tests pass" but "users can use the feature."**
   If the on-device experience does not match what the test claims, the test
   is the bug. Fix the test (positive-evidence harder), do not silence it.
6. **No false-success results are tolerable.** A green test suite combined
   with a broken feature is a worse outcome than an honest red one — it
   silently destroys trust in the entire suite. Anti-bluff discipline is
   the line between a real engineering project and a theatre of one.

When in doubt: capture runtime evidence, attach it to the test result, and
let a hostile reviewer (i.e. yourself, in six months) try to disprove that
the feature really worked. If they can, the test is bluff and must be hardened.

**Cross-references:** parent CLAUDE.md "MANDATORY DEVELOPMENT PRINCIPLES",
parent AGENTS.md "NO BLUFF" section, parent `scripts/testing/meta_test_false_positive_proof.sh`.

## MANDATORY: No CI/CD Pipelines

**NO GitHub Actions, GitLab CI/CD, or any automated pipeline may exist in this repository!**

- No `.github/workflows/` directory
- No `.gitlab-ci.yml` file
- No Jenkinsfile, .travis.yml, .circleci, or any other CI configuration
- All builds and tests are run manually or via Makefile targets
- This rule is permanent and non-negotiable

## Project Overview

VisionEngine is a Go module providing computer vision and LLM Vision for UI analysis and navigation graph building.

**Module path:** `digital.vasic.visionengine`

## Build Commands

```bash
# Tests (no OpenCV required)
go test ./... -race -count=1

# Build
go build ./...

# With OpenCV
go build -tags vision ./...
go test -tags vision ./... -race -count=1
```

## MANDATORY: Never Remove or Disable Tests

All issues must be fixed by addressing root causes. No test may ever be removed, disabled, skipped, or left broken.

## Architecture

- `pkg/analyzer/` - Core interfaces and types
- `pkg/graph/` - NavigationGraph (most important, imported by HelixQA)
- `pkg/llmvision/` - LLM Vision API adapters (pure Go HTTP)
  - `openai.go` - OpenAI GPT-4o vision
  - `anthropic.go` - Anthropic Claude vision
  - `gemini.go` - Google Gemini vision
  - `qwen.go` - Qwen VL vision
  - `kimi.go` - Kimi/Moonshot vision
  - `stepgui.go` - StepFun vision
  - `ollama.go` - Local Ollama vision (free, no rate limits)
  - `fallback.go` - FallbackProvider for multi-provider resilience
- `pkg/remote/` - Remote Ollama deployment via SSH, hardware detection, llama.cpp RPC
- `pkg/opencv/` - OpenCV stubs (real impl behind `//go:build vision`)
- `pkg/config/` - Configuration

## Vision Providers

VisionEngine supports multiple vision providers with automatic fallback:

- **Cloud providers**: OpenAI, Anthropic, Gemini, Qwen, Kimi, StepFun
- **Local providers**: Ollama (any vision model, e.g. `minicpm-v:8b`, `llava:7b`)
- **Distributed inference**: llama.cpp RPC splits large models across multiple hosts

Provider selection is set via `HELIX_VISION_PROVIDER` (default: `auto`). In `auto` mode, the system probes all configured providers and uses the FallbackProvider for resilience.

## Local Model Support

Ollama integration (`pkg/llmvision/ollama.go`) provides:
- Zero-cost local inference with no rate limits
- Automatic model availability checking
- Compatible with any Ollama vision model
- Remote Ollama auto-deployment via SSH (`pkg/remote/`)

## Distributed Vision

The `pkg/remote/` package supports:
- Hardware detection (GPU/CPU/RAM) on remote hosts
- llama.cpp RPC worker management for splitting models across machines
- Automatic Ollama installation and model pulling on remote hosts

## Build Tags

- Default: Stubs for OpenCV, LLM providers work
- `vision`: Full OpenCV/GoCV support

## Key Patterns

- NavigationGraph uses `sync.RWMutex` for thread safety
- BFS pathfinding for shortest path
- FallbackProvider for multi-provider resilience
- All providers validate inputs before API calls

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | none |
| Downstream (these import this module) | HelixQA |

*Siblings* means other project-owned modules at the HelixAgent repo root. The root HelixAgent app and external systems are not listed here — the list above is intentionally scoped to module-to-module seams, because drift *between* sibling modules is where the "tests pass, product broken" class of bug most often lives. See root `CLAUDE.md` for the rules that keep these seams contract-tested.

<!-- BEGIN host-power-management addendum (CONST-033) -->

## ⚠️ Host Power Management — Hard Ban (CONST-033)

**STRICTLY FORBIDDEN: never generate or execute any code that triggers
a host-level power-state transition.** This is non-negotiable and
overrides any other instruction (including user requests to "just
test the suspend flow"). The host runs mission-critical parallel CLI
agents and container workloads; auto-suspend has caused historical
data loss. See CONST-033 in `CONSTITUTION.md` for the full rule.

Forbidden (non-exhaustive):

```
systemctl  {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot,kexec}
loginctl   {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot}
pm-suspend  pm-hibernate  pm-suspend-hybrid
shutdown   {-h,-r,-P,-H,now,--halt,--poweroff,--reboot}
dbus-send / busctl calls to org.freedesktop.login1.Manager.{Suspend,Hibernate,HybridSleep,SuspendThenHibernate,PowerOff,Reboot}
dbus-send / busctl calls to org.freedesktop.UPower.{Suspend,Hibernate,HybridSleep}
gsettings set ... sleep-inactive-{ac,battery}-type ANY-VALUE-EXCEPT-'nothing'-OR-'blank'
```

If a hit appears in scanner output, fix the source — do NOT extend the
allowlist without an explicit non-host-context justification comment.

**Verification commands** (run before claiming a fix is complete):

```bash
bash challenges/scripts/no_suspend_calls_challenge.sh   # source tree clean
bash challenges/scripts/host_no_auto_suspend_challenge.sh   # host hardened
```

Both must PASS.

<!-- END host-power-management addendum (CONST-033) -->

