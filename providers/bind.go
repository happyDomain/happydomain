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

package providers // import "git.happydns.org/happyDomain/providers"

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/DNSControl/dnscontrol/v4/providers/bind"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

// bindAllowedDirectories lists the absolute directories the BIND provider is
// allowed to read from and write to. It is filled by the -with-bind-provider
// flag, which is also what registers the provider: there is no way to enable the
// provider without declaring at least one directory.
//
// The paths are kept as given, only made absolute and cleaned: they are resolved
// against the file system when a provider is used, not when the flag is parsed,
// because a zone directory is routinely a volume mounted after the process
// starts.
var bindAllowedDirectories []string

// bindProviderRegistered guards the registration, as the flag is repeatable.
var bindProviderRegistered bool

var errBindDirectoryNotAllowed = errors.New("this directory is not among those allowed by the administrator of this instance")

type BindServer struct {
	Directory  string `json:"directory,omitempty" happydomain:"label=Directory,placeholder=/etc/named/zones/,required,description=Local directory on the same host running happyDomain containing your zones. It has to be inside one of the directories allowed by the administrator of this instance."`
	Fileformat string `json:"fileformat,omitempty" happydomain:"label=File format,placeholder=%U.zone,description=See format at https://docs.dnscontrol.org/service-providers/providers/bind#filenameformat"`
}

func (s *BindServer) DNSControlName() string {
	return "BIND"
}

func (s *BindServer) InstantiateProvider() (happydns.ProviderActuator, error) {
	actuator, err := adapter.NewDNSControlProviderAdapter(s)
	if err != nil {
		return nil, err
	}

	return &bindActuator{ProviderActuator: actuator}, nil
}

// bindActuator screens the zone names before they reach dnscontrol, which
// derives from them the name of the file it reads and writes.
type bindActuator struct {
	happydns.ProviderActuator
}

func (a *bindActuator) GetZoneRecords(domain string) ([]happydns.Record, error) {
	if err := checkZoneName(domain); err != nil {
		return nil, err
	}

	return a.ProviderActuator.GetZoneRecords(domain)
}

func (a *bindActuator) GetZoneCorrections(domain string, wantedRecords []happydns.Record) ([]*happydns.Correction, int, error) {
	if err := checkZoneName(domain); err != nil {
		return nil, 0, err
	}

	return a.ProviderActuator.GetZoneCorrections(domain, wantedRecords)
}

func (a *bindActuator) CreateDomain(fqdn string) error {
	if err := checkZoneName(fqdn); err != nil {
		return err
	}

	return a.ProviderActuator.CreateDomain(fqdn)
}

// checkZoneName ensures the zone name can only produce a bare file name: as
// dnscontrol substitutes it in the file name format then joins the result to the
// directory with filepath.Join, which cleans it, a path separator or a .. would
// escape the allowed directory.
//
// This is specific to the BIND provider: for every other provider a zone name
// never reaches a file system, and RFC 2317 classless reverse delegations
// (0/25.2.0.192.in-addr.arpa) are valid domain names that simply cannot be
// stored in a file named after them.
func checkZoneName(domain string) error {
	name := strings.TrimSuffix(domain, ".")

	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("%q cannot be handled by the BIND provider: it derives the name of the zone file from the name of the zone, which therefore cannot contain a path separator", domain)
	}

	switch filepath.Clean(name) {
	case ".", "..":
		return fmt.Errorf("%q is not a valid zone name for the BIND provider", domain)
	}

	return nil
}

func (s *BindServer) ToDNSControlConfig() (map[string]string, error) {
	// This check runs on every instantiation, hence on every zone listing,
	// reading and writing: providers already stored with an arbitrary path are
	// neutralized too, not only the ones going through the settings form.
	directory, err := resolveAllowedDirectory(s.Directory)
	if err != nil {
		return nil, err
	}

	config := map[string]string{
		"directory": directory,
	}

	if s.Fileformat != "" {
		if err := checkFileformat(s.Fileformat); err != nil {
			return nil, err
		}

		config["filenameformat"] = s.Fileformat
	}

	return config, nil
}

