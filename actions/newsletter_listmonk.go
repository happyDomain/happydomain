// Copyright or © or Copr. happyDNS (2023)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

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
