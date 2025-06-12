// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

const (
	InsightsUpdateInterval = 24 * time.Hour
	InsightsEndpoint       = "https://insights.happydomain.org/collect"
)

type insightsCollector struct {
	cfg   *happydns.Options
	store storage.Storage
	stop  chan bool
}

func (c *insightsCollector) Close() {
	c.stop <- true
}

func (c *insightsCollector) Run() {
	if t, ok := c.LastRun(); !ok {
		select {
		case <-time.After(time.Hour):
			break
		case <-c.stop:
			return
		}
	} else {
		select {
		case <-time.After(time.Until(t.Add(InsightsUpdateInterval))):
			break
		case <-c.stop:
			return
		}
	}

	var err error
	var nextInterval time.Duration
	for {
		err = c.send()
		if err != nil {
			log.Println("Unable to send insights:", err.Error())
			nextInterval = time.Duration(90-rand.Int31n(45)) * time.Minute
		} else {
			nextInterval = InsightsUpdateInterval + time.Duration(rand.Int31n(600000)-300000)*time.Microsecond
		}

		select {
		case <-time.After(nextInterval):
			continue
		case <-c.stop:
			return
		}
	}
}

func (c *insightsCollector) LastRun() (time.Time, bool) {
	timestamp, _, err := c.store.LastInsightsRun()
	if err != nil || timestamp == nil {
		return time.Time{}, false
	}

	return *timestamp, true
}

func (c *insightsCollector) collect() (*happydns.Insights, error) {
	_, instance, _ := c.store.LastInsightsRun()

	// Basic info
	data := happydns.Insights{
		InsightsID: instance.String(),
		Version:    controller.HDVersion,
	}

	// Build info
	data.Build.Settings, data.Build.GoVersion = buildInfo()

	// OS info
	data.OS.Type = runtime.GOOS
	data.OS.Arch = runtime.GOARCH
	data.OS.NumCPU = runtime.NumCPU()

	// Config info
	data.Config.DisableEmbeddedLogin = c.cfg.DisableEmbeddedLogin
	data.Config.DisableProviders = c.cfg.DisableProviders
	data.Config.DisableRegistration = c.cfg.DisableRegistration
	data.Config.HasBaseURL = c.cfg.BasePath != ""
	data.Config.HasDevProxy = c.cfg.DevProxy != ""
	data.Config.HasExternalAuth = c.cfg.ExternalAuth.String() != ""
	data.Config.HasListmonkURL = c.cfg.ListmonkURL.String() != ""
	data.Config.LocalBind = strings.HasPrefix(c.cfg.Bind, "127.0.0.1:") || strings.HasPrefix(c.cfg.Bind, "[::1]:")
	data.Config.NbOidcProviders = len(c.cfg.OIDCClients)
	data.Config.NoAuthActive = c.cfg.NoAuth
	data.Config.NoMail = c.cfg.NoMail
	data.Config.NonUnixAdminBind = strings.Contains(c.cfg.AdminBind, ":")
	data.Config.StorageEngine = string(c.cfg.StorageEngine)

	// Database info
	data.Database.Version = c.store.SchemaVersion()

	if authusers, err := c.store.ListAllAuthUsers(); err != nil {
		return nil, err
	} else {
		for authusers.Next() {
			data.Database.NbAuthUsers++
		}
	}

	users, err := c.store.ListAllUsers()
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

		if providers, err := c.store.ListProviders(user); err == nil {
			for _, provider := range providers {
				data.Database.Providers[provider.Type] += 1
			}
		}

		if domains, err := c.store.ListDomains(user); err == nil {
			data.Database.NbDomains += len(domains)

			for _, domain := range domains {
				data.Database.NbZones += len(domain.ZoneHistory)
			}
		}
	}

	return &data, nil
}

func (c *insightsCollector) send() error {
	data, err := c.collect()
	if err != nil {
		return err
	}

	dataenc, err := json.Marshal(data)
	if err != nil {
		return err
	}

	resp, err := http.Post(InsightsEndpoint, "application/json", bytes.NewReader(dataenc))
	if err != nil {
		return fmt.Errorf("could not send insights: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected response status code: %s", resp.Status)
	}
	return c.store.InsightsRun()
}

func buildInfo() (map[string]string, string) {
	bInfo := map[string]string{}
	var version string
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Value == "" {
				continue
			}
			bInfo[setting.Key] = setting.Value
		}
		version = info.GoVersion
	}
	return bInfo, version
}
