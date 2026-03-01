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
    getChecksByCid,
    getChecksByCidOptions,
    postChecksByCidOptions,
    putChecksByCidOptions,
    getChecksByCidOptionsByOptname,
    putChecksByCidOptionsByOptname,
    postDomainsByDomainChecksByCname,
    getDomainsByDomainChecksByCnameOptions,
    getDomainsByDomainChecks,
    getDomainsByDomainChecksByCnameResults,
    getDomainsByDomainChecksByCnameResultsByResultId,
    getDomainsByDomainChecksByCnameResultsByResultIdReport,
    deleteDomainsByDomainChecksByCnameResults,
    deleteDomainsByDomainChecksByCnameResultsByResultId,
    getDomainsByDomainChecksByCnameExecutionsByExecutionId,
    postPluginsTestsSchedules,
    putPluginsTestsSchedulesByScheduleId,
} from "$lib/api-base/sdk.gen";
import { unwrapSdkResponse } from "./errors";
import type {
    CheckerList,
    CheckerInfo,
    CheckerOptions,
    AvailableCheck,
    CheckerSchedule,
    CheckResult,
    CheckExecution,
} from "$lib/model/check";
import { CheckScopeType } from "$lib/model/check";

export async function listChecks(): Promise<CheckerList> {
    return unwrapSdkResponse(await getChecks()) as CheckerList;
}

export async function getCheckStatus(checkId: string): Promise<CheckerInfo> {
    return unwrapSdkResponse(
        await getChecksByCid({
            path: { cid: checkId },
        }),
    ) as unknown as CheckerInfo;
}

export async function getCheckOptions(checkId: string): Promise<CheckerOptions> {
    return unwrapSdkResponse(
        await getChecksByCidOptions({
            path: { cid: checkId },
        }),
    ) as CheckerOptions;
}

export async function addCheckOptions(checkId: string, options: CheckerOptions): Promise<boolean> {
    return unwrapSdkResponse(
        await postChecksByCidOptions({
            path: { cid: checkId },
            body: { options } as any,
        }),
    ) as boolean;
}

export async function updateCheckOptions(checkId: string, options: CheckerOptions): Promise<boolean> {
    return unwrapSdkResponse(
        await putChecksByCidOptions({
            path: { cid: checkId },
            body: { options } as any,
        }),
    ) as boolean;
}

export async function getCheckOption(checkId: string, optionName: string): Promise<any> {
    return unwrapSdkResponse(
        await getChecksByCidOptionsByOptname({
            path: { cid: checkId, optname: optionName },
        }),
    );
}

export async function setcheckOption(
    checkId: string,
    optionName: string,
    value: any,
): Promise<boolean> {
    return unwrapSdkResponse(
        await putChecksByCidOptionsByOptname({
            path: { cid: checkId, optname: optionName },
            body: value as any,
        }),
    ) as boolean;
}

export async function getDomainCheckOptions(domainId: string, checkId: string): Promise<CheckerOptions> {
    return unwrapSdkResponse(
        await getDomainsByDomainChecksByCnameOptions({
            path: { domain: domainId, cname: checkId },
        }),
    ) as CheckerOptions;
}

export async function triggerCheck(
    domainId: string,
    checkId: string,
    options?: CheckerOptions,
): Promise<{ execution_id?: string }> {
    const filteredOptions = options
        ? Object.fromEntries(Object.entries(options).filter(([, v]) => v !== "" && v !== null && v !== undefined))
        : undefined;
    return unwrapSdkResponse(
        await postDomainsByDomainChecksByCname({
            path: { domain: domainId, cname: checkId },
            body: { options: filteredOptions } as any,
        }),
    ) as { execution_id?: string };
}

export async function listAvailableChecks(domainId: string): Promise<AvailableCheck[]> {
    return unwrapSdkResponse(
        await getDomainsByDomainChecks({
            path: { domain: domainId },
        }),
    ) as unknown as AvailableCheck[];
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
    ) as CheckerSchedule;
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

export async function deleteAllCheckResults(
    domainId: string,
    checkName: string,
): Promise<void> {
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
