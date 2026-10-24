// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package usecase

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

const (
	sshDefaultPort    = 22
	sshTotalDeadline  = 10 * time.Second
	sshPerKeyDeadline = 4 * time.Second

	// Shows up in the logs of the probed server, so its administrator can tell
	// who is knocking.
	sshClientVersion = "SSH-2.0-happyDomain"
)

// sshfpAlgorithms maps the SSH key type names to the SSHFP algorithm numbers of
// RFC 4255 sec. 3.1, RFC 6594 and RFC 8709. The key exchange is driven once per
// entry, since a server only presents the host key matching the algorithm the
// client asked for. RSA's three signature algorithms share one key, so they are
// one entry; the ECDSA curves and Ed25519 each have their own key, hence their
// own handshake.
//
// DSA (SSHFP algorithm 2) is missing on purpose: x/crypto/ssh dropped it, and
// so did every implementation that could have served it.
var sshfpAlgorithms = []struct {
	hostKeyAlgos []string
	sshfp        uint8
}{
	{[]string{ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSA}, 1},
	{[]string{ssh.KeyAlgoECDSA256}, 3},
	{[]string{ssh.KeyAlgoECDSA384}, 3},
	{[]string{ssh.KeyAlgoECDSA521}, 3},
	{[]string{ssh.KeyAlgoED25519}, 4},
	{[]string{ssh.KeyAlgoSKECDSA256}, 3},
	{[]string{ssh.KeyAlgoSKED25519}, 4},
}

// FetchSSHHostKeys implements happydns.ResolverUsecase.
//
// It runs one aborted handshake per host key algorithm; see sshHostKey.
func (us *resolverUsecase) FetchSSHHostKeys(req happydns.SSHHostKeysRequest) (*happydns.SSHHostKeysResponse, error) {
	// SSHFP owners are written FQDN-style with a trailing dot, and IPv6
	// literals may arrive bracketed.
	host := strings.TrimSuffix(strings.TrimSpace(req.Host), ".")
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	if host == "" {
		return nil, happydns.ValidationError{Msg: "host is required"}
	}

	port := req.Port
	if port == 0 {
		port = sshDefaultPort
	}

	resp := &happydns.SSHHostKeysResponse{Host: host, Port: port}

	ctx, cancel := context.WithTimeout(context.Background(), sshTotalDeadline)
	defer cancel()

	// Resolved once: re-resolving per algorithm would reopen the DNS-rebinding
	// window that pinning closes.
	addrs, err := us.outboundGuard.ResolveAllowed(ctx, host)
	if err != nil {
		resp.Status, resp.ErrorMsg = us.classifySSHError(ctx, err)
		return resp, nil
	}

	// Concurrent so that the algorithms the server does not offer do not add up
	// their timeouts.
	type probe struct {
		key ssh.PublicKey
		err error
	}
	results := make([]probe, len(sshfpAlgorithms))
	var wg sync.WaitGroup
	for i, family := range sshfpAlgorithms {
		wg.Add(1)
		go func(i int, algos []string) {
			defer wg.Done()
			key, err := us.sshHostKey(ctx, addrs, port, algos)
			results[i] = probe{key: key, err: err}
		}(i, family.hostKeyAlgos)
	}
	wg.Wait()

	var lastErr error

	for i, family := range sshfpAlgorithms {
		r := results[i]
		if r.err != nil {
			// A server simply not offering this algorithm is the common case,
			// so only the last error is kept, and only to explain an empty
			// result.
			lastErr = r.err
			continue
		}

		blob := r.key.Marshal()
		sha256sum := sha256.Sum256(blob)
		fingerprint := hex.EncodeToString(sha256sum[:])

		sha1sum := sha1.Sum(blob)
		resp.Keys = append(resp.Keys, happydns.SSHHostKey{
			Algorithm:     family.sshfp,
			AlgorithmName: r.key.Type(),
			SHA256:        fingerprint,
			SHA1:          hex.EncodeToString(sha1sum[:]),
		})
	}

	if len(resp.Keys) > 0 {
		resp.Status = "ok"
		return resp, nil
	}

	resp.Status, resp.ErrorMsg = us.classifySSHError(ctx, lastErr)
	return resp, nil
}

// sshHostKey drives one handshake restricted to the given host key
// algorithms, and returns the key the server presented. addrs is the
// caller-resolved, already-guard-checked set of addresses for the host.
func (us *resolverUsecase) sshHostKey(ctx context.Context, addrs []netip.Addr, port uint16, algos []string) (ssh.PublicKey, error) {
	dialCtx, cancel := context.WithTimeout(ctx, sshPerKeyDeadline)
	defer cancel()

	conn, err := dialAny(dialCtx, addrs, port)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if deadline, ok := dialCtx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	var hostKey ssh.PublicKey
	cfg := &ssh.ClientConfig{
		HostKeyAlgorithms: algos,
		ClientVersion:     sshClientVersion,
		// ClientConfig.Timeout is only read by ssh.Dial; the deadline that
		// actually bounds this handshake is the conn.SetDeadline call above,
		// since the connection goes through ssh.NewClientConn instead.
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			// Any non-nil error aborts the handshake before the client offers a
			// credential; the caller only wants the key and discards this error.
			hostKey = key
			return errors.New("host key captured")
		},
	}

	_, _, _, err = ssh.NewClientConn(conn, conn.RemoteAddr().String(), cfg)
	if hostKey != nil {
		return hostKey, nil
	}
	if err == nil {
		// Cannot happen: the callback always refuses. Reported rather than
		// silently returning a nil key.
		return nil, errors.New("the server presented no host key")
	}

	return nil, err
}

// dialAny connects to the first reachable of the given already-guard-checked
// addresses, mirroring the fallback loop of (*netguard.Guard).DialContext.
func dialAny(ctx context.Context, addrs []netip.Addr, port uint16) (net.Conn, error) {
	dialer := &net.Dialer{}

	var lastErr error
	for _, a := range addrs {
		conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(a.String(), fmt.Sprint(port)))
		if err == nil {
			return conn, nil
		}

		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("no usable address to reach host")
}

// classifySSHError turns a failure into the status and message of the response.
// Deadline errors count as a timeout whether they come from the overall context
// or from the shorter per-key one, which always expires first.
func (us *resolverUsecase) classifySSHError(ctx context.Context, err error) (status, msg string) {
	if errors.Is(err, netguard.ErrBlocked) {
		return "blocked", us.outboundGuard.Refusal("The SSH server")
	}

	// Before the OpError check below: a timed-out read is an OpError too.
	if ctx.Err() != nil || errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return "timeout", "the SSH server did not answer in time"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "connect-error", fmt.Sprintf("cannot resolve the host: %s", dnsErr.Err)
	}

	// Only a failed dial means the server could not be reached: a read or
	// write error comes from a connection that was established, so the
	// server answered and it is the handshake that failed.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			return "connect-error", opErr.Err.Error()
		}
		return "handshake-error", opErr.Err.Error()
	}

	if err == nil {
		return "handshake-error", "the server offered no host key"
	}

	return "handshake-error", err.Error()
}
