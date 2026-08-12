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

import { beforeEach, describe, expect, it, vi } from "vitest";

// The instance answers with a known version, so the report is predictable.
vi.mock("$lib/api/version", () => ({
    getVersion: vi.fn(async () => ({ version: "1.2.3", "last-commit": "deadbeef" })),
}));

import { buildDiagnosticsReport, diagnostics } from "./diagnostics";

describe("buildDiagnosticsReport", () => {
    beforeEach(() => diagnostics.clear());

    it("reports the version and the errors happyDomain met", async () => {
        diagnostics.record("NetworkError when attempting to fetch resource.", "An error occured!");

        const report = await buildDiagnosticsReport();

        expect(report).toContain("happyDomain: 1.2.3, deadbeef");
        expect(report).toContain("Errors recorded by happyDomain:");
        expect(report).toContain("NetworkError when attempting to fetch resource.");
    });

    it("leads with the error the user chose to report", async () => {
        diagnostics.record("an earlier hiccup");

        const report = await buildDiagnosticsReport("An error occured!: the zone did not load");

        expect(report.split("\n")[0]).toBe(
            "Reported error: An error occured!: the zone did not load",
        );
        expect(report).toContain("an earlier hiccup");
    });

    it("keeps only the last errors", async () => {
        for (let i = 0; i < 20; i++) diagnostics.record(`error ${i}`);

        const report = await buildDiagnosticsReport();

        expect(report).toContain("error 19");
        expect(report).not.toContain("error 11");
    });
});
