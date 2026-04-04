// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
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

import type { HappydnsApplyConfirm, HappydnsFieldHint, HappydnsUserSettings, HappydnsZoneView } from "$lib/api-base/types.gen";

export type FieldHint = HappydnsFieldHint;
export const FieldHintHide: FieldHint = 0;
export const FieldHintTooltip: FieldHint = 1;
export const FieldHintFocused: FieldHint = 2;
export const FieldHintAlways: FieldHint = 3;

export type ZoneView = HappydnsZoneView;
export const ZoneViewGrid: ZoneView = 0;
export const ZoneViewList: ZoneView = 1;
export const ZoneViewRecords: ZoneView = 2;

export type ApplyConfirm = HappydnsApplyConfirm;
export const ApplyConfirmUnexpected: ApplyConfirm = 0;
export const ApplyConfirmAlways: ApplyConfirm = 1;
export const ApplyConfirmNever: ApplyConfirm = 2;

export type UserSettings = HappydnsUserSettings;
