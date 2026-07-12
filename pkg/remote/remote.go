// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package remote provides multi-instance vision inference pool
// management for remote GPU hosts. It supports both Ollama and
// llama.cpp backends, with per-device slot assignment for
// zero-contention parallel vision analysis.
package remote

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ErrBackendVerificationNotImplemented is returned by
// VisionPool.EnsureReady after config validation succeeds, to
// signal that the PoolConfig is well-formed but the remote
// backend has NOT been probed for actual reachability.
//
// Round-28 §11.4 audit (2026-05-17): the previous EnsureReady
// body carried an inline `// In production, this would SSH to
// the host and verify the backend is running. For now, we
// validate config.` comment — a textbook deferred-
// implementation tell that misled callers gating on
// `pool.EnsureReady() == nil` into believing the backend was
// reachable. Config-validation passing is necessary but not
// sufficient for end-user usability per CONST-035 / Article XI
// §11.9. Callers MUST perform an independent reachability
// probe (HTTP probe against the inference endpoint, TCP dial
// against the backend port, or SSH check against the host)
// until SSH-backed verification is wired into this package.
//
// Constitutional anchors: CONST-035 (anti-bluff), CONST-050(A)
// (no-fakes-beyond-unit-tests), Article XI §11.9 (forensic
// anchor).
var ErrBackendVerificationNotImplemented = fmt.Errorf("visionengine: EnsureReady cannot verify backend availability without SSH wiring — config validation is necessary but not sufficient. Caller MUST verify backend reachability via independent health check (HTTP probe, TCP dial) until SSH-based verification is wired (§11.4 PASS-bluff: 'config-validated' ≠ 'backend-reachable')")

// ErrShutdownRemoteCleanupNotImplemented is returned by
// VisionPool.Shutdown to signal that local slot state was
// cleared but remote llama-server / Ollama-server processes on
// VisionPool.config.Host were NOT terminated.
//
// Round-28 §11.4 audit (2026-05-17): the previous Shutdown
// documented "for llama.cpp backends, this terminates remote
// server processes" but only cleared the local slots map.
// That mismatch orphaned remote inference processes (port
// leak, GPU-VRAM leak, per-host disk-cache growth) every time
// a consuming process shut down. Doc-comment in the body — "in
// production this would also SSH to the host and kill
// llama-server processes" — was a textbook §11.4 deferred-
// implementation tell that masked the live gap.
//
// Until SSH-backed remote-process termination is wired into
// this package (it requires SSH credentials that the consuming
// project's runtime holds but VisionPool does not), callers
// MUST explicitly terminate remote llama-server / Ollama-
// server processes themselves. Shutdown also emits a WARN log
// line per pool documenting the orphan state.
//
// Constitutional anchors: CONST-035 (anti-bluff), CONST-050(A)
// (no-fakes-beyond-unit-tests), Article XI §11.9 (forensic
// anchor).
var ErrShutdownRemoteCleanupNotImplemented = fmt.Errorf("visionengine: Shutdown only clears local pool state; remote llama-server processes are NOT killed (orphan-process gap, §11.4 deferred-implementation). Caller MUST manually terminate remote llama-server processes until this is wired (e.g., via SSH client in the consuming process)")

// InferenceBackend identifies the vision inference engine.
const (
	// BackendOllama uses Ollama's API for vision inference.
	BackendOllama = "ollama"
	// BackendLlamaCpp uses llama.cpp llama-server instances.
	BackendLlamaCpp = "llama-cpp"
)

// PoolConfig holds the configuration for a VisionPool.
type PoolConfig struct {
	// Host is the hostname of the remote machine running
	// the inference backend (e.g. "thinker.local").
	Host string

	// User is the SSH user for the remote host.
	User string

	// Model is the model identifier for Ollama (e.g.
	// "llava:7b") or a display name for llama.cpp.
	Model string

	// Shared indicates whether all devices share a single
	// inference endpoint. When false, one slot per device
	// is created.
	Shared bool

	// InferenceBackend selects the backend engine. Defaults
	// to BackendOllama if empty.
	InferenceBackend string

	// BasePort is the starting port for llama-server
	// instances. Each slot increments from this base.
	BasePort int

	// LlamaCpp holds llama.cpp-specific configuration.
	// Required when InferenceBackend is BackendLlamaCpp.
	LlamaCpp *LlamaCppConfig

	// MaxConcurrentPerSlot limits concurrent inference
	// calls per slot. 0 means unlimited.
	MaxConcurrentPerSlot int
}

