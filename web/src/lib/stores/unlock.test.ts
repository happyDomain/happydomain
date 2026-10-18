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

import { get } from "svelte/store";
import { describe, expect, it } from "vitest";

import { requestUnlock, settleUnlock, unlockRequest } from "./unlock";

describe("requestUnlock", () => {
    it("shows one keystore at a time, sharing the dialog of the same one", async () => {
        const a1 = requestUnlock({ ref: "a" });
        const a2 = requestUnlock({ ref: "a" });
        const b = requestUnlock({ ref: "b" });

        expect(get(unlockRequest)?.locked.ref).toBe("a");
        settleUnlock(true);
        await expect(a1).resolves.toBe(true);
        await expect(a2).resolves.toBe(true);

        expect(get(unlockRequest)?.locked.ref).toBe("b");
        settleUnlock(false);
        await expect(b).resolves.toBe(false);
        expect(get(unlockRequest)).toBeNull();
    });

    it("shares the dialog of a keystore waiting in the queue", async () => {
        const a = requestUnlock({ ref: "a" });
        const b1 = requestUnlock({ ref: "b" });
        const b2 = requestUnlock({ ref: "b" });

        settleUnlock(true);
        await expect(a).resolves.toBe(true);
        expect(get(unlockRequest)?.locked.ref).toBe("b");

        settleUnlock(true);
        await expect(b1).resolves.toBe(true);
        await expect(b2).resolves.toBe(true);
        expect(get(unlockRequest)).toBeNull();
    });
});
