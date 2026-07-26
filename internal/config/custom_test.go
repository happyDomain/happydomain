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

package config // import "git.happydns.org/happyDomain/internal/config"

import (
	"slices"
	"testing"
)

func TestProxyListSet(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		want    []string
		wantErr bool
	}{
		{
			name:   "single address",
			values: []string{"192.0.2.1"},
			want:   []string{"192.0.2.1"},
		},
		{
			name:   "comma separated, as given by an environment variable",
			values: []string{"10.0.0.0/8, 2001:db8::/32 ,192.0.2.1"},
			want:   []string{"10.0.0.0/8", "2001:db8::/32", "192.0.2.1"},
		},
		{
			name:   "repeated flag accumulates",
			values: []string{"192.0.2.1", "198.51.100.0/24"},
			want:   []string{"192.0.2.1", "198.51.100.0/24"},
		},
		{
			// It used to expand to every private range, which trusts the
			// network the proxy sits on rather than the proxy itself.
			name:    "local keyword is refused",
			values:  []string{"local"},
			wantErr: true,
		},
		{
			name:   "empty items are skipped",
			values: []string{" , "},
			want:   nil,
		},
		{
			name:    "invalid CIDR is refused",
			values:  []string{"10.0.0.0/33"},
			wantErr: true,
		},
		{
			name:    "invalid address is refused",
			values:  []string{"not-an-ip"},
			wantErr: true,
		},
		{
			// net.ParseCIDR would silently mask this down to 192.0.2.0/24,
			// trusting 254 hosts for an operator who wrote a single one.
			name:    "prefix with host bits set is refused",
			values:  []string{"192.0.2.5/24"},
			wantErr: true,
		},
		{
			// gin turns this into ::/32: the intended proxy stays untrusted
			// and ::1 becomes trusted instead.
			name:    "IPv4-mapped address is refused",
			values:  []string{"::ffff:192.0.2.1"},
			wantErr: true,
		},
		{
			name:    "IPv4-mapped block is refused",
			values:  []string{"::ffff:192.0.2.0/120"},
			wantErr: true,
		},
		{
			name:   "none clears what a lower-precedence source set",
			values: []string{"10.0.0.0/8", "none, 192.0.2.10"},
			want:   []string{"192.0.2.10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			p := &proxyList{stringSlice{&got}}

			var err error
			for _, value := range tt.values {
				if err = p.Set(value); err != nil {
					break
				}
			}

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Set(%v) = nil error, want an error", tt.values)
				}
				return
			}

			if err != nil {
				t.Fatalf("Set(%v) => %s", tt.values, err.Error())
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("Set(%v) = %v, want %v", tt.values, got, tt.want)
			}
		})
	}
}
