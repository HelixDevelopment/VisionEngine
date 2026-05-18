// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHConfig holds the credentials and connection parameters
// VisionPool uses to reach the remote GPU host for in-band
// process management (kill orphaned llama-server / Ollama
// processes on Shutdown).
//
// Round-40 §11.4 anti-bluff fix (2026-05-18): the round-27
// fix introduced ErrShutdownRemoteCleanupNotImplemented as
// a loud sentinel so the orphan-process gap was at least
// detectable. SSHConfig + the wiring in Shutdown turns that
// detection into action — when SSHConfig is populated,
// Shutdown actually SSHes in and terminates the orphan
// processes instead of merely warning about them.
//
// All fields MUST be populated from environment variables
// or a user-controlled config file — never hardcoded —
// per CONST-042 (no-secret-leak) and CONST-046
// (no-hardcoded-content).
type SSHConfig struct {
	// Host is the SSH-reachable hostname or IP of the
	// remote GPU host. Empty Host means SSH is NOT
	// configured; Shutdown / EnsureReady will return the
	// round-27/28 "not implemented" sentinels in that case
	// (preserving the round-27/28 contract for callers that
	// have not yet opted into SSH).
	Host string

	// Port is the SSH port (default 22 when zero).
	Port int

	// User is the SSH login user.
	User string

	// KeyPath is the absolute path to the PEM-encoded
	// private key (OpenSSH or RSA/Ed25519 PEM). The key
	// MUST be readable only by the invoking user
	// (chmod 0600 enforced by SSH library itself).
	KeyPath string

	// KnownHostsPath is the absolute path to the
	// known_hosts file used to verify the remote host
	// key. CONST-035 (anti-bluff) + CONST-042
	// (no-secret-leak): accepting unknown hosts is a
	// silent security PASS-bluff, so KnownHostsPath MUST
	// point to a real, curated known_hosts file. Empty
	// KnownHostsPath is an explicit configuration error
	// (ErrSSHHostKeyVerificationFailed at dial time).
	KnownHostsPath string

	// Timeout is the SSH dial + per-session timeout. Zero
	// defaults to 30 s.
	Timeout time.Duration
}

// ErrSSHKeyParseFailed signals that the SSH private-key
// material at SSHConfig.KeyPath could not be read or parsed
// (file not readable, not PEM, wrong key type, etc.). The
// error wraps the underlying os / ssh error for diagnostics.
var ErrSSHKeyParseFailed = errors.New(
	"visionengine ssh: private key PEM at SSHConfig.KeyPath could not be parsed " +
		"— verify file format (OpenSSH or PEM) and that the file is readable by the invoking user")

// ErrSSHHostKeyVerificationFailed signals that host-key
// verification against the configured known_hosts file
// failed. CONST-035: accepting unknown hosts is a silent
// security PASS-bluff — production callers MUST populate
// SSHConfig.KnownHostsPath with a curated known_hosts file
// before this code will dial out.
var ErrSSHHostKeyVerificationFailed = errors.New(
	"visionengine ssh: host key verification failed against known_hosts " +
		"— refusing to connect to potentially-malicious host (CONST-035: accepting unknown hosts " +
		"is a silent security PASS-bluff)")

// sshConn opens a fresh SSH client connection using cfg.
// Caller is responsible for Close() on the returned client.
// Lazily acquired per-call instead of long-lived to avoid
// stale-connection state across long-lived VisionPool
// lifetimes.
func sshConn(_ context.Context, cfg SSHConfig) (*ssh.Client, error) {
	if cfg.Host == "" {
		return nil, errors.New("visionengine ssh: empty Host (SSH not configured)")
	}
	if cfg.User == "" {
		return nil, errors.New("visionengine ssh: empty User in SSHConfig")
	}
	if cfg.KeyPath == "" {
		return nil, fmt.Errorf("%w: SSHConfig.KeyPath is empty", ErrSSHKeyParseFailed)
	}
	if cfg.KnownHostsPath == "" {
		return nil, fmt.Errorf("%w: SSHConfig.KnownHostsPath is empty", ErrSSHHostKeyVerificationFailed)
	}

	keyBytes, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("%w: read %q: %v", ErrSSHKeyParseFailed, cfg.KeyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: parse %q: %v", ErrSSHKeyParseFailed, cfg.KeyPath, err)
	}

	hostKeyCallback, err := knownhosts.New(cfg.KnownHostsPath)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: load known_hosts %q: %v",
			ErrSSHHostKeyVerificationFailed, cfg.KnownHostsPath, err)
	}

	port := cfg.Port
	if port == 0 {
		port = 22
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	clientCfg := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	client, err := ssh.Dial("tcp", addr, clientCfg)
	if err != nil {
		// Wrap host-key errors specifically so callers can
		// distinguish CONST-035 violations from generic
		// connectivity failures.
		if isHostKeyError(err) {
			return nil, fmt.Errorf("%w: dial %s: %v", ErrSSHHostKeyVerificationFailed, addr, err)
		}
		return nil, fmt.Errorf("visionengine ssh: dial %s: %w", addr, err)
	}
	return client, nil
}

// isHostKeyError reports whether the dial-time error chain
// looks like a host-key verification failure. The standard
// library's ssh package returns a *knownhosts.KeyError
// wrapped inside a *net.OpError-style chain; we match on
// substring as a defensive fallback in case the wrapping
// scheme changes between Go versions.
func isHostKeyError(err error) bool {
	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "knownhosts") ||
		strings.Contains(msg, "host key") ||
		strings.Contains(msg, "ssh: handshake failed")
}

// runRemote executes one command on the open SSH client,
// returning combined stdout+stderr. Each call gets its own
// session (SSH sessions are single-use by spec).
func runRemote(client *ssh.Client, cmdline string) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("visionengine ssh: new session: %w", err)
	}
	defer session.Close()
	out, err := session.CombinedOutput(cmdline)
	return out, err
}
