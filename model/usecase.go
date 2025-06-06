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

type UsecaseDependancies interface {
	AuthenticationUsecase() AuthenticationUsecase
	AuthUserUsecase() AuthUserUsecase
	DomainUsecase() DomainUsecase
	DomainLogUsecase() DomainLogUsecase
	ProviderUsecase(secure bool) ProviderUsecase
	ProviderSettingsUsecase() ProviderSettingsUsecase
	ProviderSpecsUsecase() ProviderSpecsUsecase
	RemoteZoneImporterUsecase() RemoteZoneImporterUsecase
	ResolverUsecase() ResolverUsecase
	ServiceUsecase() ServiceUsecase
	ServiceSpecsUsecase() ServiceSpecsUsecase
	SessionUsecase() SessionUsecase
	UserUsecase() UserUsecase
	ZoneCorrectionApplierUsecase() ZoneCorrectionApplierUsecase
	ZoneImporterUsecase() ZoneImporterUsecase
	ZoneServiceUsecase() ZoneServiceUsecase
	ZoneUsecase() ZoneUsecase
}
