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

import ()

type FieldHint uint8

const (
	FieldHintHide    FieldHint = iota
	FieldHintTooltip
	FieldHintFocused
	FieldHintAlways
)

type ZoneView uint8

const (
	ZoneViewGrid    ZoneView = iota
	ZoneViewList
	ZoneViewRecords
)

type ApplyConfirm uint8

const (
	ApplyConfirmUnexpected ApplyConfirm = iota
	ApplyConfirmAlways
	ApplyConfirmNever
)

// UserSettings represents the settings for an account.
type UserSettings struct {
	// Language saves the locale defined by the user.
	Language string `json:"language,omitempty"`

	// Newsletter indicates wether the user wants to receive the newsletter or not.
	Newsletter bool `json:"newsletter,omitempty"`

	// FieldHint stores the way form hints are displayed.
	FieldHint FieldHint `json:"fieldhint"`

	// ZoneView keeps the view of the zone wanted by the user.
	ZoneView ZoneView `json:"zoneview"`

	// ApplyConfirm stores when to show a confirmation step before applying changes.
	ApplyConfirm ApplyConfirm `json:"applyconfirm"`

	// ShowRRTypes tells if we show equivalent RRTypes in interface (for advanced users).
	ShowRRTypes bool `json:"showrrtypes,omitempty"`
}

func DefaultUserSettings() *UserSettings {
	return &UserSettings{
		Language:     "en",
		Newsletter:   false,
		FieldHint:    FieldHintFocused,
		ZoneView:     ZoneViewGrid,
		ApplyConfirm: ApplyConfirmUnexpected,
		ShowRRTypes:  false,
	}
}
