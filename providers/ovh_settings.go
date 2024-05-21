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
	"fmt"

	"github.com/gin-contrib/sessions"
	"github.com/ovh/go-ovh/ovh"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/forms"
)

func settingsForm(edit bool) *forms.CustomForm {
	srcFields := []*forms.Field{
		&forms.Field{
			Id:          "endpoint",
			Type:        "string",
			Label:       "Endpoint",
			Default:     "ovh-eu",
			Choices:     []string{"ovh-eu", "ovh-us", "ovh-ca", "soyoustart-eu", "soyoustart-ca", "kimsufi-eu", "kimsufi-ca"},
			Required:    true,
			Description: "The endpoint depends on your service's seller (OVH, SoYouStart, Kimsufi) and the datacenter location (eu, us, ca). Choose 'ovh-eu' if unsure.",
		},
	}

	if edit {
		srcFields = append(srcFields, &forms.Field{
			Id:          "consumerkey",
			Type:        "string",
			Label:       "Consumer Key",
			Placeholder: "xxxxxxxxxx",
			Description: "The consumer key allows us to access your domains' settings without knowing your OVH credentials. To generate a new key, remove the content of this field before continue.",
		})
	}

	form := forms.GenDefaultSettingsForm(nil)
	form.Fields = srcFields
	form.NextButtonText = "common.next"
	return form
}

func settingsAskCredentials(cfg *config.Options, recallid string, session *sessions.Session) (*forms.CustomForm, map[string]interface{}, error) {
	client, err := ovh.NewClient("ovh-eu", appKey, appSecret, "")
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to generate Consumer key, as OVH client can't be created: %w", err)
	}

	// Generate customer key
	ckReq := client.NewCkRequestWithRedirection(cfg.BuildURL_noescape("/providers/new/OVHAPI/2?nsprvid=%s", recallid))
	ckReq.AddRecursiveRules(ovh.ReadWrite, "/domain")
	ckReq.AddRules(ovh.ReadOnly, "/me")

	response, err := ckReq.Do()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to generate Consumer key; OVH returns: %w", err)
	}

	// Return some explanation to the user
	return &forms.CustomForm{
			BeforeText:          "In order allows happyDomain to get and update yours domains, you have to let us access them. To avoid storing your credentials, we will store a unique token that will be associated with your account. For this purpose, you will be redirected to an OVH login screen. The registration will automatically continue",
			NextButtonText:      "Go to OVH",
			PreviousButtonText:  "common.previous",
			NextButtonLink:      response.ValidationURL,
			PreviousButtonState: 0,
		}, map[string]interface{}{
			"consumerkey": response.ConsumerKey,
		}, nil
}

func (s *OVHAPI) DisplaySettingsForm(state int32, cfg *config.Options, session *sessions.Session, genRecallId forms.GenRecallID) (*forms.CustomForm, map[string]interface{}, error) {
	switch state {
	case 0:
		return settingsForm(s.ConsumerKey != ""), nil, nil
	case 1:
		if s.ConsumerKey == "" {
			recallid := genRecallId()
			return settingsAskCredentials(cfg, recallid, session)
		} else {
			return nil, nil, forms.DoneForm
		}
	case 2:
		if s.ConsumerKey == "" {
			return nil, nil, errors.New("Something wierd has happend, as you were not in a consumer key registration process. Please retry.")
		} else {
			return nil, nil, forms.DoneForm
		}
	default:
		return nil, nil, forms.CancelForm
	}
}
