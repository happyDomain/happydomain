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
	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/cscglobal"

	"git.happydns.org/happyDomain/model"
)

type CscGlobalAPI struct {
	ApiKey             string `json:"ApiKey,omitempty" happydomain:"label=API key,placeholder=xxxxxxxx,required,description=Your API key"`
	UserToken          string `json:"UserToken,omitempty" happydomain:"label=User token,placeholder=xxxxxxxx,required,description=Your user token"`
	NotificationEmails string `json:"NotificationEmails,omitempty" happydomain:"label=Notification emails,placeholder=xxxxxxxx,description=Optional comma-separated list of email addresses to send notifications to"`
}

func (s *CscGlobalAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"api-key":             s.ApiKey,
		"user-token":          s.UserToken,
		"notification_emails": s.NotificationEmails,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *CscGlobalAPI) DNSControlName() string {
	return "CSCGLOBAL"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &CscGlobalAPI{}
	}, ProviderInfos{
		Name:        "CSC Global",
		Description: "Corporation Service Company (CSC) provides various business, legal, and financial services.",
	})
}
