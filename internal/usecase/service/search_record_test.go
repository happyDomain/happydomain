package service_test

import (
	"strings"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

func TestExistsInService(t *testing.T) {
	uc := service.NewSearchRecordUsecase(service.NewListRecordsUsecase())

	origin := "happydns.org."

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

	txt2 := helpers.RRRelative(txt, origin).(*happydns.TXT)

	exists, err := uc.ExistsInService(s["test"][0], txt2)
	if err != nil {
		t.Fatalf("ExistsInService failed: %v", err)
	}

	if !exists {
		t.Fatalf("Expected found service, got not found")
	}
}

func TestNotExistsInService(t *testing.T) {
	uc := service.NewSearchRecordUsecase(service.NewListRecordsUsecase())

	origin := "happydns.org."

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

	rr2, err := dns.NewRR("test." + origin + " 3600 IN TXT \"autre test\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	txt2 := helpers.RRRelative(happydns.NewTXT(rr2.(*dns.TXT)), origin).(*happydns.TXT)

	exists, err := uc.ExistsInService(s["test"][0], txt2)
	if err != nil {
		t.Fatalf("ExistsInService failed: %v", err)
	}

	if exists {
		t.Fatalf("Expected not found service, got found")
	}
}

func TestExistsInRelativeService(t *testing.T) {
	uc := service.NewSearchRecordUsecase(service.NewListRecordsUsecase())

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

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}

	if len(s["test"]) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(s["test"]))
	}

	txt.Header().Name = strings.TrimSuffix(txt.Header().Name, ".")

	exists, err := uc.ExistsInService(s["test"][0], txt)
	if err != nil {
		t.Fatalf("ExistsInService failed: %v", err)
	}

	if !exists {
		t.Fatalf("Expected found service, got not found")
	}
}
