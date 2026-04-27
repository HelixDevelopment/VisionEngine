// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectStrongestModel_NoHosts(t *testing.T) {
	rec := SelectStrongestModel(nil)
	assert.Equal(t, "llava:7b", rec.ModelName)
	assert.Equal(t, "7B", rec.ModelSize)
	assert.False(t, rec.NeedsDistribution)
}

func TestSelectStrongestModel_SingleGPU_Small(t *testing.T) {
	hosts := []*HostHardware{
		{
			Host:         "gpu1.local",
			HasGPU:       true,
			GPUName:      "NVIDIA RTX 3060",
			GPUMemMB:     12000,
			GPUFreeMemMB: 8000,
			RAMTotalMB:   32000,
			RAMFreeMB:    24000,
		},
	}
	rec := SelectStrongestModel(hosts)
	assert.Equal(t, "llava:7b", rec.ModelName)
	assert.Equal(t, "7B", rec.ModelSize)
	assert.False(t, rec.NeedsDistribution)
	assert.Equal(t, 8000, rec.TotalGPUMemMB)
	assert.Equal(t, []string{"gpu1.local"}, rec.GPUHosts)
}

func TestSelectStrongestModel_SingleGPU_12GB(t *testing.T) {
	hosts := []*HostHardware{
		{
			Host:         "gpu1.local",
			HasGPU:       true,
			GPUName:      "NVIDIA RTX 4080",
			GPUMemMB:     16000,
			GPUFreeMemMB: 14000,
			RAMTotalMB:   64000,
			RAMFreeMB:    48000,
		},
	}
	rec := SelectStrongestModel(hosts)
	assert.Equal(t, "minicpm-v:8b", rec.ModelName)
	assert.Equal(t, "8B", rec.ModelSize)
	assert.False(t, rec.NeedsDistribution)
}

func TestSelectStrongestModel_SingleGPU_24GB(t *testing.T) {
	hosts := []*HostHardware{
		{
			Host:         "gpu1.local",
			HasGPU:       true,
			GPUName:      "NVIDIA RTX 4090",
			GPUMemMB:     24000,
			GPUFreeMemMB: 24000,
			RAMTotalMB:   64000,
			RAMFreeMB:    48000,
		},
	}
	rec := SelectStrongestModel(hosts)
	assert.Equal(t, "qwen2.5-vl:32b", rec.ModelName)
	assert.Equal(t, "32B", rec.ModelSize)
	assert.False(t, rec.NeedsDistribution)
}

func TestSelectStrongestModel_MultiGPU_Distributed(
	t *testing.T,
) {
	hosts := []*HostHardware{
		{
			Host:         "thinker.local",
			HasGPU:       true,
			GPUName:      "NVIDIA RTX 4090",
			GPUMemMB:     24000,
			GPUFreeMemMB: 24000,
			RAMTotalMB:   64000,
			RAMFreeMB:    48000,
		},
		{
			Host:         "amber.local",
			HasGPU:       true,
			GPUName:      "NVIDIA RTX 3090",
			GPUMemMB:     24000,
			GPUFreeMemMB: 24000,
			RAMTotalMB:   32000,
			RAMFreeMB:    24000,
		},
	}
	rec := SelectStrongestModel(hosts)
	assert.Equal(t, "qwen2.5-vl:72b", rec.ModelName)
	assert.Equal(t, "72B", rec.ModelSize)
	assert.True(t, rec.NeedsDistribution)
	assert.Equal(t, 48000, rec.TotalGPUMemMB)
	assert.Len(t, rec.GPUHosts, 2)
}

func TestSelectStrongestModel_CPUOnly_LargeRAM(
	t *testing.T,
) {
	hosts := []*HostHardware{
		{
			Host:       "cpu1.local",
			HasGPU:     false,
			CPUCores:   16,
			RAMTotalMB: 64000,
			RAMFreeMB:  48000,
		},
	}
	rec := SelectStrongestModel(hosts)
	assert.Equal(t, "llava:13b", rec.ModelName)
	assert.Equal(t, "13B", rec.ModelSize)
	assert.Empty(t, rec.GPUHosts)
}

func TestSelectStrongestModel_CPUOnly_SmallRAM(
	t *testing.T,
) {
	hosts := []*HostHardware{
		{
			Host:       "tiny.local",
			HasGPU:     false,
			CPUCores:   4,
			RAMTotalMB: 8000,
			RAMFreeMB:  6000,
		},
	}
	rec := SelectStrongestModel(hosts)
	assert.Equal(t, "llava:7b", rec.ModelName)
	assert.Equal(t, "7B", rec.ModelSize)
	assert.False(t, rec.NeedsDistribution)
}

func TestSelectStrongestModel_CPUOnly_MediumRAM(
	t *testing.T,
) {
	hosts := []*HostHardware{
		{
			Host:       "med.local",
			HasGPU:     false,
			CPUCores:   8,
			RAMTotalMB: 32000,
			RAMFreeMB:  20000,
		},
	}
	rec := SelectStrongestModel(hosts)
	assert.Equal(t, "llava:7b", rec.ModelName)
	assert.Equal(t, "7B", rec.ModelSize)
}

func TestSelectStrongestModel_MixedGPUAndCPU(
	t *testing.T,
) {
	hosts := []*HostHardware{
		{
			Host:         "gpu.local",
			HasGPU:       true,
			GPUName:      "NVIDIA RTX 4090",
			GPUMemMB:     24000,
			GPUFreeMemMB: 24000,
			RAMTotalMB:   64000,
			RAMFreeMB:    48000,
		},
		{
			Host:       "cpu.local",
			HasGPU:     false,
			CPUCores:   32,
			RAMTotalMB: 128000,
			RAMFreeMB:  100000,
		},
	}
	rec := SelectStrongestModel(hosts)
	// GPU selection path uses only GPU VRAM = 24GB.
	assert.Equal(t, "qwen2.5-vl:32b", rec.ModelName)
	assert.Equal(t, "32B", rec.ModelSize)
	assert.True(t, rec.NeedsDistribution)
	assert.Equal(t, 24000, rec.TotalGPUMemMB)
	assert.Equal(t, []string{"gpu.local"}, rec.GPUHosts)
	assert.Len(t, rec.AllHosts, 2)
}

