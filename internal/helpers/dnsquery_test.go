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

package helpers

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
)

// startTruncatingDNSServer serves example.com. on a local port, over UDP
// and TCP: over UDP, it answers truncated when truncate is set. It returns
// its address and the number of questions received over TCP.
func startTruncatingDNSServer(t *testing.T, truncate bool) (string, *atomic.Int32) {
	t.Helper()

	var tcpQuestions atomic.Int32
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(req)

		if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
			tcpQuestions.Add(1)
		} else if truncate {
			m.Truncated = true
			w.WriteMsg(m)
			return
		}

		rr, _ := dns.NewRR("example.com. 300 IN TXT \"full answer\"")
		m.Answer = append(m.Answer, rr)
		w.WriteMsg(m)
	})

	// The UDP and TCP servers share the port, as a client only knows one
	// address.
	var (
		l  net.Listener
		pc net.PacketConn
	)
	for range 10 {
		var err error
		l, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen tcp: %v", err)
		}
		pc, err = net.ListenPacket("udp", l.Addr().String())
		if err == nil {
			break
		}
		l.Close()
		l = nil
	}
	if l == nil {
		t.Fatal("unable to listen on the same UDP and TCP port")
	}

	for _, srv := range []*dns.Server{{Listener: l, Handler: handler}, {PacketConn: pc, Handler: handler}} {
		started := make(chan struct{})
		srv.NotifyStartedFunc = func() { close(started) }
		go srv.ActivateAndServe()
		<-started
		t.Cleanup(func() { srv.Shutdown() })
	}

	return l.Addr().String(), &tcpQuestions
}

func txtQuestion() *dns.Msg {
	m := new(dns.Msg)
	m.SetQuestion("example.com.", dns.TypeTXT)
	m.SetEdns0(4096, true)
	return m
}

func TestExchangeRetriesOverTCPWhenTruncated(t *testing.T) {
	addr, tcpQuestions := startTruncatingDNSServer(t, true)
	client := &dns.Client{Timeout: time.Second}

	r, err := Exchange(context.Background(), client, txtQuestion(), addr)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if r.Truncated || len(r.Answer) != 1 {
		t.Fatalf("want the full answer, got truncated=%v and %d records", r.Truncated, len(r.Answer))
	}
	if n := tcpQuestions.Load(); n != 1 {
		t.Errorf("want 1 question over TCP, got %d", n)
	}
	if client.Net != "" {
		t.Errorf("the client given was changed to %q", client.Net)
	}
}

func TestExchangeStaysOnUDPWhenNotTruncated(t *testing.T) {
	addr, tcpQuestions := startTruncatingDNSServer(t, false)

	r, err := Exchange(context.Background(), &dns.Client{Timeout: time.Second}, txtQuestion(), addr)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if len(r.Answer) != 1 {
		t.Fatalf("want 1 record, got %d", len(r.Answer))
	}
	if n := tcpQuestions.Load(); n != 0 {
		t.Errorf("want no question over TCP, got %d", n)
	}
}

func TestExchangeError(t *testing.T) {
	// Nothing listens there: the question times out.
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer pc.Close()

	if _, err := Exchange(context.Background(), &dns.Client{Timeout: 100 * time.Millisecond}, txtQuestion(), pc.LocalAddr().String()); err == nil {
		t.Fatal("want an error from a server not answering")
	}
}
