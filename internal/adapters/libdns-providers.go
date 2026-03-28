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
	"context"
	"fmt"
	"strings"

	"github.com/libdns/libdns"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// LibdnsConfigAdapter is an interface that provider configurations must implement
// to work with libdns. It allows retrieving the underlying libdns provider instance.
type LibdnsConfigAdapter interface {
	// LibdnsProvider returns the underlying libdns provider instance.
	// The returned value must implement at least libdns.RecordGetter.
	LibdnsProvider() any
}

// RegisterLibdnsProviderAdapter registers a DNS provider that uses libdns as its backend.
// It automatically populates the provider's capabilities by checking which libdns
// interfaces the provider implements.
func RegisterLibdnsProviderAdapter(creator happydns.ProviderCreatorFunc, infos happydns.ProviderInfos, registerFunc happydns.RegisterProviderFunc) {
	prvInstance := creator().(LibdnsConfigAdapter)
	infos.Capabilities = append(infos.Capabilities, GetLibdnsProviderCapabilities(prvInstance)...)

	registerFunc(creator, infos)
}

// GetLibdnsProviderCapabilities checks which libdns interfaces the provider implements
// and returns the corresponding capability strings. Since libdns providers are type-agnostic,
// common record types are declared for all providers.
func GetLibdnsProviderCapabilities(prvd LibdnsConfigAdapter) (caps []string) {
	p := prvd.LibdnsProvider()

	if _, ok := p.(libdns.ZoneLister); ok {
		caps = append(caps, "ListDomains")
	}

	// libdns providers are type-agnostic, so declare support for common RR types.
	for _, v := range []uint16{
		dns.TypeA, dns.TypeAAAA, dns.TypeCNAME, dns.TypeMX,
		dns.TypeNS, dns.TypeTXT, dns.TypeSRV, dns.TypeCAA,
		dns.TypePTR,
	} {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", v, dns.TypeToString[v]))
	}

	return
}

// NewLibdnsProviderAdapter creates a new provider actuator instance from a libdns configuration.
// It discovers the provider's capabilities by checking which libdns interfaces it implements.
// The provider must implement at least libdns.RecordGetter.
func NewLibdnsProviderAdapter(configAdapter LibdnsConfigAdapter) (happydns.ProviderActuator, error) {
	p := configAdapter.LibdnsProvider()

	adapter := &LibdnsAdapterNSProvider{
		provider: p,
	}

	if g, ok := p.(libdns.RecordGetter); ok {
		adapter.getter = g
	} else {
		return nil, fmt.Errorf("libdns provider must implement RecordGetter")
	}

	if s, ok := p.(libdns.RecordSetter); ok {
		adapter.setter = s
	}
	if a, ok := p.(libdns.RecordAppender); ok {
		adapter.appender = a
	}
	if d, ok := p.(libdns.RecordDeleter); ok {
		adapter.deleter = d
	}
	if z, ok := p.(libdns.ZoneLister); ok {
		adapter.zoneLister = z
	}

	return adapter, nil
}

// LibdnsAdapterNSProvider wraps a libdns provider to implement the happyDomain ProviderActuator interface.
type LibdnsAdapterNSProvider struct {
	provider   any
	getter     libdns.RecordGetter
	setter     libdns.RecordSetter
	appender   libdns.RecordAppender
	deleter    libdns.RecordDeleter
	zoneLister libdns.ZoneLister
}

// normalizeZone ensures the zone name has a trailing dot (FQDN format expected by libdns).
func normalizeZone(domain string) string {
	zone := strings.TrimSuffix(domain, ".")
	return zone + "."
}

// CanListZones checks if the provider supports listing zones.
func (p *LibdnsAdapterNSProvider) CanListZones() bool {
	return p.zoneLister != nil
}

// CanCreateDomain returns false since libdns has no zone creation interface.
func (p *LibdnsAdapterNSProvider) CanCreateDomain() bool {
	return false
}

// CreateDomain is not supported by libdns providers.
func (p *LibdnsAdapterNSProvider) CreateDomain(fqdn string) error {
	return fmt.Errorf("libdns provider does not support domain creation")
}

// ListZones retrieves the list of all zones managed by this provider.
func (p *LibdnsAdapterNSProvider) ListZones() ([]string, error) {
	if p.zoneLister == nil {
		return nil, fmt.Errorf("libdns provider does not support zone listing")
	}

	zones, err := p.zoneLister.ListZones(context.TODO())
	if err != nil {
		return nil, err
	}

	result := make([]string, len(zones))
	for i, z := range zones {
		result[i] = z.Name
	}
	return result, nil
}

// GetZoneRecords retrieves all DNS records for the specified domain from the provider.
func (p *LibdnsAdapterNSProvider) GetZoneRecords(domain string) ([]happydns.Record, error) {
	zone := normalizeZone(domain)

	recs, err := p.getter.GetRecords(context.TODO(), zone)
	if err != nil {
		return nil, err
	}

	return libdnsRecordsToHappyDNS(recs, zone)
}

