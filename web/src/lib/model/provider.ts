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

export interface ProviderInfos {
    name: string;
    description: string;
    capabilities: Array<string>;
    helplink: string;
}

export function getAvailableResourceTypes(pi: ProviderInfos): Array<number> {
    const availableResourceTypes = [];

    for (const cap of pi.capabilities) {
        if (cap.startsWith("rr-")) {
            availableResourceTypes.push(parseInt(cap.substring(3, cap.indexOf("-", 4))));
        }
    }

    return availableResourceTypes;
}

export type ProviderList = Record<string, ProviderInfos>;

export interface ProviderMeta {
    _srctype: string;
    _id: string;
    _ownerid: string;
    _comment: string;
}

export interface ProviderData extends ProviderMeta {
    Provider: any;
}

export interface Provider extends ProviderMeta {
    Provider: any;
}

export function isProvider(e: unknown): e is Provider {
    return typeof e === "object" && e !== null &&
        "Provider" in e && "_id" in e && "_srctype" in e;
}