// LlamaCppConfig holds configuration for llama.cpp server
// instances on the remote host.
type LlamaCppConfig struct {
	// Host is the hostname of the remote machine.
	Host string

	// User is the SSH user for the remote host.
	User string

	// RepoDir is the llama.cpp source directory on the
	// remote host (e.g. "~/llama.cpp").
	RepoDir string

	// ModelPath is the path to the GGUF model file on the
	// remote host.
	ModelPath string

	// MMProjPath is the path to the multimodal projector
	// GGUF on the remote host.
	MMProjPath string

	// BasePort is the starting port for llama-server
	// instances.
	BasePort int

	// GPULayers is the number of layers to offload to GPU.
	// Use -1 for all layers.
	GPULayers int

	// ContextSize is the context window size for the
	// server.
	ContextSize int
}

// SlotTarget identifies a platform+device combination that
// needs a dedicated vision inference slot.
type SlotTarget struct {
	// Platform is the platform identifier (e.g. "android",
	// "web").
	Platform string

	// Device is the device identifier (e.g. ADB serial or
	// "browser"). Empty for platforms with a single device.
	Device string
}

// VisionSlot represents a single inference endpoint assigned
// to a specific platform+device combination. It provides
// mutual exclusion so that only one goroutine accesses the
// endpoint at a time.
type VisionSlot struct {
	// ID is a unique identifier for this slot.
	ID string

	// Endpoint is the full HTTP URL for the inference API
	// (e.g. "http://thinker.local:8081/v1/chat/completions").
	Endpoint string

	// Port is the port number for this slot's server.
	Port int

	// mu guards exclusive endpoint access via the public
	// Lock()/Unlock() methods below. It is INTENTIONALLY separate
	// from the atomic counters below — see the 2026-07-10
	// adversarial-audit note on RecordCall/Stats for why.
	mu sync.Mutex

	// calls / totalTimeNanos / errors back the diagnostics
	// surfaced by RecordCall/Stats. They are atomic.Int64 instead
	// of plain int/time.Duration — round 2026-07-10 audit finding:
	// the previous plain-int fields were mutated by RecordCall and
	// read by Stats with NO synchronization of their own; Stats
	// documented no locking requirement and the module's own
	// existing test (TestVisionSlot_RecordCall) already calls
	// RecordCall without holding the public Lock(), so the "callers
	// serialize via Lock()" convention was not actually the real
	// contract. A concurrent Stats()-polling goroutine (a
	// perfectly ordinary usage — metrics reporter reading live
	// counters while inference goroutines record calls) raced with
	// RecordCall under `go test -race` (captured RED evidence:
	// qa-results/audit_20260710/RED_visionslot_stats_race.txt).
	// Using atomics protects the counters unconditionally,
	// independent of whether any caller also holds the public
	// Lock() — and deliberately does NOT reuse `mu` for this,
	// because a caller following the documented
	// Lock()-around-the-network-call pattern
	// (`slot.Lock(); ...; slot.RecordCall(...); slot.Unlock()`)
	// would deadlock against Go's non-reentrant sync.Mutex if
	// RecordCall also tried to acquire `mu`.
	calls          atomic.Int64
	totalTimeNanos atomic.Int64
	errors         atomic.Int64

	sem chan struct{} // concurrency limiter; nil means unlimited
}

// Lock acquires exclusive access to this slot.
func (s *VisionSlot) Lock() {
	s.mu.Lock()
}

// Unlock releases exclusive access to this slot.
func (s *VisionSlot) Unlock() {
	s.mu.Unlock()
}

