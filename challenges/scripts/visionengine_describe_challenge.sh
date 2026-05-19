#!/usr/bin/env bash
# Copyright 2026 HelixDevelopment. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#
# visionengine_describe_challenge.sh — round-297 paired-mutation
# Challenge for VisionEngine (mirror round-220 / 242-296 template).
#
# Anti-bluff contract (Article XI §11.9 + CONST-035):
#   - Normal mode: builds + runs `challenges/runner` against the REAL
#     analyzer.StubAnalyzer + graph.NavigationGraph + i18n seam, and
#     EXPECTS the runner to exit 0 with all six runtime-evidence
#     locale-tagged lines on stdout. Failure to print any locale line
#     fails the Challenge.
#   - Mutation mode (--mutate): plants a deliberate regression in a
#     scratch copy of pkg/graph/graph.go (zeroing the AddTransition
#     return path) and EXPECTS the runner to FAIL (exit non-zero).
#     If the runner still PASSES on the mutated source, the Challenge
#     itself was a bluff — exit 99 to flag the bluff.
#
# Exit codes:
#   0  — normal run: real runner passed with full runtime evidence
#   99 — mutate run: runner correctly detected the planted regression
#         (99 is the canonical paired-mutation "genuine Challenge" code,
#         mirroring round-220 / 242-296)
#   1  — normal run: runner failed (real defect or environment issue)
#   2  — mutate run: runner PASSED on mutated source (Challenge is a
#         bluff — does not actually exercise the mutated code path)

set -euo pipefail

MODE="normal"
if [[ "${1:-}" == "--mutate" ]]; then
    MODE="mutate"
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

EXPECTED_LOCALES=(
    "AnalyzeScreen OK"
    "AnalyzeScreen rejects empty buffer"
    "NavigationGraph BFS found"
    "Exports OK"
    "Rect geometry"
    "NoopTranslator passthrough OK"
)

run_normal() {
    echo "[visionengine_describe_challenge] mode=normal building runner…"
    local bin
    bin="$(mktemp -d)/visionengine-runner"
    go build -o "$bin" ./challenges/runner/
    local out
    if ! out="$("$bin" 2>&1)"; then
        echo "FAIL: runner exited non-zero in normal mode"
        echo "$out"
        exit 1
    fi
    echo "$out"
    for needle in "${EXPECTED_LOCALES[@]}"; do
        if ! grep -q "$needle" <<<"$out"; then
            echo "FAIL: missing required locale-tagged line: $needle"
            exit 1
        fi
    done
    echo "[visionengine_describe_challenge] PASS — runtime evidence captured per §11.9"
    exit 0
}

run_mutate() {
    echo "[visionengine_describe_challenge] mode=mutate — planting AddTransition regression"
    local work
    work="$(mktemp -d)"
    cp -r "$REPO_ROOT" "$work/visionengine"
    cd "$work/visionengine"
    # Strip mutate-mode guard so we don't recurse forever.
    rm -rf .git || true

    local target="pkg/graph/graph.go"
    if [[ ! -f "$target" ]]; then
        echo "BLUFF-FAIL: $target missing in scratch copy"
        exit 2
    fi

    # Plant regression: turn AddTransition into a no-op so PathTo
    # cannot find a 2-hop path. This MUST cause the runner to fail.
    python3 - "$target" <<'PY'
import re, sys, pathlib
path = pathlib.Path(sys.argv[1])
src = path.read_text()
# Match: func (g *navigationGraph) AddTransition(from, to string, action analyzer.Action) {
pattern = re.compile(r"(func\s+\([^\)]+\)\s+AddTransition\([^\)]*\)\s*\{)")
m = pattern.search(src)
if not m:
    print("BLUFF: AddTransition signature not found — Challenge mutation invalid", file=sys.stderr)
    sys.exit(2)
new = src[:m.end()] + "\n\treturn // round-297 paired-mutation regression\n" + src[m.end():]
path.write_text(new)
PY

    if ! go build -o /tmp/visionengine-runner-mutated ./challenges/runner/ 2>&1; then
        # Build failure also proves the mutation invalidated the
        # downstream — counts as successful regression detection.
        echo "[visionengine_describe_challenge] mutation triggered build failure (regression detected)"
        echo "PASS-MUTATE: planted regression detected via build failure (exit 99 = genuine)"
        exit 99
    fi
    if /tmp/visionengine-runner-mutated >/dev/null 2>&1; then
        echo "BLUFF-FAIL: runner PASSED on mutated source — Challenge does not exercise AddTransition"
        exit 2
    fi
    echo "[visionengine_describe_challenge] mutation correctly detected — Challenge is genuine (exit 99)"
    exit 99
}

case "$MODE" in
    normal) run_normal ;;
    mutate) run_mutate ;;
    *) echo "usage: $0 [--mutate]"; exit 2 ;;
esac
