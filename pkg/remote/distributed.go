// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"
	"fmt"
	"log"
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
//
// Round-48 forensic note (2026-05-18): this function currently
// returns an empty slice unconditionally — original body comment
// "Stub implementation: return empty list" was a §11.4 CONST-035
// forbidden tell in production code. Sentinel-isation is DEFERRED
// to round 49 because, unlike the four LlamaCppDeployer lifecycle
// methods (StartRPCServer / StartWithRPC / StopInstance /
// StopRPCServer) — which had `return nil` and could mislead an
// error-checking caller into "success" — this function returns a
// value-typed slice with no error channel, so a sentinel return
// would force a breaking signature change. Round 49 will redesign
// the signature to `(\[\]HardwareInfo, error)` AND wire real SSH
// hardware probing in the same change. Until then, callers
// observing an empty slice from a non-empty host list MUST treat
// it as an unimplemented signal, not as "no GPU detected".
//
// Constitutional anchors: CONST-035 (anti-bluff), CONST-050(A)
// (no-fakes-beyond-unit-tests), Article XI §11.9.
func ProbeHosts(ctx context.Context, hosts []string, sshUser string) []HardwareInfo {
	if len(hosts) > 0 {
		log.Printf("WARN visionengine/remote.ProbeHosts: called with %d host(s) but real hardware probing is not yet wired (round-49 candidate); returning empty []HardwareInfo. Caller MUST NOT interpret empty result as 'no GPU detected'.", len(hosts))
	}
	return []HardwareInfo{}
}

