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

import { describe, it, expect } from "vitest";
import "./compliance";
import {
    buildContext,
    getValidators,
    type ComplianceIssue,
} from "$lib/services/compliance";
import type { Zone } from "$lib/model/zone";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";

const ORIGIN = makeDomain();

const svc = (svctype: string, service: Record<string, unknown> = {}) =>
    makeService(svctype, service);

// An Alias service holding a record of the given type.
function alias(rrtype: number): ServiceWithValue {
    return svc("svcs.Alias", { record: { Hdr: { Rrtype: rrtype } } });
}

const zone = (services: Record<string, ServiceWithValue[]>): Zone => makeZone({ services });

function run(mx: unknown, z: Zone | null = null): ComplianceIssue[] {
    const ctx = buildContext("", ORIGIN, z);
    const v = getValidators("svcs.MXs");
    expect(v?.sync).toBeDefined();
    return v!.sync!({ mx }, ctx);
}

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

const MX = (Mx: string, Preference = 10) => ({ Mx, Preference });

describe("MX compliance: empty / well-formed", () => {
    it("returns no issues on empty list", () => {
        expect(run([])).toEqual([]);
    });
    it("returns no issues on a single valid external target", () => {
        expect(run([MX("mail.external.tld.")])).toEqual([]);
    });
    it("normalizes a single record (not wrapped in array)", () => {
        expect(run(MX("mail.external.tld."))).toEqual([]);
    });
});

describe("MX compliance: null MX (RFC 7505)", () => {
    it("accepts a sole null MX with preference 0", () => {
        const issues = run([{ Mx: ".", Preference: 0 }]);
        expect(issues).toEqual([]);
    });
    it("flags null MX coexisting with another record", () => {
        const issues = run([{ Mx: ".", Preference: 0 }, MX("mail.external.tld.")]);
        expect(ids(issues)).toContain("mx.null-mx-with-others");
    });
    it("warns when null MX preference is non-zero", () => {
        const issues = run([{ Mx: ".", Preference: 10 }]);
        expect(ids(issues)).toContain("mx.null-mx-non-zero-preference");
    });
});

describe("MX compliance: target validity", () => {
    it("flags a record left without a target", () => {
        expect(ids(run([MX("")]))).toEqual(["mx.empty-target"]);
    });
    it("does not take an empty target for a null MX", () => {
        const found = ids(run([MX(""), MX("mail.external.tld.")]));
        expect(found).toContain("mx.empty-target");
        expect(found).not.toContain("mx.null-mx-with-others");
    });
    it("also flags a whitespace-only or missing target", () => {
        expect(ids(run([MX("   ")]))).toEqual(["mx.empty-target"]);
        expect(ids(run([{ Preference: 10 }]))).toEqual(["mx.empty-target"]);
    });
    it("locates the empty record by its index", () => {
        const empty = run([MX("mail.external.tld."), MX("")]).find(
            (i) => i.id === "mx.empty-target",
        );
        expect(empty?.field).toBe("mx[1]");
        expect(empty?.severity).toBe("error");
    });
    it("keeps validating the other records next to an empty one", () => {
        expect(ids(run([MX(""), MX("mail server.tld.")]))).toEqual([
            "mx.empty-target",
            "mx.invalid-target",
        ]);
    });
    it("does not flag a sole null MX as empty", () => {
        expect(ids(run([{ Mx: ".", Preference: 0 }]))).not.toContain("mx.empty-target");
    });
    it("still reports a null MX alongside an empty record", () => {
        const found = ids(run([{ Mx: ".", Preference: 0 }, MX("")]));
        expect(found).toContain("mx.null-mx-with-others");
        expect(found).toContain("mx.empty-target");
    });
    it("flags an invalid hostname target", () => {
        const issues = run([MX("mail server.tld.")]);
        expect(ids(issues)).toContain("mx.invalid-target");
    });
    it("flags an out-of-range preference", () => {
        const issues = run([{ Mx: "mail.external.tld.", Preference: 70000 }]);
        expect(ids(issues)).toContain("mx.invalid-preference");
    });
    it("flags duplicate targets case-insensitively", () => {
        const issues = run([MX("mail.external.tld.", 10), MX("MAIL.External.tld.", 20)]);
        expect(ids(issues)).toContain("mx.duplicate-target");
    });
});

describe("MX compliance: in-zone cross checks", () => {
    it("flags MX target that is a CNAME owner in the zone", () => {
        const z = zone({ mail: [alias(5)] });
        const issues = run([MX("mail.example.com.")], z);
        expect(ids(issues)).toContain("mx.target-is-cname");
    });
    it("does not also warn about a missing address when the target is a CNAME", () => {
        const z = zone({ mail: [alias(5)] });
        expect(ids(run([MX("mail.example.com.")], z))).not.toContain("mx.target-no-address");
    });
    it("does not flag MX target that is an ALIAS owner in the zone", () => {
        // An ALIAS is resolved into addresses by the provider, RFC 5321 sec.
        // 5.1 does not stand in its way.
        const z = zone({ mail: [alias(65280)] });
        const issues = run([MX("mail.example.com.")], z);
        expect(ids(issues)).not.toContain("mx.target-is-cname");
        expect(ids(issues)).not.toContain("mx.target-no-address");
    });
    it("warns when in-zone target has no A/AAAA service", () => {
        const z = zone({});
        const issues = run([MX("mail.example.com.")], z);
        expect(ids(issues)).toContain("mx.target-no-address");
    });
    it("does not warn when in-zone target has an abstract.Server", () => {
        const z = zone({ mail: [svc("abstract.Server")] });
        const issues = run([MX("mail.example.com.")], z);
        expect(ids(issues)).not.toContain("mx.target-no-address");
    });
    it("does not warn for external targets", () => {
        const z = zone({});
        const issues = run([MX("mail.elsewhere.tld.")], z);
        expect(ids(issues)).not.toContain("mx.target-no-address");
        expect(ids(issues)).not.toContain("mx.target-is-cname");
    });
    it("matches apex target to subdomain ''", () => {
        const z = zone({ "": [svc("abstract.Server")] });
        const issues = run([MX("example.com.")], z);
        expect(ids(issues)).not.toContain("mx.target-no-address");
    });
});
