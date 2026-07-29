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

package abstract_test

import (
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/services/abstract"
)

// A service body deserialized from JSON that omits the record key leaves
// Record nil. GetRecords used to dereference it before noticing.
func TestGetRecordsWithoutRecord(t *testing.T) {
	if _, err := (&abstract.OpenPGP{Username: "hugh"}).GetRecords("example.com", 0, "example.com"); err == nil {
		t.Error("OpenPGP.GetRecords with a nil record: expected an error, got none")
	}

	if _, err := (&abstract.SMimeCert{Username: "hugh"}).GetRecords("example.com", 0, "example.com"); err == nil {
		t.Error("SMimeCert.GetRecords with a nil record: expected an error, got none")
	}
}

// The owner name has to carry the hash of the username, or the records would
// not be found by a client resolving them from an address.
func TestGetRecordsPrefixMismatch(t *testing.T) {
	// RFC 7929 sec. 4 example: hugh@example.com
	const hugh = "c93f1e400f26708f98cb19d936620da35eec8f72e57f9eec01c1afd6"

	pgp := &abstract.OpenPGP{
		Username: "hugh",
		Record:   &dns.OPENPGPKEY{Hdr: dns.RR_Header{Name: hugh + "._openpgpkey.example.com."}},
	}
	if _, err := pgp.GetRecords(hugh+"._openpgpkey.example.com.", 0, "example.com"); err != nil {
		t.Errorf("OpenPGP.GetRecords with a matching prefix: %s", err)
	}
	if _, err := pgp.GetRecords("nope._openpgpkey.example.com.", 0, "example.com"); err == nil {
		t.Error("OpenPGP.GetRecords with a mismatched prefix: expected an error, got none")
	}

	smime := &abstract.SMimeCert{
		Username: "hugh",
		Record:   &dns.SMIMEA{Hdr: dns.RR_Header{Name: hugh + "._smimecert.example.com."}},
	}
	if _, err := smime.GetRecords(hugh+"._smimecert.example.com.", 0, "example.com"); err != nil {
		t.Errorf("SMimeCert.GetRecords with a matching prefix: %s", err)
	}

	smime.Username = "someone-else"
	if _, err := smime.GetRecords(hugh+"._smimecert.example.com.", 0, "example.com"); err == nil {
		t.Error("SMimeCert.GetRecords with a mismatched prefix: expected an error, got none")
	}
}
