package zone_test

import (
	"testing"

	"github.com/miekg/dns"

	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	_ "git.happydns.org/happyDomain/services/abstract"
)

func Test_AddRecordSimple(t *testing.T) {
	uc := zoneUC.NewAddRecordUsecase(serviceUC.NewListRecordsUsecase())

	origin := "happydns.org."

	zone := happydns.Zone{
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": []*happydns.Service{},
		},
	}

	txtStr := "v=spf1 include:_spf.example.com. ~all"
	rr, err := dns.NewRR("test 3600 IN TXT \"" + txtStr + "\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	txt := happydns.NewTXT(rr.(*dns.TXT))

	err = uc.Add(&zone, origin, txt)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services[""]) != 0 {
		t.Fatalf("expected 0 service got %d", len(zone.Services[""]))
	}
	if len(zone.Services["test"]) != 1 {
		t.Fatalf("expected 1 service got %d", len(zone.Services["test"]))
	}
}

func Test_AddRecordDoubleMX(t *testing.T) {
	uc := zoneUC.NewAddRecordUsecase(serviceUC.NewListRecordsUsecase())

	origin := "happydns.org."

	zone := happydns.Zone{
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": []*happydns.Service{},
		},
	}

	mx1, err := dns.NewRR("test." + origin + " 3600 IN MX 10 mx1")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	err = uc.Add(&zone, origin, mx1)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services["test"]) != 1 {
		t.Fatalf("expected 1 service got %d", len(zone.Services["test"]))
	}

	mx2, err := dns.NewRR("test." + origin + " 3600 IN MX 42 mx2")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	err = uc.Add(&zone, origin, mx2)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services["test"]) != 1 {
		t.Fatalf("expected 1 service got %d", len(zone.Services["test"]))
	}

	mxs, ok := zone.Services["test"][0].Service.(*svcs.MXs)
	if !ok {
		t.Fatalf("expected svcs.MXs got %T", zone.Services["test"][0].Service)
	}

	if len(mxs.MXs) != 2 {
		t.Fatalf("expected 2 MX got %d", len(mxs.MXs))
	}
}

func Test_AddRecordDouble(t *testing.T) {
	uc := zoneUC.NewAddRecordUsecase(serviceUC.NewListRecordsUsecase())

	origin := "happydns.org."

	zone := happydns.Zone{
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": []*happydns.Service{},
		},
	}

	mx1, err := dns.NewRR("test 3600 IN MX 10 mx1")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	err = uc.Add(&zone, origin, mx1)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services["test"]) != 1 {
		t.Fatalf("expected 1 service got %d", len(zone.Services["test"]))
	}

	txtStr := "v=spf1 include:_spf.example.com. ~all"
	spf2, err := dns.NewRR("test 3600 IN TXT \"" + txtStr + "\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	err = uc.Add(&zone, origin, spf2)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services["test"]) != 2 {
		t.Fatalf("expected 2 service got %d", len(zone.Services["test"]))
	}

	if _, ok := zone.Services["test"][0].Service.(*svcs.MXs); !ok {
		t.Fatalf("expected svcs.MXs got %T", zone.Services["test"][0].Service)
	}

	if _, ok := zone.Services["test"][1].Service.(*svcs.SPF); !ok {
		t.Fatalf("expected svcs.SPF got %T", zone.Services["test"][1].Service)
	}
}

func Test_AddRecordDoubleA(t *testing.T) {
	uc := zoneUC.NewAddRecordUsecase(serviceUC.NewListRecordsUsecase())

	origin := "happydns.org."

	zone := happydns.Zone{
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": []*happydns.Service{},
		},
	}

	a1, err := dns.NewRR("test." + origin + " 3600 IN A 127.0.0.1")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	err = uc.Add(&zone, origin, a1)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services["test"]) != 1 {
		t.Fatalf("expected 1 service got %d", len(zone.Services["test"]))
	}

	aaaa1, err := dns.NewRR("test." + origin + " 3600 IN AAAA ::1")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	err = uc.Add(&zone, origin, aaaa1)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services["test"]) != 1 {
		t.Fatalf("expected 1 service got %d", len(zone.Services["test"]))
	}

	a2, err := dns.NewRR("test." + origin + " 3600 IN A 127.0.0.2")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	err = uc.Add(&zone, origin, a2)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services["test"]) != 3 {
		t.Fatalf("expected 3 service got %d", len(zone.Services["test"]))
	}

	_, ok := zone.Services["test"][0].Service.(*svcs.Orphan)
	if !ok {
		t.Fatalf("expected svcs.Orphan got %T", zone.Services["test"][0].Service)
	}
}

func Test_AddRecordRoot(t *testing.T) {
	uc := zoneUC.NewAddRecordUsecase(serviceUC.NewListRecordsUsecase())

	origin := "happydns.org."

	zone := happydns.Zone{
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": []*happydns.Service{},
		},
	}

	a1, err := dns.NewRR(origin + " 3600 IN A 127.0.0.1")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	err = uc.Add(&zone, origin, a1)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services[""]) != 1 {
		t.Fatalf("expected 1 service got %d", len(zone.Services["test"]))
	}

	aaaa1, err := dns.NewRR(origin + " 3600 IN AAAA ::1")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	err = uc.Add(&zone, origin, aaaa1)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services[""]) != 1 {
		t.Fatalf("expected 1 service got %d", len(zone.Services["test"]))
	}

	a2, err := dns.NewRR(origin + " 3600 IN A 127.0.0.2")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	err = uc.Add(&zone, origin, a2)
	if err != nil {
		t.Fatalf("unexpected AddRecord error: %v", err)
	}

	if len(zone.Services[""]) != 3 {
		t.Fatalf("expected 3 service got %d", len(zone.Services["test"]))
	}

	_, ok := zone.Services[""][0].Service.(*svcs.Orphan)
	if !ok {
		t.Fatalf("expected svcs.Orphan got %T", zone.Services["test"][0].Service)
	}
}
