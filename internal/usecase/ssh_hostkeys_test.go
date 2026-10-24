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
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/crypto/ssh"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

// loopbackUsecase builds a resolver usecase whose outbound guard accepts the
// loopback, which is otherwise refused as not globally routable.
func loopbackUsecase(t *testing.T) *resolverUsecase {
	t.Helper()

	guard, err := netguard.New("outbound", "-outbound-allowed-target", []string{"127.0.0.0/8", "::1/128"})
	if err != nil {
		t.Fatalf("netguard.New: %s", err)
	}

	return &resolverUsecase{config: &happydns.Options{}, outboundGuard: guard}
}

// blockingUsecase builds a resolver usecase whose outbound guard accepts
// nothing, the default policy of a fresh instance.
func blockingUsecase(t *testing.T) *resolverUsecase {
	t.Helper()

	guard, err := netguard.New("outbound", "-outbound-allowed-target", nil)
	if err != nil {
		t.Fatalf("netguard.New: %s", err)
	}

	return &resolverUsecase{config: &happydns.Options{}, outboundGuard: guard}
}

// sshTestServer starts an SSH server on the loopback, serving the given host
// keys, and returns its host and port. The handshake is expected to fail: the
// client aborts as soon as the key is presented.
func sshTestServer(t *testing.T, signers ...ssh.Signer) (string, uint16) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %s", err)
	}
	t.Cleanup(func() { listener.Close() })

	cfg := &ssh.ServerConfig{NoClientAuth: true}
	for _, signer := range signers {
		cfg.AddHostKey(signer)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				//nolint:errcheck // the client always aborts the handshake
				ssh.NewServerConn(conn, cfg)
			}()
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.IP.String(), uint16(addr.Port)
}

func ed25519Signer(t *testing.T) (ssh.Signer, ssh.PublicKey) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %s", err)
	}

	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("ssh.NewSignerFromKey: %s", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("ssh.NewPublicKey: %s", err)
	}

	return signer, sshPub
}

func TestFetchSSHHostKeys_EmptyHost(t *testing.T) {
	us := &resolverUsecase{}
	for _, host := range []string{"", "   ", "[]", "."} {
		_, err := us.FetchSSHHostKeys(happydns.SSHHostKeysRequest{Host: host})

		var verr happydns.ValidationError
		if !errors.As(err, &verr) {
			t.Errorf("host %q: err = %v, want a ValidationError", host, err)
		}
	}
}

func TestFetchSSHHostKeys_Ed25519(t *testing.T) {
	signer, pub := ed25519Signer(t)
	host, port := sshTestServer(t, signer)

	resp, err := loopbackUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{
		Host: host,
		Port: port,
	})
	if err != nil {
		t.Fatalf("FetchSSHHostKeys: %s", err)
	}

	if resp.Status != "ok" {
		t.Fatalf("status = %q (%s), want ok", resp.Status, resp.ErrorMsg)
	}
	if len(resp.Keys) != 1 {
		t.Fatalf("collected %d keys, want 1: %+v", len(resp.Keys), resp.Keys)
	}

	key := resp.Keys[0]
	if key.Algorithm != 4 {
		t.Errorf("algorithm = %d, want 4 (Ed25519)", key.Algorithm)
	}
	if key.AlgorithmName != ssh.KeyAlgoED25519 {
		t.Errorf("algorithmName = %q, want %q", key.AlgorithmName, ssh.KeyAlgoED25519)
	}

	blob := pub.Marshal()
	wantSHA256 := sha256.Sum256(blob)
	if key.SHA256 != hex.EncodeToString(wantSHA256[:]) {
		t.Errorf("sha256 = %q, want %q", key.SHA256, hex.EncodeToString(wantSHA256[:]))
	}
	wantSHA1 := sha1.Sum(blob)
	if key.SHA1 != hex.EncodeToString(wantSHA1[:]) {
		t.Errorf("sha1 = %q, want %q", key.SHA1, hex.EncodeToString(wantSHA1[:]))
	}

	if resp.Port != port {
		t.Errorf("port = %d, want %d", resp.Port, port)
	}
}

func TestFetchSSHHostKeys_DefaultsToPort22(t *testing.T) {
	// The destination is refused right away, so nothing is dialled: only the
	// normalization of the request is under test here.
	resp, err := blockingUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{Host: "127.0.0.1."})
	if err != nil {
		t.Fatalf("FetchSSHHostKeys: %s", err)
	}

	if resp.Port != 22 {
		t.Errorf("port = %d, want 22", resp.Port)
	}
	// The trailing dot is normalized away.
	if resp.Host != "127.0.0.1" {
		t.Errorf("host = %q, want 127.0.0.1", resp.Host)
	}
}

