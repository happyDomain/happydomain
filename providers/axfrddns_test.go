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

package providers // import "git.happydns.org/happyDomain/providers"

import (
	"reflect"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/model"
)

const ddnsTestKey = "hmac-sha256:ddns:a0b1c2d3e4f5"

func TestDDNSServerKeys(t *testing.T) {
	tests := []struct {
		name         string
		server       DDNSServer
		expectError  bool
		wantTransfer string
		wantUpdate   string
	}{
		{
			name:         "read-write with a key",
			server:       DDNSServer{KeyName: "ddns", KeyAlgo: "hmac-sha256", KeyBlob: happydns.NewSecret("a0b1c2d3e4f5")},
			wantTransfer: ddnsTestKey,
			wantUpdate:   ddnsTestKey,
		},
		{
			name:        "read-write without a key",
			server:      DDNSServer{KeyAlgo: "hmac-sha256"},
			expectError: true,
		},
		{
			name:         "transfer only with a key",
			server:       DDNSServer{KeyName: "ddns", KeyAlgo: "hmac-sha256", KeyBlob: happydns.NewSecret("a0b1c2d3e4f5"), TransferOnly: true},
			wantTransfer: ddnsTestKey,
		},
		{
			name:   "transfer only without a key",
			server: DDNSServer{KeyAlgo: "hmac-sha256", TransferOnly: true},
		},
		{
			name:        "a key name without its secret",
			server:      DDNSServer{KeyName: "ddns", KeyAlgo: "hmac-sha256", TransferOnly: true},
			expectError: true,
		},
		{
			name:        "a secret key that is not base64",
			server:      DDNSServer{KeyName: "ddns", KeyAlgo: "hmac-sha256", KeyBlob: happydns.NewSecret("not base64!"), TransferOnly: true},
			expectError: true,
		},
		{
			name:        "a secret without its key name",
			server:      DDNSServer{KeyAlgo: "hmac-sha256", KeyBlob: happydns.NewSecret("a0b1c2d3e4f5"), TransferOnly: true},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := tt.server.ToDNSControlConfig()
			if tt.expectError {
				if err == nil {
					t.Fatalf("ToDNSControlConfig() = %v, want an error", config)
				}
				return
			}
			if err != nil {
				t.Fatalf("ToDNSControlConfig() = %v", err)
			}

			if got, ok := config["transfer-key"]; got != tt.wantTransfer || ok != (tt.wantTransfer != "") {
				t.Errorf("transfer-key = %q (set: %t), want %q", got, ok, tt.wantTransfer)
			}
			if got, ok := config["update-key"]; got != tt.wantUpdate || ok != (tt.wantUpdate != "") {
				t.Errorf("update-key = %q (set: %t), want %q", got, ok, tt.wantUpdate)
			}
		})
	}
}

func TestDDNSServerIsReadOnly(t *testing.T) {
	if (&DDNSServer{}).IsReadOnly() {
		t.Error("IsReadOnly() = true for a read-write server")
	}
	if !(&DDNSServer{TransferOnly: true}).IsReadOnly() {
		t.Error("IsReadOnly() = false for a transfer only server")
	}

	var body happydns.ProviderBody = &DDNSServer{}
	if _, ok := body.(happydns.ReadOnlyProvider); !ok {
		t.Error("DDNSServer does not implement happydns.ReadOnlyProvider")
	}
}

// The tag of a field is split on commas: one in the description would cut it,
// the rest taken for the label.
func TestDDNSTransferOnlyField(t *testing.T) {
	sf, _ := reflect.TypeOf(DDNSServer{}).FieldByName("TransferOnly")
	f := forms.GenField(sf)
	if f.Label != "Zone transfer only" || !strings.HasSuffix(f.Description, "to the address of happyDomain.") {
		t.Errorf("label %q, description %q: want the label and the whole description", f.Label, f.Description)
	}
}
