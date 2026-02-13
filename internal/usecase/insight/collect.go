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

package insight

import (
	"strings"

	"git.happydns.org/happyDomain/model"
)

// CollectStorage defines the storage methods required for insights collection.
type CollectStorage interface {
	InsightStorage

	// SchemaVersion returns the version of the migration currently in use.
	SchemaVersion() int

	// ListAllAuthUsers retrieves all auth users in the database.
	ListAllAuthUsers() (happydns.Iterator[happydns.UserAuth], error)

	// ListAllUsers retrieves all users in the database.
	ListAllUsers() (happydns.Iterator[happydns.User], error)

	// ListProviders retrieves all providers owned by the given User.
	ListProviders(user *happydns.User) (happydns.ProviderMessages, error)

	// ListDomains retrieves all domains owned by the given User.
	ListDomains(u *happydns.User) ([]*happydns.Domain, error)
}

// Collect gathers anonymous usage statistics about the running instance.
func Collect(cfg *happydns.Options, store CollectStorage, instanceID string, version happydns.VersionResponse, buildSettings map[string]string, goVersion string) (*happydns.Insights, error) {
	data := happydns.Insights{
		InsightsID: instanceID,
		Version:    version,
	}

	// Build info
	data.Build.Settings = buildSettings
	data.Build.GoVersion = goVersion

	// Config info
	data.Config.DisableEmbeddedLogin = cfg.DisableEmbeddedLogin
	data.Config.DisableProviders = cfg.DisableProviders
	data.Config.DisableRegistration = cfg.DisableRegistration
	data.Config.HasBaseURL = cfg.BasePath != ""
	data.Config.HasDevProxy = cfg.DevProxy != ""
	data.Config.HasExternalAuth = cfg.ExternalAuth.String() != ""
	data.Config.HasListmonkURL = cfg.ListmonkURL.String() != ""
	data.Config.LocalBind = strings.HasPrefix(cfg.Bind, "127.0.0.1:") || strings.HasPrefix(cfg.Bind, "[::1]:")
	data.Config.NbOidcProviders = len(cfg.OIDCClients)
	data.Config.NoAuthActive = cfg.NoAuth
	data.Config.NoMail = cfg.NoMail
	data.Config.NonUnixAdminBind = strings.Contains(cfg.AdminBind, ":")
	data.Config.StorageEngine = string(cfg.StorageEngine)

	// Database info
	data.Database.Version = store.SchemaVersion()

	if authusers, err := store.ListAllAuthUsers(); err != nil {
		return nil, err
	} else {
		for authusers.Next() {
			data.Database.NbAuthUsers++
		}
	}

	users, err := store.ListAllUsers()
	if err != nil {
		return nil, err
	}

	data.Database.Providers = map[string]int{}
	data.UserSettings.Languages = map[string]int{}
	data.UserSettings.FieldHints = map[int]int{}
	data.UserSettings.ZoneView = map[int]int{}
	for users.Next() {
		data.Database.NbUsers++

		user := users.Item()

		if user.Settings.Language != "" {
			data.UserSettings.Languages[user.Settings.Language]++
		}
		if user.Settings.Newsletter {
			data.UserSettings.Newsletter++
		}
		data.UserSettings.FieldHints[user.Settings.FieldHint]++
		data.UserSettings.ZoneView[user.Settings.ZoneView]++

		if providers, err := store.ListProviders(user); err == nil {
			for _, provider := range providers {
				data.Database.Providers[provider.Type] += 1
			}
		}

		if domains, err := store.ListDomains(user); err == nil {
			data.Database.NbDomains += len(domains)

			for _, domain := range domains {
				data.Database.NbZones += len(domain.ZoneHistory)
			}
		}
	}

	return &data, nil
}
