// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import {
    getDomains,
    getDomainsByDomainId,
    postDomains,
    putDomainsByDomainId,
    deleteDomainsByDomainId,
    getDomainsByDomainIdLogs,
} from "$lib/api-base/sdk.gen";
import type { Domain, DomainLog } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import { unwrapSdkResponse, unwrapEmptyResponse } from "./errors";

export async function listDomains(): Promise<Array<Domain>> {
    return unwrapSdkResponse(await getDomains()) as Array<Domain>;
}

export async function getDomain(id: string): Promise<Domain> {
    return unwrapSdkResponse(
        await getDomainsByDomainId({
            path: { domainId: id },
        }),
    ) as Domain;
}

export async function addDomain(domain: string, provider: Provider | undefined): Promise<Domain> {
    const id_provider = provider ? provider._id : undefined;

    return unwrapSdkResponse(
        await postDomains({
            body: {
                domain,
                id_provider,
            } as any,
        }),
    ) as Domain;
}

export async function updateDomain(domain: Domain): Promise<Domain> {
    if (domain.id) {
        return unwrapSdkResponse(
            await putDomainsByDomainId({
                path: { domainId: domain.id },
                body: domain as any,
            }),
        ) as Domain;
    } else {
        return unwrapSdkResponse(
            await postDomains({
                body: domain as any,
            }),
        ) as Domain;
    }
}

export async function deleteDomain(id: string): Promise<boolean> {
    return unwrapEmptyResponse(
        await deleteDomainsByDomainId({
            path: { domainId: id },
        }),
    );
}

export async function getDomainLogs(id: string): Promise<Array<DomainLog>> {
    return unwrapSdkResponse(
        await getDomainsByDomainIdLogs({
            path: { domainId: id },
        }),
    ) as unknown as Array<DomainLog>;
}
