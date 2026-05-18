// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
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
// LlamaCppDeployer.StartRPCServer ONLY when SSHConfig is unset
// (round-48 sentinel preserved as the explicit "SSH is unconfigured"
// signal post round-52). For the SSH-configured path the method
// now actually launches `llama-server --rpc` over SSH (round-52
// wiring) and returns wrapped concrete errors on failure — never
// this sentinel.
//
// Round-48 §11.4 audit (2026-05-18): the previous body was
// `return nil` under a `// Stub: do nothing.` comment — a textbook
// CONST-035 forbidden tell in production code. Callers that
// checked `if err := d.StartRPCServer(ctx, port); err != nil`
// observed "success" while the remote host had no RPC server
// running, then immediately failed at the next step (StartWithRPC
// connecting to nothing). The original "Stub: do nothing." comment
// is preserved as a grep anchor so future agents can locate the
// historical bluff context from this sentinel string.
//
// Round-52 §11.4 audit (2026-05-18): real SSH-driven invocation is
// now wired using the round-40 SSH client helper (sshConn +
// runRemote, commit 1169213). The sentinel is RETAINED for the
// no-SSH-config path (semantically: "you didn't tell me how to
// reach the remote host, so I can't act"). When SSHConfig is
// populated, the method returns concrete wrapped errors for SSH
// dial failures, llama-server launch failures, PID parse failures,
// and readiness-probe failures — never this sentinel.
//
// Constitutional anchors: CONST-035 (anti-bluff forbidden-tell
// removal), CONST-042 (no-secret-leak — SSH credentials come from
// SSHConfig populated by env vars/config files; never hardcoded),
// CONST-050(A) (no-fakes-beyond-unit-tests — production code MUST
// NOT silently succeed for unimplemented work), Article XI §11.9
// forensic anchor.
var ErrRPCServerStartNotImplemented = fmt.Errorf("visionengine/remote: LlamaCppDeployer.StartRPCServer cannot act — SSHConfig is unset; call WithSSHConfig to enable real SSH-driven 'llama-server --rpc' invocation (round-52 wired the SSH-configured path using round-40 sshConn helper; this sentinel is preserved for the unconfigured-SSH signal — §11.4 CONST-035 forbidden tell: original 'Stub: do nothing' production-code comment removed round-48 2026-05-18)")

// ErrRPCServerStartWithRPCNotImplemented is returned by
// LlamaCppDeployer.StartWithRPC ONLY when SSHConfig is unset
// (round-48 sentinel preserved as the explicit "SSH is unconfigured"
// signal post round-52). For the SSH-configured path the method
// now orchestrates a real `llama-server --rpc <worker-list>`
// invocation over SSH (round-52 wiring) and returns wrapped
// concrete errors on failure — never this sentinel.
//
// Round-48 §11.4 audit (2026-05-18): see
// ErrRPCServerStartNotImplemented for the full audit narrative;
// same `// Stub: do nothing.` forbidden-tell removal applied here.
// Round-52 wires real SSH-driven invocation using the round-40
// sshConn helper (commit 1169213).
//
// Constitutional anchors: CONST-035, CONST-042, CONST-050(A),
// Article XI §11.9.
var ErrRPCServerStartWithRPCNotImplemented = fmt.Errorf("visionengine/remote: LlamaCppDeployer.StartWithRPC cannot act — SSHConfig is unset; call WithSSHConfig to enable real SSH-driven 'llama-server --rpc <worker-list>' orchestration (round-52 wired the SSH-configured path; this sentinel is preserved for the unconfigured-SSH signal — §11.4 CONST-035 forbidden tell: original 'Stub: do nothing' production-code comment removed round-48 2026-05-18)")

