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

const ORIGIN = makeDomain();

const TYPE_CNAME = 5;
const TYPE_DNAME = 39;
const TYPE_ALIAS = 65282; // one of the private use pseudo-types

function alias(rrtype: number, Target = "target.example.com.", Name = ""): ServiceWithValue {
    return makeService("svcs.Alias", { record: { Hdr: { Name, Rrtype: rrtype }, Target } });
}

const zone = (services: Record<string, ServiceWithValue[]>): Zone => makeZone({ services });

function run(record: unknown, dn = "www", z: Zone | null = null): ComplianceIssue[] {
    const ctx = buildContext(dn, ORIGIN, z);
    const v = getValidators("svcs.Alias");
    expect(v?.sync).toBeDefined();
    return v!.sync!({ record }, ctx);
}

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

const cname = (Target: string, Name = "") => ({ Hdr: { Name, Rrtype: TYPE_CNAME }, Target });

describe("Alias compliance: coexistence", () => {
    it("accepts a lone CNAME", () => {
        expect(run(cname("target.external.tld."))).toEqual([]);
    });

    it("flags a CNAME sharing its name with another service", () => {
        const z = zone({ www: [makeService("abstract.Server", {})] });
        expect(ids(run(cname("target.external.tld."), "www", z))).toContain(
            "alias.cname-not-alone",
        );
    });

    it("flags a CNAME at the apex", () => {
        expect(ids(run(cname("target.external.tld."), ""))).toContain("alias.cname-at-apex");
    });

    it("leaves the provider-resolved kinds alone at the apex", () => {
        const record = {
            Hdr: { Name: "", Rrtype: TYPE_ALIAS },
            Data: { Target: "lb.provider.tld." },
        };
        expect(run(record, "")).toEqual([]);
    });

    it("flags a DNAME conflicting with a CNAME at the same name", () => {
        const z = zone({ www: [alias(TYPE_CNAME)] });
        const record = { Hdr: { Name: "", Rrtype: TYPE_DNAME }, Target: "elsewhere.tld." };
        expect(ids(run(record, "www", z))).toContain("alias.dname-not-alone");
    });
});

describe("Alias compliance: target", () => {
    it("flags an empty target", () => {
        expect(ids(run(cname("")))).toEqual(["alias.empty-target"]);
    });

    it("flags a syntactically invalid target", () => {
        expect(ids(run(cname("not a hostname")))).toEqual(["alias.invalid-target"]);
    });

    it("flags a record aliasing its own name", () => {
        expect(ids(run(cname("www.example.com.")))).toEqual(["alias.cname-loop"]);
    });

    it("reads the target of the pseudo-types under Data", () => {
        const record = { Hdr: { Name: "", Rrtype: TYPE_ALIAS }, Data: { Target: "" } };
        expect(ids(run(record))).toEqual(["alias.empty-target"]);
    });

    it("warns when the in-zone target is itself a CNAME", () => {
        const z = zone({ other: [alias(TYPE_CNAME)] });
        expect(ids(run(cname("other.example.com."), "www", z))).toContain("alias.cname-chain");
    });

    it("spots a SubAlias as an alias target too", () => {
        const z = zone({
            other: [makeService("svcs.SpecialCNAME", { cname: { Hdr: { Name: "" } } })],
        });
        expect(ids(run(cname("other.example.com."), "www", z))).toContain("alias.cname-chain");
    });

    it("does not treat a SubAlias under an underscore name as its subdomain's own CNAME", () => {
        const z = zone({
            other: [makeService("svcs.SpecialCNAME", { cname: { Hdr: { Name: "_sip._tcp" } } })],
        });
        expect(ids(run(cname("other.example.com."), "www", z))).not.toContain("alias.cname-chain");
    });

    it("warns when the in-zone target publishes nothing", () => {
        const z = zone({ www: [] });
        expect(ids(run(cname("nowhere.example.com."), "www", z))).toContain(
            "alias.dangling-target",
        );
    });

    it("stays quiet when the in-zone target has records", () => {
        const z = zone({ other: [makeService("abstract.Server", {})] });
        expect(ids(run(cname("other.example.com."), "www", z))).toEqual([]);
    });

    it("does not follow the target of a provider-resolved alias", () => {
        const z = zone({ other: [alias(TYPE_CNAME)] });
        const record = {
            Hdr: { Name: "", Rrtype: TYPE_ALIAS },
            Data: { Target: "other.example.com." },
        };
        expect(run(record, "www", z)).toEqual([]);
    });

    it("leaves external targets alone", () => {
        expect(run(cname("target.external.tld."), "www", zone({}))).toEqual([]);
    });
});
