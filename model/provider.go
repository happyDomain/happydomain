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

package happydns

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	dnscontrol "github.com/StackExchange/dnscontrol/v4/providers"

	"git.happydns.org/happyDomain/providers"
)

// ProviderMinimal is used for swagger documentation as Provider add.
type ProviderMinimal struct {
	// Type is the string representation of the Provider's type.
	Type string `json:"_srctype"`

	Provider providers.Provider

	// Comment is a string that helps user to distinguish the Provider.
	Comment string `json:"_comment,omitempty"`
}

// ProviderMeta holds the metadata associated to a Provider.
type ProviderMeta struct {
	// Type is the string representation of the Provider's type.
	Type string `json:"_srctype"`

	// Id is the Provider's identifier.
	Id Identifier `json:"_id" swaggertype:"string"`

	// OwnerId is the User's identifier for the current Provider.
	OwnerId Identifier `json:"_ownerid" swaggertype:"string"`

	// Comment is a string that helps user to distinguish the Provider.
	Comment string `json:"_comment,omitempty"`
}

// ProviderMessage combined ProviderMeta + Provider in a parsable way
type ProviderMessage struct {
	ProviderMeta
	Provider json.RawMessage
}

func (msg *ProviderMessage) Meta() *ProviderMeta {
	return &msg.ProviderMeta
}

func (msg *ProviderMessage) ParseProvider() (p *ProviderCombined, err error) {
	p = &ProviderCombined{}

	p.ProviderMeta = msg.ProviderMeta
	p.Provider, err = providers.FindProvider(msg.Type)
	if err != nil {
		return
	}

	err = json.Unmarshal(msg.Provider, &p.Provider)
	return
}

type ProviderMessages []*ProviderMessage

func (pms *ProviderMessages) Metas() (ret []*ProviderMeta) {
	for _, pm := range *pms {
		ret = append(ret, &pm.ProviderMeta)
	}
	return
}

// ProviderCombined combined ProviderMeta + Provider
type ProviderCombined struct {
	ProviderMeta
	providers.Provider
}

func (p *ProviderCombined) ToMessage() (msg ProviderMessage, err error) {
	msg.ProviderMeta = p.ProviderMeta
	msg.Provider, err = json.Marshal(p.Provider)
	return
}

// Validate ensure the given parameters are corrects.
func (p *ProviderCombined) Validate() error {
	prv, err := p.NewDNSServiceProvider()
	if err != nil {
		return err
	}

	sr, ok := prv.(dnscontrol.ZoneLister)
	if ok {
		_, err = sr.ListZones()
	}

	return err
}

func (p *ProviderCombined) getZoneRecords(fqdn string) (rcs models.Records, err error) {
	var s dnscontrol.DNSServiceProvider
	s, err = p.NewDNSServiceProvider()
	if err != nil {
		return
	}

	defer func() {
		if a := recover(); a != nil {
			err = fmt.Errorf("%s", a)
		}
	}()

	return s.GetZoneRecords(strings.TrimSuffix(fqdn, "."), nil)
}

func (p *ProviderCombined) DomainExists(fqdn string) (err error) {
	_, err = p.getZoneRecords(fqdn)
	if err != nil {
		return
	}

	return nil
}

func (p *ProviderCombined) ImportZone(dn *Domain) (rcs models.Records, err error) {
	return p.getZoneRecords(dn.DomainName)
}

func (p *ProviderCombined) GetDomainCorrections(dn *Domain, dc *models.DomainConfig) (rrs []*models.Correction, err error) {
	var s dnscontrol.DNSServiceProvider
	s, err = p.NewDNSServiceProvider()
	if err != nil {
		return
	}

	defer func() {
		if a := recover(); a != nil {
			err = fmt.Errorf("%s", a)
		}
	}()

	rcs, err := p.getZoneRecords(dn.DomainName)

	return s.GetZoneRecordsCorrections(dc, rcs)
}