// ErrRPCServerStopInstanceNotImplemented is returned by
// LlamaCppDeployer.StopInstance ONLY when SSHConfig is unset
// (round-48 sentinel preserved as the explicit "SSH is unconfigured"
// signal post round-52). For the SSH-configured path the method
// now actually kills the remote llama-server process by PID over
// SSH (SIGTERM + SIGKILL fallback after 10 s) and returns wrapped
// concrete errors on failure — never this sentinel.
//
// Round-48 §11.4 audit (2026-05-18): see
// ErrRPCServerStartNotImplemented for the full audit narrative;
// same `// Stub: do nothing.` forbidden-tell removal applied here.
// This sentinel composes with the round-27 sibling
// ErrShutdownRemoteCleanupNotImplemented (which surfaces the same
// orphan-process gap from the VisionPool.Shutdown direction) —
// both expose the unfinished "kill remote llama-server" lifecycle
// for unconfigured-SSH callers. Round-52 wires real SSH-driven
// termination using the round-40 sshConn helper (commit 1169213).
//
// Constitutional anchors: CONST-035, CONST-042, CONST-050(A),
// Article XI §11.9.
var ErrRPCServerStopInstanceNotImplemented = fmt.Errorf("visionengine/remote: LlamaCppDeployer.StopInstance cannot act — SSHConfig is unset; call WithSSHConfig to enable real SSH-driven SIGTERM+SIGKILL of the llama-server PID (round-52 wired the SSH-configured path; composes with round-27 ErrShutdownRemoteCleanupNotImplemented for the orphan-process gap; this sentinel is preserved for the unconfigured-SSH signal — §11.4 CONST-035 forbidden tell: original 'Stub: do nothing' production-code comment removed round-48 2026-05-18)")

// ErrRPCServerStopNotImplemented is returned by
// LlamaCppDeployer.StopRPCServer ONLY when SSHConfig is unset
// (round-48 sentinel preserved as the explicit "SSH is unconfigured"
// signal post round-52). For the SSH-configured path the method
// now actually terminates the remote RPC-server llama-server
// process by port → PID lookup + SIGTERM + SIGKILL fallback over
// SSH (round-52 wiring) and returns wrapped concrete errors on
// failure — never this sentinel.
//
// Round-48 §11.4 audit (2026-05-18): see
// ErrRPCServerStartNotImplemented for the full audit narrative;
// same `// Stub: do nothing.` forbidden-tell removal applied here.
// Round-52 wires real SSH-driven RPC-server shutdown using the
// round-40 sshConn helper (commit 1169213).
//
// Constitutional anchors: CONST-035, CONST-042, CONST-050(A),
// Article XI §11.9.
var ErrRPCServerStopNotImplemented = fmt.Errorf("visionengine/remote: LlamaCppDeployer.StopRPCServer cannot act — SSHConfig is unset; call WithSSHConfig to enable real SSH-driven RPC-server termination (round-52 wired the SSH-configured path; this sentinel is preserved for the unconfigured-SSH signal — §11.4 CONST-035 forbidden tell: original 'Stub: do nothing' production-code comment removed round-48 2026-05-18)")

// ErrRPCInstanceNotFound is returned by LlamaCppDeployer.StopInstance
// when the supplied port does not correspond to any tracked instance
// in the deployer's instances map. CONST-035: silent no-op for an
// unknown instance is a PASS-bluff — caller believes the instance
// was stopped when in fact nothing happened. The honest signal is
// "I don't know what you're asking me to stop". Composes with the
// round-52 instance-tracking design.
var ErrRPCInstanceNotFound = errors.New("visionengine/remote: no RPC instance tracked at the requested port — StopInstance refuses to silently no-op (CONST-035: a no-op for unknown work would be a PASS-bluff)")

// ErrRPCLaunchFailed is returned by LlamaCppDeployer.StartRPCServer
// when the remote llama-server process could not be launched (binary
// missing, exec error, PID parse failure with no pgrep fallback hit,
// etc.). Wraps the underlying SSH session output for diagnostics.
var ErrRPCLaunchFailed = errors.New("visionengine/remote: remote llama-server launch failed — verify llama.cpp binary exists at RepoDir on the remote host and SSH user has execute permission")

