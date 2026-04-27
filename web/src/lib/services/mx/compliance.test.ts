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
import type { Domain } from "$lib/model/domain";
import type { Zone } from "$lib/model/zone";
import type { ServiceWithValue } from "$lib/model/service.svelte";

const ORIGIN = { domain: "example.com." } as unknown as Domain;

function svc(svctype: string): ServiceWithValue {
    return { _svctype: svctype, Service: {} } as unknown as ServiceWithValue;
}

function zone(services: Record<string, ServiceWithValue[]>): Zone {
    return { services } as unknown as Zone;
}

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
        const z = zone({ mail: [svc("svcs.CNAME")] });
        const issues = run([MX("mail.example.com.")], z);
        expect(ids(issues)).toContain("mx.target-is-cname");
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
