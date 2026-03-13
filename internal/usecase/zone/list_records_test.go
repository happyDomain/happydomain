package zone_test

import (
	"strings"
	"testing"

	"github.com/miekg/dns"

	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/services/providers/google"
)

func TestListRecords_SPFMerge_GSuiteOnly(t *testing.T) {
	gsuite := &google.GSuite{}
	gsuite.Initialize()

	domain := &happydns.Domain{DomainName: "example.com."}
	zone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{DefaultTTL: 3600},
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": {
				{
					ServiceMeta: happydns.ServiceMeta{Domain: ""},
					Service:     gsuite,
				},
			},
		},
	}

	uc := zoneUC.NewListRecordsUsecase(serviceUC.NewListRecordsUsecase())
	records, err := uc.List(domain, zone)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should have MX records + 1 merged SPF TXT record
	spfCount := 0
	for _, rr := range records {
		if txt, ok := rr.(*happydns.TXT); ok && strings.HasPrefix(txt.Txt, "v=spf1") {
			spfCount++
			if !strings.Contains(txt.Txt, "include:_spf.google.com") {
				t.Errorf("SPF record should contain Google include, got %q", txt.Txt)
			}
			if !strings.Contains(txt.Txt, "~all") {
				t.Errorf("SPF record should contain ~all, got %q", txt.Txt)
			}
		}
	}

	if spfCount != 1 {
		t.Errorf("expected 1 SPF record, got %d", spfCount)
	}
}

func TestListRecords_SPFMerge_SPFOnly(t *testing.T) {
	spfSvc := &svcs.SPF{
		Record: &happydns.TXT{
			Hdr: dns.RR_Header{Rrtype: dns.TypeTXT, Class: dns.ClassINET},
			Txt: "v=spf1 ip4:203.0.113.0/24 ~all",
		},
	}

	domain := &happydns.Domain{DomainName: "example.com."}
	zone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{DefaultTTL: 3600},
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": {
				{
					ServiceMeta: happydns.ServiceMeta{Domain: ""},
					Service:     spfSvc,
				},
			},
		},
	}

	uc := zoneUC.NewListRecordsUsecase(serviceUC.NewListRecordsUsecase())
	records, err := uc.List(domain, zone)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	spfCount := 0
	for _, rr := range records {
		if txt, ok := rr.(*happydns.TXT); ok && strings.HasPrefix(txt.Txt, "v=spf1") {
			spfCount++
			if !strings.Contains(txt.Txt, "ip4:203.0.113.0/24") {
				t.Errorf("SPF record should contain ip4 directive, got %q", txt.Txt)
			}
		}
	}

	if spfCount != 1 {
		t.Errorf("expected 1 SPF record, got %d", spfCount)
	}
}

func TestListRecords_SPFMerge_GSuiteAndSPF(t *testing.T) {
	gsuite := &google.GSuite{}
	gsuite.Initialize()

	spfSvc := &svcs.SPF{
		Record: &happydns.TXT{
			Hdr: dns.RR_Header{Rrtype: dns.TypeTXT, Class: dns.ClassINET},
			Txt: "v=spf1 ip4:203.0.113.0/24 -all",
		},
	}

	domain := &happydns.Domain{DomainName: "example.com."}
	zone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{DefaultTTL: 3600},
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": {
				{
					ServiceMeta: happydns.ServiceMeta{Domain: ""},
					Service:     gsuite,
				},
				{
					ServiceMeta: happydns.ServiceMeta{Domain: ""},
					Service:     spfSvc,
				},
			},
		},
	}

	uc := zoneUC.NewListRecordsUsecase(serviceUC.NewListRecordsUsecase())
	records, err := uc.List(domain, zone)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Count SPF records — should be exactly 1 merged record
	spfCount := 0
	var spfTxt string
	for _, rr := range records {
		if txt, ok := rr.(*happydns.TXT); ok && strings.HasPrefix(txt.Txt, "v=spf1") {
			spfCount++
			spfTxt = txt.Txt
		}
	}

	if spfCount != 1 {
		t.Fatalf("expected 1 SPF record, got %d", spfCount)
	}

	// Merged record should contain both directives
	if !strings.Contains(spfTxt, "include:_spf.google.com") {
		t.Errorf("merged SPF should contain Google include, got %q", spfTxt)
	}
	if !strings.Contains(spfTxt, "ip4:203.0.113.0/24") {
		t.Errorf("merged SPF should contain ip4 directive, got %q", spfTxt)
	}

	// Strictest policy wins: -all from SPF service > ~all from GSuite
	if !strings.HasSuffix(spfTxt, "-all") {
		t.Errorf("merged SPF should end with -all (strictest), got %q", spfTxt)
	}
}
