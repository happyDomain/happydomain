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

	r, err := Question(context.Background(), client, addr, "example.com.", dns.TypeA, QueryOptions{})
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

	if _, err = Question(context.Background(), client, addr, "example.com.", dns.TypeA, QueryOptions{Recursive: true}); err != nil {
		t.Fatalf("Question: %v", err)
	}
	if !last().RecursionDesired {
		t.Error("Question did not send RD although Recursive is true")
	}
}

func TestMultiQuestion(t *testing.T) {
	addr, _ := startTestDNSServer(t)
	client := &dns.Client{Timeout: time.Second}

	r, err := MultiQuestion(context.Background(), client, addr, "example.com.", []uint16{dns.TypeA, dns.TypeTXT, dns.TypeMX}, QueryOptions{Recursive: true})
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
	r, err = MultiQuestion(context.Background(), client, addr, "example.com.", []uint16{dns.TypeTXT}, QueryOptions{})
	if err != nil {
		t.Fatalf("MultiQuestion: %v", err)
	}
	if r == nil || len(r.Answer) != 0 || r.Question[0].Qtype != dns.TypeANY {
		t.Errorf("MultiQuestion with no answer = %v; want an empty ANY response", r)
	}

	r, err = MultiQuestion(context.Background(), client, addr, "example.com.", nil, QueryOptions{})
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

	if r, err := Question(context.Background(), client, closedAddr(t), "example.com.", dns.TypeA, QueryOptions{}); err == nil {
		t.Errorf("Question to a closed port = %v; want an error", r)
	}

	addr, _ := startTestDNSServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := Question(ctx, client, addr, "example.com.", dns.TypeA, QueryOptions{}); err == nil {
		t.Errorf("Question with a canceled context = %v; want an error", r)
	}
}

func TestMultiQuestionError(t *testing.T) {
	client := &dns.Client{Timeout: time.Second}

	if r, err := MultiQuestion(context.Background(), client, closedAddr(t), "example.com.", []uint16{dns.TypeA, dns.TypeMX}, QueryOptions{}); err == nil {
		t.Errorf("MultiQuestion to a closed port = %v; want an error", r)
	}
}
