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
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

const (
	sshDefaultPort     = 22
	sshTotalDeadline   = 10 * time.Second
	sshPerKeyDeadline  = 4 * time.Second
	sshClientVersion   = "SSH-2.0-happyDomain"
	sshMaxKeysExpected = 8
)

// sshfpAlgorithms maps the SSH key type names to the SSHFP algorithm numbers of
// RFC 4255 sec. 3.1, RFC 6594 and RFC 8709. The key exchange is driven once per
// entry, since a server only presents the host key matching the algorithm the
// client asked for.
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
// It opens an SSH connection to the given host, once per host key algorithm,
// and stops each of them as soon as the server has presented its key. Nothing
// is ever authenticated: the exchange is aborted before the client offers any
// credential.
func (us *resolverUsecase) FetchSSHHostKeys(req happydns.SSHHostKeysRequest) (*happydns.SSHHostKeysResponse, error) {
	host := strings.TrimSuffix(strings.TrimSpace(req.Host), ".")
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	if host == "" {
		return nil, errors.New("host is required")
	}

	port := req.Port
	if port == 0 {
		port = sshDefaultPort
	}

	resp := &happydns.SSHHostKeysResponse{Host: host, Port: port}

	ctx, cancel := context.WithTimeout(context.Background(), sshTotalDeadline)
	defer cancel()

	// Resolved once and shared by every handshake below: they all target the
	// same host, and asking the guard to re-resolve it per algorithm would
	// only reopen the DNS-rebinding window pinning is meant to close, for no
	// benefit.
	addrs, err := us.outboundGuard.ResolveAllowed(ctx, host)
	if err != nil {
		resp.Status, resp.ErrorMsg = us.classifySSHError(ctx, err)
		return resp, nil
	}

	// The algorithms are independent handshakes against the same address, so
	// they are driven concurrently: each is bounded by its own
	// sshPerKeyDeadline anyway, and probing them one at a time would pay for
	// every algorithm the server does not offer in series.
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

	seen := make(map[string]bool, sshMaxKeysExpected)
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
		if seen[fingerprint] {
			continue
		}
		seen[fingerprint] = true

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
// caller-resolved, already-guard-checked set of addresses for the host:
// dialing straight to one of those IP literals, rather than through
// outboundGuard.DialContext again, avoids resolving the name once per
// algorithm.
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
		// No user, no authentication method: the handshake is aborted from the
		// callback below, before the client would offer anything.
		User:              "",
		Auth:              nil,
		HostKeyAlgorithms: algos,
		ClientVersion:     sshClientVersion,
		// ClientConfig.Timeout is only read by ssh.Dial; the deadline that
		// actually bounds this handshake is the conn.SetDeadline call above,
		// since the connection goes through ssh.NewClientConn instead.
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			// The point is to read what the server presents, never to
			// authenticate against it: any non-nil error aborts the handshake
			// right here, before the client would offer anything, and the
			// specific error is discarded by the caller (only hostKey matters).
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
		if errors.Is(err, syscall.ECONNREFUSED) {
			continue
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("no usable address to reach host")
}

func (us *resolverUsecase) classifySSHError(ctx context.Context, err error) (status, msg string) {
	if errors.Is(err, netguard.ErrBlocked) {
		return "blocked", us.outboundGuard.Refusal("The SSH server")
	}

	if ctx.Err() != nil {
		return "timeout", "the SSH server did not answer in time"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "connect-error", fmt.Sprintf("cannot resolve the host: %s", dnsErr.Err)
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return "connect-error", opErr.Err.Error()
	}

	if err == nil {
		return "handshake-error", "the server offered no host key"
	}

	return "handshake-error", err.Error()
}
