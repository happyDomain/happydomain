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

const subalias = (Name: string, Target: string) => ({ Hdr: { Name, Rrtype: 5 }, Target });

const zone = (services: Record<string, ServiceWithValue[]>): Zone => makeZone({ services });

function run(cname: unknown, dn = "", z: Zone | null = null): ComplianceIssue[] {
    const ctx = buildContext(dn, ORIGIN, z);
    const v = getValidators("svcs.SpecialCNAME");
    expect(v?.sync).toBeDefined();
    return v!.sync!({ cname }, ctx);
}

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

describe("SubAlias compliance", () => {
    it("accepts a lone SubAlias", () => {
        expect(run(subalias("_sip._tcp", "sip.provider.tld."))).toEqual([]);
    });

    it("shares the target checks of the plain aliases", () => {
        expect(ids(run(subalias("_sip._tcp", "")))).toEqual(["alias.empty-target"]);
        expect(ids(run(subalias("_sip._tcp", "not a hostname")))).toEqual(["alias.invalid-target"]);
    });

    it("spots a loop on the underscore name it carries", () => {
        expect(ids(run(subalias("_sip._tcp", "_sip._tcp.example.com.")))).toEqual([
            "alias.cname-loop",
        ]);
    });

    it("flags another SubAlias published under the same service name", () => {
        const other = makeService("svcs.SpecialCNAME", {
            cname: subalias("_sip._tcp", "elsewhere.tld."),
        });
        const issues = run(subalias("_sip._tcp", "sip.provider.tld."), "", zone({ "": [other] }));
        expect(ids(issues)).toContain("alias.special-cname-not-alone");
    });

    it("lets two SubAliases of different services coexist", () => {
        const other = makeService("svcs.SpecialCNAME", {
            cname: subalias("_xmpp._tcp", "xmpp.provider.tld."),
        });
        expect(run(subalias("_sip._tcp", "sip.provider.tld."), "", zone({ "": [other] }))).toEqual(
            [],
        );
    });

    it("ignores the other services of the subdomain, which sit at another name", () => {
        const z = zone({ "": [makeService("abstract.Server", {})] });
        expect(run(subalias("_sip._tcp", "sip.provider.tld."), "", z)).toEqual([]);
    });
});
