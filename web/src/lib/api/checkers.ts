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
    getCheckers,
    getCheckersByCheckerId,
    getCheckersByCheckerIdOptions,
    putCheckersByCheckerIdOptions,
    getDomainsByDomainCheckers,
    getDomainsByDomainCheckersByCheckerIdExecutions,
    postDomainsByDomainCheckersByCheckerIdExecutions,
    deleteDomainsByDomainCheckersByCheckerIdExecutions,
    deleteDomainsByDomainCheckersByCheckerIdExecutionsByExecutionId,
    getDomainsByDomainCheckersByCheckerIdExecutionsByExecutionId,
    getDomainsByDomainCheckersByCheckerIdExecutionsByExecutionIdObservations,
    getDomainsByDomainCheckersByCheckerIdExecutionsByExecutionIdMetrics,
    getDomainsByDomainCheckersByCheckerIdExecutionsByExecutionIdObservationsByObsKeyReport,
    getDomainsByDomainCheckersByCheckerIdExecutionsByExecutionIdResults,
    getDomainsByDomainCheckersByCheckerIdMetrics,
    getDomainsByDomainCheckersByCheckerIdOptions,
    putDomainsByDomainCheckersByCheckerIdOptions,
    getDomainsByDomainCheckersByCheckerIdPlans,
    postDomainsByDomainCheckersByCheckerIdPlans,
    putDomainsByDomainCheckersByCheckerIdPlansByPlanId,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckers,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutions,
    postDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutions,
    deleteDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutions,
    deleteDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionId,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionId,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionIdObservations,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionIdMetrics,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionIdObservationsByObsKeyReport,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionIdResults,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdMetrics,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdOptions,
    putDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdOptions,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdPlans,
    postDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdPlans,
    putDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdPlansByPlanId,
} from "$lib/api-base/sdk.gen";
import type {
    CheckerCheckerDefinition,
    CheckerCheckerOptions,
    CheckerCheckMetric,
    HappydnsCheckEvaluation,
    HappydnsCheckPlan,
    HappydnsCheckPlanWritable,
    HappydnsCheckerOptions,
    HappydnsCheckerOptionsPositional,
    HappydnsCheckerRunRequest,
    HappydnsCheckerStatus,
    HappydnsExecution,
    HappydnsObservationSnapshot,
} from "$lib/api-base/types.gen";

// Workaround: hey-api/openapi-ts drops the `data` field from HappydnsObservationSnapshot
// because swagger generates `additionalProperties: {type: object}` which the codegen cannot handle.
// The API does return a `data` field of type Record<string, unknown>.
export type ObservationSnapshotWithData = HappydnsObservationSnapshot & {
    readonly data: Record<string, unknown>;
};
import { unwrapSdkResponse, unwrapEmptyResponse } from "./errors";

// Global (non-scoped) checker functions

export async function listCheckers(): Promise<Record<string, CheckerCheckerDefinition>> {
    return unwrapSdkResponse(await getCheckers()) as Record<string, CheckerCheckerDefinition>;
}

export async function getCheckStatus(checkerId: string): Promise<CheckerCheckerDefinition> {
    return unwrapSdkResponse(
        await getCheckersByCheckerId({ path: { checkerId } }),
    ) as CheckerCheckerDefinition;
}

export async function getCheckOptions(checkerId: string): Promise<HappydnsCheckerOptionsPositional[]> {
    return (unwrapSdkResponse(
        await getCheckersByCheckerIdOptions({ path: { checkerId } }),
    ) as HappydnsCheckerOptionsPositional[]) ?? [];
}

export async function updateCheckOptions(
    checkerId: string,
    options: HappydnsCheckerOptions,
): Promise<HappydnsCheckerOptions> {
    return unwrapSdkResponse(
        await putCheckersByCheckerIdOptions({ path: { checkerId }, body: options as CheckerCheckerOptions }),
    ) as HappydnsCheckerOptions;
}

// Scope-aware helpers

export interface CheckerScope {
    domainId: string;
    zoneId?: string;
    subdomain?: string;
    serviceId?: string;
}

function isServiceScope(scope: CheckerScope): scope is CheckerScope & { zoneId: string; subdomain: string; serviceId: string } {
    return !!(scope.zoneId && scope.subdomain !== undefined && scope.serviceId);
}

export async function listScopedCheckers(
    scope: CheckerScope,
): Promise<HappydnsCheckerStatus[]> {
    if (isServiceScope(scope)) {
        return (unwrapSdkResponse(
            await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckers({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId },
            }),
        ) as HappydnsCheckerStatus[]) ?? [];
    }
    return (unwrapSdkResponse(
        await getDomainsByDomainCheckers({ path: { domain: scope.domainId } }),
    ) as HappydnsCheckerStatus[]) ?? [];
}

