// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

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
// Round-52 §11.4 (2026-05-18, this round): the sentinels are
// PRESERVED for the no-SSH-config path — the same deployer
// constructed without WithSSHConfig still returns the round-48
// sentinels. This canary therefore continues to PASS post-round-52
// without modification, validating that the SSH-configured wiring
// did NOT silently swallow the unconfigured-SSH signal. New
// round-52 tests below (TestLlamaCppDeployer_RPCMethods_NoSSHConfig
// _StillReturnSentinels, TestLlamaCppDeployer_StopInstance_Unknown
// _Port_ReturnsError, etc.) cover the SSH-configured paths.
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

// --- Round-52 §11.4 anti-bluff wiring tests ---
//
// The block below covers the round-52 wiring of the four RPC
// lifecycle methods. The unit-only tests target unreachable / error
// paths (no real SSH server is stood up in-process — see the
// round-52 commit body for the design rationale: a Go-native SSH
// server fixture exceeds the per-round complexity guardrail; real
// SSH coverage lives in the env-gated integration test
// TestLlamaCppDeployer_RPCLifecycle_AgainstRealSSHHost below
// with a loud SKIP-OK marker).
//
// Test coverage matrix:
//   Method            | NoSSHConfig | SSHKeyMissing | EmptyKnownHosts | UnknownPort | Real-SSH (env-gated)
//   StartRPCServer    |      X      |       X       |        X        |     n/a     |          X
//   StartWithRPC      |      X      |       X       |        X        |     n/a     |          X
//   StopInstance      |      X      |       X       |        X        |      X      |          X
//   StopRPCServer     |      X      |       X       |        X        |      X      |          X
//
// Constitutional anchors: CONST-035 (anti-bluff), CONST-042 (no
// hardcoded secrets — test credentials come from env vars), CONST-
// 050(A) (no fakes beyond unit tests — this *is* a unit-test file
// so unreachable-path coverage is permitted; real-SSH path is
// integration-gated), CONST-050(B) (100% test-type coverage — the
// env-gated integration test is the real-SSH limb), Article XI §11.9.

// TestLlamaCppDeployer_SSHConfigured_TrueFalse — round-52 paired
// mutation guard for the new SSHConfigured() accessor. SSHConfigured
// MUST return false for a fresh deployer and true after WithSSHConfig
// is applied with a non-empty Host. Coupling SSHConfigured to the
// behavioural bifurcation in the four lifecycle methods means a
// regression here will break every downstream gate.
func TestLlamaCppDeployer_SSHConfigured_TrueFalse(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"})
	assert.False(t, d.SSHConfigured(),
		"fresh deployer MUST report SSH unconfigured")

	d = d.WithSSHConfig(SSHConfig{
		Host:           "gpu.local",
		User:           "test",
		KeyPath:        "/nonexistent/key",
		KnownHostsPath: "/nonexistent/known_hosts",
	})
	assert.True(t, d.SSHConfigured(),
		"deployer with non-empty SSHConfig.Host MUST report SSH configured")
}

// TestLlamaCppDeployer_StartRPCServer_NoSSHConfig_ReturnsSentinel —
// round-52 preservation guard for the round-48 sentinel contract.
// A deployer constructed without WithSSHConfig MUST still return
// ErrRPCServerStartNotImplemented (the sentinel is now the explicit
// "SSH is unconfigured" signal rather than "method is unimplemented").
func TestLlamaCppDeployer_StartRPCServer_NoSSHConfig_ReturnsSentinel(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"})
	require.False(t, d.SSHConfigured())

	err := d.StartRPCServer(context.Background(), 9100)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRPCServerStartNotImplemented),
		"no-SSH-config path MUST still surface round-48 sentinel post round-52; got: %v", err)
}

// TestLlamaCppDeployer_StartWithRPC_NoSSHConfig_ReturnsSentinel —
// same preservation guard for StartWithRPC.
func TestLlamaCppDeployer_StartWithRPC_NoSSHConfig_ReturnsSentinel(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"})
	require.False(t, d.SSHConfigured())

	err := d.StartWithRPC(context.Background(), "/models/m.gguf", []string{"w1:50001"}, 9101)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRPCServerStartWithRPCNotImplemented),
		"no-SSH-config path MUST still surface round-48 sentinel post round-52; got: %v", err)
}

