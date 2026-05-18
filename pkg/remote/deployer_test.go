// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Round-47 §11.4 anti-bluff repair (2026-05-18): the original
// deployer_test.go targeted an Ollama-flavoured Deployer with an
// HTTP client + Endpoint/Status surface. That type no longer
// exists in production — round-40 wiring (commit 1169213) settled
// the package on a llama.cpp-flavoured LlamaCppDeployer whose
// constructor is NewLlamaCppDeployer(LlamaCppConfig{...}) and
// whose surface is purely SSH-driven (FreeGPU / StartInstance /
// RestoreOllama / StartRPCServer / StartWithRPC / StopInstance /
// StopRPCServer). Old-API → new-API mapping:
//
//	OLD                          NEW
//	NewDeployer(...)             NewLlamaCppDeployer(...)
//	Config{Host,User,Port,       LlamaCppConfig{Host,User,RepoDir,
//	  Model,OllamaPort}            ModelPath,MMProjPath,BasePort,
//	                               GPULayers,ContextSize}
//	d.cfg                        d.config
//	d.client / d.Endpoint() /    (removed — Ollama HTTP surface
//	d.Status() / d.isModelPulled  was lifted out of the package)
//
// Old TestDeployer_IsModelPulled_*, TestDeployer_Endpoint*,
// TestDeployer_Status_Unreachable, TestDeployer_APICheck_Success
// scenarios are NOT expressible against the new API (the surface
// they exercised was deleted). They are recorded as skipped here
// with SKIP-OK markers per CONST-035 (loud-absence-of-coverage
// preferred over silent deletion of audit trail), referencing the
// round-27 (76452da) and round-40 (1169213) forensic anchors.
//
// Constitutional anchors: CONST-035 (anti-bluff), CONST-050(A)
// (no-fakes-beyond-unit-tests — unit-test file, mocks-free
// reconstruction), Article XI §11.9 forensic anchor (a test that
// does not compile is a §11.4 PASS-bluff equivalent: any
// "go test ./..." PASS claim is a bluff while the test file
// itself fails to build).

func TestNewLlamaCppDeployer_DefaultsViaZeroConfig(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "test.local"})
	require.NotNil(t, d)
	assert.Equal(t, "test.local", d.config.Host)
	// LlamaCppConfig has no implicit defaults applied by the
	// constructor (round-40 design: config is a literal value).
	assert.Equal(t, "", d.config.User)
	assert.Equal(t, 0, d.config.BasePort)
	assert.Equal(t, 0, d.config.GPULayers)
	assert.Equal(t, 0, d.config.ContextSize)
	assert.Equal(t, "", d.config.RepoDir)
	assert.Equal(t, "", d.config.ModelPath)
	assert.Equal(t, "", d.config.MMProjPath)
}

func TestNewLlamaCppDeployer_CustomConfig(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{
		Host:        "gpu.local",
		User:        "admin",
		RepoDir:     "~/llama.cpp",
		ModelPath:   "/models/llava.gguf",
		MMProjPath:  "/models/mmproj.gguf",
		BasePort:    8080,
		GPULayers:   -1,
		ContextSize: 4096,
	})
	require.NotNil(t, d)
	assert.Equal(t, "gpu.local", d.config.Host)
	assert.Equal(t, "admin", d.config.User)
	assert.Equal(t, "~/llama.cpp", d.config.RepoDir)
	assert.Equal(t, "/models/llava.gguf", d.config.ModelPath)
	assert.Equal(t, "/models/mmproj.gguf", d.config.MMProjPath)
	assert.Equal(t, 8080, d.config.BasePort)
	assert.Equal(t, -1, d.config.GPULayers)
	assert.Equal(t, 4096, d.config.ContextSize)
}

// TestLlamaCppDeployer_sshCmd_EmptyHostError asserts the
// behavioural guarantee documented in deployer.go: sshCmd
// returns an error when Host is unset, instead of silently
// invoking ssh with an empty target (which would otherwise be a
// silent CONST-035 PASS-bluff — the deployer would appear to
// "work" while shipping garbage to ssh).
func TestLlamaCppDeployer_sshCmd_EmptyHostError(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{})
	ctx := context.Background()
	err := d.sshCmd(ctx, "true")
	require.Error(t, err, "sshCmd MUST error when Host is empty")
	assert.Contains(t, err.Error(), "host is required",
		"error message MUST identify the missing-host condition")
}

