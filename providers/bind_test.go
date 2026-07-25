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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupAllowedDirectories points the allow list at the given directories for
// the duration of the test.
func setupAllowedDirectories(t *testing.T, directories ...string) {
	t.Helper()

	previous := bindAllowedDirectories
	t.Cleanup(func() { bindAllowedDirectories = previous })

	bindAllowedDirectories = nil
	for _, directory := range directories {
		if err := addBindAllowedDirectory(directory); err != nil {
			t.Fatalf("addBindAllowedDirectory(%q) = %v", directory, err)
		}
	}
}

func TestToDNSControlConfigDirectory(t *testing.T) {
	root := t.TempDir()
	// t.TempDir() may sit behind a symlink (/tmp -> /private/tmp on macOS).
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) = %v", root, err)
	}

	allowed := filepath.Join(root, "zones")
	sibling := filepath.Join(root, "zones-evil")
	nested := filepath.Join(allowed, "tenant")
	for _, dir := range []string{allowed, sibling, nested} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("Mkdir(%q) = %v", dir, err)
		}
	}

	escaping := filepath.Join(allowed, "escaping")
	if err := os.Symlink(root, escaping); err != nil {
		t.Fatalf("Symlink(%q) = %v", escaping, err)
	}

	setupAllowedDirectories(t, allowed)

	tests := []struct {
		name        string
		directory   string
		expectError bool
	}{
		{"the allowed directory itself", allowed, false},
		{"a subdirectory of the allowed one", nested, false},
		{"a traversal landing back inside", filepath.Join(nested, "..", "tenant"), false},
		{"a traversal escaping the allowed one", filepath.Join(allowed, "..", "zones-evil"), true},
		{"a sibling sharing the same prefix", sibling, true},
		{"an unrelated absolute directory", "/etc", true},
		{"a symlink pointing outside", escaping, true},
		{"a directory that does not exist", filepath.Join(allowed, "missing"), true},
		{"no directory at all", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := (&BindServer{Directory: tt.directory}).ToDNSControlConfig()

			if tt.expectError {
				if err == nil {
					t.Fatalf("ToDNSControlConfig() expected an error, got %v", config)
				}
				return
			}

			if err != nil {
				t.Fatalf("ToDNSControlConfig() error = %v", err)
			}

			// The directory handed to dnscontrol is the canonicalized one.
			if !filepath.IsAbs(config["directory"]) {
				t.Errorf("ToDNSControlConfig()[directory] = %q; want an absolute path", config["directory"])
			}
		})
	}
}

// TestInstantiateProviderRefusesStoredDirectory checks the whole instantiation
// path, the one taken every time a stored provider is used to list, read or
// write zones: a provider persisted with an arbitrary directory before the
// allow list existed must not be usable anymore.
func TestInstantiateProviderRefusesStoredDirectory(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) = %v", root, err)
	}

	setupAllowedDirectories(t, root)

	if _, err := (&BindServer{Directory: "/etc"}).InstantiateProvider(); err == nil {
		t.Error("InstantiateProvider() expected an error for a directory outside the allow list")
	}

	if _, err := (&BindServer{Directory: root}).InstantiateProvider(); err != nil {
		t.Errorf("InstantiateProvider() error = %v", err)
	}
}

// TestAllowedDirectoryAppearingLater covers the volume mounted after the
// process starts: the flag accepts the path, nothing is allowed while it is
// missing, and it starts working once it is there, with no restart.
func TestAllowedDirectoryAppearingLater(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) = %v", root, err)
	}

	zones := filepath.Join(root, "zones")
	setupAllowedDirectories(t, zones)

	if _, err := (&BindServer{Directory: zones}).ToDNSControlConfig(); err == nil {
		t.Error("ToDNSControlConfig() expected an error while the allowed directory does not exist")
	}

	if _, err := (&BindServer{Directory: "/etc"}).ToDNSControlConfig(); err == nil {
		t.Error("ToDNSControlConfig() expected an error for a directory outside the allow list")
	}

	if err := os.Mkdir(zones, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) = %v", zones, err)
	}

	if _, err := (&BindServer{Directory: zones}).ToDNSControlConfig(); err != nil {
		t.Errorf("ToDNSControlConfig() error = %v once the allowed directory exists", err)
	}
}

func TestToDNSControlConfigWithoutAllowedDirectory(t *testing.T) {
	setupAllowedDirectories(t)

	if _, err := (&BindServer{Directory: "/etc"}).ToDNSControlConfig(); err == nil {
		t.Error("ToDNSControlConfig() expected an error when no directory is allowed")
	}
}

func TestToDNSControlConfigFileformat(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) = %v", root, err)
	}

	setupAllowedDirectories(t, root)

	tests := []struct {
		format      string
		expectError bool
	}{
		{"", false},
		{"%U.zone", false},
		{"%D.zone", false},
		{"../../tmp/pwned.zone", true},
		{"..", true},
		{".", true},
		{"sub/%U.zone", true},
		{`sub\%U.zone`, true},
		{"/etc/cron.d/%U", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			config, err := (&BindServer{Directory: root, Fileformat: tt.format}).ToDNSControlConfig()

			if tt.expectError {
				if err == nil {
					t.Fatalf("ToDNSControlConfig() expected an error, got %v", config)
				}
				return
			}

			if err != nil {
				t.Fatalf("ToDNSControlConfig() error = %v", err)
			}

			if got := config["filenameformat"]; got != tt.format {
				t.Errorf("ToDNSControlConfig()[filenameformat] = %q; want %q", got, tt.format)
			}
		})
	}
}

