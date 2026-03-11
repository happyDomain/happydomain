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
    getChecks,
    getChecksByCname,
    getChecksByCnameOptions,
    postChecksByCnameOptions,
    putChecksByCnameOptions,
    getChecksByCnameOptionsByOptname,
    putChecksByCnameOptionsByOptname,
    postDomainsByDomainChecksByCname,
    getDomainsByDomainChecksByCnameOptions,
    postDomainsByDomainChecksByCnameOptions,
    putDomainsByDomainChecksByCnameOptions,
    getDomainsByDomainChecks,
    getDomainsByDomainChecksByCnameResults,
    getDomainsByDomainChecksByCnameResultsByResultId,
    getDomainsByDomainChecksByCnameResultsByResultIdReport,
    deleteDomainsByDomainChecksByCnameResults,
    deleteDomainsByDomainChecksByCnameResultsByResultId,
    getDomainsByDomainChecksByCnameExecutionsByExecutionId,
    postPluginsTestsSchedules,
    putPluginsTestsSchedulesByScheduleId,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecks,
    postDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCname,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameExecutionsByExecutionId,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameOptions,
    postDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameOptions,
    putDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameOptions,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameResults,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameResultsByResultId,
    deleteDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameResultsByResultId,
    deleteDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameResults,
    getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameResultsByResultIdReport,
} from "$lib/api-base/sdk.gen";
import { unwrapSdkResponse } from "./errors";
import type {
    CheckerList,
    CheckerInfo,
    CheckerOptions,
    AvailableChecker,
    CheckerSchedule,
    CheckResult,
    CheckExecution,
    CheckScopeType,
} from "$lib/model/checker";

export async function listCheckers(): Promise<CheckerList> {
    return unwrapSdkResponse(await getChecks()) as CheckerList;
}

export async function getCheckStatus(checkId: string): Promise<CheckerInfo> {
    return unwrapSdkResponse(
        await getChecksByCname({
            path: { cname: checkId },
        }),
    ) as unknown as CheckerInfo;
}

export async function getCheckOptions(checkId: string): Promise<CheckerOptions> {
    return unwrapSdkResponse(
        await getChecksByCnameOptions({
            path: { cname: checkId },
        }),
    ) as CheckerOptions;
}

export async function addCheckOptions(checkId: string, options: CheckerOptions): Promise<boolean> {
    return unwrapSdkResponse(
        await postChecksByCnameOptions({
            path: { cname: checkId },
            body: { options } as any,
        }),
    ) as boolean;
}

export async function updateCheckOptions(
    checkId: string,
    options: CheckerOptions,
): Promise<boolean> {
    return unwrapSdkResponse(
        await putChecksByCnameOptions({
            path: { cname: checkId },
            body: { options } as any,
        }),
    ) as boolean;
}

export async function getCheckOption(checkId: string, optionName: string): Promise<any> {
    return unwrapSdkResponse(
        await getChecksByCnameOptionsByOptname({
            path: { cname: checkId, optname: optionName },
        }),
    );
}

export async function setcheckOption(
    checkId: string,
    optionName: string,
    value: any,
): Promise<boolean> {
    return unwrapSdkResponse(
        await putChecksByCnameOptionsByOptname({
            path: { cname: checkId, optname: optionName },
            body: value as any,
        }),
    ) as boolean;
}

export async function getDomainCheckOptions(
    domainId: string,
    checkId: string,
): Promise<CheckerOptions> {
    return unwrapSdkResponse(
        await getDomainsByDomainChecksByCnameOptions({
            path: { domain: domainId, cname: checkId },
        }),
    ) as CheckerOptions;
}

export async function addDomainCheckOptions(
    domainId: string,
    checkId: string,
    options: CheckerOptions,
): Promise<boolean> {
    return unwrapSdkResponse(
        await postDomainsByDomainChecksByCnameOptions({
            path: { domain: domainId, cname: checkId },
            body: { options } as any,
        }),
    ) as boolean;
}

export async function updateDomainCheckOptions(
    domainId: string,
    checkId: string,
    options: CheckerOptions,
): Promise<boolean> {
    return unwrapSdkResponse(
        await putDomainsByDomainChecksByCnameOptions({
            path: { domain: domainId, cname: checkId },
            body: { options } as any,
        }),
    ) as boolean;
}

