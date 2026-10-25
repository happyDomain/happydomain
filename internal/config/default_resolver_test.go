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

package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestResolverList(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		want    []string
		wantErr bool
	}{
		{name: "IPv4 without a port", values: []string{"192.0.2.53"}, want: []string{"192.0.2.53:53"}},
		{name: "IPv4 with a port", values: []string{"192.0.2.53:5353"}, want: []string{"192.0.2.53:5353"}},
		{name: "IPv6 without a port", values: []string{"2001:db8::53"}, want: []string{"[2001:db8::53]:53"}},
		{name: "bracketed IPv6 without a port", values: []string{"[2001:db8::53]"}, want: []string{"[2001:db8::53]:53"}},
		{name: "IPv6 with a port", values: []string{"[2001:db8::53]:5353"}, want: []string{"[2001:db8::53]:5353"}},
		{name: "comma separated, in order", values: []string{"192.0.2.54, 192.0.2.53"}, want: []string{"192.0.2.54:53", "192.0.2.53:53"}},
		{name: "repeated", values: []string{"192.0.2.53", "192.0.2.54"}, want: []string{"192.0.2.53:53", "192.0.2.54:53"}},
		{name: "none clears", values: []string{"192.0.2.53", "none"}, want: []string{}},
		{name: "a host name is refused", values: []string{"ns.example.com"}, wantErr: true},
		{name: "a bad port is refused", values: []string{"192.0.2.53:dns"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			p := resolverList(&got)

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
				t.Fatalf("Set(%v) => %v", tt.values, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Set(%v) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}

func writeResolvConf(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "resolv.conf")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestSystemResolversFrom(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"servers in order", "nameserver 192.0.2.53\nnameserver 192.0.2.54\n", []string{"192.0.2.53:53", "192.0.2.54:53"}},
		{"IPv6 server", "nameserver 2001:db8::53\n", []string{"[2001:db8::53]:53"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := systemResolversFrom(writeResolvConf(t, tt.content))
			if err != nil {
				t.Fatalf("systemResolversFrom: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestSystemResolversFromErrors(t *testing.T) {
	if _, err := systemResolversFrom(writeResolvConf(t, "search example.com\n")); err == nil {
		t.Error("want an error without any name server")
	}

	if _, err := systemResolversFrom(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("want an error without the file")
	}
}