// Acquire blocks until a concurrency slot is available.
// Returns immediately if no semaphore is configured.
func (s *VisionSlot) Acquire() {
	if s.sem != nil {
		s.sem <- struct{}{}
	}
}

// Release frees a concurrency slot.
func (s *VisionSlot) Release() {
	if s.sem != nil {
		<-s.sem
	}
}

// RecordCall records a vision inference call's duration and
// error status for diagnostics. Safe for concurrent use by any
// number of callers, independent of whether they also hold the
// slot's public Lock() — see the VisionSlot field-comment above.
func (s *VisionSlot) RecordCall(duration time.Duration, err error) {
	s.calls.Add(1)
	s.totalTimeNanos.Add(int64(duration))
	if err != nil {
		s.errors.Add(1)
	}
}

// Stats returns the number of calls, total time, and error
// count for this slot. Safe for concurrent use with RecordCall.
func (s *VisionSlot) Stats() (calls int, totalTime time.Duration, errors int) {
	return int(s.calls.Load()), time.Duration(s.totalTimeNanos.Load()), int(s.errors.Load())
}

// VisionPool manages a set of inference endpoints, one per
// platform+device combination (or a single shared endpoint).
type VisionPool struct {
	config    PoolConfig
	sshConfig SSHConfig
	slots     map[string]*VisionSlot
	mu        sync.RWMutex
}

// NewVisionPool creates a VisionPool with the given
// configuration. Slots are not created until AssignSlots is
// called.
//
// SSHConfig is left zero-valued; callers that want real
// remote-process termination on Shutdown OR real backend
// reachability probes on EnsureReady MUST also call
// WithSSHConfig. When SSHConfig is zero-valued, Shutdown
// preserves its round-28 ErrShutdownRemoteCleanupNotImplemented
// contract and EnsureReady preserves its round-28
// ErrBackendVerificationNotImplemented contract.
func NewVisionPool(config PoolConfig) *VisionPool {
	if config.InferenceBackend == "" {
		config.InferenceBackend = BackendOllama
	}
	if config.BasePort == 0 {
		config.BasePort = 8080
	}
	return &VisionPool{
		config: config,
		slots:  make(map[string]*VisionSlot),
	}
}

// WithSSHConfig attaches SSH credentials so VisionPool can
// perform real remote-process management and backend
// reachability probes. Returns the same pool for chaining.
// Callers MUST source SSHConfig fields from env vars /
// config files — never hardcode them (CONST-042).
func (p *VisionPool) WithSSHConfig(cfg SSHConfig) *VisionPool {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sshConfig = cfg
	return p
}

// SSHConfigured reports whether the pool has SSH credentials
// attached. False means Shutdown's remote-cleanup path and
// EnsureReady's reachability path are disabled (round-28
// sentinels preserved).
func (p *VisionPool) SSHConfigured() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sshConfig.Host != ""
}

