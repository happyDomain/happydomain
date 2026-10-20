// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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
import { buildContext, getValidators, type ComplianceIssue } from "$lib/services/compliance";
import type { Zone } from "$lib/model/zone";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";

const REVERSE4 = makeDomain({ domain: "2.0.192.in-addr.arpa." });
const REVERSE6 = makeDomain({ domain: "8.b.d.0.1.0.0.2.ip6.arpa." });
const FORWARD = makeDomain();

const zone = (services: Record<string, ServiceWithValue[]>): Zone => makeZone({ services });

const PTR = (Ptr: string) => ({ Hdr: { Name: "" }, Ptr });

function run(
    Record: unknown,
    dn = "42",
    origin = REVERSE4,
    z: Zone | null = null,
): ComplianceIssue[] {
    const ctx = buildContext(dn, origin, z);
    const v = getValidators("svcs.PTR");
    expect(v?.sync).toBeDefined();
    return v!.sync!({ Record }, ctx);
}

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

describe("PTR compliance: target", () => {
    it("accepts an absolute host name in a reverse zone", () => {
        expect(run(PTR("mail.example.com."))).toEqual([]);
    });

    it("flags an empty target", () => {
        expect(ids(run(PTR("")))).toEqual(["ptr.empty-target"]);
    });

    it("flags an invalid target", () => {
        expect(ids(run(PTR("not a host")))).toEqual(["ptr.invalid-target"]);
    });

    it("notes a target left relative", () => {
        expect(ids(run(PTR("mail.example.com")))).toEqual(["ptr.relative-target"]);
    });
});

describe("PTR compliance: owner name", () => {
    it("accepts an IPv6 nibble under ip6.arpa", () => {
        expect(run(PTR("mail.example.com."), "1.0.0.0", REVERSE6)).toEqual([]);
    });

    it("flags a label that cannot be part of an IPv4 reverse name", () => {
        expect(ids(run(PTR("mail.example.com."), "300"))).toEqual(["ptr.bad-reverse-label"]);
    });

    it("flags a label that cannot be part of an IPv6 reverse name", () => {
        expect(ids(run(PTR("mail.example.com."), "ff", REVERSE6))).toEqual([
            "ptr.bad-reverse-label",
        ]);
    });

    it("notes a PTR published outside of any reverse zone", () => {
        expect(ids(run(PTR("mail.example.com."), "host", FORWARD))).toEqual(["ptr.forward-zone"]);
    });
});

describe("PTR compliance: coexistence", () => {
    it("warns when the reverse name carries other records", () => {
        const z = zone({ "42": [makeService("svcs.TXT", {})] });
        expect(ids(run(PTR("mail.example.com."), "42", REVERSE4, z))).toEqual(["ptr.not-alone"]);
    });

    it("does not count the record being edited as its own sibling", () => {
        const record = PTR("mail.example.com.");
        const z = zone({ "42": [makeService("svcs.PTR", { Record: record })] });
        const ctx = buildContext("42", REVERSE4, z);
        expect(getValidators("svcs.PTR")!.sync!({ Record: record }, ctx)).toEqual([]);
    });

    it("flags a target that is an alias of the edited zone", () => {
        const forward = zone({
            mail: [makeService("svcs.Alias", { record: { Hdr: { Rrtype: 5 } } })],
        });
        expect(ids(run(PTR("mail.example.com."), "host", FORWARD, forward))).toEqual([
            "ptr.forward-zone",
            "ptr.target-is-cname",
        ]);
    });
});