// ErrRPCReadinessProbeFailed is returned by
// LlamaCppDeployer.StartRPCServer when the remote llama-server
// process appears to have launched (PID captured) but does not
// accept TCP connections on the bound port within the readiness
// window (3 retries × 500ms = 1.5s).
var ErrRPCReadinessProbeFailed = errors.New("visionengine/remote: remote llama-server launched (PID captured) but readiness probe (nc -z) failed after 3 retries — process may be crash-looping, port may be firewalled, or bind may have failed silently")

// RPCInstance represents a single tracked llama-server --rpc
// process on the remote host. The deployer keeps a per-port
// instance map so StopInstance / StopRPCServer can address the
// concrete remote PID rather than relying on a port-only lookup
// at termination time (which would race against PID reuse if the
// process crashed and a new one bound the same port).
//
// Constitutional anchor: CONST-035 — tracking by PID prevents the
// "we killed the wrong process" silent-failure mode that a
// fuser-only termination path would produce.
type RPCInstance struct {
	// ID is the deployer-assigned identifier (currently the
	// stringified port; future revisions may switch to UUID).
	ID string
	// PID is the remote process ID captured at launch time.
	PID int
	// Port is the TCP port the llama-server bound to.
	Port int
	// Host is the remote host the process runs on (mirrors
	// SSHConfig.Host at launch time so post-launch SSHConfig
	// rotation does not orphan the tracked process).
	Host string
	// StartedAt is the wall-clock time when the launch command
	// completed (PID captured).
	StartedAt time.Time
}

// instanceMap is the deployer's mutex-protected per-port tracking
// of live RPCInstance records. Round-52 design: separate map +
// mutex keeps lock contention bounded to the lifecycle methods;
// FreeGPU / StartInstance / RestoreOllama do not touch it.
type instanceMap struct {
	mu sync.RWMutex
	m  map[int]*RPCInstance
}

func newInstanceMap() *instanceMap {
	return &instanceMap{m: make(map[int]*RPCInstance)}
}

func (im *instanceMap) put(port int, inst *RPCInstance) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.m[port] = inst
}

func (im *instanceMap) get(port int) (*RPCInstance, bool) {
	im.mu.RLock()
	defer im.mu.RUnlock()
	inst, ok := im.m[port]
	return inst, ok
}

func (im *instanceMap) delete(port int) {
	im.mu.Lock()
	defer im.mu.Unlock()
	delete(im.m, port)
}

