// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package adapter

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	dnscontrol "github.com/StackExchange/dnscontrol/v4/providers"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

func RegisterDNSControlProviderAdapter(creator happydns.ProviderCreatorFunc, infos happydns.ProviderInfos, registerFunc happydns.RegisterProviderFunc) {
	prvInstance := creator().(DNSControlConfigAdapter)
	infos.Capabilities = GetDNSControlProviderCapabilities(prvInstance)
	infos.HelpLink = "https://docs.dnscontrol.org/service-providers/providers/" + strings.ToLower(prvInstance.DNSControlName())

	registerFunc(creator, infos)
}

// GetProviderCapabilities lists available capabilities for the given Provider.
func GetDNSControlProviderCapabilities(prvd DNSControlConfigAdapter) (caps []string) {
	// Features
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.DocCreateDomains) {
		caps = append(caps, "CreateDomain")
	}
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.CanGetZones) {
		caps = append(caps, "ListDomains")
	}

	// Compatible RR
	for _, v := range []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeCNAME, dns.TypeMX, dns.TypeNS, dns.TypeTXT} {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", v, dns.TypeToString[v]))
	}
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.CanUseSOA) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeSOA, dns.TypeToString[dns.TypeSOA]))
	}
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.CanUseCAA) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeCAA, dns.TypeToString[dns.TypeCAA]))
	}
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.CanUseDS) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeDS, dns.TypeToString[dns.TypeDS]))
	}
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.CanUseNAPTR) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeNAPTR, dns.TypeToString[dns.TypeNAPTR]))
	}
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.CanUseOPENPGPKEY) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeOPENPGPKEY, dns.TypeToString[dns.TypeOPENPGPKEY]))
	}
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.CanUsePTR) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypePTR, dns.TypeToString[dns.TypePTR]))
	}
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.CanUseSRV) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeSRV, dns.TypeToString[dns.TypeSRV]))
	}
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.CanUseSSHFP) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeSSHFP, dns.TypeToString[dns.TypeSSHFP]))
	}
	if dnscontrol.ProviderHasCapability(prvd.DNSControlName(), dnscontrol.CanUseTLSA) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeTLSA, dns.TypeToString[dns.TypeTLSA]))
	}

	return
}

type DNSControlConfigAdapter interface {
	DNSControlName() string
	ToDNSControlConfig() (map[string]string, error)
}

func NewDNSControlProviderAdapter(configAdapter DNSControlConfigAdapter) (happydns.ProviderActuator, error) {
	config, err := configAdapter.ToDNSControlConfig()
	if err != nil {
		return nil, err
	}

	provider, err := dnscontrol.CreateDNSProvider(configAdapter.DNSControlName(), config, nil)
	if err != nil {
		return nil, err
	}

	var auditor dnscontrol.RecordAuditor
	if p, ok := dnscontrol.DNSProviderTypes[configAdapter.DNSControlName()]; ok && p.RecordAuditor != nil {
		auditor = p.RecordAuditor
	}

	return &DNSControlAdapterNSProvider{provider, auditor}, nil
}

type DNSControlAdapterNSProvider struct {
	DNSServiceProvider dnscontrol.DNSServiceProvider
	RecordAuditor      dnscontrol.RecordAuditor
}

func (p *DNSControlAdapterNSProvider) CanListZones() bool {
	_, ok := p.DNSServiceProvider.(dnscontrol.ZoneLister)
	return ok
}

func (p *DNSControlAdapterNSProvider) CanCreateDomain() bool {
	_, ok := p.DNSServiceProvider.(dnscontrol.ZoneCreator)
	return ok
}

func (p *DNSControlAdapterNSProvider) GetZoneRecords(domain string) (ret []happydns.Record, err error) {
	var records models.Records

	defer func() {
		if a := recover(); a != nil {
			err = fmt.Errorf("%s", a)
		}
	}()

	records, err = p.DNSServiceProvider.GetZoneRecords(strings.TrimSuffix(domain, "."), nil)
	if err != nil {
		return
	}

	for _, rec := range records {
		ret = append(ret, rec.ToRR())
	}

	return
}

func (p *DNSControlAdapterNSProvider) GetZoneCorrections(domain string, rrs []happydns.Record) (ret []*happydns.FCorrection, err error) {
	var dc *models.DomainConfig
	dc, err = NewDNSControlDomainConfig(strings.TrimSuffix(domain, "."), rrs)
	if err != nil {
		return
	}

	errs := p.RecordAuditor(dc.Records)
	if errs != nil {
		err = fmt.Errorf("some records are incompatibles with this NS provider: %w. Please fix those errors and retry.", errors.Join(errs...))
		return
	}

	defer func() {
		if a := recover(); a != nil {
			err = fmt.Errorf("%s", a)
		}
	}()

	// Retrieve current zone
	var records models.Records
	records, err = p.DNSServiceProvider.GetZoneRecords(strings.TrimSuffix(domain, "."), nil)
	if err != nil {
		return nil, err
	}

	// Compute needed corrections
	var corrections []*models.Correction
	corrections, err = p.DNSServiceProvider.GetZoneRecordsCorrections(dc, records)
	if err != nil {
		return nil, err
	}

	ret = make([]*happydns.FCorrection, len(corrections))
	for i, correction := range corrections {
		id := sha256.Sum224([]byte(correction.Msg))

		ret[i] = &happydns.FCorrection{
			F: correction.F,
			Correction: happydns.Correction{
				Id:   happydns.Identifier(id[:]),
				Msg:  correction.Msg,
				Kind: DNSControlCorrectionKindFromMessage(correction.Msg),
			},
		}
	}

	return ret, nil
}

func (p *DNSControlAdapterNSProvider) CreateDomain(fqdn string) error {
	zc, ok := p.DNSServiceProvider.(dnscontrol.ZoneCreator)
	if !ok {
		return fmt.Errorf("Provider doesn't support domain creation.")
	}

	return zc.EnsureZoneExists(strings.TrimSuffix(fqdn, "."))
}

func (p *DNSControlAdapterNSProvider) ListZones() ([]string, error) {
	zl, ok := p.DNSServiceProvider.(dnscontrol.ZoneLister)
	if !ok {
		return nil, fmt.Errorf("Provider doesn't support domain listing.")
	}

	return zl.ListZones()
}
