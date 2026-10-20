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

package web

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"
)

type noCaptcha struct{}

func (noCaptcha) Provider() string            { return "" }
func (noCaptcha) SiteKey() string             { return "" }
func (noCaptcha) Verify(string, string) error { return nil }

// injectedAppConfig runs DeclareRoutes and returns the app-config JSON it
// injected in the page head.
func injectedAppConfig(t *testing.T) map[string]any {
	t.Helper()

	saved := CustomHeadHTML
	CustomHeadHTML = ""
	t.Cleanup(func() { CustomHeadHTML = saved })

	gin.SetMode(gin.TestMode)
	DeclareRoutes(&happydns.Options{}, gin.New().Group("/"), noCaptcha{})

	m := regexp.MustCompile(`(?s)<script id="app-config" type="application/json">(.*)</script>`).FindStringSubmatch(CustomHeadHTML)
	if m == nil {
		t.Fatalf("no app-config injected in %q", CustomHeadHTML)
	}

	var cfg map[string]any
	if err := json.Unmarshal([]byte(m[1]), &cfg); err != nil {
		t.Fatalf("invalid app-config JSON: %s", err)
	}
	return cfg
}

func TestDeclareRoutes_MailAutoconfigHost(t *testing.T) {
	t.Cleanup(func() { abstract.SetAutoconfigHost("") })

	abstract.SetAutoconfigHost("happydomain.example.com.")
	if got := injectedAppConfig(t)["mail_autoconfig_host"]; got != "happydomain.example.com" {
		t.Errorf("mail_autoconfig_host = %v, want the host without its trailing dot", got)
	}
}

func TestDeclareRoutes_NoMailAutoconfigHost(t *testing.T) {
	abstract.SetAutoconfigHost("")
	if got, ok := injectedAppConfig(t)["mail_autoconfig_host"]; ok {
		t.Errorf("mail_autoconfig_host = %v, want the key absent when no host is configured", got)
	}
}
