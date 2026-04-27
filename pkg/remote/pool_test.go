// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVisionPool_Defaults(t *testing.T) {
	p := NewVisionPool(PoolConfig{Host: "gpu.local"})
	assert.Equal(t, 11434, p.cfg.BasePort)
	assert.Equal(t, "llava:7b", p.cfg.Model)
	assert.Equal(t, 0, p.Size())
}

func TestVisionPool_AssignSlots_Shared(t *testing.T) {
	p := NewVisionPool(PoolConfig{
		Host:   "gpu.local",
		Shared: true,
	})
	targets := []SlotTarget{
		{Platform: "androidtv", Device: "192.168.0.134:5555"},
		{Platform: "androidtv", Device: "192.168.0.214:5555"},
		{Platform: "androidtv", Device: "192.168.0.233:5555"},
		{Platform: "web", Device: "localhost:3000"},
		{Platform: "api", Device: "localhost:8080"},
	}

	slots := p.AssignSlots(targets)
	require.Len(t, slots, 5)
	assert.Equal(t, 5, p.Size())

	// In shared mode, all slots use the same port.
	for _, s := range slots {
		assert.Equal(t, 11434, s.Port)
		assert.Equal(t,
			"http://gpu.local:11434", s.Endpoint,
		)
	}

	// Each slot has a unique ID.
	ids := make(map[string]bool)
	for _, s := range slots {
		assert.False(t, ids[s.ID], "duplicate ID: %s", s.ID)
		ids[s.ID] = true
	}
}

func TestVisionPool_AssignSlots_Dedicated(t *testing.T) {
	p := NewVisionPool(PoolConfig{
		Host:     "gpu.local",
		BasePort: 11434,
		Shared:   false,
	})
	targets := []SlotTarget{
		{Platform: "androidtv", Device: "dev1"},
		{Platform: "androidtv", Device: "dev2"},
		{Platform: "web", Device: "web"},
	}

	slots := p.AssignSlots(targets)
	require.Len(t, slots, 3)

	// Dedicated mode: each slot gets its own port.
	assert.Equal(t, 11434, slots[0].Port)
	assert.Equal(t, 11435, slots[1].Port)
	assert.Equal(t, 11436, slots[2].Port)

	assert.Equal(t,
		"http://gpu.local:11434", slots[0].Endpoint,
	)
	assert.Equal(t,
		"http://gpu.local:11435", slots[1].Endpoint,
	)
	assert.Equal(t,
		"http://gpu.local:11436", slots[2].Endpoint,
	)
}

func TestVisionPool_GetSlot(t *testing.T) {
	p := NewVisionPool(PoolConfig{
		Host:   "gpu.local",
		Shared: true,
	})
	p.AssignSlots([]SlotTarget{
		{Platform: "androidtv", Device: "dev1"},
		{Platform: "web", Device: "web"},
	})

	// Exact match.
	s := p.GetSlot("androidtv", "dev1")
	require.NotNil(t, s)
	assert.Equal(t, "androidtv", s.Platform)
	assert.Equal(t, "dev1", s.Device)

	// Platform fallback.
	s2 := p.GetSlot("web", "unknown")
	require.NotNil(t, s2)
	assert.Equal(t, "web", s2.Platform)

	// No match.
	s3 := p.GetSlot("desktop", "x")
	assert.Nil(t, s3)
}

func TestVisionSlot_Lock(t *testing.T) {
	s := &VisionSlot{ID: "test-slot"}

	// Lock should be non-blocking on first call.
	done := make(chan bool, 1)
	go func() {
		s.Lock()
		done <- true
		s.Unlock()
	}()

	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Fatal("Lock blocked unexpectedly")
	}
}

func TestVisionSlot_RecordCall(t *testing.T) {
	s := &VisionSlot{ID: "test"}
	assert.Equal(t, 0, s.Stats().TotalCalls)

	s.RecordCall(500*time.Millisecond, nil)
	s.RecordCall(300*time.Millisecond, nil)
	s.RecordCall(200*time.Millisecond,
		assert.AnError,
	)

	st := s.Stats()
	assert.Equal(t, 3, st.TotalCalls)
	assert.Equal(t, time.Second, st.TotalDuration)
	assert.Equal(t, 1, st.Errors)
	assert.False(t, st.LastCallAt.IsZero())
}

func TestVisionPool_Shutdown_Shared(t *testing.T) {
	p := NewVisionPool(PoolConfig{
		Host:   "gpu.local",
		Shared: true,
	})
	p.AssignSlots([]SlotTarget{
		{Platform: "web", Device: "w"},
	})
	// Shared shutdown is a no-op (just prints stats).
	// Should not panic.
	p.Shutdown(nil)
}
