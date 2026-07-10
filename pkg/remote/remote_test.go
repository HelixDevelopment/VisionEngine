// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package remote_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.visionengine/pkg/remote"
)

func TestNewVisionPool_Defaults(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{})
	require.NotNil(t, pool)
	assert.Equal(t, 0, pool.Size())
}

func TestNewVisionPool_WithConfig(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:             "thinker.local",
		User:             "admin",
		Model:            "llava:7b",
		InferenceBackend: remote.BackendOllama,
		BasePort:         9000,
	})
	require.NotNil(t, pool)
}

func TestVisionPool_EnsureReady_EmptyHost(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{})
	err := pool.EnsureReady(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")
}

func TestVisionPool_EnsureReady_LlamaCppMissingConfig(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:             "thinker.local",
		InferenceBackend: remote.BackendLlamaCpp,
	})
	err := pool.EnsureReady(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "llama.cpp config required")
}

// TestVisionPool_EnsureReady_ConfigValidReturnsSentinel asserts the
// round-28 §11.4 audit fix: EnsureReady on a well-formed PoolConfig
// no longer returns nil (which would be a §11.4 PASS-bluff — config-
// validated ≠ backend-reachable). It returns
// ErrBackendVerificationNotImplemented instead so callers can detect
// the gap programmatically and perform an independent reachability
// probe (HTTP probe, TCP dial, SSH check).
func TestVisionPool_EnsureReady_ConfigValidReturnsSentinel(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:             "thinker.local",
		InferenceBackend: remote.BackendLlamaCpp,
		LlamaCpp: &remote.LlamaCppConfig{
			Host:      "thinker.local",
			ModelPath: "/models/llava.gguf",
		},
	})
	err := pool.EnsureReady(context.Background())
	require.Error(t, err,
		"EnsureReady MUST NOT return nil after config-only validation — would re-introduce the §11.4 bluff")
	require.ErrorIs(t, err, remote.ErrBackendVerificationNotImplemented)
}

// TestVisionPool_EnsureReady_MalformedConfigStillFailsWithDescriptiveError
// asserts that malformed-config paths still produce their original
// descriptive errors (NOT the sentinel) — the sentinel is reserved
// for the config-valid-but-unverified path.
func TestVisionPool_EnsureReady_MalformedConfigStillFailsWithDescriptiveError(t *testing.T) {
	// Malformed: empty host.
	pool := remote.NewVisionPool(remote.PoolConfig{})
	err := pool.EnsureReady(context.Background())
	require.Error(t, err)
	assert.False(t,
		errors.Is(err, remote.ErrBackendVerificationNotImplemented),
		"malformed-config errors MUST NOT be the verification sentinel")
}

func TestVisionPool_AssignSlots_Shared(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		Shared:   true,
		BasePort: 8080,
	})

	targets := []remote.SlotTarget{
		{Platform: "android", Device: "device1"},
		{Platform: "android", Device: "device2"},
		{Platform: "web"},
	}
	pool.AssignSlots(targets)

	// All targets should share the same slot endpoint.
	s1 := pool.GetSlot("android", "device1")
	s2 := pool.GetSlot("android", "device2")
	s3 := pool.GetSlot("web", "")
	require.NotNil(t, s1)
	require.NotNil(t, s2)
	require.NotNil(t, s3)

	assert.Equal(t, s1.Endpoint, s2.Endpoint,
		"shared pool: all slots should have same endpoint")
	assert.Equal(t, s1.Endpoint, s3.Endpoint)
	assert.Equal(t, 8080, s1.Port)
}

func TestVisionPool_AssignSlots_Dedicated(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		Shared:   false,
		BasePort: 9000,
	})

	targets := []remote.SlotTarget{
		{Platform: "android", Device: "device1"},
		{Platform: "android", Device: "device2"},
		{Platform: "web"},
	}
	pool.AssignSlots(targets)

	assert.Equal(t, 3, pool.Size())

	s1 := pool.GetSlot("android", "device1")
	s2 := pool.GetSlot("android", "device2")
	s3 := pool.GetSlot("web", "")
	require.NotNil(t, s1)
	require.NotNil(t, s2)
	require.NotNil(t, s3)

	assert.NotEqual(t, s1.Endpoint, s2.Endpoint,
		"dedicated pool: each slot should have different endpoint")
	assert.Equal(t, 9000, s1.Port)
	assert.Equal(t, 9001, s2.Port)
	assert.Equal(t, 9002, s3.Port)
}

