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

import { describe, expect, it } from "vitest";

import type { DomainLog } from "$lib/model/domain";
import { groupLogsByDay, isProblem, logDay, logSeverity, logZoneVersion } from "./domain_logs";

function log(id: string, date: Date, level = 8, content = ""): DomainLog {
    return { id, date, level, content, id_user: "u" };
}

describe("logSeverity", () => {
    it("maps the levels of model/logs.go", () => {
        expect(logSeverity(0)).toBe("critical");
        expect(logSeverity(1)).toBe("critical");
        expect(logSeverity(3)).toBe("error");
        expect(logSeverity(5)).toBe("warning");
        expect(logSeverity(8)).toBe("info");
        expect(logSeverity(9)).toBe("success");
        expect(logSeverity(10)).toBe("debug");
    });
});

describe("isProblem", () => {
    it("covers weird situations and worse", () => {
        expect(isProblem(log("a", new Date(), 6))).toBe(true);
        expect(isProblem(log("a", new Date(), 3))).toBe(true);
        expect(isProblem(log("a", new Date(), 7))).toBe(false);
        expect(isProblem(log("a", new Date(), 9))).toBe(false);
    });
});

describe("logDay", () => {
    it("uses the local day, zero padded", () => {
        expect(logDay(new Date(2026, 0, 5, 23, 59))).toBe("2026-01-05");
    });
});

describe("groupLogsByDay", () => {
    it("groups by day, most recent first, and keeps the worst level", () => {
        const days = groupLogsByDay([
            log("old", new Date(2026, 9, 12, 10), 8),
            log("morning", new Date(2026, 9, 24, 9), 9),
            log("evening", new Date(2026, 9, 24, 18), 3),
        ]);

        expect(days.map((d) => d.day)).toEqual(["2026-10-24", "2026-10-12"]);
        expect(days[0].logs.map((l) => l.id)).toEqual(["evening", "morning"]);
        expect(days[0].worst).toBe(3);
        expect(days[1].worst).toBe(8);
    });

    it("returns nothing without logs", () => {
        expect(groupLogsByDay([])).toEqual([]);
    });
});

describe("logZoneVersion", () => {
    it("finds the version named in the message", () => {
        const l = log("a", new Date(), 9, "Zone published (abc_-12), 3 corrections applied");
        expect(logZoneVersion(l, ["zzz", "abc_-12"])).toBe("abc_-12");
        expect(logZoneVersion(l, ["zzz"])).toBeUndefined();
        expect(logZoneVersion(l, undefined)).toBeUndefined();
    });
});
