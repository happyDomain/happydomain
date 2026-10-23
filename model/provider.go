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

// ReadOnlyProvider is implemented by provider bodies that can be set to read
// the zone without ever changing it, for instance to monitor a domain with
// credentials that grant no write access.
type ReadOnlyProvider interface {
	IsReadOnly() bool
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

	// Website is the URL of the provider's website, used for favicon fetching.
	Website string `json:"website,omitempty"`
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

// ProviderMinimal is what a client sends to create or update a Provider: the
// server chooses its identifier and owner.
type ProviderMinimal struct {
	// Type is the string representation of the Provider's type.
	Type string `json:"_srctype" binding:"required"`

	// Provider holds the settings of the provider, whose shape depends on
	// the type given above.
	Provider json.RawMessage `json:"Provider" swaggertype:"object"`

	// Comment is a string that helps user to distinguish the Provider.
	Comment string `json:"_comment,omitempty"`
}

// ToMessage returns the ProviderMessage p stands for, without identifier nor
// owner, which are the server's to set.
func (p *ProviderMinimal) ToMessage() *ProviderMessage {
	return &ProviderMessage{
		ProviderMeta: ProviderMeta{
			Type:    p.Type,
			Comment: p.Comment,
		},
		Provider: p.Provider,
	}
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

	// ReadOnly tells whether the Provider never changes the zone. The server
	// sets it from the settings each time the Provider is saved, for display
	// only: checks ask the settings themselves, see Provider.IsReadOnly.
	ReadOnly bool `json:"_readonly,omitempty" readonly:"true"`
}

// ProviderMessage combined ProviderMeta + Provider in a parsable way
type ProviderMessage struct {
	ProviderMeta
	Provider json.RawMessage `json:"Provider" swaggertype:"object"`
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

// IsReadOnly tells whether the settings of the Provider forbid any change to
// the zone.
func (p *Provider) IsReadOnly() bool {
	ro, ok := p.Provider.(ReadOnlyProvider)
	return ok && ro.IsReadOnly()
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
	// RetrieveZone retrieves the records of the zone. known are the
	// subdomains of the current zone, for providers discovering the zone
	// rather than listing it (see DiscoveryZoneGetter).
	RetrieveZone(ctx context.Context, provider *Provider, name string, known ...string) ([]Record, error)
	// ShowReferences replaces, in p, each credential a keystore of
	// references keeps by that reference, which is not secret. p is changed
	// in place: only call it on a provider about to be answered, never stored.
	ShowReferences(context.Context, *Provider) error
	TestDomainExistence(context.Context, *Provider, string) error
	UpdateProvider(context.Context, Identifier, *User, func(*Provider)) error
	UpdateProviderFromMessage(context.Context, Identifier, *User, *ProviderMessage) (*Provider, error)
}

// AdminProviderUsecase exposes administrative provider operations that are
// not scoped to a specific User. Admin callers can list every provider,
// force-delete one by ID, and wipe the table.
type AdminProviderUsecase interface {
	ListAllProviderMetas() ([]*ProviderMeta, error)
	DeleteProviderByID(Identifier) error
	ClearProviders() error
}

type ProviderActuator interface {
	CanCreateDomain() bool
	CanListZones() bool
	CreateDomain(fqdn string) error
	GetZoneRecords(domain string) ([]Record, error)
	GetZoneCorrections(domain string, wantedRecords []Record) ([]*Correction, int, error)
	ListZones() ([]string, error)
}

// DiscoveryZoneGetter is implemented by provider actuators that recompose
// the zone from what they can discover, rather than listing it. They need
// the subdomains found by previous retrievals, so that a name discovered
// once keeps being looked after, and a context, as discovering takes time.
type DiscoveryZoneGetter interface {
	// GetZoneRecordsWithContext is GetZoneRecords, also looking after the
	// known subdomains, relative to domain.
	GetZoneRecordsWithContext(ctx context.Context, domain string, known []string) ([]Record, error)
}

// DomainExistenceChecker is implemented by provider actuators able to tell a
// domain exists far more cheaply than by retrieving its zone, such as the
// ones discovering the zone rather than listing it.
type DomainExistenceChecker interface {
	TestDomainExistence(ctx context.Context, domain string) error
}
