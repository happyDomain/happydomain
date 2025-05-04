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

package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/api/controller"
	"git.happydns.org/happyDomain/api/middleware"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type ProviderController struct {
	providerService happydns.ProviderUsecase
	store           storage.Storage
}

func NewProviderController(providerService happydns.ProviderUsecase, store storage.Storage) *ProviderController {
	return &ProviderController{
		providerService,
		store,
	}
}

func (pc *ProviderController) ListProviders(c *gin.Context) {
	user := middleware.MyUser(c)
	if user != nil {
		srcmeta, err := pc.store.ListProviders(user)
		happydns.ApiResponse(c, srcmeta.Metas(), err)
		return
	}

	var providers []*happydns.ProviderMeta

	users, err := pc.store.ListAllUsers()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, fmt.Errorf("unable to list users: %w", err))
		return
	}

	for _, user := range users {
		usersProviders, err := pc.store.ListProviders(user)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, fmt.Errorf("unable to list users: %w", err))
			return
		}

		providers = append(providers, usersProviders.Metas()...)
	}

	happydns.ApiResponse(c, providers, nil)
}

func (pc *ProviderController) AddProvider(c *gin.Context) {
	apidc := controller.NewProviderController(pc.providerService)
	apidc.AddProvider(c)
	return
}

func (pc *ProviderController) DeleteProvider(c *gin.Context) {
	srcMeta := c.MustGet("providermeta").(*happydns.ProviderMeta)

	happydns.ApiResponse(c, true, pc.store.DeleteProvider(srcMeta.Id))
}

func (pc *ProviderController) GetProvider(c *gin.Context) {
	apidc := controller.NewProviderController(pc.providerService)
	apidc.GetProvider(c)
	return
}

func (pc *ProviderController) UpdateProvider(c *gin.Context) {
	apidc := controller.NewProviderController(pc.providerService)
	apidc.UpdateProvider(c)
	return
}

func (pc *ProviderController) ClearProviders(c *gin.Context) {
	user := middleware.MyUser(c)
	if user != nil {
		providers, err := pc.providerService.ListUserProviders(user)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}

		for _, p := range providers {
			e := pc.store.DeleteProvider(p.Id)
			if e != nil {
				err = errors.Join(err, e)
			}
		}

		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusOK)
		return
	}

	happydns.ApiResponse(c, true, pc.store.ClearProviders())
}
