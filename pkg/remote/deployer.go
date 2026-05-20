// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// LlamaCppDeployer manages llama.cpp server instances on a
// remote GPU host via SSH. It handles starting/stopping
// llama-server processes and freeing GPU resources.
//
// Round-52 §11.4 anti-bluff (2026-05-18): adds `sshConfig` +
// `instances` fields to support real SSH-driven RPC lifecycle
// management (see distributed.go). The legacy `sshCmd` helper
// (using `exec.Command("ssh", ...)`) is retained for the existing
// FreeGPU / StartInstance / RestoreOllama methods which predate
// the round-40 in-process SSH client and still rely on the
// host's `ssh` binary. The four RPC lifecycle methods
// (StartRPCServer / StartWithRPC / StopInstance / StopRPCServer)
// use the round-40 sshConn helper via SSHConfig instead.
//
// Constitutional anchors: CONST-035 (anti-bluff — no silent
// success), CONST-042 (no-secret-leak — SSH credentials sourced
// from SSHConfig populated by env/config; never hardcoded).
type LlamaCppDeployer struct {
	config    LlamaCppConfig
	sshConfig SSHConfig
	instances *instanceMap
}

// NewLlamaCppDeployer creates a deployer with the given
// llama.cpp configuration. SSHConfig is left zero-valued;
// callers that want real RPC lifecycle management on the
// remote host MUST also call WithSSHConfig.
//
// Round-52: when SSHConfig is unset the four RPC lifecycle
// methods preserve the round-48 sentinel-error contract
// (ErrRPCServerStart/StartWithRPC/StopInstance/StopNotImplemented)
// — the legacy FreeGPU / StartInstance / RestoreOllama methods
// continue to use the host's `ssh` binary via sshCmd.
func NewLlamaCppDeployer(config LlamaCppConfig) *LlamaCppDeployer {
	return &LlamaCppDeployer{
		config:    config,
		instances: newInstanceMap(),
	}
}

// WithSSHConfig attaches SSH credentials so the deployer can
// perform real RPC lifecycle management (StartRPCServer,
// StartWithRPC, StopInstance, StopRPCServer) against the remote
// host. Returns the same deployer for chaining. Callers MUST
// source SSHConfig fields from env vars / config files — never
// hardcode them (CONST-042 no-secret-leak).
//
// Round-52 §11.4: this is the opt-in switch that flips the four
// RPC lifecycle methods from "return round-48 sentinel" to "do
// real SSH-driven work via sshConn (round-40 helper)".
func (d *LlamaCppDeployer) WithSSHConfig(cfg SSHConfig) *LlamaCppDeployer {
	d.sshConfig = cfg
	return d
}

// SSHConfigured reports whether the deployer has SSH credentials
// attached. False means the four RPC lifecycle methods will
// return their round-48 sentinel errors (preserved unconfigured-
// SSH signal); true means they will dial out via sshConn.
func (d *LlamaCppDeployer) SSHConfigured() bool {
	return d.sshConfig.Host != ""
}

// FreeGPU stops Ollama on the remote host to free GPU VRAM
// for llama-server instances. This is a no-op if Ollama is
// not running.
func (d *LlamaCppDeployer) FreeGPU(ctx context.Context) {
	_ = d.sshCmd(ctx, "systemctl", "--user", "stop", "ollama")
}

// StartInstance launches a llama-server process on the remote
// host at the specified port. The server runs in the
// background and listens for HTTP requests.
func (d *LlamaCppDeployer) StartInstance(
	ctx context.Context, port int,
) {
	args := []string{
		fmt.Sprintf("%s/build/bin/llama-server", d.config.RepoDir),
		"-m", d.config.ModelPath,
		"--mmproj", d.config.MMProjPath,
		"--port", fmt.Sprintf("%d", port),
		"-ngl", fmt.Sprintf("%d", d.config.GPULayers),
		"-c", fmt.Sprintf("%d", d.config.ContextSize),
	}
	cmd := strings.Join(args, " ")
	_ = d.sshCmd(ctx, "nohup", cmd, ">/dev/null", "2>&1", "&")
}

// RestoreOllama restarts Ollama on the remote host after
// llama-server instances have been stopped.
func (d *LlamaCppDeployer) RestoreOllama(ctx context.Context) {
	_ = d.sshCmd(ctx, "systemctl", "--user", "start", "ollama")
}

// sshCmd runs a command on the remote host via SSH.
func (d *LlamaCppDeployer) sshCmd(
	ctx context.Context, args ...string,
) error {
	if d.config.Host == "" {
		return fmt.Errorf("remote: deployer host is required")
	}
	target := d.config.Host
	if d.config.User != "" {
		target = d.config.User + "@" + d.config.Host
	}
	sshArgs := append(
		[]string{"-o", "StrictHostKeyChecking=no", target},
		args...,
	)
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	return cmd.Run()
}