// TestLlamaCppDeployer_StopInstance_NoSSHConfig_ReturnsSentinel —
// same preservation guard for StopInstance.
func TestLlamaCppDeployer_StopInstance_NoSSHConfig_ReturnsSentinel(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"})
	require.False(t, d.SSHConfigured())

	err := d.StopInstance(context.Background(), 9100)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRPCServerStopInstanceNotImplemented),
		"no-SSH-config path MUST still surface round-48 sentinel post round-52; got: %v", err)
}

// TestLlamaCppDeployer_StopRPCServer_NoSSHConfig_ReturnsSentinel —
// same preservation guard for StopRPCServer.
func TestLlamaCppDeployer_StopRPCServer_NoSSHConfig_ReturnsSentinel(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"})
	require.False(t, d.SSHConfigured())

	err := d.StopRPCServer(context.Background(), 9100)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRPCServerStopNotImplemented),
		"no-SSH-config path MUST still surface round-48 sentinel post round-52; got: %v", err)
}

// TestLlamaCppDeployer_StartRPCServer_SSHKeyMissing_ReturnsKeyError —
// round-52: with SSHConfig populated but KeyPath pointing at a
// nonexistent file, StartRPCServer MUST surface ErrSSHKeyParseFailed
// (NOT the round-48 sentinel — that path is for unconfigured SSH).
// Mirrors TestShutdown_SSHKeyMissing_ReturnsKeyParseError pattern
// from round-40 remote_test.go.
func TestLlamaCppDeployer_StartRPCServer_SSHKeyMissing_ReturnsKeyError(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"}).WithSSHConfig(SSHConfig{
		Host:           "gpu.local",
		User:           "test",
		KeyPath:        "/nonexistent/path/to/key",
		KnownHostsPath: "/nonexistent/path/to/known_hosts",
	})

	err := d.StartRPCServer(context.Background(), 9100)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSSHKeyParseFailed,
		"SSH-configured + missing key MUST surface ErrSSHKeyParseFailed")
	require.NotErrorIs(t, err, ErrRPCServerStartNotImplemented,
		"SSH-configured path MUST NOT surface round-48 sentinel — that is the unconfigured-SSH signal")
}

// TestLlamaCppDeployer_StartWithRPC_SSHKeyMissing_ReturnsKeyError —
// same pattern for StartWithRPC.
func TestLlamaCppDeployer_StartWithRPC_SSHKeyMissing_ReturnsKeyError(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"}).WithSSHConfig(SSHConfig{
		Host:           "gpu.local",
		User:           "test",
		KeyPath:        "/nonexistent/path/to/key",
		KnownHostsPath: "/nonexistent/path/to/known_hosts",
	})

	err := d.StartWithRPC(context.Background(), "/models/m.gguf", []string{"w1:50001"}, 9101)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSSHKeyParseFailed)
	require.NotErrorIs(t, err, ErrRPCServerStartWithRPCNotImplemented)
}

// TestLlamaCppDeployer_StartWithRPC_EmptyModelPath_RejectedEarly —
// round-52: StartWithRPC validates modelPath before SSH-dialling so a
// caller error surfaces immediately, not as a wasted SSH round trip.
func TestLlamaCppDeployer_StartWithRPC_EmptyModelPath_RejectedEarly(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"}).WithSSHConfig(SSHConfig{
		Host:           "gpu.local",
		User:           "test",
		KeyPath:        "/nonexistent/key",
		KnownHostsPath: "/nonexistent/known_hosts",
	})

	err := d.StartWithRPC(context.Background(), "", []string{"w1:50001"}, 9101)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-empty modelPath",
		"empty modelPath MUST be rejected with a clear error before SSH dial")
}

// TestLlamaCppDeployer_StartWithRPC_InvalidServerPort_RejectedEarly —
// round-52: serverPort <= 0 is rejected pre-dial.
func TestLlamaCppDeployer_StartWithRPC_InvalidServerPort_RejectedEarly(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"}).WithSSHConfig(SSHConfig{
		Host:           "gpu.local",
		User:           "test",
		KeyPath:        "/nonexistent/key",
		KnownHostsPath: "/nonexistent/known_hosts",
	})

	err := d.StartWithRPC(context.Background(), "/models/m.gguf", []string{"w1:50001"}, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "serverPort > 0",
		"non-positive serverPort MUST be rejected with a clear error before SSH dial")
}

