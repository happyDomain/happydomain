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

package domain

import (
	"fmt"

	domainLogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

type CreateDomainUsecase struct {
	domainExistence   *providerUC.DomainExistenceUsecase
	domainLogAppender *domainLogUC.CreateDomainLogUsecase
	getProvider       *providerUC.GetProviderUsecase
	store             DomainStorage
}

func NewCreateDomainUsecase(store DomainStorage, getProvider *providerUC.GetProviderUsecase, domainExistence *providerUC.DomainExistenceUsecase, domainLogAppender *domainLogUC.CreateDomainLogUsecase) *CreateDomainUsecase {
	return &CreateDomainUsecase{
		domainExistence:   domainExistence,
		domainLogAppender: domainLogAppender,
		getProvider:       getProvider,
		store:             store,
	}
}

func (uc *CreateDomainUsecase) Create(user *happydns.User, uz *happydns.Domain) error {
	uz, err := happydns.NewDomain(user, uz.DomainName, uz.ProviderId)
	if err != nil {
		return err
	}

	provider, err := uc.getProvider.Get(user, uz.ProviderId)
	if err != nil {
		return happydns.ValidationError{Msg: fmt.Sprintf("unable to find the provider.")}
	}

	if err = uc.domainExistence.TestDomainExistence(provider, uz.DomainName); err != nil {
		return happydns.NotFoundError{Msg: err.Error()}
	}

	if err := uc.store.CreateDomain(uz); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateDomain: %s", err),
			UserMessage: "Sorry, we are unable to create your domain now.",
		}
	}

	// Add a log entry
	if uc.domainLogAppender != nil {
		uc.domainLogAppender.Create(uz, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Domain name %s added.", uz.DomainName)))
	}

	return nil
}