func TestVisionPool_GetSlot_NotAssigned(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		BasePort: 8080,
	})
	slot := pool.GetSlot("nonexistent", "")
	assert.Nil(t, slot)
}

// TestVisionPool_Shutdown asserts the round-28 §11.4 audit fix:
// Shutdown clears local pool state (Size()==0) AND returns the
// ErrShutdownRemoteCleanupNotImplemented sentinel so callers can
// detect the orphan-process gap (remote llama-server processes are
// NOT terminated by Shutdown).
func TestVisionPool_Shutdown(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		BasePort: 8080,
	})
	pool.AssignSlots([]remote.SlotTarget{
		{Platform: "android"},
	})
	assert.Equal(t, 1, pool.Size())

	err := pool.Shutdown(context.Background())
	require.Error(t, err,
		"Shutdown MUST surface the orphan-process sentinel — silent nil would be a §11.4 bluff")
	require.ErrorIs(t, err, remote.ErrShutdownRemoteCleanupNotImplemented)
	assert.Equal(t, 0, pool.Size(),
		"local pool state MUST still be cleared (that part of Shutdown's contract has never been the gap)")
}

// TestVisionPool_Shutdown_EmptyPool asserts that Shutdown on a pool
// with zero slots STILL returns the sentinel — the sentinel surfaces
// the contract gap (Shutdown cannot remotely kill processes) rather
// than the runtime state (how many slots were tracked).
func TestVisionPool_Shutdown_EmptyPool(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		BasePort: 8080,
	})
	err := pool.Shutdown(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, remote.ErrShutdownRemoteCleanupNotImplemented)
	assert.Equal(t, 0, pool.Size())
}

// TestShutdown_NoSSHConfigured_ReturnsSentinel — round-40 regression
// guard: a pool constructed WITHOUT WithSSHConfig() MUST still return
// the round-28 ErrShutdownRemoteCleanupNotImplemented sentinel,
// preserving the contract for legacy callers.
func TestShutdown_NoSSHConfigured_ReturnsSentinel(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		BasePort: 8080,
	})
	assert.False(t, pool.SSHConfigured(), "SSH must be unconfigured for this test")
	pool.AssignSlots([]remote.SlotTarget{{Platform: "android"}})

	err := pool.Shutdown(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, remote.ErrShutdownRemoteCleanupNotImplemented,
		"unconfigured-SSH path MUST surface the round-28 sentinel — round-40 wiring must NOT silently swallow it")
}

// TestEnsureReady_NoSSHConfigured_ReturnsSentinel — round-40
// regression guard: a pool constructed WITHOUT WithSSHConfig() MUST
// still return the round-28 ErrBackendVerificationNotImplemented
// sentinel on the well-formed-config path.
func TestEnsureReady_NoSSHConfigured_ReturnsSentinel(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:             "thinker.local",
		InferenceBackend: remote.BackendLlamaCpp,
		LlamaCpp: &remote.LlamaCppConfig{
			Host:      "thinker.local",
			ModelPath: "/models/llava.gguf",
		},
	})
	assert.False(t, pool.SSHConfigured())

	err := pool.EnsureReady(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, remote.ErrBackendVerificationNotImplemented,
		"unconfigured-SSH path MUST surface the round-28 sentinel — round-40 wiring must NOT silently swallow it")
}

// TestShutdown_SSHKeyMissing_ReturnsKeyParseError — round-40: when
// SSHConfig is populated but KeyPath points to a non-existent file,
// Shutdown returns ErrSSHKeyParseFailed (not the round-28 sentinel).
func TestShutdown_SSHKeyMissing_ReturnsKeyParseError(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		BasePort: 8080,
	}).WithSSHConfig(remote.SSHConfig{
		Host:           "thinker.local",
		User:           "test",
		KeyPath:        "/nonexistent/path/to/key",
		KnownHostsPath: "/nonexistent/path/to/known_hosts",
	})
	assert.True(t, pool.SSHConfigured())

	err := pool.Shutdown(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, remote.ErrSSHKeyParseFailed,
		"missing/unreadable key MUST surface ErrSSHKeyParseFailed")
	require.NotErrorIs(t, err, remote.ErrShutdownRemoteCleanupNotImplemented,
		"SSH-configured path MUST NOT surface the round-28 sentinel — that is the unconfigured-SSH signal")
}