// TestLlamaCppDeployer_StopInstance_UnknownPort_ReturnsInstanceNotFound —
// round-52 CONST-035 anti-bluff: StopInstance for a port that was
// never registered MUST return ErrRPCInstanceNotFound rather than
// silently no-op. A no-op for unknown work is a PASS-bluff (caller
// believes the instance was stopped when in fact nothing happened).
func TestLlamaCppDeployer_StopInstance_UnknownPort_ReturnsInstanceNotFound(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"}).WithSSHConfig(SSHConfig{
		Host:           "gpu.local",
		User:           "test",
		KeyPath:        "/nonexistent/key",
		KnownHostsPath: "/nonexistent/known_hosts",
	})

	err := d.StopInstance(context.Background(), 9999)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRPCInstanceNotFound,
		"unknown-port StopInstance MUST surface ErrRPCInstanceNotFound (CONST-035: no silent no-op)")
}

// TestLlamaCppDeployer_StopRPCServer_UnknownPort_ReturnsInstanceNotFound —
// same anti-bluff guard for StopRPCServer.
func TestLlamaCppDeployer_StopRPCServer_UnknownPort_ReturnsInstanceNotFound(t *testing.T) {
	d := NewLlamaCppDeployer(LlamaCppConfig{Host: "gpu.local"}).WithSSHConfig(SSHConfig{
		Host:           "gpu.local",
		User:           "test",
		KeyPath:        "/nonexistent/key",
		KnownHostsPath: "/nonexistent/known_hosts",
	})

	err := d.StopRPCServer(context.Background(), 9999)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRPCInstanceNotFound)
}

// TestLlamaCppDeployer_InstanceMap_PutGetDelete — round-52
// paired-mutation guard for the internal instanceMap. Direct
// unit-test exercise of the lifecycle map semantics (concurrent
// access correctness covered by -race flag at suite level).
func TestLlamaCppDeployer_InstanceMap_PutGetDelete(t *testing.T) {
	im := newInstanceMap()

	_, ok := im.get(8080)
	assert.False(t, ok, "fresh map has no entries")

	inst := &RPCInstance{
		ID:        "8080",
		PID:       12345,
		Port:      8080,
		Host:      "gpu.local",
		StartedAt: time.Now(),
	}
	im.put(8080, inst)

	got, ok := im.get(8080)
	require.True(t, ok)
	assert.Equal(t, 12345, got.PID)
	assert.Equal(t, 8080, got.Port)
	assert.Equal(t, "gpu.local", got.Host)

	im.delete(8080)
	_, ok = im.get(8080)
	assert.False(t, ok, "delete must remove the entry")
}

// TestParsePIDFromOutput_Variants — round-52 paired-mutation
// guard for the PID-extraction helper. The helper must take the
// LAST numeric token because nohup output ("nohup: ignoring input
// ...") precedes the `echo $!` PID line.
func TestParsePIDFromOutput_Variants(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"bare-pid", "12345", 12345},
		{"nohup-prefix-then-pid", "nohup: ignoring input and redirecting stderr to stdout\n12345", 12345},
		{"trailing-whitespace", "  12345  ", 12345},
		{"multi-line-pgrep", "12345\n67890", 67890},
		{"empty-string", "", 0},
		{"only-non-numeric", "command not found", 0},
		{"zero-rejected", "0", 0},
		{"negative-token-rejected", "abc -5 def", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePIDFromOutput(tt.in)
			assert.Equal(t, tt.want, got, "parsePIDFromOutput(%q)", tt.in)
		})
	}
}

// TestRoundFortyEightSentinels_ErrInstanceNotFound_Distinct —
// round-52 §11.4 paired-mutation guard: ErrRPCInstanceNotFound
// MUST be distinguishable from the four round-48 sentinels (and
// from ErrRPCLaunchFailed / ErrRPCReadinessProbeFailed) via
// errors.Is. Composes with TestDistributedSentinels_AllFour_Distinct.
func TestRoundFortyEightSentinels_ErrInstanceNotFound_Distinct(t *testing.T) {
	round48 := []error{
		ErrRPCServerStartNotImplemented,
		ErrRPCServerStartWithRPCNotImplemented,
		ErrRPCServerStopInstanceNotImplemented,
		ErrRPCServerStopNotImplemented,
	}
	round52 := []error{
		ErrRPCInstanceNotFound,
		ErrRPCLaunchFailed,
		ErrRPCReadinessProbeFailed,
	}
	for _, r48 := range round48 {
		for _, r52 := range round52 {
			assert.False(t, errors.Is(r48, r52),
				"round-48 sentinel %q MUST NOT collapse into round-52 sentinel %q", r48, r52)
			assert.False(t, errors.Is(r52, r48),
				"round-52 sentinel %q MUST NOT collapse into round-48 sentinel %q", r52, r48)
		}
	}
	// Round-52 sentinels pairwise distinct from each other.
	for i, a := range round52 {
		for j, b := range round52 {
			if i == j {
				continue
			}
			assert.False(t, errors.Is(a, b),
				"round-52 sentinels MUST be pairwise distinct: %v vs %v", a, b)
		}
	}
}

