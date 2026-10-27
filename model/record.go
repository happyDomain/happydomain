// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
)

type UpdateRecordForm struct {
	NewRR string `json:"newrr"`
	OldRR string `json:"oldrr"`
}

type Record interface {
	Header() *dns.RR_Header
	String() string
}

type CopiableRecord interface {
	Copy() Record
}

type ConvertibleRecord interface {
	ToRR() dns.RR
}

func RecordToServiceRecord(rr Record) *ServiceRecord {
	return &ServiceRecord{
		Type:   dns.TypeToString[rr.Header().Rrtype],
		String: rr.String(),
		RR:     rr,
	}
}

// TXT is an abstraction of TXT record, with a whole string.
type SPF struct {
	Hdr dns.RR_Header
	Txt string
}

func NewSPF(rr *dns.SPF) *SPF {
	return &SPF{
		Hdr: rr.Hdr,
		Txt: strings.Join(rr.Txt, ""),
	}
}

func (rr *SPF) Copy() Record {
	return &SPF{
		Hdr: rr.Hdr,
		Txt: rr.Txt,
	}
}

func (rr *SPF) Header() *dns.RR_Header {
	return &rr.Hdr
}

func (rr *SPF) String() string {
	return rr.ToRR().String()
}

func (rr *SPF) ToRR() dns.RR {
	txtLen := len(rr.Txt)
	numSegments := (txtLen + TXT_SEGMENT_LEN - 1) / TXT_SEGMENT_LEN
	txts := make([]string, numSegments)

	for i := range numSegments {
		start := i * TXT_SEGMENT_LEN
		end := min(start+TXT_SEGMENT_LEN, txtLen)
		txts[i] = rr.Txt[start:end]
	}

	return &dns.SPF{
		Hdr: rr.Hdr,
		Txt: txts,
	}
}

// TXT is an abstraction of TXT record, with a whole string.
type TXT struct {
	Hdr dns.RR_Header
	Txt string
}

func NewTXT(rr *dns.TXT) *TXT {
	return &TXT{
		Hdr: rr.Hdr,
		Txt: strings.Join(rr.Txt, ""),
	}
}

func (rr *TXT) Copy() Record {
	return &TXT{
		Hdr: rr.Hdr,
		Txt: rr.Txt,
	}
}

const TXT_SEGMENT_LEN = 255

func (rr *TXT) Header() *dns.RR_Header {
	return &rr.Hdr
}

func (rr *TXT) String() string {
	return rr.ToRR().String()
}

func (rr *TXT) ToRR() dns.RR {
	txtLen := len(rr.Txt)
	numSegments := (txtLen + TXT_SEGMENT_LEN - 1) / TXT_SEGMENT_LEN
	txts := make([]string, numSegments)

	for i := range numSegments {
		start := i * TXT_SEGMENT_LEN
		end := min(start+TXT_SEGMENT_LEN, txtLen)
		txts[i] = rr.Txt[start:end]
	}

	return &dns.TXT{
		Hdr: rr.Hdr,
		Txt: txts,
	}
}

// SVCB is an abstraction of the SVCB record, that can be stored.
//
// miekg/dns holds its parameters behind the SVCBKeyValue interface, which
// encoding/json writes but cannot read back; and once flattened, nothing tells
// which key each parameter was (both address hints become {"Hint": [...]}).
// They are written in their zone file form instead, e.g. alpn="h3,h2".
type SVCB struct {
	Hdr      dns.RR_Header
	Priority uint16
	// Tagged like in miekg/dns, for the helpers making it relative to the zone.
	Target string `dns:"domain-name"`
	Value  []dns.SVCBKeyValue
}

func NewSVCB(rr *dns.SVCB) *SVCB {
	rr = dns.Copy(rr).(*dns.SVCB)

	return &SVCB{
		Hdr:      rr.Hdr,
		Priority: rr.Priority,
		Target:   rr.Target,
		Value:    rr.Value,
	}
}

func (rr *SVCB) Copy() Record {
	return NewSVCB(rr.toSVCB())
}

func (rr *SVCB) Header() *dns.RR_Header {
	return &rr.Hdr
}

func (rr *SVCB) String() string {
	return rr.ToRR().String()
}