export async function triggerCheck(
    domainId: string,
    checkId: string,
    options?: CheckerOptions,
): Promise<{ execution_id?: string }> {
    const filteredOptions = options
        ? Object.fromEntries(
              Object.entries(options).filter(([, v]) => v !== "" && v !== null && v !== undefined),
          )
        : undefined;
    return unwrapSdkResponse(
        await postDomainsByDomainChecksByCname({
            path: { domain: domainId, cname: checkId },
            body: { options: filteredOptions } as any,
        }),
    ) as { execution_id?: string };
}

export async function listAvailableCheckers(domainId: string): Promise<AvailableChecker[]> {
    return unwrapSdkResponse(
        await getDomainsByDomainChecks({
            path: { domain: domainId },
        }),
    ) as unknown as AvailableChecker[];
}

export async function createCheckSchedule(data: {
    checker_name: string;
    target_type: CheckScopeType;
    target_id: string;
    interval: number;
    enabled: boolean;
    options?: CheckerOptions;
}): Promise<CheckerSchedule> {
    return unwrapSdkResponse(
        await postPluginsTestsSchedules({
            body: {
                checker_name: data.checker_name,
                target_type: data.target_type,
                target_id: data.target_id,
                interval: data.interval,
                enabled: data.enabled,
                options: data.options,
            },
        }),
    ) as CheckerSchedule;
}

export async function updateCheckSchedule(
    scheduleId: string,
    schedule: CheckerSchedule,
): Promise<CheckerSchedule> {
    return unwrapSdkResponse(
        await putPluginsTestsSchedulesByScheduleId({
            path: { schedule_id: scheduleId },
            body: {
                id: schedule.id,
                checker_name: schedule.checker_name,
                target_type: schedule.target_type,
                target_id: schedule.target_id,
                interval: schedule.interval,
                enabled: schedule.enabled,
                last_run: schedule.last_run,
                next_run: schedule.next_run,
                options: schedule.options,
            },
        }),
    ) as unknown as AvailableChecker[];
}
export async function getCheckResult(
    domainId: string,
    checkName: string,
    resultId: string,
): Promise<CheckResult> {
    return unwrapSdkResponse(
        await getDomainsByDomainChecksByCnameResultsByResultId({
            path: { domain: domainId, cname: checkName, result_id: resultId },
        }),
    ) as unknown as CheckResult;
}

export async function deleteCheckResult(
    domainId: string,
    checkName: string,
    resultId: string,
): Promise<void> {
    await deleteDomainsByDomainChecksByCnameResultsByResultId({
        path: { domain: domainId, cname: checkName, result_id: resultId },
    });
}

export async function listCheckResults(
    domainId: string,
    checkName: string,
): Promise<CheckResult[]> {
    return unwrapSdkResponse(
        await getDomainsByDomainChecksByCnameResults({
            path: { domain: domainId, cname: checkName },
        }),
    ) as unknown as CheckResult[];
}

export async function deleteAllCheckResults(domainId: string, checkName: string): Promise<void> {
    await deleteDomainsByDomainChecksByCnameResults({
        path: { domain: domainId, cname: checkName },
    });
}

export async function getCheckResultHTMLReport(
    domainId: string,
    checkName: string,
    resultId: string,
): Promise<string> {
    return unwrapSdkResponse(
        await getDomainsByDomainChecksByCnameResultsByResultIdReport({
            path: { domain: domainId, cname: checkName, result_id: resultId },
        }),
    ) as string;
}

export async function getCheckExecution(
    domainId: string,
    checkName: string,
    executionId: string,
): Promise<CheckExecution> {
    return unwrapSdkResponse(
        await getDomainsByDomainChecksByCnameExecutionsByExecutionId({
            path: { domain: domainId, cname: checkName, execution_id: executionId },
        }),
    ) as unknown as CheckExecution;
}

// --- Service-scoped check API functions ---