// EnsureReady validates the pool's PoolConfig AND (when
// SSHConfig is populated) actually probes the remote inference
// backend's TCP port over the SSH connection.
//
// Round-40 §11.4 audit (2026-05-18): wires real SSH-based
// backend reachability probing into the round-28 sentinel-
// returning skeleton. Behaviour now bifurcates:
//
//  1. Malformed PoolConfig (missing host, missing llama.cpp
//     config when backend == BackendLlamaCpp, etc.): a
//     descriptive fmt.Errorf is returned (unchanged from
//     round-28).
//
//  2. Well-formed PoolConfig + SSHConfig.Host == "" (round-28
//     contract): returns ErrBackendVerificationNotImplemented
//     so callers know config-validation passed but reachability
//     is unproven.
//
//  3. Well-formed PoolConfig + SSHConfig populated (round-40
//     wiring): dials SSH, runs "nc -z <host> <port>" against
//     SSHConfig.BackendProbePort (defaults to PoolConfig.BasePort),
//     and asserts the probe succeeds. Returns nil on success;
//     ErrBackendNotReachable on probe failure;
//     ErrSSHKeyParseFailed / ErrSSHHostKeyVerificationFailed
//     on SSH-setup failures.
func (p *VisionPool) EnsureReady(ctx context.Context) error {
	p.mu.RLock()
	cfg := p.config
	sshCfg := p.sshConfig
	p.mu.RUnlock()

	if cfg.Host == "" {
		return fmt.Errorf("remote: vision pool host is required")
	}
	if cfg.InferenceBackend == BackendLlamaCpp && cfg.LlamaCpp == nil {
		return fmt.Errorf(
			"remote: llama.cpp config required for backend %q",
			BackendLlamaCpp)
	}

	// Round-28 path: SSH NOT configured → preserve sentinel.
	if sshCfg.Host == "" {
		return ErrBackendVerificationNotImplemented
	}

	// Round-40 path: SSH configured → actually probe backend port.
	client, err := sshConn(ctx, sshCfg)
	if err != nil {
		return fmt.Errorf("visionengine: EnsureReady SSH dial failed: %w", err)
	}
	defer client.Close()

	probePort := sshCfg.BackendProbePort
	if probePort == 0 {
		probePort = cfg.BasePort
	}
	// "nc -z" returns 0 if port open, non-zero otherwise.
	// We append a literal "READY" marker on success so a partial
	// connection (e.g. nc absent → 127) is distinguishable from
	// a successful zero-RTT probe.
	cmdline := fmt.Sprintf(
		"nc -z -w 5 %s %d 2>&1 && echo PROBE_READY",
		shellEscape(cfg.Host), probePort)
	out, runErr := runRemote(client, cmdline)
	output := strings.TrimSpace(string(out))
	if runErr != nil || !strings.Contains(output, "PROBE_READY") {
		return fmt.Errorf("%w: probed %s:%d via %s — output=%q err=%v",
			ErrBackendNotReachable, cfg.Host, probePort, sshCfg.Host, output, runErr)
	}
	return nil
}

// shellEscape wraps an argument in single quotes for safe
// embedding into an SSH-invoked shell command. Single quotes
// inside the argument are escaped via the standard
// `'\”` sequence.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// AssignSlots creates inference endpoint slots for each target
// platform+device combination. If the pool is shared, all
// targets map to the same endpoint.
func (p *VisionPool) AssignSlots(targets []SlotTarget) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.config.Shared {
		shared := &VisionSlot{
			ID:       "shared",
			Endpoint: fmt.Sprintf("http://%s:%d", p.config.Host, p.config.BasePort),
			Port:     p.config.BasePort,
		}
		if p.config.MaxConcurrentPerSlot > 0 {
			shared.sem = make(chan struct{}, p.config.MaxConcurrentPerSlot)
		}
		for _, t := range targets {
			key := slotKey(t.Platform, t.Device)
			p.slots[key] = shared
		}
		return
	}

	port := p.config.BasePort
	for _, t := range targets {
		key := slotKey(t.Platform, t.Device)
		slot := &VisionSlot{
			ID:       key,
			Endpoint: fmt.Sprintf("http://%s:%d", p.config.Host, port),
			Port:     port,
		}
		if p.config.MaxConcurrentPerSlot > 0 {
			slot.sem = make(chan struct{}, p.config.MaxConcurrentPerSlot)
		}
		p.slots[key] = slot
		port++
	}
}

// GetSlot returns the VisionSlot assigned to the given
// platform+device combination, or nil if no slot exists.
func (p *VisionPool) GetSlot(platform, device string) *VisionSlot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.slots[slotKey(platform, device)]
}

// Size returns the number of assigned slots.
func (p *VisionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.slots)
}