func TestSelectStrongestModel_MultiCPU_Distributed(
	t *testing.T,
) {
	hosts := []*HostHardware{
		{
			Host:       "cpu1.local",
			HasGPU:     false,
			CPUCores:   16,
			RAMTotalMB: 32000,
			RAMFreeMB:  24000,
		},
		{
			Host:       "cpu2.local",
			HasGPU:     false,
			CPUCores:   16,
			RAMTotalMB: 32000,
			RAMFreeMB:  24000,
		},
	}
	rec := SelectStrongestModel(hosts)
	// Combined RAM = 48GB free -> llava:13b.
	assert.Equal(t, "llava:13b", rec.ModelName)
	assert.Equal(t, "13B", rec.ModelSize)
	assert.True(t, rec.NeedsDistribution)
	assert.Equal(t, 48000, rec.TotalRAMMB)
}

func TestPlanDistribution_NoHosts(t *testing.T) {
	cfg := PlanDistribution(nil, "/model.gguf", 0, 0)
	assert.Nil(t, cfg)
}

func TestPlanDistribution_SingleHost(t *testing.T) {
	hosts := []*HostHardware{
		{
			Host:         "gpu.local",
			HasGPU:       true,
			GPUFreeMemMB: 24000,
			LlamaCppDir:  "~/llama.cpp",
		},
	}
	cfg := PlanDistribution(
		hosts, "/models/test.gguf", 8090, 50052,
	)
	require.NotNil(t, cfg)
	assert.Equal(t, "gpu.local", cfg.MasterHost)
	assert.Equal(t, "~/llama.cpp", cfg.MasterDir)
	assert.Equal(t, "/models/test.gguf", cfg.ModelPath)
	assert.Equal(t, 8090, cfg.ServerPort)
	assert.Equal(t, 8192, cfg.ContextSize)
	require.Len(t, cfg.RPCWorkers, 1)
	assert.Equal(t, "gpu.local:50052", cfg.RPCWorkers[0])
}

func TestPlanDistribution_MultiHost_MasterSelection(
	t *testing.T,
) {
	hosts := []*HostHardware{
		{
			Host:         "cpu.local",
			HasGPU:       false,
			GPUFreeMemMB: 0,
			RAMFreeMB:    32000,
			LlamaCppDir:  "~/llama.cpp",
		},
		{
			Host:         "gpu.local",
			HasGPU:       true,
			GPUFreeMemMB: 24000,
			RAMFreeMB:    48000,
			LlamaCppDir:  "~/llama.cpp",
		},
		{
			Host:         "gpu2.local",
			HasGPU:       true,
			GPUFreeMemMB: 12000,
			RAMFreeMB:    16000,
			LlamaCppDir:  "/opt/llama.cpp",
		},
	}
	cfg := PlanDistribution(
		hosts, "/models/big.gguf", 8090, 50052,
	)
	require.NotNil(t, cfg)
	// gpu.local has most GPU VRAM, should be master.
	assert.Equal(t, "gpu.local", cfg.MasterHost)
	assert.Equal(t, "~/llama.cpp", cfg.MasterDir)
	require.Len(t, cfg.RPCWorkers, 3)
	assert.Equal(t, "cpu.local:50052", cfg.RPCWorkers[0])
	assert.Equal(t, "gpu.local:50053", cfg.RPCWorkers[1])
	assert.Equal(t, "gpu2.local:50054", cfg.RPCWorkers[2])
}

func TestPlanDistribution_DefaultPorts(t *testing.T) {
	hosts := []*HostHardware{
		{Host: "h1.local"},
	}
	cfg := PlanDistribution(hosts, "/m.gguf", 0, 0)
	require.NotNil(t, cfg)
	assert.Equal(t, 8090, cfg.ServerPort)
	assert.Equal(t, "h1.local:50052", cfg.RPCWorkers[0])
}

func TestPlanDistribution_DefaultDir(t *testing.T) {
	hosts := []*HostHardware{
		{Host: "h1.local", LlamaCppDir: ""},
	}
	cfg := PlanDistribution(hosts, "/m.gguf", 8090, 50052)
	require.NotNil(t, cfg)
	assert.Equal(t, "~/llama.cpp", cfg.MasterDir)
}

func TestHostHardware_ZeroValues(t *testing.T) {
	hw := &HostHardware{}
	assert.Empty(t, hw.Host)
	assert.False(t, hw.HasGPU)
	assert.Zero(t, hw.GPUMemMB)
	assert.Zero(t, hw.CPUCores)
	assert.Zero(t, hw.RAMTotalMB)
}

func TestModelRecommendation_Fields(t *testing.T) {
	rec := ModelRecommendation{
		ModelName:         "qwen2.5-vl:72b",
		ModelSize:         "72B",
		NeedsDistribution: true,
		TotalRAMMB:        96000,
		TotalGPUMemMB:     48000,
		GPUHosts:          []string{"a", "b"},
		AllHosts:          []string{"a", "b", "c"},
	}
	assert.Equal(t, "qwen2.5-vl:72b", rec.ModelName)
	assert.True(t, rec.NeedsDistribution)
	assert.Len(t, rec.GPUHosts, 2)
	assert.Len(t, rec.AllHosts, 3)
}
