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
	"time"

	"git.happydns.org/happyDomain/internal/api/controller"
	insightUC "git.happydns.org/happyDomain/internal/usecase/insight"
	"git.happydns.org/happyDomain/model"
)

const (
	InsightsUpdateInterval = 24 * time.Hour
	InsightsEndpoint       = "https://insights.happydomain.org/collect"

	insightsHTTPTimeout = 10 * time.Second
)

// allowedBuildSettings lists the Go build settings that are safe to include in
// telemetry. Other settings (e.g. -ldflags values, module paths) are omitted.
var allowedBuildSettings = map[string]struct{}{
	"CGO_ENABLED":  {},
	"GOARCH":       {},
	"GOAMD64":      {},
	"GOARM":        {},
	"GOMIPS":       {},
	"GOOS":         {},
	"vcs":          {},
	"vcs.time":     {},
	"vcs.modified": {},
}

type insightsCollector struct {
	cfg   *happydns.Options
	store insightUC.CollectStorage
	stop  chan struct{}
}

func (c *insightsCollector) Close() {
	c.stop <- struct{}{}
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

func (c *insightsCollector) send() error {
	_, instance, err := c.store.LastInsightsRun()
	if err != nil {
		return fmt.Errorf("could not retrieve instance ID: %w", err)
	}

	buildSettings, goVersion := buildInfo()

	data, err := insightUC.Collect(c.cfg, c.store, instance.String(), controller.HDVersion, buildSettings, goVersion)
	if err != nil {
		return err
	}

	dataenc, err := json.Marshal(data)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: insightsHTTPTimeout}
	resp, err := client.Post(InsightsEndpoint, "application/json", bytes.NewReader(dataenc))
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
			if _, allowed := allowedBuildSettings[setting.Key]; !allowed {
				continue
			}
			bInfo[setting.Key] = setting.Value
		}
		version = info.GoVersion
	}
	// Include runtime arch/OS even without build info
	bInfo["runtime.GOOS"] = runtime.GOOS
	bInfo["runtime.GOARCH"] = runtime.GOARCH
	return bInfo, version
}