// Shutdown clears local inference-slot bookkeeping for this pool AND
// (when SSHConfig is populated) actually SSHes to the remote host and
// terminates llama-server / Ollama-server processes.
//
// Round-40 §11.4 audit (2026-05-18): wires real SSH-based remote
// process termination into the round-28 sentinel-returning skeleton.
// Behaviour now bifurcates:
//
//  1. SSHConfig.Host == "" (round-28 contract): local pool state is
//     cleared, ErrShutdownRemoteCleanupNotImplemented is returned,
//     and a WARN log line is emitted listing the leaked endpoints.
//     This preserves the round-28 sentinel for callers that have not
//     yet opted into SSH-backed cleanup.
//
//  2. SSHConfig.Host != "" (round-40 wiring): an SSH session is
//     dialled per-call (lazy connection, avoids long-lived state),
//     authenticated via SSHConfig.KeyPath, and host-key-verified
//     against SSHConfig.KnownHostsPath. For each tracked slot port a
//     "fuser -k <port>/tcp" command is issued; a final
//     "pkill -f llama-server|ollama serve" sweeps any straggler.
//     Per-command errors aggregate via errors.Join.
//
// Local pool state is ALWAYS cleared — that part of Shutdown's
// contract has never been the gap.
func (p *VisionPool) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	host := p.config.Host
	backend := p.config.InferenceBackend
	basePort := p.config.BasePort
	slotCount := len(p.slots)

	leakedEndpoints := make([]string, 0, slotCount)
	leakedPorts := make([]int, 0, slotCount)
	for _, slot := range p.slots {
		leakedEndpoints = append(leakedEndpoints, slot.Endpoint)
		leakedPorts = append(leakedPorts, slot.Port)
	}

	// Local pool state is always cleared — that part of Shutdown's
	// contract has never been the gap; the gap is remote cleanup.
	p.slots = make(map[string]*VisionSlot)

	// Round-28 path: SSH NOT configured → preserve sentinel + WARN log.
	if p.sshConfig.Host == "" {
		log.Printf("WARN visionengine/remote.VisionPool.Shutdown: local pool state cleared but %d remote %s slot(s) on host=%q (base port=%d, endpoints=%v) were NOT terminated — SSHConfig is unset; call WithSSHConfig to enable real remote-cleanup. See ErrShutdownRemoteCleanupNotImplemented.",
			slotCount, backend, host, basePort, leakedEndpoints)
		return ErrShutdownRemoteCleanupNotImplemented
	}

	// Round-40 path: SSH configured → actually kill remote processes.
	client, err := sshConn(ctx, p.sshConfig)
	if err != nil {
		log.Printf("WARN visionengine/remote.VisionPool.Shutdown: SSH dial to %s@%s failed; %d remote %s slot(s) on host=%q (endpoints=%v) NOT terminated: %v",
			p.sshConfig.User, p.sshConfig.Host, slotCount, backend, host, leakedEndpoints, err)
		return fmt.Errorf("visionengine: Shutdown SSH dial failed: %w", err)
	}
	defer client.Close()

	var killErrs []error
	for _, port := range leakedPorts {
		cmdline := fmt.Sprintf("fuser -k -n tcp %d 2>&1 || true", port)
		out, err := runRemote(client, cmdline)
		if err != nil {
			killErrs = append(killErrs, fmt.Errorf("kill port %d: %w (stderr: %s)", port, err, strings.TrimSpace(string(out))))
		}
	}

	cmdline := "pkill -f 'llama-server|ollama serve' 2>&1 || true"
	if out, err := runRemote(client, cmdline); err != nil {
		killErrs = append(killErrs, fmt.Errorf("pkill sweep: %w (stderr: %s)", err, strings.TrimSpace(string(out))))
	}

	if len(killErrs) > 0 {
		aggregated := errors.Join(killErrs...)
		log.Printf("WARN visionengine/remote.VisionPool.Shutdown: SSH connected to %s but %d kill operation(s) reported errors on host=%q (endpoints=%v): %v",
			p.sshConfig.Host, len(killErrs), host, leakedEndpoints, aggregated)
		return fmt.Errorf("visionengine: Shutdown remote kill errors: %w", aggregated)
	}

	log.Printf("INFO visionengine/remote.VisionPool.Shutdown: SSH-terminated %d remote %s slot(s) on host=%q (base port=%d, endpoints=%v) — orphan-process gap closed.",
		slotCount, backend, host, basePort, leakedEndpoints)
	return nil
}

// slotKey generates a unique key for a platform+device pair.
func slotKey(platform, device string) string {
	if device == "" {
		return platform
	}
	return platform + ":" + device
}
