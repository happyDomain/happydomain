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

package usecase

import (
	"strings"
	"time"

	"git.happydns.org/happyDomain/internal/favicon"
	"git.happydns.org/happyDomain/internal/forms"
	providerReg "git.happydns.org/happyDomain/internal/provider"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

type providerSpecsUsecase struct {
	faviconService *favicon.FaviconService
}

func NewProviderSpecsUsecase(faviconService *favicon.FaviconService) happydns.ProviderSpecsUsecase {
	return &providerSpecsUsecase{
		faviconService: faviconService,
	}
}

func (psu *providerSpecsUsecase) ListProviders() map[string]happydns.ProviderInfos {
	srcs := providerReg.GetProviders()

	ret := map[string]happydns.ProviderInfos{}
	for k, src := range srcs {
		ret[k] = src.Infos
	}

	return ret
}

func (psu *providerSpecsUsecase) GetProviderIcon(psid string) ([]byte, string, error) {
	providerType := strings.TrimSuffix(psid, ".png")

	// Try fetching favicon from provider's website
	if psu.faviconService != nil {
		if creator, ok := providerReg.GetProviders()[providerType]; ok && creator.Infos.Website != "" {
			if iconBytes, contentType, err := psu.faviconService.Fetch(creator.Infos.Website, 7*24*time.Hour); err == nil {
				return iconBytes, contentType, nil
			}
		}
	}

	// Fallback to embedded PNG
	cnt, ok := providers.Icons[providerType]
	if !ok {
		return nil, "", happydns.NotFoundError{Msg: "provider icon not found"}
	}

	return cnt, "image/png", nil
}

func (psu *providerSpecsUsecase) GetProviderSpecs(psid string) (*happydns.ProviderSpecs, error) {
	pcreator, ok := providerReg.GetProviders()[psid]
	if !ok {
		return nil, happydns.NotFoundError{Msg: happydns.ErrProviderNotFound.Error()}
	}

	return &happydns.ProviderSpecs{
		Fields:       forms.GenStructFields(pcreator.Creator()),
		Capabilities: pcreator.Infos.Capabilities,
	}, nil
}