// resolveAllowedDirectory canonicalizes the user-given directory and ensures it
// lies inside one of the directories allowed by the instance administrator.
//
// Symlinks are resolved before the comparison, so a symlink planted inside an
// allowed directory cannot be used to escape it. A symlink created after this
// check would not be caught, but allowed directories are expected to be owned
// by the administrator, not by the users of the instance.
func resolveAllowedDirectory(directory string) (string, error) {
	if len(bindAllowedDirectories) == 0 {
		return "", errors.New("no directory has been allowed by the administrator of this instance")
	}

	if directory == "" {
		return "", errors.New("no directory given")
	}

	abs, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("unable to resolve %q: %w", directory, err)
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// Don't leak whether the path exists: outside of the allowed
		// directories, every failure has to look the same.
		return "", errBindDirectoryNotAllowed
	}

	for _, allowed := range bindAllowedDirectories {
		// The allowed directory is canonicalized here rather than at flag
		// parsing time, so that it only has to exist when a zone is actually
		// read or written. One that cannot be resolved allows nothing.
		allowed, err := filepath.EvalSymlinks(allowed)
		if err != nil {
			continue
		}

		if resolved == allowed || strings.HasPrefix(resolved, allowed+string(os.PathSeparator)) {
			return resolved, nil
		}
	}

	return "", errBindDirectoryNotAllowed
}

// checkFileformat ensures the file name format stays a bare file name: as
// dnscontrol joins it to the directory with filepath.Join, which cleans the
// result, any path separator or .. would escape the allowed directory.
func checkFileformat(format string) error {
	if strings.ContainsAny(format, `/\`) {
		return errors.New("the file format cannot contain a path separator")
	}

	switch filepath.Clean(format) {
	case ".", "..":
		return errors.New("the file format is not a valid file name")
	}

	return nil
}

// addBindAllowedDirectory records one directory the BIND provider is allowed to
// use.
//
// Only the shape of the path is validated: a directory that does not exist yet,
// or that is not a directory, is reported as a warning and not as a startup
// failure, because the flag is commonly pointed at a volume mounted after the
// process starts. It allows nothing until it resolves to a directory anyway.
func addBindAllowedDirectory(directory string) error {
	if !filepath.IsAbs(directory) {
		return fmt.Errorf("%q is not an absolute path", directory)
	}

	directory = filepath.Clean(directory)

	if st, err := os.Stat(directory); err != nil {
		log.Printf("with-bind-provider: %q cannot be accessed for now (%s); the BIND provider will refuse it until it can be", directory, err)
	} else if !st.IsDir() {
		log.Printf("with-bind-provider: %q is not a directory; the BIND provider will refuse it until it is", directory)
	}

	bindAllowedDirectories = append(bindAllowedDirectories, directory)

	return nil
}

func registerBindProvider() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &BindServer{}
	}, happydns.ProviderInfos{
		Name:        "Bind files/RFC 1035",
		Description: "Use zone files saved in the RFC 1035 format.",
	}, providerReg.RegisterProvider)
}

// enableBindProvider handles one occurrence of the -with-bind-provider flag.
// The flag both enables the provider and declares the directories it may
// access, so that it can never be enabled without a confinement.
func enableBindProvider(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "true", "1", "on", "yes", "false", "0", "off", "no":
		return fmt.Errorf("with-bind-provider no longer takes a boolean (%q) but the directory holding your zone files, so that the provider is always confined to it: HAPPYDOMAIN_WITH_BIND_PROVIDER=/etc/named/zones, or -with-bind-provider=/etc/named/zones on the command line. Several directories can be given at once, separated by %q (/etc/named/zones%s/srv/zones), or by repeating the flag. See docs/bind-provider.md", value, string(os.PathListSeparator), string(os.PathListSeparator))
	}

	for _, directory := range filepath.SplitList(value) {
		if directory == "" {
			continue
		}

		if err := addBindAllowedDirectory(directory); err != nil {
			return err
		}
	}

	if len(bindAllowedDirectories) == 0 {
		return errors.New("with-bind-provider expects at least one directory")
	}

	if !bindProviderRegistered {
		bindProviderRegistered = true
		registerBindProvider()
	}

	return nil
}

func init() {
	flag.Func("with-bind-provider", "Enable the BIND provider, confined to the given directory, eg. /etc/named/zones (not suitable for cloud/shared instance as it'll access the local file system). No longer a boolean: repeat the flag or separate the paths with "+string(os.PathListSeparator)+" to allow several directories.", enableBindProvider)
}