// TestEnsureReady_SSHKeyMissing_ReturnsKeyParseError — round-40:
// same as Shutdown variant but for the EnsureReady path.
func TestEnsureReady_SSHKeyMissing_ReturnsKeyParseError(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		BasePort: 8080,
	}).WithSSHConfig(remote.SSHConfig{
		Host:           "thinker.local",
		User:           "test",
		KeyPath:        "/nonexistent/path/to/key",
		KnownHostsPath: "/nonexistent/path/to/known_hosts",
	})

	err := pool.EnsureReady(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, remote.ErrSSHKeyParseFailed)
	require.NotErrorIs(t, err, remote.ErrBackendVerificationNotImplemented)
}

// TestShutdown_EmptyKnownHostsPath_ReturnsHostKeyError — CONST-035
// paired-mutation guard: empty KnownHostsPath MUST be rejected.
func TestShutdown_EmptyKnownHostsPath_ReturnsHostKeyError(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		BasePort: 8080,
	}).WithSSHConfig(remote.SSHConfig{
		Host:           "thinker.local",
		User:           "test",
		KeyPath:        "/nonexistent/key",
		KnownHostsPath: "", // CONST-035 violation: must be rejected
	})

	err := pool.Shutdown(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, remote.ErrSSHHostKeyVerificationFailed,
		"empty KnownHostsPath MUST surface ErrSSHHostKeyVerificationFailed (CONST-035)")
}

// TestSSHSentinels_AreDistinct — paired-mutation: each sentinel
// MUST be distinguishable from the others via errors.Is.
func TestSSHSentinels_AreDistinct(t *testing.T) {
	all := []error{
		remote.ErrSSHKeyParseFailed,
		remote.ErrSSHHostKeyVerificationFailed,
		remote.ErrBackendNotReachable,
		remote.ErrShutdownRemoteCleanupNotImplemented,
		remote.ErrBackendVerificationNotImplemented,
	}
	for i, a := range all {
		for j, b := range all {
			if i == j {
				continue
			}
			assert.Falsef(t, errors.Is(a, b),
				"sentinel[%d] (%v) MUST NOT be errors.Is sentinel[%d] (%v)", i, a, j, b)
		}
	}
}

// TestShutdown_AgainstRealSSHHost — integration test gated on real
// SSH host env vars per the round-40 spec. Loud SKIP-OK marker so
// `make no-silent-skips` surfaces the conditional coverage.
//
// To run: export VISIONENGINE_TEST_SSH_HOST=<host>
//
//	VISIONENGINE_TEST_SSH_USER=<user>
//	VISIONENGINE_TEST_SSH_KEY=/path/to/key
//	VISIONENGINE_TEST_SSH_KNOWN_HOSTS=/path/to/known_hosts
func TestShutdown_AgainstRealSSHHost(t *testing.T) {
	host := os.Getenv("VISIONENGINE_TEST_SSH_HOST")
	user := os.Getenv("VISIONENGINE_TEST_SSH_USER")
	keyPath := os.Getenv("VISIONENGINE_TEST_SSH_KEY")
	knownHosts := os.Getenv("VISIONENGINE_TEST_SSH_KNOWN_HOSTS")
	if host == "" || user == "" || keyPath == "" || knownHosts == "" {
		t.Skip("SKIP-OK: #VISIONENGINE-SSH-REAL-ROUND40 — requires real SSH host; set VISIONENGINE_TEST_SSH_{HOST,USER,KEY,KNOWN_HOSTS} to enable")
	}

	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     host,
		BasePort: 18080,
	}).WithSSHConfig(remote.SSHConfig{
		Host:           host,
		User:           user,
		KeyPath:        keyPath,
		KnownHostsPath: knownHosts,
		Timeout:        15 * time.Second,
	})
	pool.AssignSlots([]remote.SlotTarget{{Platform: "test"}})

	err := pool.Shutdown(context.Background())
	require.NoError(t, err, "real SSH Shutdown must succeed against %s", host)
}

