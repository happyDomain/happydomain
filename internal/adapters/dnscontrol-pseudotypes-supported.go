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

package adapter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	dnscontrol "github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// capabilityNone stands for the pseudo-types no provider ever declares. They are
// read back from a zone, but never proposed to the user.
const capabilityNone = dnscontrol.Capability(-1)

// pseudoTypeAdapter tells how a pseudo-type happyDomain represents crosses the
// DNSControl boundary.
type pseudoTypeAdapter struct {
	// rrtype is the private use type code happyDomain gives to this type.
	rrtype uint16

	// capability is the one a provider must declare to accept this type, or
	// capabilityNone.
	capability dnscontrol.Capability

	// toRecord and toRecordConfig convert in both directions the types whose
	// rdata does not fit in a RecordConfig target alone. They are nil for the
	// bare target ones, which the generic conversions of DNSControl handle
	// correctly: see happydns.PseudoType.BareTarget.
	toRecord       func(rc *models.RecordConfig) (happydns.Record, error)
	toRecordConfig func(rr *dns.PrivateRR, origin string) (*models.RecordConfig, error)
}

// supportedPseudoTypes lists the DNSControl pseudo-types happyDomain is able to
// represent. Adding an entry here requires the matching registration in
// happydns.PseudoTypes, which the init() below checks.
var supportedPseudoTypes = map[string]*pseudoTypeAdapter{
	"ALIAS":     {rrtype: happydns.TypeALIAS, capability: dnscontrol.CanUseAlias},
	"AKAMAICDN": {rrtype: happydns.TypeAKAMAICDN, capability: dnscontrol.CanUseAKAMAICDN},

	// ANAME is how dnsmadeeasy, namedotcom and cnr spell an ALIAS on their
	// side: no capability stands for it, it is only ever read back.
	"ANAME": {rrtype: happydns.TypeANAME, capability: capabilityNone},

	"R53_ALIAS": {
		rrtype:         happydns.TypeR53ALIAS,
		capability:     dnscontrol.CanUseRoute53Alias,
		toRecord:       r53AliasToRecord,
		toRecordConfig: r53AliasToRecordConfig,
	},
	"AZURE_ALIAS": {
		rrtype:         happydns.TypeAZUREALIAS,
		capability:     dnscontrol.CanUseAzureAlias,
		toRecord:       azureAliasToRecord,
		toRecordConfig: azureAliasToRecordConfig,
	},
	"AKAMAITLC": {
		rrtype:         happydns.TypeAKAMAITLC,
		capability:     dnscontrol.CanUseAKAMAITLC,
		toRecord:       akamaiTLCToRecord,
		toRecordConfig: akamaiTLCToRecordConfig,
	},
}

// supportedPseudoTypesByRrtype indexes supportedPseudoTypes the other way
// around, to recognize the records happyDomain hands back.
var supportedPseudoTypesByRrtype = map[uint16]string{}

func init() {
	// A pseudo-type happyDomain builds records of, but does not know how to hand
	// back to DNSControl, would only fail when publishing a zone.
	for name := range happydns.PseudoTypes() {
		if _, ok := supportedPseudoTypes[name]; !ok {
			panic(fmt.Sprintf("pseudo-type %s is registered in the model but the DNSControl adapter ignores it", name))
		}
	}

	for name, adapter := range supportedPseudoTypes {
		supportedPseudoTypesByRrtype[adapter.rrtype] = name

		// These two invariants are what keeps a supported pseudo-type away from
		// the log.Fatalf of models.RecordConfig.ToRR(): either happyDomain owns
		// the conversion, or miekg/dns knows how to build the record.
		pt, ok := happydns.PseudoTypeByName(name)
		if !ok || pt.Rrtype != adapter.rrtype {
			panic(fmt.Sprintf("pseudo-type %s is supported by the DNSControl adapter but not registered in the model", name))
		}
		if dns.TypeToRR[adapter.rrtype] == nil {
			panic(fmt.Sprintf("pseudo-type %s has no constructor in dns.TypeToRR", name))
		}
		if _, known := dns.StringToType[name]; !known && (adapter.toRecord == nil || adapter.toRecordConfig == nil) {
			panic(fmt.Sprintf("pseudo-type %s is unknown to dns.StringToType, it needs both converters", name))
		}
	}
}