// SelectStrongestModel selects the best model across hosts.
//
// Round-48 forensic note (2026-05-18): unlike the four
// LlamaCppDeployer lifecycle stubs sentinel-ised this round, this
// function returns a value-typed pointer with no error channel.
// Sentinel-isation requires a signature change deferred to round
// 49 alongside real model-selection logic. Original body comment
// "Stub implementation: return a default recommendation" was a
// §11.4 CONST-035 forbidden tell preserved here so the historical
// bluff context is grep-able from this file.
//
// Constitutional anchors: CONST-035, CONST-050(A), Article XI §11.9.
func SelectStrongestModel(hwList []HardwareInfo) *ModelRecommendation {
	if len(hwList) > 0 {
		log.Printf("WARN visionengine/remote.SelectStrongestModel: called with %d HardwareInfo entry/entries but real model-selection logic is not yet wired (round-49 candidate); returning zero-valued ModelRecommendation. Caller MUST NOT interpret empty fields as 'no suitable model'.", len(hwList))
	}
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
//
// Round-48 forensic note (2026-05-18): identical reasoning to
// ProbeHosts / SelectStrongestModel — value-typed return + no error
// channel = signature change required for sentinel; deferred to
// round 49. Original body comment "Stub implementation: return
// empty configuration" was the §11.4 CONST-035 forbidden tell.
//
// Constitutional anchors: CONST-035, CONST-050(A), Article XI §11.9.
func PlanDistribution(hwList []HardwareInfo, modelPath string, serverPort, rpcBasePort int) *DistributionConfig {
	if len(hwList) > 0 {
		log.Printf("WARN visionengine/remote.PlanDistribution: called with %d HardwareInfo entry/entries but real distribution-planning logic is not yet wired (round-49 candidate); returning zero-valued DistributionConfig with caller-supplied modelPath=%q serverPort=%d. Caller MUST NOT interpret empty RPCWorkers as 'single-host plan'.", len(hwList), modelPath, serverPort)
	}
	return &DistributionConfig{
		MasterHost:  "",
		MasterDir:   "",
		ModelPath:   modelPath,
		ServerPort:  serverPort,
		ContextSize: 4096,
		RPCWorkers:  []string{},
	}
}

// ErrRPCServerStartNotImplemented is returned by
// LlamaCppDeployer.StartRPCServer to signal that the RPC server
// lifecycle has not been wired and the method intentionally
// performs no work.
//
// Round-48 §11.4 audit (2026-05-18): the previous body was
// `return nil` under a `// Stub: do nothing.` comment — a textbook
// CONST-035 forbidden tell in production code. Callers that
// checked `if err := d.StartRPCServer(ctx, port); err != nil`
// observed "success" while the remote host had no RPC server
// running, then immediately failed at the next step (StartWithRPC
// connecting to nothing). The original "Stub: do nothing." comment
// is preserved as a grep anchor so future agents can locate the
// historical bluff context from this sentinel string. Round 47
// (commit 5496b2d) added the canary `TestLlamaCppDeployer_
// RPCStubs_NoCrash` documenting the "no error" contract; round 48
// tightens that canary to assert this sentinel.
//
// Round 49 will wire real `llama-server --rpc` invocation over
// SSH using the round-40 SSH client (commit 1169213) and replace
// this sentinel with positive-success behaviour.
//
// Constitutional anchors: CONST-035 (anti-bluff forbidden-tell
// removal), CONST-050(A) (no-fakes-beyond-unit-tests — production
// code MUST NOT silently succeed for unimplemented work), Article
// XI §11.9 forensic anchor.
var ErrRPCServerStartNotImplemented = fmt.Errorf("visionengine/remote: LlamaCppDeployer.StartRPCServer has not been wired — function previously returned nil silently while doing nothing (§11.4 CONST-035 forbidden tell: original 'Stub: do nothing' production-code comment removed round-48 2026-05-18). Real RPC server lifecycle requires SSH-driven 'llama-server --rpc' invocation; wire via round-49 follow-up using round-40 SSH client. Until then, callers MUST treat this method as unimplemented")

// ErrRPCServerStartWithRPCNotImplemented is returned by
// LlamaCppDeployer.StartWithRPC to signal that the RPC-enabled
// llama-server start path has not been wired.
//
// Round-48 §11.4 audit (2026-05-18): see
// ErrRPCServerStartNotImplemented for the full audit narrative;
// same `// Stub: do nothing.` forbidden-tell removal applied here.
// Round 49 will wire real `llama-server --rpc <worker-list>`
// invocation using the round-40 SSH client (commit 1169213).
//
// Constitutional anchors: CONST-035, CONST-050(A), Article XI §11.9.
var ErrRPCServerStartWithRPCNotImplemented = fmt.Errorf("visionengine/remote: LlamaCppDeployer.StartWithRPC has not been wired — function previously returned nil silently while doing nothing (§11.4 CONST-035 forbidden tell: original 'Stub: do nothing' production-code comment removed round-48 2026-05-18). Real RPC-enabled llama-server start requires SSH-driven invocation with --rpc worker-list flag; wire via round-49 follow-up using round-40 SSH client. Until then, callers MUST treat this method as unimplemented")

// ErrRPCServerStopInstanceNotImplemented is returned by
// LlamaCppDeployer.StopInstance to signal that the
// llama-server-stop lifecycle has not been wired.
//
// Round-48 §11.4 audit (2026-05-18): see
// ErrRPCServerStartNotImplemented for the full audit narrative;
// same `// Stub: do nothing.` forbidden-tell removal applied here.
// This sentinel composes with the round-27 sibling
// ErrShutdownRemoteCleanupNotImplemented (which surfaces the same
// orphan-process gap from the VisionPool.Shutdown direction) —
// both expose the unfinished "kill remote llama-server" lifecycle.
// Round 49 will wire real SSH-driven process termination
// (pgrep + kill) using the round-40 SSH client (commit 1169213).
//
// Constitutional anchors: CONST-035, CONST-050(A), Article XI §11.9.
var ErrRPCServerStopInstanceNotImplemented = fmt.Errorf("visionengine/remote: LlamaCppDeployer.StopInstance has not been wired — function previously returned nil silently while doing nothing (§11.4 CONST-035 forbidden tell: original 'Stub: do nothing' production-code comment removed round-48 2026-05-18). Composes with round-27 ErrShutdownRemoteCleanupNotImplemented (sibling orphan-process gap). Real llama-server stop requires SSH-driven pgrep + kill; wire via round-49 follow-up using round-40 SSH client. Until then, callers MUST treat this method as unimplemented")

// ErrRPCServerStopNotImplemented is returned by
// LlamaCppDeployer.StopRPCServer to signal that the RPC-server-
// stop lifecycle has not been wired.
//
// Round-48 §11.4 audit (2026-05-18): see
// ErrRPCServerStartNotImplemented for the full audit narrative;
// same `// Stub: do nothing.` forbidden-tell removal applied here.
// Round 49 will wire real SSH-driven RPC-server shutdown using
// the round-40 SSH client (commit 1169213).
//
// Constitutional anchors: CONST-035, CONST-050(A), Article XI §11.9.
var ErrRPCServerStopNotImplemented = fmt.Errorf("visionengine/remote: LlamaCppDeployer.StopRPCServer has not been wired — function previously returned nil silently while doing nothing (§11.4 CONST-035 forbidden tell: original 'Stub: do nothing' production-code comment removed round-48 2026-05-18). Real RPC server stop requires SSH-driven termination of the llama-server --rpc process; wire via round-49 follow-up using round-40 SSH client. Until then, callers MUST treat this method as unimplemented")

// StartRPCServer starts an RPC server on the remote host.
//
// Round-48 §11.4 audit (2026-05-18): the previous body was
// `// Stub: do nothing.` + `return nil` — a CONST-035 forbidden
// tell in production code. Now returns
// ErrRPCServerStartNotImplemented so callers route remediation
// correctly. See the sentinel's GoDoc for the full audit trail.
func (d *LlamaCppDeployer) StartRPCServer(ctx context.Context, port int) error {
	log.Printf("WARN visionengine/remote.LlamaCppDeployer.StartRPCServer: not wired — returning ErrRPCServerStartNotImplemented for host=%q port=%d (round-48 sentinel; round-49 will wire real SSH-driven 'llama-server --rpc' invocation).", d.config.Host, port)
	return ErrRPCServerStartNotImplemented
}

// StartWithRPC starts a llama-server with RPC support.
//
// Round-48 §11.4 audit (2026-05-18): the previous body was
// `// Stub: do nothing.` + `return nil` — a CONST-035 forbidden
// tell in production code. Now returns
// ErrRPCServerStartWithRPCNotImplemented so callers route
// remediation correctly. See the sentinel's GoDoc for the full
// audit trail.
func (d *LlamaCppDeployer) StartWithRPC(ctx context.Context, modelPath string, rpcWorkers []string, serverPort int) error {
	log.Printf("WARN visionengine/remote.LlamaCppDeployer.StartWithRPC: not wired — returning ErrRPCServerStartWithRPCNotImplemented for host=%q modelPath=%q workers=%d serverPort=%d (round-48 sentinel; round-49 will wire real SSH-driven 'llama-server --rpc' invocation).", d.config.Host, modelPath, len(rpcWorkers), serverPort)
	return ErrRPCServerStartWithRPCNotImplemented
}

// StopInstance stops the llama-server instance.
//
// Round-48 §11.4 audit (2026-05-18): the previous body was
// `// Stub: do nothing.` + `return nil` — a CONST-035 forbidden
// tell in production code. Now returns
// ErrRPCServerStopInstanceNotImplemented so callers route
// remediation correctly. See the sentinel's GoDoc for the full
// audit trail.
func (d *LlamaCppDeployer) StopInstance(ctx context.Context, port int) error {
	log.Printf("WARN visionengine/remote.LlamaCppDeployer.StopInstance: not wired — returning ErrRPCServerStopInstanceNotImplemented for host=%q port=%d (round-48 sentinel; round-49 will wire real SSH-driven pgrep+kill of llama-server).", d.config.Host, port)
	return ErrRPCServerStopInstanceNotImplemented
}

// StopRPCServer stops the RPC server.
//
// Round-48 §11.4 audit (2026-05-18): the previous body was
// `// Stub: do nothing.` + `return nil` — a CONST-035 forbidden
// tell in production code. Now returns
// ErrRPCServerStopNotImplemented so callers route remediation
// correctly. See the sentinel's GoDoc for the full audit trail.
func (d *LlamaCppDeployer) StopRPCServer(ctx context.Context, port int) error {
	log.Printf("WARN visionengine/remote.LlamaCppDeployer.StopRPCServer: not wired — returning ErrRPCServerStopNotImplemented for host=%q port=%d (round-48 sentinel; round-49 will wire real SSH-driven RPC-server shutdown).", d.config.Host, port)
	return ErrRPCServerStopNotImplemented
}