// TestLlamaCppDeployer_RPCStubs_ReturnSentinels — round-48 §11.4
// anti-bluff canary tightening (2026-05-18, supersedes round-47
// commit 5496b2d's `TestLlamaCppDeployer_RPCStubs_NoCrash`). The
// four RPC-related lifecycle methods on LlamaCppDeployer
// (StartRPCServer / StartWithRPC / StopInstance / StopRPCServer)
// previously returned nil under `// Stub: do nothing.` comments —
// a CONST-035 forbidden tell in production code. Round 48 replaces
// each `return nil` with a distinct sentinel error per
// pkg/remote/distributed.go. This test asserts each method now
// returns its named sentinel via errors.Is.
//
// When round 49 wires real SSH-driven RPC lifecycle, this test
// MUST be re-targeted (or retired alongside its sibling
// integration test that exercises real `llama-server --rpc`) —
// that is the next anti-bluff canary moment.
//
// Constitutional anchors: CONST-035 (anti-bluff), CONST-050(A)
// (no-fakes-beyond-unit-tests), Article XI §11.9 forensic anchor.
func TestLlamaCppDeployer_RPCStubs_ReturnSentinels(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"})
	ctx := context.Background()

	t.Run("StartRPCServer", func(t *testing.T) {
		err := d.StartRPCServer(ctx, 9000)
		require.Error(t, err,
			"StartRPCServer stub MUST return non-nil error per round-48 sentinel contract")
		require.True(t, errors.Is(err, ErrRPCServerStartNotImplemented),
			"StartRPCServer MUST return ErrRPCServerStartNotImplemented; got: %v", err)
	})

	t.Run("StartWithRPC", func(t *testing.T) {
		err := d.StartWithRPC(ctx, "/models/m.gguf", []string{}, 9001)
		require.Error(t, err,
			"StartWithRPC stub MUST return non-nil error per round-48 sentinel contract")
		require.True(t, errors.Is(err, ErrRPCServerStartWithRPCNotImplemented),
			"StartWithRPC MUST return ErrRPCServerStartWithRPCNotImplemented; got: %v", err)
	})

	t.Run("StopInstance", func(t *testing.T) {
		err := d.StopInstance(ctx, 9001)
		require.Error(t, err,
			"StopInstance stub MUST return non-nil error per round-48 sentinel contract")
		require.True(t, errors.Is(err, ErrRPCServerStopInstanceNotImplemented),
			"StopInstance MUST return ErrRPCServerStopInstanceNotImplemented; got: %v", err)
	})

	t.Run("StopRPCServer", func(t *testing.T) {
		err := d.StopRPCServer(ctx, 9000)
		require.Error(t, err,
			"StopRPCServer stub MUST return non-nil error per round-48 sentinel contract")
		require.True(t, errors.Is(err, ErrRPCServerStopNotImplemented),
			"StopRPCServer MUST return ErrRPCServerStopNotImplemented; got: %v", err)
	})
}