// TestInstantiatedProviderScreensZoneName checks the screening is really wired
// on the actuator handed to the rest of the application, and not only reachable
// through checkZoneName.
func TestInstantiatedProviderScreensZoneName(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) = %v", root, err)
	}

	setupAllowedDirectories(t, root)

	// %a substitutes the zone name in the file name, which is exactly the
	// substitution the screening protects.
	actuator, err := (&BindServer{Directory: root, Fileformat: "%a.zone"}).InstantiateProvider()
	if err != nil {
		t.Fatalf("InstantiateProvider() error = %v", err)
	}

	escaping := "../../../etc/cron.d/pwned"

	if _, err := actuator.GetZoneRecords(escaping); err == nil {
		t.Error("GetZoneRecords() expected an error for a zone name escaping the directory")
	}

	if _, _, err := actuator.GetZoneCorrections(escaping, nil); err == nil {
		t.Error("GetZoneCorrections() expected an error for a zone name escaping the directory")
	}

	if err := actuator.CreateDomain(escaping); err == nil {
		t.Error("CreateDomain() expected an error for a zone name escaping the directory")
	}

	// A zone name that is a plain file name still goes through to dnscontrol,
	// which reads it from the allowed directory.
	zonefile := "example.com.\t3600\tIN\tSOA\tns.example.com. hostmaster.example.com. 1 7200 3600 86400 3600\n" +
		"www.example.com.\t3600\tIN\tA\t192.0.2.1\n"
	if err := os.WriteFile(filepath.Join(root, "example.com.zone"), []byte(zonefile), 0o644); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}

	records, err := actuator.GetZoneRecords("example.com.")
	if err != nil {
		t.Fatalf("GetZoneRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Errorf("GetZoneRecords() = %v; want the 2 records of the zone file", records)
	}
}

// TestInstantiatedProviderUsesOneFilePerZone guards against the zone name never
// reaching the file name: dnscontrol substitutes it from the Metadata map of the
// DomainConfig, and with the default format (%c.zone) an unpopulated map made
// every zone read and write <directory>/.zone.
func TestInstantiatedProviderUsesOneFilePerZone(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) = %v", root, err)
	}

	setupAllowedDirectories(t, root)

	zones := map[string]string{
		"example.com.": "192.0.2.1",
		"other.tld.":   "192.0.2.2",
	}
	for zone, address := range zones {
		content := zone + "\t3600\tIN\tSOA\tns." + zone + " hostmaster." + zone + " 1 7200 3600 86400 3600\n" +
			"www." + zone + "\t3600\tIN\tA\t" + address + "\n"
		if err := os.WriteFile(filepath.Join(root, strings.TrimSuffix(zone, ".")+".zone"), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile() = %v", err)
		}
	}

	// No Fileformat: dnscontrol falls back to its default, %c.zone.
	actuator, err := (&BindServer{Directory: root}).InstantiateProvider()
	if err != nil {
		t.Fatalf("InstantiateProvider() error = %v", err)
	}

	for zone, address := range zones {
		records, err := actuator.GetZoneRecords(zone)
		if err != nil {
			t.Fatalf("GetZoneRecords(%q) error = %v", zone, err)
		}

		var found bool
		for _, record := range records {
			if strings.Contains(record.String(), address) {
				found = true
			}
		}
		if !found {
			t.Errorf("GetZoneRecords(%q) = %v; want the zone file holding %s", zone, records, address)
		}
	}
}

func TestCheckZoneName(t *testing.T) {
	tests := []struct {
		zone        string
		expectError bool
	}{
		{"example.com.", false},
		{"example.com", false},
		{"sub.example.com.", false},
		{"2.0.192.in-addr.arpa.", false},
		// A valid RFC 2317 delegation, but not a valid file name.
		{"0/25.2.0.192.in-addr.arpa.", true},
		{`etc\057cron.d.`, true},
		{"../../etc/passwd", true},
		{"..", true},
		{".", true},
	}

	for _, tt := range tests {
		t.Run(tt.zone, func(t *testing.T) {
			err := checkZoneName(tt.zone)

			if tt.expectError && err == nil {
				t.Errorf("checkZoneName(%q) expected an error", tt.zone)
			} else if !tt.expectError && err != nil {
				t.Errorf("checkZoneName(%q) = %v", tt.zone, err)
			}
		})
	}
}

func TestEnableBindProvider(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "notadir")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) = %v", file, err)
	}

	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		{"an existing directory", root, false},
		{"several directories", root + string(os.PathListSeparator) + root, false},
		{"the legacy boolean form", "true", true},
		{"the legacy env boolean form", "1", true},
		{"an empty value", "", true},
		{"a relative path", "zones", true},
		// A directory mounted after the process starts must not stop it, so
		// the flag accepts it and the confinement rejects it until it exists.
		{"a directory that does not exist yet", filepath.Join(root, "missing"), false},
		{"a file", file, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupAllowedDirectories(t)
			registered := bindProviderRegistered
			t.Cleanup(func() { bindProviderRegistered = registered })

			err := enableBindProvider(tt.value)

			if tt.expectError {
				if err == nil {
					t.Fatalf("enableBindProvider(%q) expected an error", tt.value)
				}
				if len(bindAllowedDirectories) != 0 {
					t.Errorf("enableBindProvider(%q) allowed %v", tt.value, bindAllowedDirectories)
				}
				return
			}

			if err != nil {
				t.Fatalf("enableBindProvider(%q) = %v", tt.value, err)
			}

			if len(bindAllowedDirectories) == 0 {
				t.Errorf("enableBindProvider(%q) allowed no directory", tt.value)
			}
		})
	}
}
