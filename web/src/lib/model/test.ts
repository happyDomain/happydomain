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

import type { CheckerOptions } from "./check";

export enum TestScopeType {
    TestScopeInstance = 0,
    TestScopeUser = 1,
    TestScopeDomain = 2,
    TestScopeZone = 3,
    TestScopeService = 4,
    TestScopeOnDemand = 5,
}

export enum TestExecutionStatus {
    TestExecutionPending = 0,
    TestExecutionRunning = 1,
    TestExecutionCompleted = 2,
    TestExecutionFailed = 3,
}

export enum PluginResultStatus {
    KO = 0,
    Warn = 1,
    Info = 2,
    OK = 3,
}

export interface TestResult {
    id: string;
    plugin_name: string;
    test_type: TestScopeType;
    target_id: string;
    user_id: string;
    executed_at: string;
    scheduled_test: boolean;
    options?: CheckerOptions;
    status: PluginResultStatus;
    status_line: string;
    report?: any;
    duration?: number;
    error?: string;
}

export interface TestSchedule {
    id: string;
    plugin_name: string;
    user_id: string;
    target_type: TestScopeType;
    target_id: string;
    interval: number;
    enabled: boolean;
    last_run?: string;
    next_run: string;
    options?: CheckerOptions;
}

export interface TestExecution {
    id: string;
    schedule_id?: string;
    plugin_name: string;
    user_id: string;
    target_id: string;
    status: TestExecutionStatus;
    started_at: string;
    completed_at?: string;
    result_id?: string;
}

export interface AvailableTest {
    plugin_name: string;
    enabled: boolean;
    schedule?: TestSchedule;
    last_result?: TestResult;
}

export interface TriggerTestRequest {
    options?: CheckerOptions;
}

export interface CreateScheduleRequest {
    plugin_name: string;
    target_type: TestScopeType;
    target_id: string;
    interval: number;
    enabled: boolean;
    options?: CheckerOptions;
}