// isSupportedPseudoType reports whether happyDomain represents the given
// DNSControl pseudo-type.
func isSupportedPseudoType(rtype string) bool {
	_, ok := supportedPseudoTypes[rtype]
	return ok
}

// pseudoTypeCapabilities returns the "rr-<num>-<NAME>" capability strings for
// the pseudo-types the given provider declares supporting.
func pseudoTypeCapabilities(providerName string) (caps []string) {
	names := make([]string, 0, len(supportedPseudoTypes))
	for name := range supportedPseudoTypes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		adapter := supportedPseudoTypes[name]
		if adapter.capability == capabilityNone {
			continue
		}

		if dnscontrol.ProviderHasCapability(providerName, adapter.capability) {
			caps = append(caps, fmt.Sprintf("rr-%d-%s", adapter.rrtype, name))
		}
	}

	return
}

// recordFromRecordConfig converts a record read from a provider into the
// happyDomain representation.
func recordFromRecordConfig(rc *models.RecordConfig) (happydns.Record, error) {
	if adapter, ok := supportedPseudoTypes[rc.Type]; ok && adapter.toRecord != nil {
		return adapter.toRecord(rc)
	}

	if IsPseudoRecordType(rc.Type) && !isSupportedPseudoType(rc.Type) {
		// dropPseudoRecords is supposed to have removed those already. Refusing
		// here is what keeps the promise that ToRR() is never reached with a
		// type it would call log.Fatalf on.
		return nil, fmt.Errorf("%s is a pseudo-type happyDomain is unable to represent", rc.Type)
	}

	rr := rc.ToRR()
	if rr == nil {
		// ToRR() builds some of its records by parsing their zone file
		// representation, and discards the error when it fails.
		return nil, fmt.Errorf("unable to build a %s record from the provider answer", rc.Type)
	}

	// rc.ToRR() for modern types (DS, RP, …) returns the rtype wrapper (e.g.
	// *rtype.DS) rather than the canonical *dns.DS. When these are later passed
	// back through dnsrr.RRtoRC → DS.FromStruct, the type assertion on *dns.DS
	// fails. dns.Copy invokes the promoted copy() method from the embedded
	// *dns.DS, which returns the canonical type.
	return dns.Copy(rr), nil
}

// checkPseudoTypesSupported refuses the pseudo-types the given provider does not
// declare it handles.
//
// happyDomain does not run DNSControl's own capability check
// (pkg/normalize.capabilityCheck), and a provider's RecordAuditor audits the
// content of the records, not their type. The frontend only offers the kinds a
// provider advertises, but the API is reachable on its own.
func checkPseudoTypesSupported(providerName string, rrs []happydns.Record) error {
	for _, rr := range rrs {
		pt, isPseudo := happydns.PseudoTypeByRrtype(rr.Header().Rrtype)
		if !isPseudo {
			continue
		}

		adapter, known := supportedPseudoTypes[pt.Name]
		if !known || adapter.capability == capabilityNone ||
			!dnscontrol.ProviderHasCapability(providerName, adapter.capability) {
			return fmt.Errorf("%s records are not supported by this provider", pt.Name)
		}
	}

	return nil
}

// recordsFromRecordConfigs converts one side of a correction, leaving out the
// pseudo-types happyDomain is unable to represent, symmetrically on both sides.
func recordsFromRecordConfigs(rcs models.Records) (ret []happydns.Record, err error) {
	for _, rc := range dropPseudoRecords(rcs) {
		var rr happydns.Record

		rr, err = recordFromRecordConfig(rc)
		if err != nil {
			return nil, err
		}

		ret = append(ret, rr)
	}

	return
}

// pseudoRecordConfigFromRecord converts a pseudo-type record happyDomain holds
// into the RecordConfig DNSControl expects, carrying the fields no target can
// hold. It returns a nil RecordConfig for the records the generic conversion
// handles.
func pseudoRecordConfigFromRecord(rr happydns.Record, origin string) (*models.RecordConfig, error) {
	prr, ok := rr.(*dns.PrivateRR)
	if !ok {
		return nil, nil
	}

	name, known := supportedPseudoTypesByRrtype[prr.Hdr.Rrtype]
	if !known {
		return nil, fmt.Errorf("unable to publish a record of type %s: happyDomain doesn't know it", dns.TypeToString[prr.Hdr.Rrtype])
	}

	adapter := supportedPseudoTypes[name]
	if adapter.toRecordConfig == nil {
		return nil, nil
	}

	return adapter.toRecordConfig(prr, origin)
}