export async function listScopedExecutions(
    scope: CheckerScope,
    checkerId: string,
    options?: { includePlanned?: boolean; limit?: number },
): Promise<HappydnsExecution[]> {
    const query = {
        ...(options?.includePlanned ? { include_planned: true } : {}),
        ...(options?.limit ? { limit: options.limit } : {}),
    };
    if (isServiceScope(scope)) {
        return (unwrapSdkResponse(
            await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutions({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId },
                query,
            }),
        ) as HappydnsExecution[]) ?? [];
    }
    return (unwrapSdkResponse(
        await getDomainsByDomainCheckersByCheckerIdExecutions({
            path: { domain: scope.domainId, checkerId },
            query,
        }),
    ) as HappydnsExecution[]) ?? [];
}

export async function triggerScopedCheck(
    scope: CheckerScope,
    checkerId: string,
    request?: HappydnsCheckerRunRequest,
): Promise<HappydnsExecution> {
    if (isServiceScope(scope)) {
        return unwrapSdkResponse(
            await postDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutions({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId },
                body: request,
            }),
        ) as HappydnsExecution;
    }
    return unwrapSdkResponse(
        await postDomainsByDomainCheckersByCheckerIdExecutions({
            path: { domain: scope.domainId, checkerId },
            body: request,
        }),
    ) as HappydnsExecution;
}

export async function getScopedExecution(
    scope: CheckerScope,
    checkerId: string,
    executionId: string,
): Promise<HappydnsExecution> {
    if (isServiceScope(scope)) {
        return unwrapSdkResponse(
            await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionId({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId, executionId },
            }),
        ) as HappydnsExecution;
    }
    return unwrapSdkResponse(
        await getDomainsByDomainCheckersByCheckerIdExecutionsByExecutionId({
            path: { domain: scope.domainId, checkerId, executionId },
        }),
    ) as HappydnsExecution;
}

export async function deleteScopedExecution(
    scope: CheckerScope,
    checkerId: string,
    executionId: string,
): Promise<boolean> {
    if (isServiceScope(scope)) {
        return unwrapEmptyResponse(
            await deleteDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionId({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId, executionId },
            }),
        );
    }
    return unwrapEmptyResponse(
        await deleteDomainsByDomainCheckersByCheckerIdExecutionsByExecutionId({
            path: { domain: scope.domainId, checkerId, executionId },
        }),
    );
}

export async function deleteAllScopedExecutions(
    scope: CheckerScope,
    checkerId: string,
): Promise<boolean> {
    if (isServiceScope(scope)) {
        return unwrapEmptyResponse(
            await deleteDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutions({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId },
            }),
        );
    }
    return unwrapEmptyResponse(
        await deleteDomainsByDomainCheckersByCheckerIdExecutions({
            path: { domain: scope.domainId, checkerId },
        }),
    );
}

export async function getScopedExecutionResults(
    scope: CheckerScope,
    checkerId: string,
    executionId: string,
): Promise<HappydnsCheckEvaluation> {
    if (isServiceScope(scope)) {
        return unwrapSdkResponse(
            await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionIdResults({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId, executionId },
            }),
        ) as HappydnsCheckEvaluation;
    }
    return unwrapSdkResponse(
        await getDomainsByDomainCheckersByCheckerIdExecutionsByExecutionIdResults({
            path: { domain: scope.domainId, checkerId, executionId },
        }),
    ) as HappydnsCheckEvaluation;
}

export async function getScopedExecutionObservations(
    scope: CheckerScope,
    checkerId: string,
    executionId: string,
): Promise<ObservationSnapshotWithData> {
    if (isServiceScope(scope)) {
        return unwrapSdkResponse(
            await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionIdObservations({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId, executionId },
            }),
        ) as ObservationSnapshotWithData;
    }
    return unwrapSdkResponse(
        await getDomainsByDomainCheckersByCheckerIdExecutionsByExecutionIdObservations({
            path: { domain: scope.domainId, checkerId, executionId },
        }),
    ) as ObservationSnapshotWithData;
}

export async function getScopedCheckOptions(
    scope: CheckerScope,
    checkerId: string,
): Promise<HappydnsCheckerOptionsPositional[]> {
    if (isServiceScope(scope)) {
        return (unwrapSdkResponse(
            await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdOptions({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId },
            }),
        ) as HappydnsCheckerOptionsPositional[]) ?? [];
    }
    return (unwrapSdkResponse(
        await getDomainsByDomainCheckersByCheckerIdOptions({
            path: { domain: scope.domainId, checkerId },
        }),
    ) as HappydnsCheckerOptionsPositional[]) ?? [];
}

