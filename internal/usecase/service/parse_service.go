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

package service

import (
	"encoding/json"

	"git.happydns.org/happyDomain/internal/forms"
	intsvc "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
)

// ParseService deserialises a ServiceMessage into a typed Service value.
// It looks up the concrete ServiceBody type by msg.Type, then JSON-decodes
// msg.Service into it.
func ParseService(msg *happydns.ServiceMessage) (svc *happydns.Service, err error) {
	svc = &happydns.Service{}

	svc.ServiceMeta = msg.ServiceMeta
	svc.Service, err = intsvc.FindService(msg.Type)
	if err != nil {
		return
	}

	err = json.Unmarshal(msg.Service, &svc.Service)
	if err != nil {
		return
	}

	err = forms.ValidateStructValues(svc.Service)
	return
}
