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

package domaininfo

import (
	"strings"
)

// NormalizeStatus joins the words of a registry status in camel case, as EPP
// (RFC 5731) and WHOIS servers spell it. RDAP spells with spaced words what
// EPP writes in camel case (RFC 8056, section 2): "client transfer
// prohibited" is clientTransferProhibited. A status already in camel case, or
// made of a single word, is only trimmed: the RDAP "active", which EPP calls
// ok, stays "active".
func NormalizeStatus(status string) string {
	words := strings.Fields(status)
	if len(words) < 2 {
		return strings.TrimSpace(status)
	}

	var b strings.Builder
	b.WriteString(strings.ToLower(words[0]))
	for _, w := range words[1:] {
		w = strings.ToLower(w)
		b.WriteString(strings.ToUpper(w[:1]))
		b.WriteString(w[1:])
	}
	return b.String()
}

var statusKeyReplacer = strings.NewReplacer(" ", "", "-", "", "_", "")

// StatusKey reduces a registry status to a comparison key: lowercased,
// without spaces, dashes nor underscores, so that "client transfer
// prohibited", "client-transfer-prohibited" and "clientTransferProhibited"
// compare equal.
func StatusKey(s string) string {
	return statusKeyReplacer.Replace(strings.ToLower(s))
}

// HasStatus tells whether status, however it is spelled, is part of the
// statuses.
func HasStatus(statuses []string, status string) bool {
	key := StatusKey(status)
	for _, s := range statuses {
		if StatusKey(s) == key {
			return true
		}
	}
	return false
}

// normalizeStatuses applies NormalizeStatus to each status, keeping a nil
// list nil.
func normalizeStatuses(status []string) []string {
	if status == nil {
		return nil
	}

	out := make([]string, len(status))
	for i, s := range status {
		out[i] = NormalizeStatus(s)
	}
	return out
}
