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
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/miekg/dns"
)

// IsPseudoRecordType reports whether rtype is one of DNSControl's pseudo-types:
// a type used internally by DNSControl (ALIAS, IMPORT_TRANSFORM) or registered
// by a provider for its own needs (R53_ALIAS, AZURE_ALIAS, URL, URL301, FRAME,
// LUA, CF_WORKER_ROUTE, …), which has no counterpart in the DNS wire format.
//
// Such records must never reach models.RecordConfig.ToRR(): that function
// resolves the type through dns.StringToType and calls log.Fatalf on a miss,
// terminating the whole process instead of returning an error. Since log.Fatalf
// calls os.Exit, no recover() can catch it.
//
// The test is a lookup miss rather than a list of known names, so it also covers
// the pseudo-types DNSControl will introduce in future releases, the "UNKNOWN"
// type it uses for rdata it cannot parse, and the empty type DNSControl gives to
// the records miekg/dns could not name (dns.RFC3597, ie. private-use types).
//
// A few names do resolve through dns.StringToType while having no record
// implementation behind them: the meta-types and QTYPEs (AXFR, IXFR, MAILA,
// MAILB, ATMA, UNSPEC, None, Reserved). ToRR() calls their nil constructor and
// panics, so they are reported as pseudo-types too.
func IsPseudoRecordType(rtype string) bool {
	rdtype, ok := dns.StringToType[rtype]
	if !ok {
		return true
	}

	return dns.TypeToRR[rdtype] == nil
}

// dropPseudoRecords returns records without the entries happyDomain is unable to
// represent, ie. those carrying a pseudo-type.
//
// Callers must apply it symmetrically to both sides of a comparison. Dropping
// such a record from the zone read from a provider, while the desired zone was
// itself built from records already dropped at import time, keeps it out of the
// diff entirely: DNSControl generates no correction for it and leaves it
// untouched on the provider. Dropping it from only one side would instead make
// the diff engine believe the record has to be deleted.
func dropPseudoRecords(records models.Records) models.Records {
	ret := make(models.Records, 0, len(records))

	for _, rec := range records {
		if IsPseudoRecordType(rec.Type) {
			continue
		}

		ret = append(ret, rec)
	}

	return ret
}
