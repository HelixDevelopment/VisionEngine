// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package remote provides multi-instance vision inference pool
// management for remote GPU hosts. It supports both Ollama and
// llama.cpp backends, with per-device slot assignment for
// zero-contention parallel vision analysis.
package remote

import (
	"context"
	"fmt"
	"log"
	"sync"
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
// HelixCode runtime holds but VisionPool does not), callers
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

	mu        sync.Mutex
	calls     int
	totalTime time.Duration
	errors    int
	sem       chan struct{} // concurrency limiter; nil means unlimited
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
// error status for diagnostics.
func (s *VisionSlot) RecordCall(duration time.Duration, err error) {
	s.calls++
	s.totalTime += duration
	if err != nil {
		s.errors++
	}
}

// Stats returns the number of calls, total time, and error
// count for this slot.
func (s *VisionSlot) Stats() (calls int, totalTime time.Duration, errors int) {
	return s.calls, s.totalTime, s.errors
}

// VisionPool manages a set of inference endpoints, one per
// platform+device combination (or a single shared endpoint).
type VisionPool struct {
	config PoolConfig
	slots  map[string]*VisionSlot
	mu     sync.RWMutex
}

// NewVisionPool creates a VisionPool with the given
// configuration. Slots are not created until AssignSlots is
// called.
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

// EnsureReady validates the pool's PoolConfig and returns a
// sentinel signalling that backend reachability has NOT been
// probed.
//
// Round-28 §11.4 audit (2026-05-17): an earlier doc-comment
// claimed EnsureReady "verifies that the inference backend is
// available and responsive on the remote host", but its body
// only validated configuration well-formedness. Any caller
// gating on `pool.EnsureReady() == nil` therefore PASSed
// against a non-reachable backend with no error surfaced.
//
// The honest contract is:
//   - If PoolConfig is malformed (missing host, missing
//     llama.cpp config when backend == BackendLlamaCpp, etc.)
//     a descriptive `fmt.Errorf` value is returned.
//   - If PoolConfig is well-formed, the function returns
//     ErrBackendVerificationNotImplemented — config-validation
//     passed, but backend reachability remains unproven until
//     SSH-backed health-checks are wired into this package.
//     Callers MUST perform an independent reachability probe
//     (HTTP probe, TCP dial, SSH check) before treating the
//     pool as usable.
//
// Returning the sentinel from the well-formed path is
// deliberate per the operator anti-bluff mandate: a silent
// `nil` return would re-introduce the bluff that motivated
// this round-28 fix.
func (p *VisionPool) EnsureReady(ctx context.Context) error {
	_ = ctx // intentionally unused: this is a config-level check
	if p.config.Host == "" {
		return fmt.Errorf("remote: vision pool host is required")
	}
	if p.config.InferenceBackend == BackendLlamaCpp &&
		p.config.LlamaCpp == nil {
		return fmt.Errorf(
			"remote: llama.cpp config required for backend %q",
			BackendLlamaCpp)
	}
	return ErrBackendVerificationNotImplemented
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

// Shutdown clears local inference-slot bookkeeping for this pool.
//
// IMPORTANT — round-28 §11.4 audit (2026-05-17): Shutdown does
// NOT terminate remote llama-server / Ollama-server processes
// on VisionPool.config.Host. Earlier revisions documented
// termination but only cleared the local map, orphaning remote
// processes (port + GPU-VRAM + disk-cache leak). The honest
// behaviour is now reflected here, and Shutdown returns
// ErrShutdownRemoteCleanupNotImplemented so callers can detect
// the gap programmatically. A WARN log line is also emitted per
// pool summarising the orphan-process state (host, backend,
// base port, slot count, leaked endpoints) so operational
// dashboards surface the leak.
//
// Callers MUST explicitly terminate remote inference processes
// themselves (e.g. via the SSH client they already hold to
// deploy llama-server) until SSH-backed cleanup is wired into
// this package.
//
// Local pool state is ALWAYS cleared — Shutdown never
// short-circuits on the sentinel. The sentinel exists purely
// to give callers visibility into the orphan-process gap.
func (p *VisionPool) Shutdown(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	host := p.config.Host
	backend := p.config.InferenceBackend
	basePort := p.config.BasePort
	slotCount := len(p.slots)

	// Snapshot the per-slot endpoint range so the WARN line can
	// list concrete leaked endpoints rather than just a count.
	leakedEndpoints := make([]string, 0, slotCount)
	for _, slot := range p.slots {
		leakedEndpoints = append(leakedEndpoints, slot.Endpoint)
	}

	// Local pool state is always cleared — that part of
	// Shutdown's contract has never been the gap; the gap is
	// remote cleanup.
	p.slots = make(map[string]*VisionSlot)

	log.Printf("WARN visionengine/remote.VisionPool.Shutdown: local pool state cleared but %d remote %s slot(s) on host=%q (base port=%d, endpoints=%v) were NOT terminated — caller must SSH-kill llama-server/ollama-server processes manually. See ErrShutdownRemoteCleanupNotImplemented.",
		slotCount, backend, host, basePort, leakedEndpoints)

	return ErrShutdownRemoteCleanupNotImplemented
}

// slotKey generates a unique key for a platform+device pair.
func slotKey(platform, device string) string {
	if device == "" {
		return platform
	}
	return platform + ":" + device
}
