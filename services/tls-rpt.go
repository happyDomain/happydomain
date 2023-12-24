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

package svcs

import (
	"fmt"
	"strconv"
	"strings"
)

type TLS_RPT struct {
	Version uint     `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of TLSRPT to use.,default=1,hidden"`
	Rua     []string `json:"rua" happydomain:"label=Aggregate Report URI,placeholder=https://example.com/path|mailto:name@example.com"`
}

func (t *TLS_RPT) Analyze(txt string) error {
	fields := strings.Split(txt, ";")

	if len(fields) < 2 {
		return fmt.Errorf("not a valid TLS-RPT record: should have a version AND a rua, only one field found")
	}
	if len(fields) > 3 || (len(fields) == 3 && fields[2] != "") {
		return fmt.Errorf("not a valid TLS-RPT record: should have exactly 2 fields: seen %d", len(fields))
	}

	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}

	if !strings.HasPrefix(fields[0], "v=TLSRPTv") {
		return fmt.Errorf("not a valid TLS-RPT record: should begin with v=TLSRPTv1, seen %q", fields[0])
	}

	version, err := strconv.ParseUint(fields[0][9:], 10, 32)
	if err != nil {
		return fmt.Errorf("not a valid TLS-RPT record: bad version number: %w", err)
	}
	t.Version = uint(version)

	if !strings.HasPrefix(fields[1], "rua=") {
		return fmt.Errorf("not a valid TLS-RPT record: expected rua=, found %q", fields[1])
	}

	t.Rua = strings.Split(strings.TrimPrefix(fields[1], "rua="), ",")

	for i := range t.Rua {
		t.Rua[i] = strings.TrimSpace(t.Rua[i])
	}

	return nil
}

func (t *TLS_RPT) String() string {
	return fmt.Sprintf("v=TLSRPTv%d; rua=%s", t.Version, strings.Join(t.Rua, ","))
}