describe("PTR compliance: edge cases", () => {
    const CNAME = () => makeService("svcs.Alias", { record: { Hdr: { Rrtype: 5 } } });

    it("ignores a service without a record", () => {
        expect(run(undefined)).toEqual([]);
    });

    it("treats a missing or blank Ptr as empty", () => {
        expect(ids(run({ Hdr: { Name: "" } }))).toEqual(["ptr.empty-target"]);
        expect(ids(run(PTR("   ")))).toEqual(["ptr.empty-target"]);
    });

    it("trims the target before validating it", () => {
        expect(run(PTR(" mail.example.com. "))).toEqual([]);
    });

    it.each([".", "_dmarc.example.com.", "a..b."])("flags %s as an invalid target", (t) => {
        expect(ids(run(PTR(t)))).toEqual(["ptr.invalid-target"]);
    });

    it("keeps the owner and coexistence issues next to an empty target", () => {
        const z = zone({ "300": [makeService("svcs.TXT", {})] });
        expect(ids(run(PTR(""), "300", REVERSE4, z))).toEqual([
            "ptr.bad-reverse-label",
            "ptr.not-alone",
            "ptr.empty-target",
        ]);
    });

    it("reports the field on the target issues", () => {
        expect(run(PTR(""))[0].field).toBe("ptr");
        expect(run(PTR("not a host"))[0]).toMatchObject({
            field: "ptr",
            params: { target: "not a host" },
        });
    });

    it.each(["0", "9", "255"])("accepts %s as an IPv4 reverse label", (l) => {
        expect(run(PTR("mail.example.com."), l)).toEqual([]);
    });

    it.each(["256", "01", "007", "a", "-1", "1x"])("flags %s as an IPv4 reverse label", (l) => {
        expect(ids(run(PTR("mail.example.com."), l))).toEqual(["ptr.bad-reverse-label"]);
    });

    it("accepts uppercase nibbles", () => {
        expect(run(PTR("mail.example.com."), "A.F.0.9", REVERSE6)).toEqual([]);
    });

    it("flags a multi-digit decimal under ip6.arpa", () => {
        expect(ids(run(PTR("mail.example.com."), "10", REVERSE6))).toEqual([
            "ptr.bad-reverse-label",
        ]);
    });

    it("reports the first bad label and the RFC matching the tree", () => {
        const v4 = run(PTR("mail.example.com."), "1.999.2");
        expect(v4[0].params).toMatchObject({ label: "999" });
        expect(v4[0].docUrl).toContain("rfc1035");
        const v6 = run(PTR("mail.example.com."), "zz", REVERSE6);
        expect(v6[0].docUrl).toContain("rfc3596");
    });

    it("accepts the apex of a reverse zone", () => {
        expect(run(PTR("mail.example.com."), "")).toEqual([]);
    });

    it("does not take a lookalike suffix for a reverse tree", () => {
        const origin = makeDomain({ domain: "notin-addr.arpa." });
        expect(ids(run(PTR("mail.example.com."), "host", origin))).toEqual(["ptr.forward-zone"]);
    });

    it("reports the owner FQDN in forward-zone", () => {
        expect(run(PTR("mail.example.com."), "host", FORWARD)[0].params).toEqual({
            subdomain: "host.example.com",
        });
    });

    it("warns about another PTR at the same name", () => {
        const other = makeService("svcs.PTR", { Record: PTR("other.example.com.") });
        const z = zone({ "42": [other] });
        expect(ids(run(PTR("mail.example.com."), "42", REVERSE4, z))).toEqual(["ptr.not-alone"]);
    });

    it("reports @ as the subdomain when the apex carries other records", () => {
        const z = zone({ "": [makeService("svcs.TXT", {})] });
        const issue = run(PTR("mail.example.com."), "", REVERSE4, z).find(
            (i) => i.id === "ptr.not-alone",
        );
        expect(issue?.params).toEqual({ subdomain: "@" });
    });

    it("ignores a target outside the edited zone", () => {
        const forward = zone({ mail: [CNAME()] });
        expect(ids(run(PTR("mail.other.org."), "host", FORWARD, forward))).toEqual([
            "ptr.forward-zone",
        ]);
    });

    it("checks the alias on a target left relative", () => {
        const forward = zone({ mail: [CNAME()] });
        expect(ids(run(PTR("mail.example.com"), "host", FORWARD, forward))).toEqual([
            "ptr.forward-zone",
            "ptr.relative-target",
            "ptr.target-is-cname",
        ]);
    });

    it("flags a target that is a SubAlias owner", () => {
        const forward = zone({
            mail: [makeService("svcs.SpecialCNAME", { cname: { Hdr: { Name: "" } } })],
        });
        expect(ids(run(PTR("mail.example.com."), "host", FORWARD, forward))).toContain(
            "ptr.target-is-cname",
        );
    });
});
