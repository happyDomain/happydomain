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

import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("$lib/api/resolver", () => ({
    fetchMTAStsPolicy: vi.fn(),
}));

import "./compliance";
import { buildContext, getValidators, type ComplianceIssue } from "$lib/services/compliance";
import type { Domain } from "$lib/model/domain";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import type { Zone } from "$lib/model/zone";
import { fetchMTAStsPolicy } from "$lib/api/resolver";

const ORIGIN = { domain: "example.com." } as unknown as Domain;
const CTX = buildContext("_mta-sts", ORIGIN, null);

function run(txt: string, name = "_mta-sts.example.com."): ComplianceIssue[] {
    const v = getValidators("svcs.MTA_STS");
    return v!.sync!({ txt: { Hdr: { Name: name }, Txt: txt } }, CTX);
}
const ids = (issues: ComplianceIssue[]) => issues.map((i) => i.id);

function mxSvc(targets: string[]): ServiceWithValue {
    return {
        _svctype: "svcs.MXs",
        Service: { mx: targets.map((t, i) => ({ Mx: t, Preference: 10 + i })) },
    } as unknown as ServiceWithValue;
}
function makeZone(apexMx: string[]): Zone {
    return { services: { "": [mxSvc(apexMx)] } } as unknown as Zone;
}

async function runAsync(zone: Zone | null, policyOverride: Record<string, any>): Promise<ComplianceIssue[]> {
    const v = getValidators("svcs.MTA_STS");
    expect(v?.async).toBeDefined();
    (fetchMTAStsPolicy as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        status: "ok",
        url: "https://mta-sts.example.com/.well-known/mta-sts.txt",
        version: "STSv1",
        mode: "enforce",
        maxAge: 604800,
        ...policyOverride,
    });
    const ctx = buildContext("_mta-sts", ORIGIN, zone);
    return v!.async!(
        { txt: { Hdr: { Name: "_mta-sts.example.com." }, Txt: "v=STSv1;id=20240101" } },
        ctx,
        new AbortController().signal,
    );
}

describe("MTA-STS compliance", () => {
    it("accepts a clean record", () => {
        expect(ids(run("v=STSv1;id=20240101"))).toEqual([]);
    });
    it("flags a wrong owner name", () => {
        expect(ids(run("v=STSv1;id=2024", "example.com."))).toContain("mta_sts.wrong-owner-name");
    });
    it("flags missing version", () => {
        expect(ids(run("id=2024"))).toContain("mta_sts.missing-version");
    });
    it("flags non-STSv1 version", () => {
        expect(ids(run("v=STSv2;id=2024"))).toContain("mta_sts.invalid-version");
    });
    it("flags missing id", () => {
        expect(ids(run("v=STSv1"))).toContain("mta_sts.missing-id");
    });
    it("flags an id with non-alphanumeric chars", () => {
        expect(ids(run("v=STSv1;id=2024-01-01"))).toContain("mta_sts.invalid-id");
    });
    it("flags an id longer than 32 chars", () => {
        expect(ids(run("v=STSv1;id=" + "a".repeat(33)))).toContain("mta_sts.invalid-id");
    });
    it("returns no issue on empty TXT", () => {
        expect(run("")).toEqual([]);
    });
});

describe("MTA-STS cross-check: policy mx vs zone MX", () => {
    beforeEach(() => {
        (fetchMTAStsPolicy as unknown as ReturnType<typeof vi.fn>).mockReset();
    });

    it("does not flag when every zone MX matches a policy pattern", async () => {
        const issues = await runAsync(makeZone(["mx1.example.com.", "mx2.example.com."]), {
            mx: ["mx1.example.com", "mx2.example.com"],
        });
        expect(ids(issues)).not.toContain("mta_sts.zone-mx-not-covered");
        expect(ids(issues)).not.toContain("mta_sts.policy-mx-unused");
        expect(ids(issues)).not.toContain("mta_sts.zone-no-mx");
    });

    it("flags zone MX not covered by any policy pattern (error in enforce)", async () => {
        const issues = await runAsync(makeZone(["mx1.example.com.", "rogue.example.com."]), {
            mode: "enforce",
            mx: ["mx1.example.com"],
        });
        const e = issues.find((i) => i.id === "mta_sts.zone-mx-not-covered");
        expect(e).toBeDefined();
        expect(e!.severity).toBe("error");
        expect(e!.params?.host).toBe("rogue.example.com.");
    });

    it("downgrades to warning in testing mode", async () => {
        const issues = await runAsync(makeZone(["rogue.example.com."]), {
            mode: "testing",
            mx: ["mx1.example.com"],
        });
        const e = issues.find((i) => i.id === "mta_sts.zone-mx-not-covered");
        expect(e?.severity).toBe("warning");
    });

    it("supports wildcard patterns (one label only)", async () => {
        const issues = await runAsync(
            makeZone(["mx1.mail.example.com.", "deep.nested.mail.example.com."]),
            { mx: ["*.mail.example.com"] },
        );
        const flagged = issues.filter((i) => i.id === "mta_sts.zone-mx-not-covered");
        expect(flagged).toHaveLength(1);
        expect(flagged[0].params?.host).toBe("deep.nested.mail.example.com.");
    });

    it("flags policy patterns that match no MX (info)", async () => {
        const issues = await runAsync(makeZone(["mx1.example.com."]), {
            mx: ["mx1.example.com", "ghost.example.com"],
        });
        const u = issues.find((i) => i.id === "mta_sts.policy-mx-unused");
        expect(u?.severity).toBe("info");
        expect(u?.params?.pattern).toBe("ghost.example.com");
    });

    it("warns when policy lists mx but the zone has none", async () => {
        const issues = await runAsync(makeZone([]), { mx: ["mx1.example.com"] });
        expect(ids(issues)).toContain("mta_sts.zone-no-mx");
    });

    it("skips cross-check when mode is none", async () => {
        const issues = await runAsync(makeZone(["rogue.example.com."]), {
            mode: "none",
            mx: ["mx1.example.com"],
        });
        expect(ids(issues)).not.toContain("mta_sts.zone-mx-not-covered");
        expect(ids(issues)).not.toContain("mta_sts.policy-mx-unused");
    });

    it("skips cross-check when zone is unknown", async () => {
        const issues = await runAsync(null, { mx: ["mx1.example.com"] });
        expect(ids(issues)).not.toContain("mta_sts.zone-mx-not-covered");
        expect(ids(issues)).not.toContain("mta_sts.zone-no-mx");
    });
});
