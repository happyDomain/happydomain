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

//go:build listmonk

package actions

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
)

var (
	ListmonkURL config.URL
	ListmonkId  int
)

type ListmonkSubscriber struct {
	Email                   string                 `json:"email"`
	Name                    string                 `json:"name"`
	Status                  string                 `json:"status,omitempty"`
	Lists                   []int                  `json:"lists"`
	Attribs                 map[string]interface{} `json:"attribs,omitempty"`
	PreconfirmSubscriptions bool                   `json:"preconfirm_subscriptions,omitempty"`
}

func init() {
	flag.Var(&ListmonkURL, "newsletter-server-url", "Base URL of the listmonk newsletter server")
	flag.IntVar(&ListmonkId, "newsletter-id", 1, "Listmonk identifier of the list receiving the new user")
}

func SubscribeToNewsletter(u *happydns.User) (err error) {
	if ListmonkURL.URL == nil {
		if ListmonkId != 0 {
			log.Println("SubscribeToNewsletter: not subscribing user as newsletter server is not defined.")
		}
		return nil
	}

	url := ListmonkURL.URL
	url.Path = filepath.Join(url.Path, "api/subscribers")

	jsonForm := &ListmonkSubscriber{
		Email:                   u.Email,
		Name:                    genUsername(u.Email),
		Status:                  "enabled",
		Lists:                   []int{ListmonkId},
		PreconfirmSubscriptions: true,
	}

	j, err := json.Marshal(jsonForm)
	if err != nil {
		log.Printf("SubscribeToNewsletter: unable to encode the first request body: %s", err.Error())
		return fmt.Errorf("an error occured when trying to subscribe to the newsletter. Please try again later.")
	}
	req, err := http.NewRequest("POST", url.String(), bytes.NewReader(j))
	if err != nil {
		log.Printf("SubscribeToNewsletter: unable to create the first request: %s", err.Error())
		return fmt.Errorf("an error occured when trying to subscribe to the newsletter. Please try again later.")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("SubscribeToNewsletter: unable to perform the first request body: %s", err.Error())
		return fmt.Errorf("an error occured when trying to subscribe to the newsletter. Please try again later.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var msg map[string]string
		dec := json.NewDecoder(resp.Body)
		dec.Decode(&msg)

		log.Printf("SubscribeToNewsletter: unable to perform the first request body: %s", msg["message"])
		return fmt.Errorf("an error occured when trying to subscribe to the newsletter. Please try again later.")
	}

	return
}
