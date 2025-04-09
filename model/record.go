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
	"strings"

	"github.com/miekg/dns"
)

type Record interface {
	Header() *dns.RR_Header
	String() string
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

func (rr *SPF) Header() *dns.RR_Header {
	return &rr.Hdr
}

func (rr *SPF) String() string {
	return rr.ToRR().String()
}

func (rr *SPF) ToRR() dns.RR {
	var txts []string

	txts = make([]string, len(rr.Txt)/TXT_SEGMENT_LEN)

	for i := len(rr.Txt)/TXT_SEGMENT_LEN - 1; i >= 0; i-- {
		if i == len(rr.Txt)-1 {
			txts[i] = rr.Txt[i*TXT_SEGMENT_LEN:]
		} else {
			txts[i] = rr.Txt[i*TXT_SEGMENT_LEN : (i+1)*TXT_SEGMENT_LEN]
		}
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

const TXT_SEGMENT_LEN = 255

func (rr *TXT) Header() *dns.RR_Header {
	return &rr.Hdr
}

func (rr *TXT) String() string {
	return rr.ToRR().String()
}

func (rr *TXT) ToRR() dns.RR {
	var txts []string

	txts = make([]string, len(rr.Txt)/TXT_SEGMENT_LEN)

	for i := len(rr.Txt)/TXT_SEGMENT_LEN - 1; i >= 0; i-- {
		if i == len(rr.Txt)-1 {
			txts[i] = rr.Txt[i*TXT_SEGMENT_LEN:]
		} else {
			txts[i] = rr.Txt[i*TXT_SEGMENT_LEN : (i+1)*TXT_SEGMENT_LEN]
		}
	}

	return &dns.TXT{
		Hdr: rr.Hdr,
		Txt: txts,
	}
}