func TestFetchSSHHostKeys_Blocked(t *testing.T) {
	signer, _ := ed25519Signer(t)
	host, port := sshTestServer(t, signer)

	resp, err := blockingUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{
		Host: host,
		Port: port,
	})
	if err != nil {
		t.Fatalf("FetchSSHHostKeys: %s", err)
	}

	if resp.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", resp.Status)
	}
	if len(resp.Keys) != 0 {
		t.Errorf("collected %d keys from a refused destination", len(resp.Keys))
	}
	if resp.ErrorMsg == "" {
		t.Error("a refusal must come with an explanation")
	}
	// The refusal must never echo the address that was refused.
	if strings.Contains(resp.ErrorMsg, host) {
		t.Errorf("refusal leaks the address: %q", resp.ErrorMsg)
	}
}

func TestFetchSSHHostKeys_NotAnSSHServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %s", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	resp, err := loopbackUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{
		Host: addr.IP.String(),
		Port: uint16(addr.Port),
	})
	if err != nil {
		t.Fatalf("FetchSSHHostKeys: %s", err)
	}

	if resp.Status != "handshake-error" {
		t.Fatalf("status = %q (%s), want handshake-error", resp.Status, resp.ErrorMsg)
	}
	if resp.ErrorMsg == "" {
		t.Error("a failure must come with an explanation")
	}
}

// A server resetting the connection after accepting it was reached: the
// failure belongs to the handshake, not to the connection.
func TestFetchSSHHostKeys_ResetAfterAccept(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %s", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Waiting for the client banner guarantees the dial completed:
			// a RST sent earlier can make connect() itself fail.
			conn.Read(make([]byte, 256))
			// A zero linger turns Close into a RST instead of a FIN.
			conn.(*net.TCPConn).SetLinger(0)
			conn.Close()
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	resp, err := loopbackUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{
		Host: addr.IP.String(),
		Port: uint16(addr.Port),
	})
	if err != nil {
		t.Fatalf("FetchSSHHostKeys: %s", err)
	}

	if resp.Status != "handshake-error" {
		t.Fatalf("status = %q (%s), want handshake-error", resp.Status, resp.ErrorMsg)
	}
}

func TestFetchSSHHostKeys_ConnectionRefused(t *testing.T) {
	// A port nothing listens on: bound then released.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %s", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	listener.Close()

	resp, err := loopbackUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{
		Host: addr.IP.String(),
		Port: uint16(addr.Port),
	})
	if err != nil {
		t.Fatalf("FetchSSHHostKeys: %s", err)
	}

	if resp.Status != "connect-error" {
		t.Fatalf("status = %q, want connect-error (%s)", resp.Status, resp.ErrorMsg)
	}
}

func TestFetchSSHHostKeys_KeyTypes(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %s", err)
	}

	tests := []struct {
		name      string
		key       any
		algorithm uint8
		algoName  string
	}{
		{"rsa", rsaKey, 1, ssh.KeyAlgoRSA},
		{"ecdsa-p256", mustECDSA(t, elliptic.P256()), 3, ssh.KeyAlgoECDSA256},
		{"ecdsa-p384", mustECDSA(t, elliptic.P384()), 3, ssh.KeyAlgoECDSA384},
		{"ecdsa-p521", mustECDSA(t, elliptic.P521()), 3, ssh.KeyAlgoECDSA521},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := ssh.NewSignerFromKey(tt.key)
			if err != nil {
				t.Fatalf("ssh.NewSignerFromKey: %s", err)
			}
			host, port := sshTestServer(t, signer)

			resp, err := loopbackUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{Host: host, Port: port})
			if err != nil {
				t.Fatalf("FetchSSHHostKeys: %s", err)
			}
			if len(resp.Keys) != 1 {
				t.Fatalf("collected %d keys, want 1 (%s): %+v", len(resp.Keys), resp.ErrorMsg, resp.Keys)
			}
			if resp.Keys[0].Algorithm != tt.algorithm || resp.Keys[0].AlgorithmName != tt.algoName {
				t.Errorf("got algorithm %d (%s), want %d (%s)", resp.Keys[0].Algorithm, resp.Keys[0].AlgorithmName, tt.algorithm, tt.algoName)
			}
		})
	}
}

func mustECDSA(t *testing.T, curve elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey: %s", err)
	}
	return key
}