func (rr *SVCB) ToRR() dns.RR {
	return rr.toSVCB()
}

func (rr *SVCB) toSVCB() *dns.SVCB {
	return &dns.SVCB{
		Hdr:      rr.Hdr,
		Priority: rr.Priority,
		Target:   rr.Target,
		Value:    rr.Value,
	}
}

type svcbJSON[V any] struct {
	Hdr      dns.RR_Header
	Priority uint16
	Target   string
	Value    []V
}

func (rr SVCB) MarshalJSON() ([]byte, error) {
	values := make([]string, len(rr.Value))
	for i, kv := range rr.Value {
		values[i] = kv.Key().String() + "=\"" + kv.String() + "\""
	}

	return json.Marshal(svcbJSON[string]{
		Hdr:      rr.Hdr,
		Priority: rr.Priority,
		Target:   rr.Target,
		Value:    values,
	})
}

func (rr *SVCB) UnmarshalJSON(b []byte) error {
	var stored svcbJSON[json.RawMessage]
	if err := json.Unmarshal(b, &stored); err != nil {
		return err
	}

	values := make([]dns.SVCBKeyValue, 0, len(stored.Value))
	for _, raw := range stored.Value {
		kv, err := unmarshalSVCBKeyValue(raw)
		if err != nil {
			return fmt.Errorf("SVCB parameter %s: %w", raw, err)
		}
		values = append(values, kv)
	}

	rr.Hdr = stored.Hdr
	rr.Priority = stored.Priority
	rr.Target = stored.Target
	rr.Value = values

	return nil
}

func unmarshalSVCBKeyValue(raw json.RawMessage) (dns.SVCBKeyValue, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return unmarshalLegacySVCBKeyValue(raw)
	}

	// miekg/dns keeps the parser of each key to itself: go through a whole record.
	rr, err := dns.NewRR(". IN SVCB 1 . " + text)
	if err != nil {
		return nil, err
	}

	values := rr.(*dns.SVCB).Value
	if len(values) != 1 {
		return nil, fmt.Errorf("expected exactly one parameter")
	}

	return values[0], nil
}

// unmarshalLegacySVCBKeyValue reads the objects encoding/json made of the
// miekg/dns key types, recognizing them by their field names.
func unmarshalLegacySVCBKeyValue(raw json.RawMessage) (dns.SVCBKeyValue, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}

	var kv dns.SVCBKeyValue
	switch {
	case fields["KeyCode"] != nil:
		kv = new(dns.SVCBLocal)
	case fields["Code"] != nil:
		kv = new(dns.SVCBMandatory)
	case fields["Alpn"] != nil:
		kv = new(dns.SVCBAlpn)
	case fields["Port"] != nil:
		kv = new(dns.SVCBPort)
	case fields["ECH"] != nil:
		kv = new(dns.SVCBECHConfig)
	case fields["Template"] != nil:
		kv = new(dns.SVCBDoHPath)
	case fields["Hint"] != nil:
		var hints []net.IP
		if err := json.Unmarshal(fields["Hint"], &hints); err != nil {
			return nil, err
		}
		if len(hints) > 0 && hints[0].To4() == nil {
			return &dns.SVCBIPv6Hint{Hint: hints}, nil
		}
		return &dns.SVCBIPv4Hint{Hint: hints}, nil
	case len(fields) == 0:
		// no-default-alpn and ohttp both carry no value, the former is far more common.
		return new(dns.SVCBNoDefaultAlpn), nil
	default:
		return nil, fmt.Errorf("unknown SVCB parameter")
	}

	if err := json.Unmarshal(raw, kv); err != nil {
		return nil, err
	}

	return kv, nil
}

// HTTPS is an abstraction of the HTTPS record, that can be stored (see SVCB).
type HTTPS struct {
	SVCB
}

func NewHTTPS(rr *dns.HTTPS) *HTTPS {
	return &HTTPS{*NewSVCB(&rr.SVCB)}
}

func (rr *HTTPS) Copy() Record {
	return &HTTPS{*NewSVCB(rr.toSVCB())}
}

func (rr *HTTPS) String() string {
	return rr.ToRR().String()
}

func (rr *HTTPS) ToRR() dns.RR {
	return &dns.HTTPS{SVCB: *rr.toSVCB()}
}