// TestEnsureReady_AgainstRealSSHHost — integration test gated on
// real SSH host env vars. Probes a known-open port (defaults to 22,
// the SSH port itself, which is reachable iff the SSH dial worked).
func TestEnsureReady_AgainstRealSSHHost(t *testing.T) {
	host := os.Getenv("VISIONENGINE_TEST_SSH_HOST")
	user := os.Getenv("VISIONENGINE_TEST_SSH_USER")
	keyPath := os.Getenv("VISIONENGINE_TEST_SSH_KEY")
	knownHosts := os.Getenv("VISIONENGINE_TEST_SSH_KNOWN_HOSTS")
	if host == "" || user == "" || keyPath == "" || knownHosts == "" {
		t.Skip("SKIP-OK: #VISIONENGINE-SSH-REAL-ROUND40 — requires real SSH host; set VISIONENGINE_TEST_SSH_{HOST,USER,KEY,KNOWN_HOSTS} to enable")
	}

	// Probe SSH port (22) — guaranteed open if SSH dialled.
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     host,
		BasePort: 22,
	}).WithSSHConfig(remote.SSHConfig{
		Host:             host,
		User:             user,
		KeyPath:          keyPath,
		KnownHostsPath:   knownHosts,
		Timeout:          15 * time.Second,
		BackendProbePort: 22,
	})

	err := pool.EnsureReady(context.Background())
	require.NoError(t, err, "real SSH EnsureReady must succeed against %s:22", host)
}

func TestVisionSlot_LockUnlock(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		BasePort: 8080,
	})
	pool.AssignSlots([]remote.SlotTarget{
		{Platform: "android", Device: "dev1"},
	})

	slot := pool.GetSlot("android", "dev1")
	require.NotNil(t, slot)

	// Lock/unlock should not deadlock.
	slot.Lock()
	slot.Unlock()
}

func TestVisionSlot_RecordCall(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		BasePort: 8080,
	})
	pool.AssignSlots([]remote.SlotTarget{
		{Platform: "web"},
	})

	slot := pool.GetSlot("web", "")
	require.NotNil(t, slot)

	slot.RecordCall(100*time.Millisecond, nil)
	slot.RecordCall(200*time.Millisecond, assert.AnError)

	calls, totalTime, errors := slot.Stats()
	assert.Equal(t, 2, calls)
	assert.Equal(t, 300*time.Millisecond, totalTime)
	assert.Equal(t, 1, errors)
}

// TestVisionSlot_RecordCall_Stats_ConcurrentNoRace is a permanent
// regression guard for a real data race found in the 2026-07-10
// adversarial audit: RecordCall/Stats mutated/read plain int and
// time.Duration fields with no synchronization of their own. A
// concurrent Stats()-polling goroutine (an ordinary usage pattern —
// e.g. a metrics reporter) racing against RecordCall() from inference
// goroutines was flagged by `go test -race`. Captured RED evidence:
// qa-results/audit_20260710/RED_visionslot_stats_race.txt. Run with
// `go test -race` to exercise the guard.
func TestVisionSlot_RecordCall_Stats_ConcurrentNoRace(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:     "thinker.local",
		BasePort: 8080,
	})
	pool.AssignSlots([]remote.SlotTarget{
		{Platform: "web"},
	})
	slot := pool.GetSlot("web", "")
	require.NotNil(t, slot)

	const n = 60
	var wg sync.WaitGroup
	wg.Add(n * 2)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			slot.RecordCall(time.Millisecond, nil)
		}()
		go func() {
			defer wg.Done()
			_, _, _ = slot.Stats()
		}()
	}
	wg.Wait()

	calls, totalTime, errs := slot.Stats()
	assert.Equal(t, n, calls)
	assert.Equal(t, time.Duration(n)*time.Millisecond, totalTime)
	assert.Equal(t, 0, errs)
}

