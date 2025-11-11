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

// RegisterProviderFunc abstract the registration of a Provider
type RegisterProviderFunc func(ProviderCreatorFunc, ProviderInfos)

// ProviderCreatorFunc abstract the instanciation of a Provider
type ProviderCreatorFunc func() ProviderBody

// Provider aggregates way of create a Provider and information about it.
type ProviderCreator struct {
	Creator ProviderCreatorFunc
	Infos   ProviderInfos
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

func (p *Provider) Meta() (meta ProviderMeta) {
	meta.UnderscoreId = p.UnderscoreId
	meta.UnderscoreOwnerid = p.UnderscoreOwnerid
	meta.Type = p.Type
	meta.Comment = p.Comment

	return
}

func (p *Provider) SetMeta(meta *ProviderMeta) {
	p.UnderscoreId = meta.UnderscoreId
	p.UnderscoreOwnerid = meta.UnderscoreOwnerid
	p.Type = meta.Type
	p.Comment = meta.Comment
}

func (p *Provider) InstantiateProvider() (ProviderActuator, error) {
	return p.Provider.InstantiateProvider()
}

func (p *Provider) ToMessage() (msg ProviderMessage, err error) {
	msg.ProviderMeta = ProviderMeta{
		UnderscoreId:      p.UnderscoreId,
		UnderscoreOwnerid: p.UnderscoreOwnerid,
		Type:              p.Type,
		Comment:           p.Comment,
	}
	msg.Provider, err = json.Marshal(p.Provider)
	return
}

type ProviderUsecase interface {
	CreateProvider(*User, *ProviderMessage) (*Provider, error)
	CreateDomainOnProvider(*Provider, string) error
	DeleteProvider(*User, Identifier) error
	GetUserProvider(*User, Identifier) (*Provider, error)
	GetUserProviderMeta(*User, Identifier) (*ProviderMeta, error)
	ListHostedDomains(*Provider) ([]string, error)
	ListUserProviders(*User) ([]*ProviderMeta, error)
	ListZoneCorrections(provider *Provider, domain *Domain, records []Record) ([]*FCorrection, error)
	RetrieveZone(*Provider, string) ([]Record, error)
	TestDomainExistence(*Provider, string) error
	UpdateProvider(Identifier, *User, func(*Provider)) error
	UpdateProviderFromMessage(Identifier, *User, *ProviderMessage) error
}

type ProviderActuator interface {
	CanCreateDomain() bool
	CanListZones() bool
	CreateDomain(fqdn string) error
	GetZoneRecords(domain string) ([]Record, error)
	GetZoneCorrections(domain string, wantedRecords []Record) ([]*FCorrection, error)
	ListZones() ([]string, error)
}
