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

import type { AbstractMTASTSBody } from "$lib/services_bodies";

export { parseMTASTS, stringifyMTASTS, type MTASTSValue } from "../svcs.MTA_STS/model";

/** One week, the lower bound RFC 8461 sec. 3.2 recommends. */
export const DEFAULT_MAX_AGE = 604800;

/**
 * The policy id RFC 8461 sec. 3.1 wants changed on every policy update.
 *
 * A UTC timestamp is the convention the RFC's own examples use, and it makes
 * the DNS record self-documenting: `20260814T101500Z` says when the policy
 * behind it was last touched.
 */
export function newPolicyId(now: Date = new Date()): string {
    return now.toISOString().replace(/[-:]/g, "").replace(/\.\d+/, "");
}

/**
 * Renders the policy exactly as happyDomain will serve it (RFC 8461 sec. 3.2),
 * so the editor can show the user the file itself rather than a description of
 * it. Kept byte-for-byte in step with (*abstract.MTASTS).PolicyFile.
 */
export function renderPolicy(value: AbstractMTASTSBody): string {
    const mode = value.mode || "testing";
    const maxAge = value.maxAge || DEFAULT_MAX_AGE;

    const lines = ["version: STSv1", "mode: " + mode];
    for (const raw of value.mx ?? []) {
        const mx = raw.trim();
        if (!mx) continue;
        lines.push("mx: " + mx.replace(/\.$/, ""));
    }
    lines.push("max_age: " + maxAge);

    return lines.join("\r\n") + "\r\n";
}

/**
 * A snapshot of everything that ends up in the policy file, used to tell an
 * edit apart from merely opening the form.
 */
export function policyFingerprint(value: AbstractMTASTSBody): string {
    return JSON.stringify({
        mode: value.mode ?? "",
        maxAge: value.maxAge ?? 0,
        mx: (value.mx ?? []).map((mx) => mx.trim()).filter(Boolean),
    });
}