// newPseudoRecord builds the private use record standing for the given
// RecordConfig, with its header filled the way models.RecordConfig.ToRR() does.
func newPseudoRecord(rrtype uint16, rc *models.RecordConfig) *dns.PrivateRR {
	rr := dns.TypeToRR[rrtype]().(*dns.PrivateRR)

	rr.Hdr.Name = rc.NameFQDN + "."
	rr.Hdr.Rrtype = rrtype
	rr.Hdr.Class = dns.ClassINET
	rr.Hdr.Ttl = rc.TTL
	if rc.TTL == 0 {
		rr.Hdr.Ttl = models.DefaultTTL
	}

	return rr
}

// newPseudoRecordConfig builds the RecordConfig standing for the given private
// use record, target apart.
func newPseudoRecordConfig(rr *dns.PrivateRR, rtype, origin string) *models.RecordConfig {
	rc := &models.RecordConfig{
		Type: rtype,
		TTL:  rr.Hdr.Ttl,
	}

	rc.SetLabelFromFQDN(strings.TrimSuffix(rr.Hdr.Name, "."), strings.TrimSuffix(origin, "."))

	return rc
}

func r53AliasToRecord(rc *models.RecordConfig) (happydns.Record, error) {
	rr := newPseudoRecord(happydns.TypeR53ALIAS, rc)

	rdata := rr.Data.(*happydns.R53AliasRdata)
	rdata.Target = rc.GetTargetField()
	rdata.AType = rc.R53Alias["type"]
	rdata.ZoneID = rc.R53Alias["zone_id"]
	rdata.EvaluateTargetHealth = rc.R53Alias["evaluate_target_health"]

	return rr, nil
}

func r53AliasToRecordConfig(rr *dns.PrivateRR, origin string) (*models.RecordConfig, error) {
	rdata, ok := rr.Data.(*happydns.R53AliasRdata)
	if !ok {
		return nil, fmt.Errorf("unexpected rdata %T in a R53_ALIAS record", rr.Data)
	}

	rc := newPseudoRecordConfig(rr, "R53_ALIAS", origin)
	rc.R53Alias = map[string]string{
		"type":                   rdata.AType,
		"zone_id":                rdata.ZoneID,
		"evaluate_target_health": rdata.EvaluateTargetHealth,
	}

	return rc, rc.SetTarget(rdata.Target)
}

func azureAliasToRecord(rc *models.RecordConfig) (happydns.Record, error) {
	rr := newPseudoRecord(happydns.TypeAZUREALIAS, rc)

	rdata := rr.Data.(*happydns.AzureAliasRdata)
	rdata.Target = rc.GetTargetField()
	rdata.AType = rc.AzureAlias["type"]

	return rr, nil
}

func azureAliasToRecordConfig(rr *dns.PrivateRR, origin string) (*models.RecordConfig, error) {
	rdata, ok := rr.Data.(*happydns.AzureAliasRdata)
	if !ok {
		return nil, fmt.Errorf("unexpected rdata %T in an AZURE_ALIAS record", rr.Data)
	}

	rc := newPseudoRecordConfig(rr, "AZURE_ALIAS", origin)
	rc.AzureAlias = map[string]string{
		"type": rdata.AType,
	}

	return rc, rc.SetTarget(rdata.Target)
}

func akamaiTLCToRecord(rc *models.RecordConfig) (happydns.Record, error) {
	rr := newPseudoRecord(happydns.TypeAKAMAITLC, rc)

	rdata := rr.Data.(*happydns.AkamaiTLCRdata)
	rdata.Target = rc.GetTargetField()
	rdata.AnswerType = rc.AnswerType

	return rr, nil
}

func akamaiTLCToRecordConfig(rr *dns.PrivateRR, origin string) (*models.RecordConfig, error) {
	rdata, ok := rr.Data.(*happydns.AkamaiTLCRdata)
	if !ok {
		return nil, fmt.Errorf("unexpected rdata %T in an AKAMAITLC record", rr.Data)
	}

	rc := newPseudoRecordConfig(rr, "AKAMAITLC", origin)
	rc.AnswerType = rdata.AnswerType

	return rc, rc.SetTarget(rdata.Target)
}
