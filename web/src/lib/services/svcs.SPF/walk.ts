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

import { flattenSPF } from "$lib/api/resolver";
import type { HappydnsSpfFlattenResponse } from "$lib/api-base/types.gen";

/** DNS lookups a verifier performs at most, RFC 7208 sec. 4.6.4. */
export const SPF_LOOKUP_MAX = 10;

let last: { key: string; walk: Promise<HappydnsSpfFlattenResponse> } | null = null;

/**
 * Resolves every include and redirect of `record`, as if published at
 * `domain`. The compliance check and the lookup counter of the editor both
 * need the walk of the record being edited, so the last one is shared rather
 * than asked twice. An aborted caller only stops waiting for it.
 */
export function walkSPF(
    domain: string,
    record: string,
    signal: AbortSignal,
): Promise<HappydnsSpfFlattenResponse> {
    const key = domain + " " + record;
    if (last?.key !== key) {
        const walk = flattenSPF({ domain, record });
        // A failed walk is forgotten, so that the next caller tries again.
        walk.catch(() => {
            if (last?.walk === walk) last = null;
        });
        last = { key, walk };
    }
    const walk = last.walk;

    if (signal.aborted) return Promise.reject(signal.reason);
    return new Promise((resolve, reject) => {
        const onAbort = () => reject(signal.reason);
        signal.addEventListener("abort", onAbort, { once: true });
        walk.then(resolve, reject).finally(() => signal.removeEventListener("abort", onAbort));
    });
}

/** Forgets the shared walk, for tests to start afresh. */
export function forgetSPFWalk(): void {
    last = null;
}