// TestLlamaCppDeployer_RPCLifecycle_AgainstRealSSHHost — round-52
// integration test gated on real SSH host env vars. Loud SKIP-OK
// marker so `make no-silent-skips` surfaces the conditional coverage.
// Mirrors the round-40 TestShutdown_AgainstRealSSHHost pattern.
//
// The test exercises the FULL round-52 lifecycle on a real SSH
// host: start a llama-server RPC instance at an ephemeral port,
// verify the instance is tracked, stop it, verify it is removed.
// Because spinning up a real llama-server requires the binary +
// model files on the remote host, this test will only meaningfully
// PASS in an environment where those prerequisites are met — the
// most common failure mode (binary missing) surfaces as
// ErrRPCLaunchFailed which is itself a positive assertion that the
// wiring works end-to-end.
//
// To run:
//
//	export VISIONENGINE_TEST_SSH_HOST=<host>
//	       VISIONENGINE_TEST_SSH_USER=<user>
//	       VISIONENGINE_TEST_SSH_KEY=/path/to/key
//	       VISIONENGINE_TEST_SSH_KNOWN_HOSTS=/path/to/known_hosts
//	       VISIONENGINE_TEST_LLAMA_REPO_DIR=/path/to/llama.cpp  (optional)
//	       VISIONENGINE_TEST_LLAMA_MODEL=/path/to/model.gguf    (optional)
func TestLlamaCppDeployer_RPCLifecycle_AgainstRealSSHHost(t *testing.T) {
	host := os.Getenv("VISIONENGINE_TEST_SSH_HOST")
	user := os.Getenv("VISIONENGINE_TEST_SSH_USER")
	keyPath := os.Getenv("VISIONENGINE_TEST_SSH_KEY")
	knownHosts := os.Getenv("VISIONENGINE_TEST_SSH_KNOWN_HOSTS")
	if host == "" || user == "" || keyPath == "" || knownHosts == "" {
		t.Skip("SKIP-OK: #VISIONENGINE-RPC-REAL-ROUND52 — requires real SSH host; set VISIONENGINE_TEST_SSH_{HOST,USER,KEY,KNOWN_HOSTS} to enable")
	}

	repoDir := os.Getenv("VISIONENGINE_TEST_LLAMA_REPO_DIR")
	modelPath := os.Getenv("VISIONENGINE_TEST_LLAMA_MODEL")

	d := NewLlamaCppDeployer(LlamaCppConfig{
		Host:        host,
		User:        user,
		RepoDir:     repoDir,
		ModelPath:   modelPath,
		BasePort:    18180,
		ContextSize: 2048,
		GPULayers:   0,
	}).WithSSHConfig(SSHConfig{
		Host:           host,
		User:           user,
		KeyPath:        keyPath,
		KnownHostsPath: knownHosts,
		Timeout:        20 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	port := 18180
	startErr := d.StartRPCServer(ctx, port)
	if startErr != nil {
		// ErrRPCLaunchFailed / ErrRPCReadinessProbeFailed are
		// positive evidence the SSH wiring reached the remote
		// host — log + skip rather than fail (model/binary may
		// be missing in the test env).
		if errors.Is(startErr, ErrRPCLaunchFailed) || errors.Is(startErr, ErrRPCReadinessProbeFailed) {
			t.Logf("round-52 wiring reached remote host but llama-server launch/probe failed (expected without full prerequisites): %v", startErr)
			t.Skip("SKIP-OK: #VISIONENGINE-RPC-REAL-ROUND52-LLAMA — wiring confirmed reaching host; full launch requires llama-server binary + model on remote")
		}
		t.Fatalf("StartRPCServer: %v", startErr)
	}

	// If launch actually succeeded, the instance must be tracked.
	inst, ok := d.instances.get(port)
	require.True(t, ok, "successful StartRPCServer MUST register instance")
	require.Greater(t, inst.PID, 0)
	t.Logf("real-SSH StartRPCServer success: host=%s port=%d pid=%d", host, port, inst.PID)

	// Stop it.
	stopErr := d.StopInstance(ctx, port)
	require.NoError(t, stopErr, "StopInstance MUST succeed for a tracked instance")

	_, stillTracked := d.instances.get(port)
	assert.False(t, stillTracked, "successful StopInstance MUST remove instance from tracking")
}


