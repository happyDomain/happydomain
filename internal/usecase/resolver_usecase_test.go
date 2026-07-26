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

package usecase

import (
	"testing"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

// The cases use IP literals throughout: pickResolver must decide on those
// without any name resolution, so the test needs no DNS.
func TestPickResolver(t *testing.T) {
	tests := []struct {
		name     string
		allowed  []string
		resolver string
		custom   string
		want     string
		wantErr  bool
	}{
		{
			name: "no resolver requested falls back to a public one",
			want: "1.1.1.1:53",
		},
		{
			// The frontend puts the address straight in `resolver`, so this
			// path is exactly as caller-controlled as `custom` is.
			name:     "a public resolver named directly",
			resolver: "9.9.9.10",
			want:     "9.9.9.10:53",
		},
		{
			name:     "loopback named directly is refused",
			resolver: "127.0.0.1",
			wantErr:  true,
		},
		{
			name:     "loopback as a custom resolver is refused",
			resolver: "custom",
			custom:   "127.0.0.1",
			wantErr:  true,
		},
		{
			name:     "the admin socket's host is refused",
			resolver: "custom",
			custom:   "::1",
			wantErr:  true,
		},
		{
			name:     "the cloud metadata endpoint is refused",
			resolver: "custom",
			custom:   "169.254.169.254",
			wantErr:  true,
		},
		{
			name:     "an internal resolver the operator allowed",
			allowed:  []string{"10.0.0.0/8"},
			resolver: "custom",
			custom:   "10.0.0.53",
			want:     "10.0.0.53:53",
		},
		{
			name:     "an address outside what the operator allowed",
			allowed:  []string{"10.0.0.0/8"},
			resolver: "custom",
			custom:   "192.168.1.53",
			wantErr:  true,
		},
		{
			name:     "an IPv6 literal is bracketed, not mangled",
			resolver: "custom",
			custom:   "2606:4700:4700::1111",
			want:     "[2606:4700:4700::1111]:53",
		},
		{
			// The old string surgery turned this into "[1.1.1.1:5353]:53".
			name:     "an explicit port is kept",
			resolver: "custom",
			custom:   "1.1.1.1:5353",
			want:     "1.1.1.1:5353",
		},
		{
			name:     "custom without an address",
			resolver: "custom",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard, err := netguard.New("resolver", "-resolver-allowed-target", tt.allowed)
			if err != nil {
				t.Fatalf("netguard.New(%v) => %v", tt.allowed, err)
			}

			ru := &resolverUsecase{
				config:        &happydns.Options{},
				resolverGuard: guard,
			}

			got, err := ru.pickResolver(t.Context(), tt.resolver, tt.custom)
			if (err != nil) != tt.wantErr {
				t.Fatalf("pickResolver(%q, %q) error = %v, wantErr %v", tt.resolver, tt.custom, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("pickResolver(%q, %q) = %q, want %q", tt.resolver, tt.custom, got, tt.want)
			}
		})
	}
}
