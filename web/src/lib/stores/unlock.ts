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

import type { HappydnsKeystoreLock } from "$lib/api-base/types.gen";

// A keystore to unlock, and the requests waiting for it.
interface UnlockRequest {
    locked: HappydnsKeystoreLock;
    waiters: Array<(unlocked: boolean) => void>;
}

// The request the unlock dialog of the root layout shows, one at a time.
export const unlockRequest = writable<UnlockRequest | null>(null);

const queue: Array<UnlockRequest> = [];

/**
 * Asks the user to unlock a keystore a request found locked. Resolves to true
 * once unlocked, false when they gave up. Requests waiting for the same
 * keystore share one dialog, so concurrent calls do not prompt twice.
 */
export function requestUnlock(locked: HappydnsKeystoreLock): Promise<boolean> {
    return new Promise((resolve) => {
        const current = get(unlockRequest);
        const waiting = [current, ...queue].find((r) => r && r.locked.ref === locked.ref);
        if (waiting) {
            waiting.waiters.push(resolve);
            return;
        }
        const request = { locked, waiters: [resolve] };
        if (current) {
            queue.push(request);
        } else {
            unlockRequest.set(request);
        }
    });
}

/** Ends the request shown, unlocked or given up, and shows the next one. */
export function settleUnlock(unlocked: boolean) {
    const current = get(unlockRequest);
    if (!current) return;
    for (const resolve of current.waiters) resolve(unlocked);
    unlockRequest.set(queue.shift() ?? null);
}
