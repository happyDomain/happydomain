// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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

package ovh // import "happydns.org/sources/ovh"

import (
	"errors"
	"fmt"

	"github.com/ovh/go-ovh/ovh"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
)

const (
	SESSION_CKEY = "ovh-consumerkey"
)

func settingsForm(edit bool) *sources.CustomForm {
	srcFields := []*sources.SourceField{
		&sources.SourceField{
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
		srcFields = append(srcFields, &sources.SourceField{
			Id:          "consumerkey",
			Type:        "string",
			Label:       "Consumer Key",
			Placeholder: "xxxxxxxxxx",
			Description: "The consumer key allows us to access your domains' settings without knowing your OVH credentials. To generate a new key, remove the content of this field before continue.",
		})
	}

	form := sources.GenDefaultSettingsForm(nil)
	form.Fields = srcFields
	form.NextButtonText = "Next >"
	return form
}

func settingsAskCredentials(cfg *config.Options, recallid int64, endpoint string, session *happydns.Session) (*sources.CustomForm, error) {
	client, err := ovh.NewClient(endpoint, appKey, appSecret, "")
	if err != nil {
		return nil, fmt.Errorf("Unable to generate Consumer key, as OVH client can't be created: %w", err)
	}

	// Generate customer key
	ckReq := client.NewCkRequestWithRedirection(cfg.BuildURL_noescape("/sources/new/ovh.OVHAPI/2?recall=%d", recallid))
	ckReq.AddRecursiveRules(ovh.ReadWrite, "/domain")
	ckReq.AddRules(ovh.ReadOnly, "/me")

	response, err := ckReq.Do()
	if err != nil {
		return nil, fmt.Errorf("Unable to generate Consumer key; OVH returns: %w", err)
	}

	// Store the key in user's session
	session.SetValue(SESSION_CKEY, response.ConsumerKey)

	// Return some explanation to the user
	return &sources.CustomForm{
		BeforeText:          "In order allows happyDNS to get and update yours domains, you have to let us access them. To avoid storing your credentials, we will store a unique token that will be associated with your account. For this purpose, you will be redirected to an OVH login screen. The registration will automatically continue",
		NextButtonText:      "Go to OVH",
		PreviousButtonText:  "< Previous",
		NextButtonLink:      response.ValidationURL,
		PreviousButtonState: 0,
	}, nil
}

func (s *OVHAPI) DisplaySettingsForm(state int32, cfg *config.Options, session *happydns.Session, genRecallId sources.GenRecallID) (*sources.CustomForm, error) {
	switch state {
	case 0:
		return settingsForm(s.ConsumerKey != ""), nil
	case 1:
		if s.Endpoint == "" {
			return nil, errors.New("You need to fill the End Point. Choose 'ovh-eu' if you're unsure.")
		} else if s.ConsumerKey == "" {
			recallid := genRecallId()
			return settingsAskCredentials(cfg, recallid, s.Endpoint, session)
		} else {
			return nil, sources.DoneForm
		}
	case 2:
		var consumerKey string
		if s.Endpoint == "" {
			return nil, errors.New("You need to fill the End Point. Choose 'ovh-eu' if you're unsure.")
		} else if ok := session.GetValue(SESSION_CKEY, &consumerKey); !ok {
			return nil, errors.New("Something wierd has happend, as you were not in a consumer key registration process. Please retry.")
		} else {
			s.ConsumerKey = consumerKey
			session.DropKey(SESSION_CKEY)
			return nil, sources.DoneForm
		}
	default:
		return nil, sources.CancelForm
	}
}