func TestNewLlamaCppDeployer(t *testing.T) {
	deployer := remote.NewLlamaCppDeployer(remote.LlamaCppConfig{
		Host:        "thinker.local",
		User:        "admin",
		ModelPath:   "/models/llava.gguf",
		MMProjPath:  "/models/mmproj.gguf",
		BasePort:    8080,
		GPULayers:   -1,
		ContextSize: 4096,
	})
	require.NotNil(t, deployer)
}

func TestBackendConstants(t *testing.T) {
	assert.Equal(t, "ollama", remote.BackendOllama)
	assert.Equal(t, "llama-cpp", remote.BackendLlamaCpp)
}

func TestSlotTarget_Fields(t *testing.T) {
	target := remote.SlotTarget{
		Platform: "android",
		Device:   "emulator-5554",
	}
	assert.Equal(t, "android", target.Platform)
	assert.Equal(t, "emulator-5554", target.Device)
}

func TestPoolConfig_AllFields(t *testing.T) {
	cfg := remote.PoolConfig{
		Host:             "gpu-host",
		User:             "user",
		Model:            "model",
		Shared:           true,
		InferenceBackend: remote.BackendLlamaCpp,
		BasePort:         9000,
		LlamaCpp: &remote.LlamaCppConfig{
			Host:        "gpu-host",
			User:        "user",
			RepoDir:     "~/llama.cpp",
			ModelPath:   "/models/model.gguf",
			MMProjPath:  "/models/proj.gguf",
			BasePort:    9000,
			GPULayers:   32,
			ContextSize: 2048,
		},
	}
	assert.Equal(t, "gpu-host", cfg.Host)
	assert.Equal(t, remote.BackendLlamaCpp, cfg.InferenceBackend)
	assert.NotNil(t, cfg.LlamaCpp)
	assert.Equal(t, 32, cfg.LlamaCpp.GPULayers)
}

func TestVisionSlot_AcquireRelease(t *testing.T) {
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:                 "thinker.local",
		BasePort:             8080,
		MaxConcurrentPerSlot: 2,
	})
	pool.AssignSlots([]remote.SlotTarget{
		{Platform: "android", Device: "dev1"},
	})

	slot := pool.GetSlot("android", "dev1")
	require.NotNil(t, slot)

	// Basic acquire/release cycle must not deadlock.
	slot.Acquire()
	slot.Release()

	slot.Acquire()
	slot.Acquire()
	slot.Release()
	slot.Release()
}

func TestVisionSlot_Semaphore_Disabled(t *testing.T) {
	// MaxConcurrentPerSlot=0 means unlimited; Acquire must never block.
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:                 "thinker.local",
		BasePort:             8080,
		MaxConcurrentPerSlot: 0,
	})
	pool.AssignSlots([]remote.SlotTarget{
		{Platform: "web"},
	})

	slot := pool.GetSlot("web", "")
	require.NotNil(t, slot)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// All of these must return immediately.
		for range 100 {
			slot.Acquire()
			slot.Release()
		}
	}()

	select {
	case <-done:
		// pass
	case <-time.After(2 * time.Second):
		t.Fatal("Acquire blocked with semaphore disabled")
	}
}

func TestVisionSlot_Semaphore_Limits(t *testing.T) {
	const maxConcurrent = 2
	pool := remote.NewVisionPool(remote.PoolConfig{
		Host:                 "thinker.local",
		BasePort:             8080,
		MaxConcurrentPerSlot: maxConcurrent,
	})
	pool.AssignSlots([]remote.SlotTarget{
		{Platform: "android", Device: "dev1"},
	})

	slot := pool.GetSlot("android", "dev1")
	require.NotNil(t, slot)

	// Saturate the semaphore.
	slot.Acquire()
	slot.Acquire()

	// A third Acquire must block; confirm it does not complete
	// within a short window.
	blocked := make(chan struct{})
	go func() {
		slot.Acquire()
		close(blocked)
	}()

	select {
	case <-blocked:
		t.Fatal("third Acquire should have blocked but returned immediately")
	case <-time.After(100 * time.Millisecond):
		// Expected: still waiting.
	}

	// Release one slot; the blocked goroutine should now proceed.
	slot.Release()

	select {
	case <-blocked:
		// pass
	case <-time.After(2 * time.Second):
		t.Fatal("blocked Acquire did not unblock after Release")
	}

	// Clean up.
	slot.Release()
}
