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

export interface Field {
    id: string;
    type: string;
    label?: string;
    placeholder?: string;
    default?: string;
    choices?: Array<string>;
    required?: boolean;
    secret?: boolean;
    description?: string;
}

export interface CustomForm {
    beforeText?: string;
    sideText?: string;
    afterText?: string;
    fields: Array<Field>;
    nextButtonText?: string;
    nextEditButtonText?: string;
    previousButtonText?: string;
    previousEditButtonText?: string;
    nextButtonLink?: string;
    nextButtonState?: number;
    previousButtonLink?: string;
    previousButtonState?: number;
}

export interface FormState {
    _id?: any;
    _comment?: string;
    state: number;
    recall?: string;
    redirect?: string;
}

export interface FormResponse<T> {
    form?: CustomForm;
    values?: T;
    redirect?: string;
}
