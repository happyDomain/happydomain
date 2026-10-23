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

	r, err := Exchange(context.Background(), client, txtQuestion(), []string{addr})
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

	r, err := Exchange(context.Background(), &dns.Client{Timeout: time.Second}, txtQuestion(), []string{addr})
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

// silentServer returns the address of a UDP socket that never answers.
func silentServer(t *testing.T) string {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { pc.Close() })
	return pc.LocalAddr().String()
}

func TestExchangeError(t *testing.T) {
	client := &dns.Client{Timeout: 100 * time.Millisecond}

	if _, err := Exchange(context.Background(), client, txtQuestion(), []string{silentServer(t), silentServer(t)}); err == nil {
		t.Fatal("want an error when no server answers")
	}

	if _, err := Exchange(context.Background(), client, txtQuestion(), nil); err == nil {
		t.Fatal("want an error without any server")
	}
}

func TestExchangeFallsBackToTheNextServer(t *testing.T) {
	addr, _ := startTruncatingDNSServer(t, false)

	r, err := Exchange(context.Background(), &dns.Client{Timeout: 100 * time.Millisecond}, txtQuestion(), []string{silentServer(t), addr})
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if len(r.Answer) != 1 {
		t.Fatalf("want 1 record from the second server, got %d", len(r.Answer))
	}
}

func TestExchangeKeepsAnAnswerWhateverItsRcode(t *testing.T) {
	var asked atomic.Int32
	refusing := func(t *testing.T) string {
		pc, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		srv := &dns.Server{PacketConn: pc, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
			asked.Add(1)
			m := new(dns.Msg)
			m.SetRcode(req, dns.RcodeRefused)
			w.WriteMsg(m)
		})}
		started := make(chan struct{})
		srv.NotifyStartedFunc = func() { close(started) }
		go srv.ActivateAndServe()
		<-started
		t.Cleanup(func() { srv.Shutdown() })
		return pc.LocalAddr().String()
	}

	r, err := Exchange(context.Background(), &dns.Client{Timeout: time.Second}, txtQuestion(), []string{refusing(t), refusing(t)})
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if r.Rcode != dns.RcodeRefused {
		t.Errorf("want REFUSED, got %s", dns.RcodeToString[r.Rcode])
	}
	if n := asked.Load(); n != 1 {
		t.Errorf("want only the first server asked, got %d questions", n)
	}
}

// startTestDNSServer serves answers for example.com. on a local UDP port,
// and returns its address and a function giving the last question it
// received.
func startTestDNSServer(t *testing.T) (string, func() *dns.Msg) {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	answers := map[uint16]string{
		dns.TypeA:  "example.com. 300 IN A 192.0.2.1",
		dns.TypeMX: "example.com. 300 IN MX 10 mail.example.com.",
	}

	var last atomic.Pointer[dns.Msg]
	mux := dns.NewServeMux()
	mux.HandleFunc("example.com.", func(w dns.ResponseWriter, req *dns.Msg) {
		last.Store(req)

		m := new(dns.Msg)
		m.SetReply(req)
		if s, ok := answers[req.Question[0].Qtype]; ok {
			rr, _ := dns.NewRR(s)
			m.Answer = append(m.Answer, rr)
		}
		w.WriteMsg(m)
	})

	started := make(chan struct{})
	srv := &dns.Server{PacketConn: pc, Handler: mux, NotifyStartedFunc: func() { close(started) }}
	go srv.ActivateAndServe()
	<-started
	t.Cleanup(func() { srv.Shutdown() })

	return pc.LocalAddr().String(), last.Load
}

func TestQuestion(t *testing.T) {
	addr, last := startTestDNSServer(t)
	client := &dns.Client{Timeout: time.Second}

	r, err := Question(context.Background(), client, []string{addr}, "example.com.", dns.TypeA, QueryOptions{})
	if err != nil {
		t.Fatalf("Question: %v", err)
	}
	if len(r.Answer) != 1 || r.Answer[0].Header().Rrtype != dns.TypeA {
		t.Errorf("Question answer = %v; want one A record", r.Answer)
	}
	if last().RecursionDesired {
		t.Error("Question sent RD although Recursive is false")
	}
	if opt := last().IsEdns0(); opt == nil || !opt.Do() {
		t.Error("Question sent no EDNS0 record with the DO bit")
	}

	if _, err = Question(context.Background(), client, []string{addr}, "example.com.", dns.TypeA, QueryOptions{Recursive: true}); err != nil {
		t.Fatalf("Question: %v", err)
	}
	if !last().RecursionDesired {
		t.Error("Question did not send RD although Recursive is true")
	}
}

func TestMultiQuestion(t *testing.T) {
	addr, _ := startTestDNSServer(t)
	client := &dns.Client{Timeout: time.Second}

	r, err := MultiQuestion(context.Background(), client, []string{addr}, "example.com.", []uint16{dns.TypeA, dns.TypeTXT, dns.TypeMX}, QueryOptions{Recursive: true})
	if err != nil {
		t.Fatalf("MultiQuestion: %v", err)
	}
	if r.Question[0].Qtype != dns.TypeANY {
		t.Errorf("MultiQuestion question type = %s; want ANY", dns.TypeToString[r.Question[0].Qtype])
	}
	if len(r.Answer) != 2 {
		t.Errorf("MultiQuestion merged %d answers; want 2 (A and MX)", len(r.Answer))
	}

	// No type has an answer: the last response is still returned, as ANY.
	r, err = MultiQuestion(context.Background(), client, []string{addr}, "example.com.", []uint16{dns.TypeTXT}, QueryOptions{})
	if err != nil {
		t.Fatalf("MultiQuestion: %v", err)
	}
	if r == nil || len(r.Answer) != 0 || r.Question[0].Qtype != dns.TypeANY {
		t.Errorf("MultiQuestion with no answer = %v; want an empty ANY response", r)
	}

	r, err = MultiQuestion(context.Background(), client, []string{addr}, "example.com.", nil, QueryOptions{})
	if r != nil || err != nil {
		t.Errorf("MultiQuestion with no type = %v, %v; want nothing", r, err)
	}
}

// closedAddr returns a local UDP address nothing listens on.
func closedAddr(t *testing.T) string {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := pc.LocalAddr().String()
	pc.Close()
	return addr
}

func TestQuestionErrors(t *testing.T) {
	client := &dns.Client{Timeout: time.Second}

	if r, err := Question(context.Background(), client, []string{closedAddr(t)}, "example.com.", dns.TypeA, QueryOptions{}); err == nil {
		t.Errorf("Question to a closed port = %v; want an error", r)
	}

	addr, _ := startTestDNSServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := Question(ctx, client, []string{addr}, "example.com.", dns.TypeA, QueryOptions{}); err == nil {
		t.Errorf("Question with a canceled context = %v; want an error", r)
	}
}

func TestMultiQuestionError(t *testing.T) {
	client := &dns.Client{Timeout: time.Second}

	if r, err := MultiQuestion(context.Background(), client, []string{closedAddr(t)}, "example.com.", []uint16{dns.TypeA, dns.TypeMX}, QueryOptions{}); err == nil {
		t.Errorf("MultiQuestion to a closed port = %v; want an error", r)
	}
}
