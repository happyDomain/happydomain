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
	"context"
	"encoding/json"
)

// ProviderBody is where Domains and Zones can be managed.
type ProviderBody interface {
	InstantiateProvider() (ProviderActuator, error)
}

// ProviderInfos describes the purpose of a user usable provider.
type ProviderInfos struct {
	// Name is the name displayed.
	Name string `json:"name" binding:"required"`

	// Description is a brief description of what the provider is.
	Description string `json:"description" binding:"required"`

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

	Provider Provider `json:"Provider"`

	// Comment is a string that helps user to distinguish the Provider.
	Comment string `json:"_comment,omitempty"`
}

// ProviderMeta holds the metadata associated to a Provider.
type ProviderMeta struct {
	// Type is the string representation of the Provider's type.
	Type string `json:"_srctype" binding:"required"`

	// Id is the Provider's identifier.
	Id Identifier `json:"_id" swaggertype:"string" binding:"required" readonly:"true"`

	// Owner is the User's identifier for the current Provider.
	Owner Identifier `json:"_ownerid" swaggertype:"string" binding:"required" readonly:"true"`

	// Comment is a string that helps user to distinguish the Provider.
	Comment string `json:"_comment,omitempty"`
}

// ProviderMessage combined ProviderMeta + Provider in a parsable way
type ProviderMessage struct {
	ProviderMeta
	Provider json.RawMessage `json:"Provider"`
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
	Provider ProviderBody `json:"Provider"`
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
	CreateProvider(context.Context, *User, *ProviderMessage) (*Provider, error)
	CreateDomainOnProvider(context.Context, *Provider, string) error
	DeleteProvider(context.Context, *User, Identifier) error
	GetUserProvider(context.Context, *User, Identifier) (*Provider, error)
	GetUserProviderMeta(context.Context, *User, Identifier) (*ProviderMeta, error)
	ListHostedDomains(context.Context, *Provider) ([]string, error)
	ListUserProviders(context.Context, *User) ([]*ProviderMeta, error)
	ListZoneCorrections(context.Context, *Provider, *Domain, []Record) ([]*Correction, int, error)
	RetrieveZone(context.Context, *Provider, string) ([]Record, error)
	TestDomainExistence(context.Context, *Provider, string) error
	UpdateProvider(context.Context, Identifier, *User, func(*Provider)) error
	UpdateProviderFromMessage(context.Context, Identifier, *User, *ProviderMessage) error
}

type ProviderActuator interface {
	CanCreateDomain() bool
	CanListZones() bool
	CreateDomain(fqdn string) error
	GetZoneRecords(domain string) ([]Record, error)
	GetZoneCorrections(domain string, wantedRecords []Record) ([]*Correction, int, error)
	ListZones() ([]string, error)
}
