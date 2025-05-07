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

import { handleApiResponse } from "$lib/errors";
import type { ProviderInfos, ProviderList } from "$lib/model/provider";

export async function listProviders(): Promise<ProviderList> {
    const res = await fetch("/api/providers/_specs", {
        method: "GET",
        headers: { Accept: "application/json" },
    });
    return await handleApiResponse<ProviderList>(res);
}

export async function getProviderSpec(psid: string): Promise<ProviderInfos> {
    const res = await fetch(`/api/providers/_specs/` + psid, {
        method: "GET",
        headers: { Accept: "application/json" },
    });
    return await handleApiResponse<ProviderInfos>(res);
}
