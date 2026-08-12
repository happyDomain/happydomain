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

import { get, writable } from "svelte/store";

import { getVersion } from "$lib/api/version";

// Everything a reporter would otherwise have to hunt down themselves: we
// collect it as it happens, so that reporting a bug is one click and one
// sentence.

export interface DiagnosticEvent {
    at: Date;
    title?: string;
    message: string;
    // Route the user was on, never the full URL: a domain name may be
    // something they'd rather not publish.
    route: string;
}

// Only the last few events are worth reporting, and keeping the buffer small
// keeps the report readable.
const MAX_EVENTS = 8;

function createDiagnosticsStore() {
    const { subscribe, update } = writable<DiagnosticEvent[]>([]);

    const record = (message: string, title?: string) => {
        if (!message) return;

        update((all: DiagnosticEvent[]) => {
            all.unshift({
                at: new Date(),
                title,
                message,
                route: typeof window === "undefined" ? "" : window.location.pathname,
            });
            return all.slice(0, MAX_EVENTS);
        });
    };

    const clear = () => update(() => []);

    return { subscribe, record, clear };
}

export const diagnostics = createDiagnosticsStore();

function formatEvent(evt: DiagnosticEvent): string {
    const when = evt.at.toISOString();
    const what = evt.title ? `${evt.title}: ${evt.message}` : evt.message;
    return `- ${when} on ${evt.route}\n  ${what}`;
}

/**
 * Build the technical part of a bug report, so that the reporter only has to
 * describe what they were doing.
 *
 * It deliberately contains no domain name, no provider settings and no
 * identifier: what we need to route a bug is the version, the browser and the
 * errors happyDomain itself raised.
 */
export async function buildDiagnosticsReport(reportedError?: string): Promise<string> {
    const lines: string[] = [];

    // The failure the user clicked on: it deserves to be read first, before
    // the errors that merely preceded it.
    if (reportedError) {
        lines.push(`Reported error: ${reportedError}`);
        lines.push("");
    }

    try {
        const version = await getVersion();
        const build = [version.version];
        if (version["last-commit"]) build.push(version["last-commit"]);
        if (version["dirty-build"]) build.push("dirty build");
        lines.push(`happyDomain: ${build.join(", ")}`);
    } catch {
        // An instance too old to expose /api/version, or unreachable: that is
        // in itself a useful thing to read in a report.
        lines.push("happyDomain: version unavailable");
    }

    if (typeof window !== "undefined") {
        lines.push(`Instance: ${window.location.origin}`);
        lines.push(`Page: ${window.location.pathname}`);
    }
    if (typeof navigator !== "undefined") {
        lines.push(`Browser: ${navigator.userAgent}`);
        lines.push(`Language: ${navigator.language}`);
    }

    const events = get(diagnostics);
    if (events.length) {
        lines.push("");
        lines.push("Errors recorded by happyDomain:");
        lines.push(...events.map(formatEvent));
    }

    return lines.join("\n");
}
