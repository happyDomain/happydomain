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

package api

import (
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/forms"
	"git.happydns.org/happyDomain/model"
)

type FormState struct {
	// Id for an already existing element.
	Id *happydns.Identifier `json:"_id,omitempty" swaggertype:"string"`

	// User defined name of the element.
	Name string `json:"_comment"`

	// State is the desired form to shows next (starting at 0).
	State int32 `json:"state"`

	// Recall is the identifier for a saved FormState you want to retrieve.
	Recall string `json:"recall,omitempty"`
}

func formDoState(cfg *config.Options, c *gin.Context, fs *FormState, data interface{}, defaultForm func(interface{}) *forms.CustomForm) (form *forms.CustomForm, d map[string]interface{}, err error) {
	session := c.MustGet("MySession").(*happydns.Session)

	csf, ok := data.(forms.CustomSettingsForm)
	if !ok {
		if fs.State == 1 {
			err = forms.DoneForm
		} else {
			form = defaultForm(data)
		}
		return
	} else {
		return csf.DisplaySettingsForm(fs.State, cfg, session, func() string {
			return fs.Recall
		})
	}
}
