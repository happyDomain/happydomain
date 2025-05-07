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

package newsletter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

type ListmonkNewsletterSubscription struct {
	ListmonkURL *url.URL
	ListmonkId  int
}

type ListmonkSubscriber struct {
	Email                   string                 `json:"email"`
	Name                    string                 `json:"name"`
	Status                  string                 `json:"status,omitempty"`
	Lists                   []int                  `json:"lists"`
	Attribs                 map[string]interface{} `json:"attribs,omitempty"`
	PreconfirmSubscriptions bool                   `json:"preconfirm_subscriptions,omitempty"`
}

func (ns *ListmonkNewsletterSubscription) SubscribeToNewsletter(u happydns.UserInfo) error {
	if ns.ListmonkId == 0 {
		log.Println("SubscribeToNewsletter: not subscribing user as newsletter list id is not defined.")
		return nil
	}

	url := *ns.ListmonkURL
	url.Path = filepath.Join(url.Path, "api/subscribers")

	jsonForm := &ListmonkSubscriber{
		Email:                   u.GetEmail(),
		Name:                    helpers.GenUsername(u.GetEmail()),
		Status:                  "enabled",
		Lists:                   []int{ns.ListmonkId},
		PreconfirmSubscriptions: true,
	}

	j, err := json.Marshal(jsonForm)
	if err != nil {
		return fmt.Errorf("unable to encode the first request body: %s", err.Error())
	}
	req, err := http.NewRequest("POST", url.String(), bytes.NewReader(j))
	if err != nil {
		return fmt.Errorf("unable to create the first request: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to perform the first request body: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var msg map[string]string
		dec := json.NewDecoder(resp.Body)
		dec.Decode(&msg)

		return fmt.Errorf("unable to perform the first request body: %s", msg["message"])
	}

	return nil
}