// StartRPCServer starts an RPC server on the remote host.
//
// Round-52 §11.4 anti-bluff wiring (2026-05-18): real
// `llama-server --rpc <bind>` invocation over SSH using the
// round-40 sshConn helper (commit 1169213). Replaces the round-48
// sentinel-only stub. Behaviour bifurcates:
//
//  1. SSHConfig is unset (round-48 sentinel path preserved):
//     ErrRPCServerStartNotImplemented is returned. This remains
//     the explicit signal "you did not give me SSH credentials,
//     so I cannot act on the remote host". Callers wanting real
//     behaviour MUST call WithSSHConfig.
//
//  2. SSHConfig is set (round-52 wiring): an SSH session is
//     dialled per-call via sshConn (lazy, host-key-verified per
//     CONST-035). A backgrounded `nohup llama-server --rpc
//     0.0.0.0:<port> --model <path> --ctx-size <N> --n-gpu-layers
//     <N> > /tmp/llama-rpc-<port>.log 2>&1 & echo $!` invocation
//     captures the PID via stdout. If PID parse fails the method
//     falls back to `pgrep -f 'llama-server.*--rpc.*:<port>'` to
//     locate the process. The instance is registered in d.instances
//     keyed by port. Finally a `nc -z <host> <port>` readiness
//     probe runs with 3 retries × 500ms; failure surfaces
//     ErrRPCReadinessProbeFailed (instance still registered so
//     subsequent StopInstance can clean up).
//
// Constitutional anchors: CONST-035, CONST-042 (SSH credentials
// from SSHConfig — never hardcoded), CONST-050(A), Article XI §11.9.
func (d *LlamaCppDeployer) StartRPCServer(ctx context.Context, port int) error {
	if d.sshConfig.Host == "" {
		log.Printf("WARN visionengine/remote.LlamaCppDeployer.StartRPCServer: SSHConfig unset for host=%q port=%d — returning ErrRPCServerStartNotImplemented (round-48 sentinel preserved as the unconfigured-SSH signal; call WithSSHConfig to enable real launch).", d.config.Host, port)
		return ErrRPCServerStartNotImplemented
	}

	client, err := sshConn(ctx, d.sshConfig)
	if err != nil {
		log.Printf("WARN visionengine/remote.LlamaCppDeployer.StartRPCServer: SSH dial to %s@%s failed for port=%d: %v",
			d.sshConfig.User, d.sshConfig.Host, port, err)
		return fmt.Errorf("visionengine/remote: StartRPCServer SSH dial failed: %w", err)
	}
	defer client.Close()

	// Compose the llama-server invocation. RepoDir may be empty —
	// in that case the binary must already be on the SSH user's
	// PATH on the remote host.
	binary := "llama-server"
	if d.config.RepoDir != "" {
		binary = fmt.Sprintf("%s/build/bin/llama-server", d.config.RepoDir)
	}

	parts := []string{
		"nohup", binary,
		"--rpc", fmt.Sprintf("0.0.0.0:%d", port),
	}
	if d.config.ModelPath != "" {
		parts = append(parts, "--model", d.config.ModelPath)
	}
	if d.config.ContextSize > 0 {
		parts = append(parts, "--ctx-size", strconv.Itoa(d.config.ContextSize))
	}
	if d.config.GPULayers != 0 {
		parts = append(parts, "--n-gpu-layers", strconv.Itoa(d.config.GPULayers))
	}
	parts = append(parts,
		">", fmt.Sprintf("/tmp/llama-rpc-%d.log", port),
		"2>&1", "&", "echo", "$!",
	)
	cmdline := strings.Join(parts, " ")

	out, err := runRemote(client, cmdline)
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		// nohup-backgrounded shell normally returns 0 even on
		// failure of the backgrounded process; an err here means
		// the SSH session itself failed.
		return fmt.Errorf("%w: SSH session error for port=%d cmd=%q output=%q underlying=%v",
			ErrRPCLaunchFailed, port, cmdline, outStr, err)
	}

	// Parse PID from "echo $!" output. The output may also contain
	// preceding warnings from nohup (e.g. "nohup: ignoring input and
	// redirecting stderr to stdout"); we take the LAST non-empty
	// numeric token.
	pid := parsePIDFromOutput(outStr)
	if pid <= 0 {
		// Fallback: pgrep -f 'llama-server.*--rpc.*:<port>'.
		pgrepCmd := fmt.Sprintf("pgrep -f 'llama-server.*--rpc.*:%d' | head -n 1", port)
		pgrepOut, pgrepErr := runRemote(client, pgrepCmd)
		pgrepStr := strings.TrimSpace(string(pgrepOut))
		if pgrepErr != nil || pgrepStr == "" {
			return fmt.Errorf("%w: PID parse failed (echo $! output=%q) AND pgrep fallback failed (out=%q err=%v) for port=%d",
				ErrRPCLaunchFailed, outStr, pgrepStr, pgrepErr, port)
		}
		pid = parsePIDFromOutput(pgrepStr)
		if pid <= 0 {
			return fmt.Errorf("%w: PID parse failed for both echo $! (%q) and pgrep fallback (%q) for port=%d",
				ErrRPCLaunchFailed, outStr, pgrepStr, port)
		}
	}

	inst := &RPCInstance{
		ID:        strconv.Itoa(port),
		PID:       pid,
		Port:      port,
		Host:      d.sshConfig.Host,
		StartedAt: time.Now(),
	}
	d.instances.put(port, inst)

	// Readiness probe: nc -z over SSH, 3 retries × 500 ms.
	probeCmd := fmt.Sprintf("nc -z 127.0.0.1 %d && echo READY || echo NOTREADY", port)
	var lastProbeOut string
	probed := false
	for i := 0; i < 3; i++ {
		// Honour ctx cancellation between retries.
		select {
		case <-ctx.Done():
			return fmt.Errorf("visionengine/remote: StartRPCServer readiness probe cancelled for port=%d pid=%d: %w", port, pid, ctx.Err())
		default:
		}
		probeOut, probeErr := runRemote(client, probeCmd)
		lastProbeOut = strings.TrimSpace(string(probeOut))
		if probeErr == nil && strings.Contains(lastProbeOut, "READY") && !strings.Contains(lastProbeOut, "NOTREADY") {
			probed = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !probed {
		return fmt.Errorf("%w: port=%d pid=%d host=%s last_probe_output=%q",
			ErrRPCReadinessProbeFailed, port, pid, d.sshConfig.Host, lastProbeOut)
	}

	log.Printf("INFO visionengine/remote.LlamaCppDeployer.StartRPCServer: launched llama-server --rpc on host=%s port=%d pid=%d (instance registered).",
		d.sshConfig.Host, port, pid)
	return nil
}

// StartWithRPC starts a llama-server with RPC support against the
// supplied worker list.
//
// Round-52 §11.4 anti-bluff wiring (2026-05-18): real
// SSH-driven invocation using the round-40 sshConn helper.
// Bifurcates on SSHConfig set/unset just like StartRPCServer.
//
// SSH-configured behaviour: composes a `llama-server --model <path>
// --rpc <worker1,worker2,...> --port <serverPort>` command,
// backgrounds it via nohup with PID capture, registers an
// RPCInstance keyed by serverPort, runs the same nc -z readiness
// probe. The worker-list flag is the differentiator vs
// StartRPCServer (which launches a single RPC-server endpoint).
//
// Constitutional anchors: CONST-035, CONST-042, CONST-050(A),
// Article XI §11.9.
func (d *LlamaCppDeployer) StartWithRPC(ctx context.Context, modelPath string, rpcWorkers []string, serverPort int) error {
	if d.sshConfig.Host == "" {
		log.Printf("WARN visionengine/remote.LlamaCppDeployer.StartWithRPC: SSHConfig unset for host=%q modelPath=%q workers=%d serverPort=%d — returning ErrRPCServerStartWithRPCNotImplemented (round-48 sentinel preserved as the unconfigured-SSH signal; call WithSSHConfig to enable real launch).",
			d.config.Host, modelPath, len(rpcWorkers), serverPort)
		return ErrRPCServerStartWithRPCNotImplemented
	}
	if modelPath == "" {
		return fmt.Errorf("visionengine/remote: StartWithRPC requires non-empty modelPath")
	}
	if serverPort <= 0 {
		return fmt.Errorf("visionengine/remote: StartWithRPC requires serverPort > 0; got %d", serverPort)
	}

	client, err := sshConn(ctx, d.sshConfig)
	if err != nil {
		log.Printf("WARN visionengine/remote.LlamaCppDeployer.StartWithRPC: SSH dial to %s@%s failed for serverPort=%d: %v",
			d.sshConfig.User, d.sshConfig.Host, serverPort, err)
		return fmt.Errorf("visionengine/remote: StartWithRPC SSH dial failed: %w", err)
	}
	defer client.Close()

	binary := "llama-server"
	if d.config.RepoDir != "" {
		binary = fmt.Sprintf("%s/build/bin/llama-server", d.config.RepoDir)
	}

	parts := []string{
		"nohup", binary,
		"--model", modelPath,
		"--port", strconv.Itoa(serverPort),
	}
	if len(rpcWorkers) > 0 {
		parts = append(parts, "--rpc", strings.Join(rpcWorkers, ","))
	}
	if d.config.ContextSize > 0 {
		parts = append(parts, "--ctx-size", strconv.Itoa(d.config.ContextSize))
	}
	if d.config.GPULayers != 0 {
		parts = append(parts, "--n-gpu-layers", strconv.Itoa(d.config.GPULayers))
	}
	parts = append(parts,
		">", fmt.Sprintf("/tmp/llama-rpc-server-%d.log", serverPort),
		"2>&1", "&", "echo", "$!",
	)
	cmdline := strings.Join(parts, " ")

	out, err := runRemote(client, cmdline)
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("%w: SSH session error for serverPort=%d cmd=%q output=%q underlying=%v",
			ErrRPCLaunchFailed, serverPort, cmdline, outStr, err)
	}

	pid := parsePIDFromOutput(outStr)
	if pid <= 0 {
		pgrepCmd := fmt.Sprintf("pgrep -f 'llama-server.*--port[= ]%d' | head -n 1", serverPort)
		pgrepOut, pgrepErr := runRemote(client, pgrepCmd)
		pgrepStr := strings.TrimSpace(string(pgrepOut))
		if pgrepErr != nil || pgrepStr == "" {
			return fmt.Errorf("%w: PID parse failed (echo $! output=%q) AND pgrep fallback failed (out=%q err=%v) for serverPort=%d",
				ErrRPCLaunchFailed, outStr, pgrepStr, pgrepErr, serverPort)
		}
		pid = parsePIDFromOutput(pgrepStr)
		if pid <= 0 {
			return fmt.Errorf("%w: PID parse failed for both echo $! (%q) and pgrep fallback (%q) for serverPort=%d",
				ErrRPCLaunchFailed, outStr, pgrepStr, serverPort)
		}
	}

	inst := &RPCInstance{
		ID:        strconv.Itoa(serverPort),
		PID:       pid,
		Port:      serverPort,
		Host:      d.sshConfig.Host,
		StartedAt: time.Now(),
	}
	d.instances.put(serverPort, inst)

	probeCmd := fmt.Sprintf("nc -z 127.0.0.1 %d && echo READY || echo NOTREADY", serverPort)
	var lastProbeOut string
	probed := false
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("visionengine/remote: StartWithRPC readiness probe cancelled for serverPort=%d pid=%d: %w", serverPort, pid, ctx.Err())
		default:
		}
		probeOut, probeErr := runRemote(client, probeCmd)
		lastProbeOut = strings.TrimSpace(string(probeOut))
		if probeErr == nil && strings.Contains(lastProbeOut, "READY") && !strings.Contains(lastProbeOut, "NOTREADY") {
			probed = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !probed {
		return fmt.Errorf("%w: serverPort=%d pid=%d host=%s workers=%d last_probe_output=%q",
			ErrRPCReadinessProbeFailed, serverPort, pid, d.sshConfig.Host, len(rpcWorkers), lastProbeOut)
	}

	log.Printf("INFO visionengine/remote.LlamaCppDeployer.StartWithRPC: launched llama-server --rpc on host=%s serverPort=%d pid=%d workers=%d (instance registered).",
		d.sshConfig.Host, serverPort, pid, len(rpcWorkers))
	return nil
}

