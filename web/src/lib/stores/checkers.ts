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

import { derived, writable, type Readable, type Writable } from "svelte/store";
import { listCheckers, type ObservationSnapshotWithData } from "$lib/api/checkers";
import type {
    CheckerCheckerDefinition,
    HappydnsExecution,
} from "$lib/api-base/types.gen";

export const checkers: Writable<Record<string, CheckerCheckerDefinition> | undefined> =
    writable(undefined);

export async function refreshCheckers() {
    const data = await listCheckers();
    checkers.set(data);
    return data;
}

// Stores for the currently viewed execution detail page
export const currentExecution: Writable<HappydnsExecution | undefined> = writable(undefined);
export const currentCheckInfo: Writable<CheckerCheckerDefinition | undefined> = writable(undefined);
export const currentObservations: Writable<ObservationSnapshotWithData | undefined> = writable(undefined);

// Report view mode: which panel the main area shows
export type ReportViewMode = "json" | "html" | "metrics";
export const reportViewMode: Writable<ReportViewMode> = writable("json");
export const showHTMLReport: Readable<boolean> = derived(reportViewMode, ($m) => $m === "html");

// Cached HTML report content, shared between the report card and sidebar download button.
export const cachedHTMLReport: Writable<string | null> = writable(null);
