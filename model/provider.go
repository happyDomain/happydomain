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
)

// ProviderBody is where Domains and Zones can be managed.
type ProviderBody interface {
	InstantiateProvider() (ProviderActuator, error)
}

// ProviderInfos describes the purpose of a user usable provider.
type ProviderInfos struct {
	// Name is the name displayed.
	Name string `json:"name"`

	// Description is a brief description of what the provider is.
	Description string `json:"description"`

	// Capabilites is a list of special ability of the provider (automatically filled).
	Capabilities []string `json:"capabilities,omitempty"`

	// HelpLink is the link to the documentation of the provider configuration.
	HelpLink string `json:"helplink,omitempty"`
}

// RegisterProviderFunc abstract the registration of a Provider
type RegisterProviderFunc func(ProviderCreatorFunc, ProviderInfos)

// ProviderCreatorFunc abstract the instanciation of a Provider
type ProviderCreatorFunc func() ProviderBody

// Provider aggregates way of create a Provider and information about it.
type ProviderCreator struct {
	Creator ProviderCreatorFunc
	Infos   ProviderInfos
}

// ProviderMinimal is used for swagger documentation as Provider add.
type ProviderMinimal struct {
	// Type is the string representation of the Provider's type.
	Type string `json:"_srctype"`

	Provider Provider

	// Comment is a string that helps user to distinguish the Provider.
	Comment string `json:"_comment,omitempty"`
}

// ProviderMeta holds the metadata associated to a Provider.
type ProviderMeta struct {
	// Type is the string representation of the Provider's type.
	Type string `json:"_srctype"`

	// Id is the Provider's identifier.
	Id Identifier `json:"_id" swaggertype:"string"`

	// Owner is the User's identifier for the current Provider.
	Owner Identifier `json:"_ownerid" swaggertype:"string"`

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

type ProviderMessages []*ProviderMessage

func (pms *ProviderMessages) Metas() (ret []*ProviderMeta) {
	for _, pm := range *pms {
		ret = append(ret, &pm.ProviderMeta)
	}
	return
}

// ProviderCombined combined ProviderMeta + Provider
type Provider struct {
	ProviderMeta
	Provider ProviderBody
}

func (p *Provider) InstantiateProvider() (ProviderActuator, error) {
	return p.Provider.InstantiateProvider()
}

func (p *Provider) ToMessage() (msg ProviderMessage, err error) {
	msg.ProviderMeta = p.ProviderMeta
	msg.Provider, err = json.Marshal(p.Provider)
	return
}

func (p *Provider) Meta() *ProviderMeta {
	return &p.ProviderMeta
}

type ProviderUsecase interface {
	CreateProvider(*User, *ProviderMessage) (*Provider, error)
	DeleteProvider(*User, Identifier) error
	GetUserProvider(*User, Identifier) (*Provider, error)
	GetUserProviderMeta(*User, Identifier) (*ProviderMeta, error)
	GetZoneCorrections(provider *Provider, domain *Domain, records []Record) ([]*Correction, error)
	ListUserProviders(*User) ([]*ProviderMeta, error)
	RetrieveZone(*Provider, string) ([]Record, error)
	TestDomainExistence(*Provider, string) error
	UpdateProvider(Identifier, *User, func(*Provider)) error
	UpdateProviderFromMessage(Identifier, *User, *ProviderMessage) error
	ValidateProvider(*Provider) error
}

type ProviderActuator interface {
	GetZoneRecords(domain string) ([]Record, error)
	GetZoneCorrections(domain string, wantedRecords []Record) ([]*Correction, error)
}

type ZoneListerActuator interface {
	ListZones() ([]string, error)
}
