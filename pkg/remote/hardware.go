// Copyright 2026 Milos Vasic. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// HostHardware describes the hardware capabilities of a
// single host, discovered via SSH probing.
type HostHardware struct {
	// Host is the hostname or IP address.
	Host string
	// HasGPU indicates whether an NVIDIA GPU was detected.
	HasGPU bool
	// GPUName is the human-readable GPU name (e.g.
	// "NVIDIA GeForce RTX 4090").
	GPUName string
	// GPUMemMB is the total GPU VRAM in megabytes.
	GPUMemMB int
	// GPUFreeMemMB is the available GPU VRAM in megabytes.
	GPUFreeMemMB int
	// CPUCores is the number of logical CPU cores.
	CPUCores int
	// RAMTotalMB is the total system RAM in megabytes.
	RAMTotalMB int
	// RAMFreeMB is the available system RAM in megabytes.
	RAMFreeMB int
	// HasLlamaCpp indicates whether llama.cpp is built
	// (llama-server binary exists).
	HasLlamaCpp bool
	// HasRPCServer indicates whether the llama.cpp
	// rpc-server binary exists.
	HasRPCServer bool
	// LlamaCppDir is the llama.cpp install directory.
	LlamaCppDir string
}

// sshProbe runs a command on a remote host via SSH and
// returns stdout. Uses BatchMode to avoid password prompts.
func sshProbe(
	ctx context.Context,
	host, user, command string,
) (string, error) {
	args := []string{
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes",
	}
	target := host
	if user != "" {
		target = user + "@" + host
	}
	args = append(args, target, command)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ProbeHost connects to the given host via SSH and detects
// its hardware capabilities: GPU (via nvidia-smi), CPU cores
// (via nproc), and RAM (via free). It also checks for
// llama.cpp binaries.
func ProbeHost(
	ctx context.Context,
	host, user string,
) (*HostHardware, error) {
	probeCtx, cancel := context.WithTimeout(
		ctx, 15*time.Second,
	)
	defer cancel()

	hw := &HostHardware{
		Host: host,
	}

	// Probe GPU via nvidia-smi.
	gpuOut, gpuErr := sshProbe(probeCtx, host, user,
		"nvidia-smi --query-gpu=name,memory.total,memory.free "+
			"--format=csv,noheader,nounits 2>/dev/null",
	)
	if gpuErr == nil && strings.TrimSpace(gpuOut) != "" {
		// Format: "GPU Name, TotalMB, FreeMB"
		// May have multiple lines for multi-GPU; take first.
		for _, line := range strings.Split(
			strings.TrimSpace(gpuOut), "\n",
		) {
			parts := strings.SplitN(
				strings.TrimSpace(line), ", ", 3,
			)
			if len(parts) >= 3 {
				hw.HasGPU = true
				hw.GPUName = strings.TrimSpace(parts[0])
				hw.GPUMemMB, _ = strconv.Atoi(
					strings.TrimSpace(parts[1]),
				)
				hw.GPUFreeMemMB, _ = strconv.Atoi(
					strings.TrimSpace(parts[2]),
				)
				break // Use first GPU.
			}
		}
	}

	// Probe CPU cores via nproc.
	cpuOut, cpuErr := sshProbe(probeCtx, host, user,
		"nproc 2>/dev/null || echo 0",
	)
	if cpuErr == nil {
		hw.CPUCores, _ = strconv.Atoi(
			strings.TrimSpace(cpuOut),
		)
	}

	// Probe RAM via free -m.
	ramOut, ramErr := sshProbe(probeCtx, host, user,
		"free -m 2>/dev/null | awk '/^Mem:/ {print $2, $7}'",
	)
	if ramErr == nil && strings.TrimSpace(ramOut) != "" {
		fields := strings.Fields(
			strings.TrimSpace(ramOut),
		)
		if len(fields) >= 2 {
			hw.RAMTotalMB, _ = strconv.Atoi(fields[0])
			hw.RAMFreeMB, _ = strconv.Atoi(fields[1])
		} else if len(fields) == 1 {
			hw.RAMTotalMB, _ = strconv.Atoi(fields[0])
		}
	}

	// Check for llama.cpp binaries.
	for _, dir := range []string{
		"~/llama.cpp", "/opt/llama.cpp",
	} {
		binOut, binErr := sshProbe(probeCtx, host, user,
			fmt.Sprintf(
				"test -x %s/build/bin/llama-server && "+
					"echo yes || echo no",
				dir,
			),
		)
		if binErr == nil &&
			strings.TrimSpace(binOut) == "yes" {
			hw.HasLlamaCpp = true
			hw.LlamaCppDir = dir
			// Also check for rpc-server.
			rpcOut, _ := sshProbe(probeCtx, host, user,
				fmt.Sprintf(
					"test -x %s/build/bin/rpc-server && "+
						"echo yes || echo no",
					dir,
				),
			)
			if strings.TrimSpace(rpcOut) == "yes" {
				hw.HasRPCServer = true
			}
			break
		}
	}

	return hw, nil
}

// ProbeHosts probes multiple hosts in sequence and returns
// only the hosts that were reachable. Unreachable hosts are
// logged and skipped.
func ProbeHosts(
	ctx context.Context,
	hosts []string,
	user string,
) []*HostHardware {
	var result []*HostHardware
	for _, host := range hosts {
		h := strings.TrimSpace(host)
		if h == "" {
			continue
		}
		info, err := ProbeHost(ctx, h, user)
		if err != nil {
			fmt.Printf(
				"[hardware] %s unreachable: %v\n",
				h, err,
			)
			continue
		}
		result = append(result, info)
		fmt.Printf(
			"[hardware] %s: GPU=%s (%dMB total, "+
				"%dMB free), CPU=%d cores, "+
				"RAM=%dMB/%dMB, llama.cpp=%v, "+
				"rpc=%v\n",
			h,
			info.GPUName, info.GPUMemMB,
			info.GPUFreeMemMB,
			info.CPUCores,
			info.RAMFreeMB, info.RAMTotalMB,
			info.HasLlamaCpp, info.HasRPCServer,
		)
	}
	return result
}

// ModelRecommendation holds the result of automatic model
// selection based on combined hardware resources.
type ModelRecommendation struct {
	// ModelName is the recommended vision model identifier.
	ModelName string
	// ModelSize is a human-readable description (e.g.
	// "72B", "32B", "8B").
	ModelSize string
	// NeedsDistribution indicates the model requires
	// distributed inference across multiple hosts (RPC).
	NeedsDistribution bool
	// TotalRAMMB is the combined available RAM across hosts.
	TotalRAMMB int
	// TotalGPUMemMB is the combined available GPU VRAM.
	TotalGPUMemMB int
	// GPUHosts lists hosts that have a GPU.
	GPUHosts []string
	// AllHosts lists all reachable hosts.
	AllHosts []string
}

// SelectStrongestModel picks the best vision model for the
// combined hardware of all hosts. It considers total RAM,
// GPU VRAM, and the number of hosts to determine whether
// distributed inference (RPC) is needed.
//
// Model selection tiers:
//
//	>48GB GPU VRAM total: qwen2.5-vl:72b (72B, best)
//	>24GB GPU VRAM total: qwen2.5-vl:32b (32B)
//	>12GB GPU VRAM total: minicpm-v:8b (8B, strong vision)
//	>6GB GPU VRAM total:  llava:7b (7B)
//	CPU-only >32GB RAM:   llava:13b (13B)
//	CPU-only >16GB RAM:   llava:7b (7B)
//	CPU-only <16GB RAM:   llava:7b (7B, degraded)
func SelectStrongestModel(
	hosts []*HostHardware,
) ModelRecommendation {
	rec := ModelRecommendation{}

	if len(hosts) == 0 {
		rec.ModelName = "llava:7b"
		rec.ModelSize = "7B"
		return rec
	}

	for _, h := range hosts {
		rec.AllHosts = append(rec.AllHosts, h.Host)
		rec.TotalRAMMB += h.RAMFreeMB
		if h.HasGPU {
			rec.TotalGPUMemMB += h.GPUFreeMemMB
			rec.GPUHosts = append(rec.GPUHosts, h.Host)
		}
	}

	hasGPU := len(rec.GPUHosts) > 0

	// GPU-based selection: use combined VRAM across all
	// GPU hosts for distributed inference sizing.
	if hasGPU {
		switch {
		case rec.TotalGPUMemMB >= 48000:
			rec.ModelName = "qwen2.5-vl:72b"
			rec.ModelSize = "72B"
			rec.NeedsDistribution = len(hosts) > 1
		case rec.TotalGPUMemMB >= 24000:
			rec.ModelName = "qwen2.5-vl:32b"
			rec.ModelSize = "32B"
			rec.NeedsDistribution = len(hosts) > 1
		case rec.TotalGPUMemMB >= 12000:
			rec.ModelName = "minicpm-v:8b"
			rec.ModelSize = "8B"
		case rec.TotalGPUMemMB >= 6000:
			rec.ModelName = "llava:7b"
			rec.ModelSize = "7B"
		default:
			rec.ModelName = "llava:7b"
			rec.ModelSize = "7B"
		}
		return rec
	}

	// CPU-only selection: based on combined available RAM.
	switch {
	case rec.TotalRAMMB >= 32000:
		rec.ModelName = "llava:13b"
		rec.ModelSize = "13B"
		rec.NeedsDistribution = len(hosts) > 1
	case rec.TotalRAMMB >= 16000:
		rec.ModelName = "llava:7b"
		rec.ModelSize = "7B"
	default:
		rec.ModelName = "llava:7b"
		rec.ModelSize = "7B"
	}

	return rec
}

// DistributedConfig holds the configuration needed to start
// distributed llama.cpp inference across multiple hosts.
type DistributedConfig struct {
	// MasterHost is the host that runs llama-server with
	// the --rpc flag.
	MasterHost string
	// MasterUser is the SSH user for the master host.
	MasterUser string
	// MasterDir is the llama.cpp directory on the master.
	MasterDir string
	// RPCWorkers lists "host:port" addresses for rpc-server
	// instances on each participating host.
	RPCWorkers []string
	// ModelPath is the path to the GGUF model on the master.
	ModelPath string
	// ServerPort is the port for the master llama-server.
	ServerPort int
	// ContextSize is the context window size.
	ContextSize int
}

// PlanDistribution creates a DistributedConfig by selecting
// the strongest GPU host as master and setting up RPC workers
// on all hosts. The master host runs llama-server with --rpc
// pointing to all workers.
func PlanDistribution(
	hosts []*HostHardware,
	modelPath string,
	serverPort int,
	rpcBasePort int,
) *DistributedConfig {
	if len(hosts) == 0 {
		return nil
	}
	if serverPort == 0 {
		serverPort = 8090
	}
	if rpcBasePort == 0 {
		rpcBasePort = 50052
	}

	// Select master: prefer the host with the most GPU VRAM.
	// If no GPU host, pick the one with the most RAM.
	master := hosts[0]
	for _, h := range hosts[1:] {
		if h.GPUFreeMemMB > master.GPUFreeMemMB {
			master = h
		} else if h.GPUFreeMemMB == master.GPUFreeMemMB &&
			h.RAMFreeMB > master.RAMFreeMB {
			master = h
		}
	}

	cfg := &DistributedConfig{
		MasterHost:  master.Host,
		MasterDir:   master.LlamaCppDir,
		ModelPath:   modelPath,
		ServerPort:  serverPort,
		ContextSize: 8192,
	}
	if cfg.MasterDir == "" {
		cfg.MasterDir = "~/llama.cpp"
	}

	// All hosts run rpc-server (including the master, which
	// also serves its own local compute via RPC).
	for i, h := range hosts {
		port := rpcBasePort + i
		worker := fmt.Sprintf("%s:%d", h.Host, port)
		cfg.RPCWorkers = append(cfg.RPCWorkers, worker)
	}

	return cfg
}
