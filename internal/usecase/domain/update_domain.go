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
	"git.happydns.org/happyDomain/model"
)

type UpdateDomainUsecase struct {
	domainLogAppender domainLogUC.DomainLogAppender
	getDomain         *GetDomainUsecase
	store             DomainStorage
}

func NewUpdateDomainUsecase(store DomainStorage, getDomain *GetDomainUsecase, domainLogAppender domainLogUC.DomainLogAppender) *UpdateDomainUsecase {
	return &UpdateDomainUsecase{
		domainLogAppender: domainLogAppender,
		getDomain:         getDomain,
		store:             store,
	}
}

func (uc *UpdateDomainUsecase) Update(domainid happydns.Identifier, user *happydns.User, updateFn func(*happydns.Domain)) error {
	domain, err := uc.getDomain.ByID(user, domainid)
	if err != nil {
		return err
	}

	updateFn(domain)
	//domain.ModifiedOn = time.Now()

	if !domain.Id.Equals(domainid) {
		return happydns.ValidationError{Msg: "you cannot change the domain identifier"}
	}

	err = uc.store.UpdateDomain(domain)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateDomain in UpdateDomain: %w", err),
			UserMessage: "Sorry, we are currently unable to update your domain. Please retry later.",
		}
	}

	// Add a log entry
	if uc.domainLogAppender != nil {
		uc.domainLogAppender.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Domain name %s properties changed.", domain.DomainName)))
	}

	return nil
}