// StopInstance stops the llama-server instance bound at the given
// port.
//
// Round-52 §11.4 anti-bluff wiring (2026-05-18): real
// SSH-driven SIGTERM + SIGKILL fallback. Bifurcates on SSHConfig:
//
//  1. SSHConfig unset: round-48 sentinel
//     ErrRPCServerStopInstanceNotImplemented returned (preserved
//     unconfigured-SSH signal).
//
//  2. SSHConfig set: lookup instance in d.instances; absence
//     surfaces ErrRPCInstanceNotFound (CONST-035: silent no-op for
//     unknown work would be a PASS-bluff). For known instances:
//     `kill <pid>` (SIGTERM) → poll `kill -0 <pid>` every 1s for
//     up to 10s → if still alive `kill -9 <pid>` (SIGKILL). Remove
//     from instances map on terminal success (either SIGTERM-took
//     or SIGKILL-took path).
//
// Constitutional anchors: CONST-035, CONST-042, CONST-050(A),
// Article XI §11.9.
func (d *LlamaCppDeployer) StopInstance(ctx context.Context, port int) error {
	if d.sshConfig.Host == "" {
		log.Printf("WARN visionengine/remote.LlamaCppDeployer.StopInstance: SSHConfig unset for host=%q port=%d — returning ErrRPCServerStopInstanceNotImplemented (round-48 sentinel preserved as the unconfigured-SSH signal; call WithSSHConfig to enable real termination).", d.config.Host, port)
		return ErrRPCServerStopInstanceNotImplemented
	}

	inst, ok := d.instances.get(port)
	if !ok {
		return fmt.Errorf("%w: port=%d host=%s (StartRPCServer or StartWithRPC was never called for this port, or the instance was already stopped)",
			ErrRPCInstanceNotFound, port, d.sshConfig.Host)
	}

	client, err := sshConn(ctx, d.sshConfig)
	if err != nil {
		log.Printf("WARN visionengine/remote.LlamaCppDeployer.StopInstance: SSH dial to %s@%s failed for port=%d pid=%d: %v",
			d.sshConfig.User, d.sshConfig.Host, port, inst.PID, err)
		return fmt.Errorf("visionengine/remote: StopInstance SSH dial failed: %w", err)
	}
	defer client.Close()

	// SIGTERM.
	termCmd := fmt.Sprintf("kill %d 2>&1 || true", inst.PID)
	if out, killErr := runRemote(client, termCmd); killErr != nil {
		log.Printf("WARN visionengine/remote.LlamaCppDeployer.StopInstance: SIGTERM SSH session error for host=%s pid=%d: %v (out=%q)",
			d.sshConfig.Host, inst.PID, killErr, strings.TrimSpace(string(out)))
		// Continue to polling — kill may have actually delivered
		// the signal despite the session reporting an error.
	}

	// Poll `kill -0 <pid>` every 1 s for up to 10 s.
	checkCmd := fmt.Sprintf("kill -0 %d 2>&1 && echo ALIVE || echo DEAD", inst.PID)
	dead := false
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("visionengine/remote: StopInstance SIGTERM poll cancelled for port=%d pid=%d: %w", port, inst.PID, ctx.Err())
		default:
		}
		out, _ := runRemote(client, checkCmd)
		if strings.Contains(strings.TrimSpace(string(out)), "DEAD") {
			dead = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !dead {
		// SIGKILL fallback.
		killCmd := fmt.Sprintf("kill -9 %d 2>&1 || true", inst.PID)
		if out, killErr := runRemote(client, killCmd); killErr != nil {
			log.Printf("WARN visionengine/remote.LlamaCppDeployer.StopInstance: SIGKILL SSH session error for host=%s pid=%d: %v (out=%q)",
				d.sshConfig.Host, inst.PID, killErr, strings.TrimSpace(string(out)))
		}
		// Re-check once after SIGKILL.
		out, _ := runRemote(client, checkCmd)
		if !strings.Contains(strings.TrimSpace(string(out)), "DEAD") {
			return fmt.Errorf("visionengine/remote: StopInstance failed to terminate port=%d pid=%d host=%s after SIGTERM (10s poll) + SIGKILL — process may be uninterruptible (D state) or kill -0 / -9 may lack permission; instance kept in tracking map for manual remediation",
				port, inst.PID, d.sshConfig.Host)
		}
	}

	d.instances.delete(port)
	log.Printf("INFO visionengine/remote.LlamaCppDeployer.StopInstance: terminated llama-server on host=%s port=%d pid=%d (instance removed from tracking).",
		d.sshConfig.Host, port, inst.PID)
	return nil
}

