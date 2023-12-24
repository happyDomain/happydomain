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

package config // import "git.happydns.org/happyDomain/config"

import (
	"flag"
	"fmt"

	"git.happydns.org/happyDomain/storage"
)

// declareFlags registers flags for the structure Options.
func (o *Options) declareFlags() {
	flag.StringVar(&o.DevProxy, "dev", o.DevProxy, "Proxify traffic to this host for static assets")
	flag.StringVar(&o.AdminBind, "admin-bind", o.AdminBind, "Bind port/socket for administration interface")
	flag.StringVar(&o.Bind, "bind", ":8081", "Bind port/socket")
	flag.Var(&o.ExternalURL, "externalurl", "Begining of the URL, before the base, that should be used eg. in mails")
	flag.StringVar(&o.BaseURL, "baseurl", o.BaseURL, "URL prepended to each URL")
	flag.StringVar(&o.DefaultNameServer, "default-ns", o.DefaultNameServer, "Adress to the default name server")
	flag.Var(&o.StorageEngine, "storage-engine", fmt.Sprintf("Select the storage engine between %v", storage.GetStorageEngines()))
	flag.BoolVar(&o.NoAuth, "no-auth", false, "Disable user access control, use default account")
	flag.Var(&o.JWTSecretKey, "jwt-secret-key", "Secret key used to verify JWT authentication tokens (a random secret is used if undefined)")
	flag.Var(&o.ExternalAuth, "external-auth", "Base URL to use for login and registration (use embedded forms if left empty)")
	flag.Var(&o.OryKratosServer, "ory-kratos-server", "URL to the Ory Kratos server (default: none, use classical auth)")

	// Others flags are declared in some other files likes sources, storages, ... when they need specials configurations
}

// parseCLI parse the flags and treats extra args as configuration filename.
func (o *Options) parseCLI() error {
	flag.Parse()

	for _, conf := range flag.Args() {
		err := o.parseFile(conf)
		if err != nil {
			return err
		}
	}

	return nil
}
