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

//go:build linux || darwin || freebsd

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"testing"
)

// fakeSymbols is a pluginSymbols implementation backed by a static map. It
// lets the loader tests exercise their behaviour without having to compile a
// real .so file via `go build -buildmode=plugin`.
type fakeSymbols struct {
	syms map[string]plugin.Symbol
}

func (f *fakeSymbols) Lookup(name string) (plugin.Symbol, error) {
	if s, ok := f.syms[name]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("symbol %q not found", name)
}

// TestLoadPlugin_NoRecognisedSymbols verifies that when a .so file exports
// none of the known plugin symbols, every loader returns (false, nil), i.e.
// the file is silently skipped rather than reported as an error. loadPlugin
// itself logs a warning in that situation; we exercise the inner loop here
// because the outer call requires plugin.Open and a real .so file.
func TestLoadPlugin_NoRecognisedSymbols(t *testing.T) {
	fs := &fakeSymbols{}
	for _, loader := range pluginLoaders {
		found, err := loader(fs, "empty.so")
		if found || err != nil {
			t.Fatalf("loader returned (%v, %v) for empty symbol set, expected (false, nil)", found, err)
		}
	}
}

func TestCheckPluginDirectoryPermissions(t *testing.T) {
	dir := t.TempDir()

	// A freshly-created TempDir is owner-only on every platform we run on,
	// so this must be accepted.
	if err := os.Chmod(dir, 0o750); err != nil {
		t.Fatalf("chmod 0750: %v", err)
	}
	if err := checkPluginDirectoryPermissions(dir); err != nil {
		t.Errorf("expected 0750 directory to be accepted, got %v", err)
	}

	// World-writable: must be refused.
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatalf("chmod 0777: %v", err)
	}
	if err := checkPluginDirectoryPermissions(dir); err == nil {
		t.Errorf("expected 0777 directory to be refused")
	}

	// Group-writable: must also be refused.
	if err := os.Chmod(dir, 0o770); err != nil {
		t.Fatalf("chmod 0770: %v", err)
	}
	if err := checkPluginDirectoryPermissions(dir); err == nil {
		t.Errorf("expected 0770 directory to be refused")
	}

	// Restore permissions so t.TempDir cleanup can remove the directory.
	_ = os.Chmod(dir, 0o700)

	// Non-existent path: must be refused.
	if err := checkPluginDirectoryPermissions(filepath.Join(dir, "does-not-exist")); err == nil {
		t.Errorf("expected missing directory to be refused")
	}

	// Symlink to a valid directory: must be refused.
	target := t.TempDir()
	link := filepath.Join(dir, "symlink-plugins")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := checkPluginDirectoryPermissions(link); err == nil {
		t.Errorf("expected symlink directory to be refused")
	}
}

func TestCheckPluginFilePermissions(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.so")
	if err := os.WriteFile(f, []byte("fake"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Owner-writable, not group/world-writable: accepted.
	if err := checkPluginFilePermissions(f); err != nil {
		t.Errorf("expected 0644 file to be accepted, got %v", err)
	}

	// Group-writable: refused.
	if err := os.Chmod(f, 0o664); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if err := checkPluginFilePermissions(f); err == nil {
		t.Errorf("expected 0664 file to be refused")
	}

	// World-writable: refused.
	if err := os.Chmod(f, 0o646); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if err := checkPluginFilePermissions(f); err == nil {
		t.Errorf("expected 0646 file to be refused")
	}

	// Non-existent: refused.
	if err := checkPluginFilePermissions(filepath.Join(dir, "nope.so")); err == nil {
		t.Errorf("expected missing file to be refused")
	}

	// Symlink to a safe regular file: accepted (we follow the link and
	// check the target's permissions, not the link itself).
	regular := filepath.Join(dir, "real.so")
	if err := os.WriteFile(regular, []byte("real"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	link := filepath.Join(dir, "link.so")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := checkPluginFilePermissions(link); err != nil {
		t.Errorf("expected symlink to safe file to be accepted, got %v", err)
	}

	// Symlink to a writable target: refused.
	writable := filepath.Join(dir, "writable.so")
	if err := os.WriteFile(writable, []byte("bad"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(writable, 0o666); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	linkBad := filepath.Join(dir, "link-bad.so")
	if err := os.Symlink(writable, linkBad); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := checkPluginFilePermissions(linkBad); err == nil {
		t.Errorf("expected symlink to writable file to be refused")
	}
}