export async function listServiceAvailableCheckers(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
): Promise<AvailableChecker[]> {
    return unwrapSdkResponse(
        await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecks({
            path: { domain: domainId, zoneid: zoneId, subdomain, serviceid },
        }),
    ) as unknown as AvailableChecker[];
}

export async function triggerServiceCheck(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
    checkId: string,
    options?: CheckerOptions,
): Promise<{ execution_id?: string }> {
    const filteredOptions = options
        ? Object.fromEntries(
              Object.entries(options).filter(([, v]) => v !== "" && v !== null && v !== undefined),
          )
        : undefined;
    return unwrapSdkResponse(
        await postDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCname({
            path: { domain: domainId, zoneid: zoneId, subdomain, serviceid, cname: checkId } as any,
            body: { options: filteredOptions } as any,
        }),
    ) as { execution_id?: string };
}

export async function getServiceCheckExecution(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
    checkName: string,
    executionId: string,
): Promise<CheckExecution> {
    return unwrapSdkResponse(
        await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameExecutionsByExecutionId(
            {
                path: {
                    domain: domainId,
                    zoneid: zoneId,
                    subdomain,
                    serviceid,
                    cname: checkName,
                    execution_id: executionId,
                } as any,
            },
        ),
    ) as unknown as CheckExecution;
}

export async function getServiceCheckOptions(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
    checkId: string,
): Promise<CheckerOptions> {
    return unwrapSdkResponse(
        await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameOptions({
            path: { domain: domainId, zoneid: zoneId, subdomain, serviceid, cname: checkId } as any,
        }),
    ) as CheckerOptions;
}

export async function addServiceCheckOptions(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
    checkId: string,
    options: CheckerOptions,
): Promise<boolean> {
    return unwrapSdkResponse(
        await postDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameOptions({
            path: { domain: domainId, zoneid: zoneId, subdomain, serviceid, cname: checkId } as any,
            body: { options } as any,
        }),
    ) as boolean;
}

export async function updateServiceCheckOptions(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
    checkId: string,
    options: CheckerOptions,
): Promise<boolean> {
    return unwrapSdkResponse(
        await putDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameOptions({
            path: { domain: domainId, zoneid: zoneId, subdomain, serviceid, cname: checkId } as any,
            body: { options } as any,
        }),
    ) as boolean;
}

export async function listServiceCheckResults(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
    checkName: string,
): Promise<CheckResult[]> {
    return unwrapSdkResponse(
        await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameResults({
            path: {
                domain: domainId,
                zoneid: zoneId,
                subdomain,
                serviceid,
                cname: checkName,
            } as any,
        }),
    ) as unknown as CheckResult[];
}

export async function getServiceCheckResult(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
    checkName: string,
    resultId: string,
): Promise<CheckResult> {
    return unwrapSdkResponse(
        await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameResultsByResultId(
            {
                path: {
                    domain: domainId,
                    zoneid: zoneId,
                    subdomain,
                    serviceid,
                    cname: checkName,
                    result_id: resultId,
                } as any,
            },
        ),
    ) as unknown as CheckResult;
}

export async function deleteServiceCheckResult(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
    checkName: string,
    resultId: string,
): Promise<void> {
    await deleteDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameResultsByResultId(
        {
            path: {
                domain: domainId,
                zoneid: zoneId,
                subdomain,
                serviceid,
                cname: checkName,
                result_id: resultId,
            } as any,
        },
    );
}

export async function deleteAllServiceCheckResults(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
    checkName: string,
): Promise<void> {
    await deleteDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameResults({
        path: { domain: domainId, zoneid: zoneId, subdomain, serviceid, cname: checkName } as any,
    });
}

export async function getServiceCheckResultHTMLReport(
    domainId: string,
    zoneId: string,
    subdomain: string,
    serviceid: string,
    checkName: string,
    resultId: string,
): Promise<string> {
    return unwrapSdkResponse(
        await getDomainsByDomainZoneByZoneidBySubdomainServicesByServiceidChecksByCnameResultsByResultIdReport(
            {
                path: {
                    domain: domainId,
                    zoneid: zoneId,
                    subdomain,
                    serviceid,
                    cname: checkName,
                    result_id: resultId,
                } as any,
            },
        ),
    ) as string;
}
