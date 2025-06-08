// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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
	"errors"
	"time"
)

var (
	DomainDoesNotExist = errors.New("domain name doesn't exist")
)

type DomainInfo struct {
	Name           string     `json:"name"`
	Nameservers    []string   `json:"nameservers"`
	CreationDate   *time.Time `json:"creation"`
	ExpirationDate *time.Time `json:"expiration"`
	Registrar      string     `json:"registrar"`
	RegistrarURL   *string    `json:"registrar_url"`
	Status         []string   `json:"status"`
}

type DomainInfoGetter func(Origin) (*DomainInfo, error)

type DomainInfoUsecase interface {
	GetDomainInfo(Origin) (*DomainInfo, error)
}