export async function updateScopedCheckOptions(
    scope: CheckerScope,
    checkerId: string,
    options: HappydnsCheckerOptions,
): Promise<HappydnsCheckerOptions> {
    if (isServiceScope(scope)) {
        return unwrapSdkResponse(
            await putDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdOptions({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId },
                body: options,
            }),
        ) as HappydnsCheckerOptions;
    }
    return unwrapSdkResponse(
        await putDomainsByDomainCheckersByCheckerIdOptions({
            path: { domain: scope.domainId, checkerId },
            body: options,
        }),
    ) as HappydnsCheckerOptions;
}

export async function getScopedCheckPlans(
    scope: CheckerScope,
    checkerId: string,
): Promise<HappydnsCheckPlan[]> {
    if (isServiceScope(scope)) {
        return (unwrapSdkResponse(
            await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdPlans({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId },
            }),
        ) as HappydnsCheckPlan[]) ?? [];
    }
    return (unwrapSdkResponse(
        await getDomainsByDomainCheckersByCheckerIdPlans({
            path: { domain: scope.domainId, checkerId },
        }),
    ) as HappydnsCheckPlan[]) ?? [];
}

export async function createScopedCheckPlan(
    scope: CheckerScope,
    checkerId: string,
    plan: HappydnsCheckPlan | HappydnsCheckPlanWritable,
): Promise<HappydnsCheckPlan> {
    if (isServiceScope(scope)) {
        return unwrapSdkResponse(
            await postDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdPlans({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId },
                body: plan as HappydnsCheckPlanWritable,
            }),
        ) as HappydnsCheckPlan;
    }
    return unwrapSdkResponse(
        await postDomainsByDomainCheckersByCheckerIdPlans({
            path: { domain: scope.domainId, checkerId },
            body: plan as HappydnsCheckPlanWritable,
        }),
    ) as HappydnsCheckPlan;
}

export async function updateScopedCheckPlan(
    scope: CheckerScope,
    checkerId: string,
    planId: string,
    plan: HappydnsCheckPlan | HappydnsCheckPlanWritable,
): Promise<HappydnsCheckPlan> {
    if (isServiceScope(scope)) {
        return unwrapSdkResponse(
            await putDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdPlansByPlanId({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId, planId },
                body: plan as HappydnsCheckPlanWritable,
            }),
        ) as HappydnsCheckPlan;
    }
    return unwrapSdkResponse(
        await putDomainsByDomainCheckersByCheckerIdPlansByPlanId({
            path: { domain: scope.domainId, checkerId, planId },
            body: plan as HappydnsCheckPlanWritable,
        }),
    ) as HappydnsCheckPlan;
}

// --- Metrics types and API functions ---

export type CheckMetric = CheckerCheckMetric;

export async function getScopedCheckerMetrics(
    scope: CheckerScope,
    checkerId: string,
    limit: number = 100,
): Promise<CheckMetric[]> {
    if (isServiceScope(scope)) {
        return unwrapSdkResponse(
            await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdMetrics({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId },
                query: { limit },
            }),
        ) as CheckMetric[];
    }
    return unwrapSdkResponse(
        await getDomainsByDomainCheckersByCheckerIdMetrics({
            path: { domain: scope.domainId, checkerId },
            query: { limit },
        }),
    ) as CheckMetric[];
}

export async function getScopedExecutionMetrics(
    scope: CheckerScope,
    checkerId: string,
    executionId: string,
): Promise<CheckMetric[]> {
    if (isServiceScope(scope)) {
        return unwrapSdkResponse(
            await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionIdMetrics({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId, executionId },
            }),
        ) as CheckMetric[];
    }
    return unwrapSdkResponse(
        await getDomainsByDomainCheckersByCheckerIdExecutionsByExecutionIdMetrics({
            path: { domain: scope.domainId, checkerId, executionId },
        }),
    ) as CheckMetric[];
}

// HTML report functions

export async function getScopedExecutionHTMLReport(
    scope: CheckerScope,
    checkerId: string,
    executionId: string,
    obsKey: string,
): Promise<string> {
    if (isServiceScope(scope)) {
        return unwrapSdkResponse(
            await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidCheckersByCheckerIdExecutionsByExecutionIdObservationsByObsKeyReport({
                path: { domain: scope.domainId, zoneid: scope.zoneId, subdomain: scope.subdomain, serviceid: scope.serviceId, checkerId, executionId, obsKey },
            }),
        ) as string;
    }
    return unwrapSdkResponse(
        await getDomainsByDomainCheckersByCheckerIdExecutionsByExecutionIdObservationsByObsKeyReport({
            path: { domain: scope.domainId, checkerId, executionId, obsKey },
        }),
    ) as string;
}
