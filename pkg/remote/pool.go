// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package remote provides automatic deployment and lifecycle
// management of Ollama vision models on remote hosts via SSH.

package remote

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// VisionSlot represents a dedicated vision inference slot
// assigned to one platform/device. Each slot queues requests
// sequentially so one slow inference doesn't block other
// platforms' vision calls.
type VisionSlot struct {
	// ID is the slot identifier (e.g. "androidtv-192.168.0.134:5555").
	ID string

	// Platform is the target platform (e.g. "androidtv", "web").
	Platform string

	// Device is the device identifier (e.g. ADB serial or URL).
	Device string

	// Endpoint is the Ollama API URL this slot uses.
	Endpoint string

	// Port is the Ollama port for this slot.
	Port int

	// mu serializes vision calls for this slot.
	mu sync.Mutex

	// stats tracks slot usage.
	stats SlotStats
}

// SlotStats tracks usage metrics for a vision slot.
type SlotStats struct {
	TotalCalls    int           `json:"total_calls"`
	TotalDuration time.Duration `json:"total_duration"`
	LastCallAt    time.Time     `json:"last_call_at,omitempty"`
	Errors        int           `json:"errors"`
}

// Lock acquires exclusive access to this slot. Each platform's
// curiosity phase should hold the lock during its vision call
// to prevent contention.
func (s *VisionSlot) Lock() {
	s.mu.Lock()
}

// Unlock releases the slot.
func (s *VisionSlot) Unlock() {
	s.mu.Unlock()
}

// RecordCall updates slot statistics after a vision call.
func (s *VisionSlot) RecordCall(dur time.Duration, err error) {
	s.stats.TotalCalls++
	s.stats.TotalDuration += dur
	s.stats.LastCallAt = time.Now()
	if err != nil {
		s.stats.Errors++
	}
}

// Stats returns a copy of the slot's usage statistics.
func (s *VisionSlot) Stats() SlotStats {
	return s.stats
}

// PoolConfig configures a VisionPool.
type PoolConfig struct {
	// Host is the Ollama server hostname.
	Host string

	// User is the SSH user for deployment.
	User string

	// BasePort is the starting port for Ollama instances.
	// The pool allocates BasePort, BasePort+1, ... for each
	// slot. When running against a single shared Ollama
	// instance, all slots share BasePort.
	BasePort int

	// Model is the vision model to use.
	Model string

	// Shared indicates all slots use one Ollama instance
	// (true) or each slot gets its own instance on a
	// dedicated port (false). Shared mode is the default
	// since Ollama handles concurrent requests natively.
	Shared bool
}

// VisionPool manages a set of VisionSlots, one per
// platform/device being tested. It ensures the Ollama backend
// is ready and assigns dedicated slots to each QA target.
type VisionPool struct {
	cfg      PoolConfig
	deployer *Deployer
	slots    map[string]*VisionSlot
	mu       sync.Mutex
}

// NewVisionPool creates a pool backed by the given config.
func NewVisionPool(cfg PoolConfig) *VisionPool {
	if cfg.BasePort == 0 {
		cfg.BasePort = 11434
	}
	if cfg.Model == "" {
		cfg.Model = "llava:7b"
	}
	return &VisionPool{
		cfg: cfg,
		deployer: NewDeployer(Config{
			Host:       cfg.Host,
			User:       cfg.User,
			Model:      cfg.Model,
			OllamaPort: cfg.BasePort,
		}),
		slots: make(map[string]*VisionSlot),
	}
}

// EnsureReady verifies the Ollama backend is running and the
// model is available. Call this before assigning slots.
func (p *VisionPool) EnsureReady(
	ctx context.Context,
) error {
	endpoint, err := p.deployer.EnsureReady(ctx)
	if err != nil {
		return fmt.Errorf("vision pool: %w", err)
	}
	fmt.Printf(
		"[vision-pool] backend ready at %s\n",
		endpoint,
	)
	return nil
}

// SlotTarget describes a QA target that needs a vision slot.
type SlotTarget struct {
	Platform string // e.g. "androidtv", "web", "api"
	Device   string // e.g. "192.168.0.134:5555", "localhost:3000"
}

// AssignSlots creates dedicated VisionSlots for each target.
// In shared mode, all slots point to the same Ollama endpoint
// but have independent locks for request serialization.
// In dedicated mode, each slot gets its own port.
func (p *VisionPool) AssignSlots(
	targets []SlotTarget,
) []*VisionSlot {
	p.mu.Lock()
	defer p.mu.Unlock()

	var result []*VisionSlot
	for i, t := range targets {
		id := fmt.Sprintf("%s-%s", t.Platform, t.Device)
		if id == fmt.Sprintf("%s-", t.Platform) {
			id = fmt.Sprintf("%s-%d", t.Platform, i)
		}

		port := p.cfg.BasePort
		if !p.cfg.Shared {
			port = p.cfg.BasePort + i
		}

		endpoint := fmt.Sprintf(
			"http://%s:%d", p.cfg.Host, port,
		)

		slot := &VisionSlot{
			ID:       id,
			Platform: t.Platform,
			Device:   t.Device,
			Endpoint: endpoint,
			Port:     port,
		}
		p.slots[id] = slot
		result = append(result, slot)

		fmt.Printf(
			"[vision-pool] slot %s -> %s\n",
			id, endpoint,
		)
	}
	return result
}

// GetSlot returns the slot for the given platform and device.
func (p *VisionPool) GetSlot(
	platform, device string,
) *VisionSlot {
	p.mu.Lock()
	defer p.mu.Unlock()

	id := fmt.Sprintf("%s-%s", platform, device)
	if slot, ok := p.slots[id]; ok {
		return slot
	}
	// Fallback: find by platform only.
	for _, slot := range p.slots {
		if slot.Platform == platform {
			return slot
		}
	}
	return nil
}

// AllSlots returns all assigned slots.
func (p *VisionPool) AllSlots() []*VisionSlot {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]*VisionSlot, 0, len(p.slots))
	for _, s := range p.slots {
		result = append(result, s)
	}
	return result
}

// Size returns the number of assigned slots.
func (p *VisionPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.slots)
}

// PrintStats logs per-slot usage statistics.
func (p *VisionPool) PrintStats() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, s := range p.slots {
		st := s.Stats()
		avg := time.Duration(0)
		if st.TotalCalls > 0 {
			avg = st.TotalDuration / time.Duration(
				st.TotalCalls,
			)
		}
		fmt.Printf(
			"[vision-pool] %s: %d calls, "+
				"avg %v, %d errors\n",
			s.ID, st.TotalCalls, avg.Round(
				time.Millisecond,
			), st.Errors,
		)
	}
}

// Shutdown stops any dedicated Ollama instances started by
// the pool. In shared mode, this is a no-op (the main
// Ollama service is not affected).
func (p *VisionPool) Shutdown(ctx context.Context) {
	if p.cfg.Shared {
		p.PrintStats()
		return
	}
	// Kill dedicated instances.
	for _, slot := range p.AllSlots() {
		if slot.Port != p.cfg.BasePort {
			cmd := fmt.Sprintf(
				"pkill -f 'ollama.*%d'",
				slot.Port,
			)
			_, _ = p.deployer.sshRun(ctx, cmd)
		}
	}
	p.PrintStats()
}
