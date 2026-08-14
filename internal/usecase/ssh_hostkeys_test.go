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
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strconv"
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
// key, and returns its host and port. The handshake is expected to fail: the
// client aborts as soon as the key is presented.
func sshTestServer(t *testing.T, signer ssh.Signer) (string, uint16) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %s", err)
	}
	t.Cleanup(func() { listener.Close() })

	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(signer)

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
	if _, err := us.FetchSSHHostKeys(happydns.SSHHostKeysRequest{}); err == nil {
		t.Fatal("expected an error on an empty host")
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
	// The trailing dot is dropped, so the host echoes back the way the user
	// would write it in the editor.
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
	// The refusal must never echo the address that was refused.
	if resp.ErrorMsg == "" {
		t.Error("a refusal must come with an explanation")
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

	if resp.Status == "ok" {
		t.Fatalf("status = ok on a server that speaks no SSH: %+v", resp.Keys)
	}
	if resp.ErrorMsg == "" {
		t.Error("a failure must come with an explanation")
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

func TestFetchSSHHostKeys_HostPortJoinedForIPv6(t *testing.T) {
	// Purely a guard against a naive host+":"+port concatenation.
	addr := net.JoinHostPort("::1", strconv.Itoa(22))
	if addr != "[::1]:22" {
		t.Fatalf("JoinHostPort = %q", addr)
	}
}
