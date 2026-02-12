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

import type { PostDomainsByDomainTestsByTnameResponse } from "$lib/api-base/types.gen";
import {
    getDomainsByDomainTests,
    getDomainsByDomainTestsByTname,
    postDomainsByDomainTestsByTname,
    getDomainsByDomainTestsByTnameExecutionsByExecutionId,
    getDomainsByDomainTestsByTnameOptions,
    putDomainsByDomainTestsByTnameOptions,
    getDomainsByDomainTestsByTnameResults,
    getDomainsByDomainTestsByTnameResultsByResultId,
    deleteDomainsByDomainTestsByTnameResultsByResultId,
    deleteDomainsByDomainTestsByTnameResults,
    getPluginsTestsSchedules,
    getPluginsTestsSchedulesByScheduleId,
    postPluginsTestsSchedules,
    putPluginsTestsSchedulesByScheduleId,
    deletePluginsTestsSchedulesByScheduleId,
} from "$lib/api-base/sdk.gen";
import type {
    TestResult,
    TestExecution,
    TestSchedule,
    AvailableTest,
    CreateScheduleRequest,
} from "$lib/model/test";
import type { PluginOptions } from "$lib/model/plugin";
import { unwrapSdkResponse, unwrapEmptyResponse } from "./errors";

// Domain test operations
export async function listAvailableTests(domainId: string): Promise<AvailableTest[]> {
    return unwrapSdkResponse(
        await getDomainsByDomainTests({ path: { domain: domainId } }),
    ) as unknown as AvailableTest[];
}

export async function listTestResults(
    domainId: string,
    testName: string,
    limit?: number,
): Promise<TestResult[]> {
    return unwrapSdkResponse(
        await getDomainsByDomainTestsByTnameResults({
            path: { domain: domainId, tname: testName },
            query: limit !== undefined ? { limit } : undefined,
        }),
    ) as TestResult[];
}

export async function getLatestTestResults(
    domainId: string,
    testName: string,
): Promise<TestResult[]> {
    return unwrapSdkResponse(
        await getDomainsByDomainTestsByTname({ path: { domain: domainId, tname: testName } }),
    ) as TestResult[];
}

export async function triggerTest(
    domainId: string,
    testName: string,
    options?: PluginOptions,
): Promise<PostDomainsByDomainTestsByTnameResponse> {
    return unwrapSdkResponse(
        await postDomainsByDomainTestsByTname({
            path: { domain: domainId, tname: testName },
            body: { options } as any,
        }),
    ) as PostDomainsByDomainTestsByTnameResponse;
}

export async function getTestExecution(
    domainId: string,
    testName: string,
    executionId: string,
): Promise<TestExecution> {
    return unwrapSdkResponse(
        await getDomainsByDomainTestsByTnameExecutionsByExecutionId({
            path: { domain: domainId, tname: testName, execution_id: executionId },
        }),
    ) as TestExecution;
}

export async function getTestResult(
    domainId: string,
    testName: string,
    resultId: string,
): Promise<TestResult> {
    return unwrapSdkResponse(
        await getDomainsByDomainTestsByTnameResultsByResultId({
            path: { domain: domainId, tname: testName, result_id: resultId },
        }),
    ) as TestResult;
}

export async function deleteTestResult(
    domainId: string,
    testName: string,
    resultId: string,
): Promise<void> {
    unwrapEmptyResponse(
        await deleteDomainsByDomainTestsByTnameResultsByResultId({
            path: { domain: domainId, tname: testName, result_id: resultId },
        }),
    );
}

export async function deleteAllTestResults(domainId: string, testName: string): Promise<void> {
    unwrapEmptyResponse(
        await deleteDomainsByDomainTestsByTnameResults({
            path: { domain: domainId, tname: testName },
        }),
    );
}

export async function getTestOptions(domainId: string, testName: string): Promise<PluginOptions> {
    return unwrapSdkResponse(
        await getDomainsByDomainTestsByTnameOptions({
            path: { domain: domainId, tname: testName },
        }),
    ) as PluginOptions;
}

export async function updateTestOptions(
    domainId: string,
    testName: string,
    options: PluginOptions,
): Promise<void> {
    unwrapEmptyResponse(
        await putDomainsByDomainTestsByTnameOptions({
            path: { domain: domainId, tname: testName },
            body: { options } as any,
        }),
    );
}

// Schedule operations
export async function listUserSchedules(): Promise<TestSchedule[]> {
    return unwrapSdkResponse(await getPluginsTestsSchedules()) as TestSchedule[];
}

export async function getTestSchedule(scheduleId: string): Promise<TestSchedule> {
    return unwrapSdkResponse(
        await getPluginsTestsSchedulesByScheduleId({ path: { schedule_id: scheduleId } }),
    ) as TestSchedule;
}

export async function createTestSchedule(schedule: CreateScheduleRequest): Promise<TestSchedule> {
    return unwrapSdkResponse(
        await postPluginsTestsSchedules({ body: schedule as any }),
    ) as TestSchedule;
}

export async function updateTestSchedule(
    scheduleId: string,
    schedule: Partial<TestSchedule>,
): Promise<void> {
    unwrapEmptyResponse(
        await putPluginsTestsSchedulesByScheduleId({
            path: { schedule_id: scheduleId },
            body: schedule as any,
        }),
    );
}

export async function deleteTestSchedule(scheduleId: string): Promise<void> {
    unwrapEmptyResponse(
        await deletePluginsTestsSchedulesByScheduleId({ path: { schedule_id: scheduleId } }),
    );
}
