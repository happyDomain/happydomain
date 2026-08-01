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

package happydns

import (
	"errors"
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

// DNSControl calls pseudo-types the record types it handles that have no
// counterpart in the DNS wire format: ALIAS and friends, defined either by its
// core or by a provider for its own needs.
//
// happyDomain represents them as private use resource records (RFC 6895 § 3.1),
// which is the only way to obtain something that satisfies dns.RR: that
// interface has unexported methods, so no type declared outside miekg/dns can
// implement it, but *dns.PrivateRR can carry any rdata we define.
//
// Only the types DNSControl spells in upper case can be registered: dns.
// PrivateHandle upcases the name it is given, so a NETLIFYv6 would come back as
// NETLIFYV6, a name DNSControl no longer recognizes. That leaves out NETLIFY
// and NETLIFYv6, which the netlify provider ignores anyway.
//
// The numeric values below are persisted (Hdr.Rrtype, in the JSON of the
// services holding such a record) and exposed to the frontend (the
// "rr-<num>-<NAME>" provider capabilities, ServiceInfos.RecordTypes,
// ServiceRestrictions.NeedTypes and the generated web/src/lib/dns_rr.ts). They
// must never change: 65280-65299 is reserved for them.
const (
	TypeALIAS      uint16 = 65280
	TypeANAME      uint16 = 65281
	TypeR53ALIAS   uint16 = 65282
	TypeAZUREALIAS uint16 = 65283
	TypeAKAMAICDN  uint16 = 65284
	TypeAKAMAITLC  uint16 = 65285
)

// ErrNoWireFormat is returned when something tries to put a pseudo-type on the
// wire. These records only exist in the API of a provider, never in a DNS
// message.
var ErrNoWireFormat = errors.New("this record type is a pseudo-type: it has no wire format")

// TargetRdata is implemented by the pseudo-type rdata carrying a target domain
// name, so that the callers relativizing or absolutizing a zone can handle them
// all the same way.
type TargetRdata interface {
	GetTarget() string
	SetTarget(string)
}

// PseudoType describes a pseudo-type happyDomain is able to represent.
type PseudoType struct {
	// Name is the type name, as DNSControl spells it.
	Name string

	// Rrtype is the private use type code standing for Name inside happyDomain.
	Rrtype uint16

	// New instanciates the rdata of this pseudo-type.
	New func() dns.PrivateRdata

	// BareTarget tells whether the whole rdata is the target, ie. whether
	// DNSControl represents this type with nothing more than a RecordConfig
	// target.
	//
	// This decides how far the type is registered into miekg/dns, and the two
	// cases are not interchangeable. DNSControl recognizes a pseudo-type by a
	// dns.StringToType lookup miss, in RecordConfig.ToRR() and, more
	// importantly, in RecordConfig.GetTargetCombined(), the value providers
	// store on their side and the diff engine compares.
	//
	// A bare target type is registered in dns.StringToType as well: DNSControl
	// then builds its target through the zone file representation of our rdata,
	// which gives back the very same string, and the generic conversions of
	// pkg/dnsrr work in both directions. Zone file parsing works too.
	//
	// A type with additional fields is deliberately kept out of
	// dns.StringToType, so that DNSControl keeps building its composite target
	// (`<target> atype=… zone_id=…`) instead of truncating it to the bare
	// target. Converting these records is then the job of the adapter, which
	// has to carry the extra fields to and from the dedicated RecordConfig
	// members.
	BareTarget bool
}

var (
	pseudoTypesByName   = map[string]*PseudoType{}
	pseudoTypesByRrtype = map[uint16]*PseudoType{}
)

// RegisterPseudoType makes a pseudo-type known to happyDomain and to miekg/dns.
func RegisterPseudoType(pt *PseudoType) {
	pseudoTypesByName[pt.Name] = pt
	pseudoTypesByRrtype[pt.Rrtype] = pt

	// PrivateHandle is the only way to get a *dns.PrivateRR whose internal
	// generator is set, which dns.Copy needs.
	dns.PrivateHandle(pt.Name, pt.Rrtype, pt.New)

	if !pt.BareTarget {
		// See PseudoType.BareTarget: DNSControl must keep seeing this one as a
		// pseudo-type. The type stays in dns.TypeToRR and dns.TypeToString, so
		// we can still instanciate and print it.
		delete(dns.StringToType, pt.Name)
	}
}

// PseudoTypeByName returns the pseudo-type spelled name, if any.
func PseudoTypeByName(name string) (*PseudoType, bool) {
	pt, ok := pseudoTypesByName[name]
	return pt, ok
}

// PseudoTypeByRrtype returns the pseudo-type standing behind the given private
// use type code, if any.
func PseudoTypeByRrtype(rrtype uint16) (*PseudoType, bool) {
	pt, ok := pseudoTypesByRrtype[rrtype]
	return pt, ok
}

// PseudoTypes returns all the registered pseudo-types.
func PseudoTypes() map[string]*PseudoType {
	return pseudoTypesByName
}

func init() {
	for _, pt := range []*PseudoType{
		// ALIAS is the CNAME-like record DNSControl asks providers to resolve
		// at the apex.
		{Name: "ALIAS", Rrtype: TypeALIAS, New: newHostnameRdata, BareTarget: true},

		// ANAME is how dnsmadeeasy, namedotcom and cnr spell an ALIAS on their
		// side: happyDomain never proposes it, but has to read it back.
		{Name: "ANAME", Rrtype: TypeANAME, New: newHostnameRdata, BareTarget: true},

		{Name: "AKAMAICDN", Rrtype: TypeAKAMAICDN, New: newHostnameRdata, BareTarget: true},

		{Name: "R53_ALIAS", Rrtype: TypeR53ALIAS, New: newR53AliasRdata},
		{Name: "AZURE_ALIAS", Rrtype: TypeAZUREALIAS, New: newAzureAliasRdata},
		{Name: "AKAMAITLC", Rrtype: TypeAKAMAITLC, New: newAkamaiTLCRdata},
	} {
		RegisterPseudoType(pt)
	}
}

// HostnameRdata is the rdata of the pseudo-types whose content is just a target
// domain name.
type HostnameRdata struct {
	Target string
}

func newHostnameRdata() dns.PrivateRdata { return new(HostnameRdata) }

func (rd *HostnameRdata) GetTarget() string          { return rd.Target }
func (rd *HostnameRdata) SetTarget(t string)         { rd.Target = t }
func (rd *HostnameRdata) String() string             { return rd.Target }
func (rd *HostnameRdata) Len() int                   { return len(rd.Target) + 1 }
func (rd *HostnameRdata) Pack([]byte) (int, error)   { return 0, ErrNoWireFormat }
func (rd *HostnameRdata) Unpack([]byte) (int, error) { return 0, ErrNoWireFormat }

func (rd *HostnameRdata) Parse(txt []string) error {
	if len(txt) != 1 {
		return fmt.Errorf("expected a single target, got %d fields", len(txt))
	}

	rd.Target = txt[0]

	return nil
}

func (rd *HostnameRdata) Copy(dest dns.PrivateRdata) error {
	d, ok := dest.(*HostnameRdata)
	if !ok {
		return fmt.Errorf("unable to copy a %T into a %T", rd, dest)
	}

	*d = *rd

	return nil
}

// R53AliasRdata is the rdata of the Route53 specific alias, which points at an
// AWS resource rather than at a domain name.
type R53AliasRdata struct {
	Target               string
	AType                string
	ZoneID               string
	EvaluateTargetHealth string
}

func newR53AliasRdata() dns.PrivateRdata { return new(R53AliasRdata) }

func (rd *R53AliasRdata) GetTarget() string  { return rd.Target }
func (rd *R53AliasRdata) SetTarget(t string) { rd.Target = t }

// String returns the very representation DNSControl gives to a R53_ALIAS in
// models.RecordConfig.GetTargetCombined, so that both sides of a comparison
// agree.
func (rd *R53AliasRdata) String() string {
	return fmt.Sprintf("%s atype=%s zone_id=%s evaluate_target_health=%s", rd.Target, rd.AType, rd.ZoneID, rd.EvaluateTargetHealth)
}

func (rd *R53AliasRdata) Len() int                   { return len(rd.String()) + 1 }
func (rd *R53AliasRdata) Pack([]byte) (int, error)   { return 0, ErrNoWireFormat }
func (rd *R53AliasRdata) Unpack([]byte) (int, error) { return 0, ErrNoWireFormat }

func (rd *R53AliasRdata) Parse(txt []string) error {
	if len(txt) == 0 {
		return fmt.Errorf("expected a target, got nothing")
	}

	rd.Target = txt[0]
	rd.AType = extractKeyValue(txt[1:], "atype")
	rd.ZoneID = extractKeyValue(txt[1:], "zone_id")
	rd.EvaluateTargetHealth = extractKeyValue(txt[1:], "evaluate_target_health")

	return nil
}

func (rd *R53AliasRdata) Copy(dest dns.PrivateRdata) error {
	d, ok := dest.(*R53AliasRdata)
	if !ok {
		return fmt.Errorf("unable to copy a %T into a %T", rd, dest)
	}

	*d = *rd

	return nil
}

// AzureAliasRdata is the rdata of the Azure specific alias.
type AzureAliasRdata struct {
	Target string
	AType  string
}

func newAzureAliasRdata() dns.PrivateRdata { return new(AzureAliasRdata) }

func (rd *AzureAliasRdata) GetTarget() string  { return rd.Target }
func (rd *AzureAliasRdata) SetTarget(t string) { rd.Target = t }

// String returns the very representation DNSControl gives to an AZURE_ALIAS in
// models.RecordConfig.GetTargetCombined.
func (rd *AzureAliasRdata) String() string {
	return fmt.Sprintf("%s atype=%s", rd.Target, rd.AType)
}

func (rd *AzureAliasRdata) Len() int                   { return len(rd.String()) + 1 }
func (rd *AzureAliasRdata) Pack([]byte) (int, error)   { return 0, ErrNoWireFormat }
func (rd *AzureAliasRdata) Unpack([]byte) (int, error) { return 0, ErrNoWireFormat }

func (rd *AzureAliasRdata) Parse(txt []string) error {
	if len(txt) == 0 {
		return fmt.Errorf("expected a target, got nothing")
	}

	rd.Target = txt[0]
	rd.AType = extractKeyValue(txt[1:], "atype")

	return nil
}

func (rd *AzureAliasRdata) Copy(dest dns.PrivateRdata) error {
	d, ok := dest.(*AzureAliasRdata)
	if !ok {
		return fmt.Errorf("unable to copy a %T into a %T", rd, dest)
	}

	*d = *rd

	return nil
}

// AkamaiTLCRdata is the rdata of the Akamai EdgeDNS traffic love control
// record.
type AkamaiTLCRdata struct {
	Target     string
	AnswerType string
}

func newAkamaiTLCRdata() dns.PrivateRdata { return new(AkamaiTLCRdata) }

func (rd *AkamaiTLCRdata) GetTarget() string  { return rd.Target }
func (rd *AkamaiTLCRdata) SetTarget(t string) { rd.Target = t }

// String returns the very representation DNSControl gives to an AKAMAITLC in
// models.RecordConfig.GetTargetCombined, answer type first.
func (rd *AkamaiTLCRdata) String() string {
	return fmt.Sprintf("%s %s", rd.AnswerType, rd.Target)
}

func (rd *AkamaiTLCRdata) Len() int                   { return len(rd.String()) + 1 }
func (rd *AkamaiTLCRdata) Pack([]byte) (int, error)   { return 0, ErrNoWireFormat }
func (rd *AkamaiTLCRdata) Unpack([]byte) (int, error) { return 0, ErrNoWireFormat }

func (rd *AkamaiTLCRdata) Parse(txt []string) error {
	if len(txt) != 2 {
		return fmt.Errorf("expected an answer type and a target, got %d fields", len(txt))
	}

	rd.AnswerType = txt[0]
	rd.Target = txt[1]

	return nil
}

func (rd *AkamaiTLCRdata) Copy(dest dns.PrivateRdata) error {
	d, ok := dest.(*AkamaiTLCRdata)
	if !ok {
		return fmt.Errorf("unable to copy a %T into a %T", rd, dest)
	}

	*d = *rd

	return nil
}

// extractKeyValue returns the value of the `key=value` field of fields, or an
// empty string when there is none.
func extractKeyValue(fields []string, key string) string {
	for _, field := range fields {
		if value, ok := strings.CutPrefix(field, key+"="); ok {
			return value
		}
	}

	return ""
}
