package service_test

import (
	"strings"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

func TestListOneRecord(t *testing.T) {
	uc := service.NewListRecordsUsecase()

	origin := "happydns.org."
	var defaultTTL uint32 = 1234

	txtStr := "v=spf1 include:_spf.example.com. ~all"
	rr, err := dns.NewRR("test." + origin + " 3600 IN TXT \"" + txtStr + "\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	txt := happydns.NewTXT(rr.(*dns.TXT))

	s, _, err := svcs.AnalyzeZone(origin, []happydns.Record{txt})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}

	if len(s["test"]) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(s["test"]))
	}

	records, err := uc.List(s["test"][0], origin, defaultTTL)
	if err != nil {
		t.Fatalf("uc.List failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	if records[0].Header().Name != "test."+origin {
		t.Fatalf("Bad domain name: expected \"test.%s\", got %q", origin, records[0].Header().Name)
	}

	if records[0].Header().Ttl != 3600 {
		t.Fatalf("Bad TTL: expected 3600, got %d", records[0].Header().Ttl)
	}

	if !strings.HasSuffix(records[0].String(), rr.String()) {
		t.Fatalf("Bad record: expected %q, got %q", rr.String(), records[0].String())
	}
}

func TestListRecordDefaultTTL(t *testing.T) {
	uc := service.NewListRecordsUsecase()

	origin := "happydns.org."
	var defaultTTL uint32 = 1234

	txtStr := "v=spf1 include:_spf.example.com. ~all"
	rr, err := dns.NewRR("test." + origin + " 0 IN TXT \"" + txtStr + "\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	txt := happydns.NewTXT(rr.(*dns.TXT))

	s, _, err := svcs.AnalyzeZone(origin, []happydns.Record{txt})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	records, err := uc.List(s["test"][0], origin, defaultTTL)
	if err != nil {
		t.Fatalf("uc.List failed: %v", err)
	}

	if records[0].Header().Ttl != defaultTTL {
		t.Fatalf("Bad TTL: expected %d, got %d", defaultTTL, records[0].Header().Ttl)
	}
}

func TestListRecordRelative(t *testing.T) {
	uc := service.NewListRecordsUsecase()

	var defaultTTL uint32 = 1234

	txtStr := "v=spf1 include:_spf.example.com. ~all"
	rr, err := dns.NewRR("test 3600 IN TXT \"" + txtStr + "\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	txt := happydns.NewTXT(rr.(*dns.TXT))

	s, _, err := svcs.AnalyzeZone("", []happydns.Record{txt})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	records, err := uc.List(s["test"][0], "", defaultTTL)
	if err != nil {
		t.Fatalf("uc.List failed: %v", err)
	}

	if records[0].Header().Name != "test" {
		t.Fatalf("Bad domain name: expected %q, got %q", "test", records[0].Header().Name)
	}
}
