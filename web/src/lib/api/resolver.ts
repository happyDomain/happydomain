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
    postResolver,
    postResolverDmarcReportAuth,
    postResolverMtaStsPolicy,
    postResolverSpfFlatten,
} from "$lib/api-base/sdk.gen";
import type {
    HappydnsDmarcReportAuthRequest,
    HappydnsDmarcReportAuthResponse,
    HappydnsMtastsPolicyRequest,
    HappydnsMtastsPolicyResponse,
    HappydnsResolverResponse,
    HappydnsSpfFlattenRequest,
    HappydnsSpfFlattenResponse,
} from "$lib/api-base/types.gen";
import type { ResolverForm } from "$lib/model/resolver";
import { unwrapSdkResponse } from "./errors";

export async function resolve(form: ResolverForm): Promise<HappydnsResolverResponse> {
    return unwrapSdkResponse(
        await postResolver({
            body: form,
        }),
    ) as HappydnsResolverResponse;
}

export async function flattenSPF(
    body: HappydnsSpfFlattenRequest,
    signal?: AbortSignal,
): Promise<HappydnsSpfFlattenResponse> {
    return unwrapSdkResponse(
        await postResolverSpfFlatten({
            body,
            signal,
        }),
    ) as HappydnsSpfFlattenResponse;
}

export async function fetchMTAStsPolicy(
    body: HappydnsMtastsPolicyRequest,
    signal?: AbortSignal,
): Promise<HappydnsMtastsPolicyResponse> {
    return unwrapSdkResponse(
        await postResolverMtaStsPolicy({
            body,
            signal,
        }),
    ) as HappydnsMtastsPolicyResponse;
}

export async function checkDMARCReportAuth(
    body: HappydnsDmarcReportAuthRequest,
    signal?: AbortSignal,
): Promise<HappydnsDmarcReportAuthResponse> {
    return unwrapSdkResponse(
        await postResolverDmarcReportAuth({
            body,
            signal,
        }),
    ) as HappydnsDmarcReportAuthResponse;
}
