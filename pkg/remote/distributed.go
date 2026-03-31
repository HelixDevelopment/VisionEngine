// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"
)

// HardwareInfo represents a remote host's hardware capabilities.
type HardwareInfo struct {
	Host        string
	GPUMemMB    int
	RAMMB       int
	ModelName   string
	ModelSize   string
	SupportsRPC bool
	LlamaCppDir string
}

// ModelRecommendation is the result of SelectStrongestModel.
type ModelRecommendation struct {
	ModelName         string
	ModelSize         string
	AllHosts          []string
	GPUHosts          []string
	TotalGPUMemMB     int
	TotalRAMMB        int
	NeedsDistribution bool
}

// DistributionConfig holds configuration for distributed RPC.
type DistributionConfig struct {
	MasterHost  string
	MasterDir   string
	ModelPath   string
	ServerPort  int
	ContextSize int
	RPCWorkers  []string
}

// ProbeHosts probes remote hosts for hardware capabilities.
func ProbeHosts(ctx context.Context, hosts []string, sshUser string) []HardwareInfo {
	// Stub implementation: return empty list.
	return []HardwareInfo{}
}

// SelectStrongestModel selects the best model across hosts.
func SelectStrongestModel(hwList []HardwareInfo) *ModelRecommendation {
	// Stub implementation: return a default recommendation.
	return &ModelRecommendation{
		ModelName:         "",
		ModelSize:         "",
		AllHosts:          []string{},
		GPUHosts:          []string{},
		TotalGPUMemMB:     0,
		TotalRAMMB:        0,
		NeedsDistribution: false,
	}
}

// PlanDistribution creates a distribution configuration.
func PlanDistribution(hwList []HardwareInfo, modelPath string, serverPort, rpcBasePort int) *DistributionConfig {
	// Stub implementation: return empty configuration.
	return &DistributionConfig{
		MasterHost:  "",
		MasterDir:   "",
		ModelPath:   modelPath,
		ServerPort:  serverPort,
		ContextSize: 4096,
		RPCWorkers:  []string{},
	}
}

// StartRPCServer starts an RPC server on the remote host.
func (d *LlamaCppDeployer) StartRPCServer(ctx context.Context, port int) error {
	// Stub: do nothing.
	return nil
}

// StartWithRPC starts a llama-server with RPC support.
func (d *LlamaCppDeployer) StartWithRPC(ctx context.Context, modelPath string, rpcWorkers []string, serverPort int) error {
	// Stub: do nothing.
	return nil
}

// StopInstance stops the llama-server instance.
func (d *LlamaCppDeployer) StopInstance(ctx context.Context, port int) error {
	// Stub: do nothing.
	return nil
}

// StopRPCServer stops the RPC server.
func (d *LlamaCppDeployer) StopRPCServer(ctx context.Context, port int) error {
	// Stub: do nothing.
	return nil
}
