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
	"strings"

	"github.com/StackExchange/dnscontrol/v4/pkg/spflib"
)

type SPF struct {
	Version    uint     `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of SPF to use.,default=1,hidden"`
	Directives []string `json:"directives" happydomain:"label=Directives,placeholder=ip4:203.0.113.12"`
}

func (t *SPF) Analyze(txt string) error {
	_, err := spflib.Parse(txt, nil)
	if err != nil {
		return err
	}

	t.Version = 1

	fields := strings.Fields(txt)

	// Avoid doublon
	for _, directive := range fields[1:] {
		exists := false
		for _, known := range t.Directives {
			if known == directive {
				exists = true
				break
			}
		}

		if !exists {
			t.Directives = append(t.Directives, directive)
		}
	}

	return nil
}

func (t *SPF) String() string {
	directives := append([]string{fmt.Sprintf("v=spf%d", t.Version)}, t.Directives...)
	return strings.Join(directives, " ")
}
