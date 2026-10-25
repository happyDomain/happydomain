// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
    daysUntil,
    expirationColor,
    expirationLabel,
    expirationProgress,
    formatDate,
    statusColor,
} from "./domaininfo";

const NOW = new Date("2026-10-24T12:00:00Z");
const inDays = (d: number) => new Date(NOW.getTime() + d * 86400000).toISOString();

describe("domaininfo helpers", () => {
    beforeEach(() => {
        vi.useFakeTimers();
        vi.setSystemTime(NOW);
    });
    afterEach(() => {
        vi.useRealTimers();
    });

    it("colors registry statuses", () => {
        expect(statusColor("ok")).toBe("success");
        expect(statusColor("active")).toBe("success");
        expect(statusColor("clientHold")).toBe("danger");
        expect(statusColor("clientTransferProhibited")).toBe("info");
        expect(statusColor("pendingTransfer")).toBe("info");
        expect(statusColor("renewPeriod")).toBe("secondary");
    });

    it("counts days until a date", () => {
        expect(daysUntil(inDays(10))).toBe(10);
        expect(daysUntil(inDays(-3))).toBe(-3);
        expect(daysUntil(NOW)).toBe(0);
    });

    it("colors an expiration date", () => {
        expect(expirationColor(undefined)).toBe("secondary");
        expect(expirationColor(inDays(-1))).toBe("danger");
        expect(expirationColor(inDays(10))).toBe("danger");
        expect(expirationColor(inDays(60))).toBe("warning");
        expect(expirationColor(inDays(200))).toBe("success");
    });

    it("measures the year left", () => {
        expect(expirationProgress(undefined)).toBe(0);
        expect(expirationProgress(inDays(-5))).toBe(0);
        expect(expirationProgress(inDays(73))).toBe(20);
        expect(expirationProgress(inDays(800))).toBe(100);
    });

    it("labels an expiration", () => {
        expect(expirationLabel(-4)).toEqual(["domaininfo.expired", { days: 4 }]);
        expect(expirationLabel(0)).toEqual(["domaininfo.expires-today", {}]);
        expect(expirationLabel(12)).toEqual(["domaininfo.expires-in", { days: 12 }]);
    });

    it("formats a date", () => {
        expect(formatDate(undefined)).toBe("");
        expect(formatDate("2026-10-24T12:00:00Z")).toBe(
            NOW.toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" }),
        );
    });
});