// StopRPCServer stops the RPC server bound at the given port.
//
// Round-52 §11.4 anti-bluff wiring (2026-05-18): semantically
// equivalent to StopInstance — both terminate a tracked
// llama-server PID — but kept as a distinct method so callers can
// express intent ("I'm shutting down the RPC-server endpoint" vs
// "I'm shutting down a worker-pool instance"). Bifurcates on
// SSHConfig identically; returns ErrRPCServerStopNotImplemented
// (NOT ErrRPCServerStopInstanceNotImplemented) for the unconfigured
// path so caller's `errors.Is` switch over the four round-48
// sentinels remains exhaustive.
//
// Constitutional anchors: CONST-035, CONST-042, CONST-050(A),
// Article XI §11.9.
func (d *LlamaCppDeployer) StopRPCServer(ctx context.Context, port int) error {
	if d.sshConfig.Host == "" {
		log.Printf("WARN visionengine/remote.LlamaCppDeployer.StopRPCServer: SSHConfig unset for host=%q port=%d — returning ErrRPCServerStopNotImplemented (round-48 sentinel preserved as the unconfigured-SSH signal; call WithSSHConfig to enable real termination).", d.config.Host, port)
		return ErrRPCServerStopNotImplemented
	}

	inst, ok := d.instances.get(port)
	if !ok {
		return fmt.Errorf("%w: port=%d host=%s (StartRPCServer or StartWithRPC was never called for this port, or the instance was already stopped)",
			ErrRPCInstanceNotFound, port, d.sshConfig.Host)
	}

	client, err := sshConn(ctx, d.sshConfig)
	if err != nil {
		log.Printf("WARN visionengine/remote.LlamaCppDeployer.StopRPCServer: SSH dial to %s@%s failed for port=%d pid=%d: %v",
			d.sshConfig.User, d.sshConfig.Host, port, inst.PID, err)
		return fmt.Errorf("visionengine/remote: StopRPCServer SSH dial failed: %w", err)
	}
	defer client.Close()

	termCmd := fmt.Sprintf("kill %d 2>&1 || true", inst.PID)
	if out, killErr := runRemote(client, termCmd); killErr != nil {
		log.Printf("WARN visionengine/remote.LlamaCppDeployer.StopRPCServer: SIGTERM SSH session error for host=%s pid=%d: %v (out=%q)",
			d.sshConfig.Host, inst.PID, killErr, strings.TrimSpace(string(out)))
	}

	checkCmd := fmt.Sprintf("kill -0 %d 2>&1 && echo ALIVE || echo DEAD", inst.PID)
	dead := false
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("visionengine/remote: StopRPCServer SIGTERM poll cancelled for port=%d pid=%d: %w", port, inst.PID, ctx.Err())
		default:
		}
		out, _ := runRemote(client, checkCmd)
		if strings.Contains(strings.TrimSpace(string(out)), "DEAD") {
			dead = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !dead {
		killCmd := fmt.Sprintf("kill -9 %d 2>&1 || true", inst.PID)
		if out, killErr := runRemote(client, killCmd); killErr != nil {
			log.Printf("WARN visionengine/remote.LlamaCppDeployer.StopRPCServer: SIGKILL SSH session error for host=%s pid=%d: %v (out=%q)",
				d.sshConfig.Host, inst.PID, killErr, strings.TrimSpace(string(out)))
		}
		out, _ := runRemote(client, checkCmd)
		if !strings.Contains(strings.TrimSpace(string(out)), "DEAD") {
			return fmt.Errorf("visionengine/remote: StopRPCServer failed to terminate port=%d pid=%d host=%s after SIGTERM (10s poll) + SIGKILL — process may be uninterruptible (D state) or kill -0 / -9 may lack permission; instance kept in tracking map for manual remediation",
				port, inst.PID, d.sshConfig.Host)
		}
	}

	d.instances.delete(port)
	log.Printf("INFO visionengine/remote.LlamaCppDeployer.StopRPCServer: terminated llama-server --rpc on host=%s port=%d pid=%d (instance removed from tracking).",
		d.sshConfig.Host, port, inst.PID)
	return nil
}

// parsePIDFromOutput extracts the last numeric token from an SSH
// command's combined stdout+stderr output. nohup may prefix
// "nohup: ignoring input and redirecting stderr to stdout" before
// the `echo $!` PID line, and pgrep may return multiple lines for
// ambiguous matches; taking the last numeric token handles both
// cases conservatively. Returns 0 on no-numeric-found.
func parsePIDFromOutput(out string) int {
	// Walk tokens from the end; first all-digit token wins.
	tokens := strings.Fields(out)
	for i := len(tokens) - 1; i >= 0; i-- {
		t := strings.TrimSpace(tokens[i])
		if t == "" {
			continue
		}
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			return n
		}
	}
	return 0
}