func TestFetchSSHHostKeys_SeveralKeys(t *testing.T) {
	edSigner, edPub := ed25519Signer(t)
	ecSigner, err := ssh.NewSignerFromKey(mustECDSA(t, elliptic.P256()))
	if err != nil {
		t.Fatalf("ssh.NewSignerFromKey: %s", err)
	}
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %s", err)
	}
	rsaSigner, err := ssh.NewSignerFromKey(rsaKey)
	if err != nil {
		t.Fatalf("ssh.NewSignerFromKey: %s", err)
	}
	host, port := sshTestServer(t, edSigner, ecSigner, rsaSigner)

	resp, err := loopbackUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{Host: host, Port: port})
	if err != nil {
		t.Fatalf("FetchSSHHostKeys: %s", err)
	}

	// In the order of sshfpAlgorithms, whatever the order the handshakes end in.
	var got []uint8
	for _, k := range resp.Keys {
		got = append(got, k.Algorithm)
	}
	if fmt.Sprint(got) != "[1 3 4]" {
		t.Fatalf("algorithms = %v, want [1 3 4]", got)
	}

	want := sha256.Sum256(edPub.Marshal())
	if resp.Keys[2].SHA256 != hex.EncodeToString(want[:]) {
		t.Errorf("ed25519 sha256 = %q, want %q", resp.Keys[2].SHA256, hex.EncodeToString(want[:]))
	}
}

func TestFetchSSHHostKeys_HostNormalization(t *testing.T) {
	// Refused right away by the guard: only the normalization is under test.
	tests := map[string]string{
		"  127.0.0.1 ": "127.0.0.1",
		"[::1]":        "::1",
		"example.org.": "example.org",
		"127.0.0.1":    "127.0.0.1",
	}
	for in, want := range tests {
		resp, err := blockingUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{Host: in})
		if err != nil {
			t.Errorf("host %q: %s", in, err)
			continue
		}
		if resp.Host != want {
			t.Errorf("host %q normalized to %q, want %q", in, resp.Host, want)
		}
	}
}

func TestFetchSSHHostKeys_SilentServer(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the per-key deadline")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %s", err)
	}
	defer listener.Close()

	// Accepts and never writes a banner.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	resp, err := loopbackUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{
		Host: addr.IP.String(),
		Port: uint16(addr.Port),
	})
	if err != nil {
		t.Fatalf("FetchSSHHostKeys: %s", err)
	}

	if resp.Status != "timeout" {
		t.Fatalf("status = %q (%s), want timeout", resp.Status, resp.ErrorMsg)
	}
}

func TestFetchSSHHostKeys_UnresolvableHost(t *testing.T) {
	resp, err := loopbackUsecase(t).FetchSSHHostKeys(happydns.SSHHostKeysRequest{Host: "nonexistent.invalid"})
	if err != nil {
		t.Fatalf("FetchSSHHostKeys: %s", err)
	}

	if resp.Status != "connect-error" {
		t.Fatalf("status = %q (%s), want connect-error", resp.Status, resp.ErrorMsg)
	}
}

func TestDialAny(t *testing.T) {
	t.Run("falls through a refused address", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.2:0")
		if err != nil {
			t.Skipf("127.0.0.2 not available: %s", err)
		}
		defer listener.Close()
		go func() {
			if conn, err := listener.Accept(); err == nil {
				conn.Close()
			}
		}()

		port := uint16(listener.Addr().(*net.TCPAddr).Port)
		// Nothing listens on 127.0.0.1 at this port: the first address is refused.
		conn, err := dialAny(context.Background(), []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("127.0.0.2")}, port)
		if err != nil {
			t.Fatalf("dialAny: %s", err)
		}
		conn.Close()
	})

	t.Run("fails without any address", func(t *testing.T) {
		_, err := dialAny(context.Background(), nil, 22)
		if err == nil || !strings.Contains(err.Error(), "no usable address") {
			t.Fatalf("err = %v, want no usable address", err)
		}
	})
}

func TestClassifySSHError(t *testing.T) {
	us := blockingUsecase(t)
	expired, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want string
	}{
		{"blocked", context.Background(), fmt.Errorf("x: %w", netguard.ErrBlocked), "blocked"},
		{"context expired", expired, errors.New("x"), "timeout"},
		{"read deadline", context.Background(), &net.OpError{Op: "read", Err: os.ErrDeadlineExceeded}, "timeout"},
		{"context deadline", context.Background(), fmt.Errorf("x: %w", context.DeadlineExceeded), "timeout"},
		{"dns", context.Background(), &net.DNSError{Err: "no such host"}, "connect-error"},
		{"op error", context.Background(), &net.OpError{Op: "dial", Err: errors.New("refused")}, "connect-error"},
		{"reset on read", context.Background(), &net.OpError{Op: "read", Err: syscall.ECONNRESET}, "handshake-error"},
		{"broken pipe on write", context.Background(), &net.OpError{Op: "write", Err: syscall.EPIPE}, "handshake-error"},
		{"no error", context.Background(), nil, "handshake-error"},
		{"other", context.Background(), errors.New("ssh: handshake failed"), "handshake-error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := us.classifySSHError(tt.ctx, tt.err); got != tt.want {
				t.Errorf("status = %q, want %q", got, tt.want)
			}
		})
	}
}