// GetZoneCorrections compares desired records against the current zone state and returns
// the changes needed to synchronize them. It uses the DNSControl diff engine to compute
// the diff, then creates correction functions that call the libdns provider's API.
func (p *LibdnsAdapterNSProvider) GetZoneCorrections(domain string, wantedRecords []happydns.Record) ([]*happydns.Correction, int, error) {
	zone := normalizeZone(domain)

	// Step 1: Fetch current records from the provider.
	currentLibdnsRecs, err := p.getter.GetRecords(context.TODO(), zone)
	if err != nil {
		return nil, 0, fmt.Errorf("unable to get current zone records: %w", err)
	}

	currentRecords, err := libdnsRecordsToHappyDNS(currentLibdnsRecs, zone)
	if err != nil {
		return nil, 0, fmt.Errorf("unable to convert current zone records: %w", err)
	}

	// Step 2: Compute diff using existing DNSControl diff engine.
	diffs, nbDiffs, err := DNSControlDiffByRecord(currentRecords, wantedRecords, domain)
	if err != nil {
		return nil, nbDiffs, fmt.Errorf("unable to compute zone diff: %w", err)
	}

	// Build a lookup from happydns Record string → original libdns records (with ProviderData).
	// This ensures delete operations use the provider's record IDs.
	libdnsRecordsByKey := make(map[string][]libdns.Record)
	for _, rec := range currentLibdnsRecs {
		rr := rec.RR()
		key := fmt.Sprintf("%s\t%s\t%s", rr.Name, rr.Type, rr.Data)
		libdnsRecordsByKey[key] = append(libdnsRecordsByKey[key], rec)
	}

	// Step 3: Create corrections with executable F closures.
	corrections := make([]*happydns.Correction, len(diffs))
	for i, diff := range diffs {
		corrections[i] = &happydns.Correction{
			Id:         diff.Id,
			Msg:        diff.Msg,
			Kind:       diff.Kind,
			OldRecords: diff.OldRecords,
			NewRecords: diff.NewRecords,
		}

		corrections[i].F = p.makeCorrectionFunc(zone, diff, libdnsRecordsByKey)
	}

	return corrections, nbDiffs, nil
}

// makeCorrectionFunc creates an executable function for a single correction.
func (p *LibdnsAdapterNSProvider) makeCorrectionFunc(
	zone string,
	diff *happydns.Correction,
	libdnsRecordsByKey map[string][]libdns.Record,
) func() error {
	kind := diff.Kind

	// Resolve old records to their original libdns Records (with ProviderData).
	oldRecs := p.resolveOriginalRecords(diff.OldRecords, zone, libdnsRecordsByKey)
	newRecs := happyDNSRecordsToLibdnsRecords(diff.NewRecords, zone)

	// If we have both appender and deleter, use granular operations.
	if p.appender != nil && p.deleter != nil {
		return func() error {
			ctx := context.TODO()
			switch kind {
			case happydns.CorrectionKindAddition:
				_, err := p.appender.AppendRecords(ctx, zone, newRecs)
				return err
			case happydns.CorrectionKindDeletion:
				_, err := p.deleter.DeleteRecords(ctx, zone, oldRecs)
				return err
			case happydns.CorrectionKindUpdate:
				if _, err := p.deleter.DeleteRecords(ctx, zone, oldRecs); err != nil {
					return fmt.Errorf("delete phase of update: %w", err)
				}
				_, err := p.appender.AppendRecords(ctx, zone, newRecs)
				if err != nil {
					return fmt.Errorf("append phase of update: %w", err)
				}
				return nil
			}
			return nil
		}
	}

	// Fallback: use SetRecords if available.
	if p.setter != nil {
		return func() error {
			ctx := context.TODO()
			switch kind {
			case happydns.CorrectionKindAddition:
				// SetRecords with the new records will add them to the zone
				// for their (name, type) pair.
				_, err := p.setter.SetRecords(ctx, zone, newRecs)
				return err
			case happydns.CorrectionKindDeletion:
				// To delete, we need to set the (name, type) pair to empty.
				// DeleteRecords would be better, but we only have SetRecords.
				// Use DeleteRecords-style wildcard via setter: set with empty set
				// is not directly possible with SetRecords semantics.
				// Fall through to delete if we have deleter, otherwise error.
				return fmt.Errorf("cannot delete records: provider only supports SetRecords, not DeleteRecords")
			case happydns.CorrectionKindUpdate:
				// SetRecords replaces all records for the (name, type) pair.
				_, err := p.setter.SetRecords(ctx, zone, newRecs)
				return err
			}
			return nil
		}
	}

	return func() error {
		return fmt.Errorf("libdns provider does not support record modification")
	}
}

// resolveOriginalRecords tries to find the original libdns Records (with ProviderData)
// for the given happydns Records. This ensures that delete operations use the provider's
// record identifiers.
func (p *LibdnsAdapterNSProvider) resolveOriginalRecords(
	records []happydns.Record,
	zone string,
	libdnsRecordsByKey map[string][]libdns.Record,
) []libdns.Record {
	result := make([]libdns.Record, 0, len(records))
	for _, rec := range records {
		rr := happyDNSRecordToLibdnsRR(rec, zone)
		key := fmt.Sprintf("%s\t%s\t%s", rr.Name, rr.Type, rr.Data)

		if originals, ok := libdnsRecordsByKey[key]; ok && len(originals) > 0 {
			// Use the original record and consume it from the map.
			result = append(result, originals[0])
			libdnsRecordsByKey[key] = originals[1:]
		} else {
			// Fallback: use the converted RR (without ProviderData).
			result = append(result, rr)
		}
	}
	return result
}

// happyDNSRecordsToLibdnsRecords converts happydns Records to libdns Records (the interface).
func happyDNSRecordsToLibdnsRecords(rrs []happydns.Record, zone string) []libdns.Record {
	result := make([]libdns.Record, len(rrs))
	for i, rr := range rrs {
		result[i] = happyDNSRecordToLibdnsRR(rr, zone)
	}
	return result
}
