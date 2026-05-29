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
    getAvailability,
    getAvailabilityByWatchId,
    getAvailabilityByWatchIdStatus,
    postAvailability,
    postAvailabilityByWatchIdCheck,
    deleteAvailabilityByWatchId,
} from "$lib/api-base/sdk.gen";
import type {
    HappydnsDomainAvailabilityWatch,
    HappydnsDomainAvailabilityWatchCreationInput,
} from "$lib/api-base/types.gen";
import { unwrapSdkResponse, unwrapEmptyResponse } from "./errors";

// AvailabilityWatchStatus is the latest known availability result for a watch,
// as returned by GET /availability/{watchId}/status.
export interface AvailabilityWatchStatus {
    // available is true once the domain is free to register, false while still
    // registered, and undefined when the watch has never been checked.
    available?: boolean;
    // checking is true while an availability check is currently running.
    checking: boolean;
    // last_checked is the RFC3339 timestamp of the most recent finished check.
    last_checked?: string;
    // error carries the failure message when the last check could not run.
    error?: string;
}

export async function listAvailabilityWatches(): Promise<
    Array<HappydnsDomainAvailabilityWatch>
> {
    return unwrapSdkResponse(
        await getAvailability(),
    ) as Array<HappydnsDomainAvailabilityWatch>;
}

export async function getAvailabilityWatch(
    id: string,
): Promise<HappydnsDomainAvailabilityWatch> {
    return unwrapSdkResponse(
        await getAvailabilityByWatchId({
            path: { watchId: id },
        }),
    ) as HappydnsDomainAvailabilityWatch;
}

export async function addAvailabilityWatch(
    body: HappydnsDomainAvailabilityWatchCreationInput,
): Promise<HappydnsDomainAvailabilityWatch> {
    return unwrapSdkResponse(
        await postAvailability({
            body,
        }),
    ) as HappydnsDomainAvailabilityWatch;
}

export async function deleteAvailabilityWatch(id: string): Promise<boolean> {
    return unwrapEmptyResponse(
        await deleteAvailabilityByWatchId({
            path: { watchId: id },
        }),
    );
}

export async function getAvailabilityWatchStatus(
    id: string,
): Promise<AvailabilityWatchStatus> {
    return unwrapSdkResponse(
        await getAvailabilityByWatchIdStatus({
            path: { watchId: id },
        }),
    ) as AvailabilityWatchStatus;
}

export async function triggerAvailabilityWatchCheck(id: string): Promise<boolean> {
    return unwrapEmptyResponse(
        await postAvailabilityByWatchIdCheck({
            path: { watchId: id },
        }),
    );
}