// TestDistributedSentinels_AllFour_Distinct — round-48 §11.4
// paired-mutation guard (2026-05-18) following the round-44
// 5-sentinel distinctness pattern in remote_test.go's
// TestSSHSentinels_AreDistinct. Each of the four round-48
// distributed-RPC sentinels MUST be pairwise distinguishable via
// errors.Is, otherwise callers cannot route remediation correctly
// (e.g. "StartRPCServer failed" vs "StopInstance failed" carry
// very different operational meanings even though both are
// currently unimplemented).
//
// The test also asserts cross-package distinctness against the
// round-27 sibling ErrShutdownRemoteCleanupNotImplemented (declared
// in remote.go) — the four round-48 sentinels MUST NOT collapse
// into that older one despite the overlapping "remote llama-server
// lifecycle is not wired" semantic.
//
// Constitutional anchors: CONST-035 (anti-bluff distinctness),
// CONST-050(A), Article XI §11.9.
func TestDistributedSentinels_AllFour_Distinct(t *testing.T) {
	sentinels := map[string]error{
		"ErrRPCServerStartNotImplemented":        ErrRPCServerStartNotImplemented,
		"ErrRPCServerStartWithRPCNotImplemented": ErrRPCServerStartWithRPCNotImplemented,
		"ErrRPCServerStopInstanceNotImplemented": ErrRPCServerStopInstanceNotImplemented,
		"ErrRPCServerStopNotImplemented":         ErrRPCServerStopNotImplemented,
	}

	// Pairwise distinctness across the four round-48 sentinels.
	for nameA, a := range sentinels {
		for nameB, b := range sentinels {
			if nameA == nameB {
				assert.True(t, errors.Is(a, b),
					"sentinel %s MUST satisfy errors.Is against itself", nameA)
				continue
			}
			assert.False(t, errors.Is(a, b),
				"sentinel %s MUST NOT be confusable with %s via errors.Is", nameA, nameB)
		}
	}

	// Cross-package distinctness vs round-27 sibling.
	for name, s := range sentinels {
		assert.False(t, errors.Is(s, ErrShutdownRemoteCleanupNotImplemented),
			"round-48 sentinel %s MUST NOT collapse into round-27 ErrShutdownRemoteCleanupNotImplemented", name)
		assert.False(t, errors.Is(ErrShutdownRemoteCleanupNotImplemented, s),
			"round-27 ErrShutdownRemoteCleanupNotImplemented MUST NOT collapse into round-48 sentinel %s", name)
	}
}

// --- Skipped scenarios (preserved audit trail per CONST-035) ---
//
// The four tests below were valid against the old NewDeployer /
// Config API which exposed an embedded HTTP client + Endpoint() +
// Status() + isModelPulled() surface tailored to Ollama. That
// surface was deleted from pkg/remote when the package was
// re-scoped to llama.cpp-only SSH lifecycle management
// (round-40 / commit 1169213). The Ollama HTTP surface now lives
// in pkg/llmvision/ollama.go and is exercised by that package's
// own tests. These markers keep the audit trail loud rather than
// silently deleting the historical scenarios.

func TestDeployer_Endpoint(t *testing.T) {
	t.Skip("SKIP-OK: #round-47-api-drift — Endpoint() surface removed in round-40 re-scope to llama.cpp-only; Ollama HTTP polling now lives in pkg/llmvision/ollama.go (see round-27 76452da, round-40 1169213)")
}

func TestDeployer_Endpoint_CustomPort(t *testing.T) {
	t.Skip("SKIP-OK: #round-47-api-drift — Endpoint() surface removed in round-40 re-scope to llama.cpp-only (see round-27 76452da, round-40 1169213)")
}

func TestDeployer_IsModelPulled_Found(t *testing.T) {
	t.Skip("SKIP-OK: #round-47-api-drift — isModelPulled / d.client HTTP surface removed in round-40 re-scope; Ollama model-presence checks now live in pkg/llmvision/ollama.go (see round-27 76452da, round-40 1169213)")
}

func TestDeployer_IsModelPulled_NotFound(t *testing.T) {
	t.Skip("SKIP-OK: #round-47-api-drift — isModelPulled / d.client HTTP surface removed in round-40 re-scope (see round-27 76452da, round-40 1169213)")
}

func TestDeployer_Status_Unreachable(t *testing.T) {
	t.Skip("SKIP-OK: #round-47-api-drift — Status() / OllamaRunning / ModelAvailable surface removed in round-40 re-scope to llama.cpp-only; reachability checks belong to the consuming runtime that holds the SSH creds (see round-27 76452da, round-40 1169213)")
}

func TestDeployer_APICheck_Success(t *testing.T) {
	t.Skip("SKIP-OK: #round-47-api-drift — d.client HTTP surface removed in round-40 re-scope to llama.cpp-only; Ollama API health now lives in pkg/llmvision/ollama.go (see round-27 76452da, round-40 1169213)")
}
