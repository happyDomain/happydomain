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
import "../abstract.SIP/compliance";
import "../abstract.Kerberos/compliance";
import { buildContext, getValidators, type ComplianceIssue } from "$lib/services/compliance";
import type { Zone } from "$lib/model/zone";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";

const ORIGIN = makeDomain();

const zone = (services: Record<string, ServiceWithValue[]>): Zone => makeZone({ services });

const SRV = (
    Target: string,
    Port = 5060,
    overrides: Record<string, unknown> = {},
): Record<string, unknown> => ({
    Hdr: { Name: "_sip._tcp" },
    Priority: 10,
    Weight: 10,
    Port,
    Target,
    ...overrides,
});

function run(
    raw: Record<string, unknown>,
    svctype = "svcs.UnknownSRV",
    z: Zone | null = null,
): ComplianceIssue[] {
    const ctx = buildContext("", ORIGIN, z);
    const v = getValidators(svctype);
    expect(v?.sync).toBeDefined();
    return v!.sync!(raw, ctx);
}

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

describe("SRV compliance: owner name", () => {
    it("accepts a well-formed record set", () => {
        expect(run({ srv: [SRV("sip.provider.tld.")] })).toEqual([]);
    });

    it("flags an owner that is not _service._proto", () => {
        const issues = run({ srv: [SRV("sip.provider.tld.", 5060, { Hdr: { Name: "sip" } })] });
        expect(ids(issues)).toEqual(["srv.invalid-owner"]);
    });

    it("accepts an owner carrying the served name too", () => {
        const record = SRV("sip.provider.tld.", 5060, { Hdr: { Name: "_sip._tcp.branch" } });
        expect(run({ srv: [record] })).toEqual([]);
    });
});

describe("SRV compliance: fields", () => {
    it("flags an out-of-range priority, weight and port", () => {
        const record = SRV("sip.provider.tld.", 70000, { Priority: -1, Weight: 1.5 });
        expect(ids(run({ srv: [record] }))).toEqual([
            "srv.invalid-priority",
            "srv.invalid-weight",
            "srv.invalid-port",
        ]);
    });

    it("flags port 0 on a real target", () => {
        expect(ids(run({ srv: [SRV("sip.provider.tld.", 0)] }))).toEqual(["srv.port-zero"]);
    });

    it("flags an invalid target", () => {
        expect(ids(run({ srv: [SRV("not a host")] }))).toEqual(["srv.invalid-target"]);
    });

    it("warns on a duplicate host and port", () => {
        const issues = run({ srv: [SRV("sip.provider.tld."), SRV("SIP.provider.tld")] });
        expect(ids(issues)).toEqual(["srv.duplicate"]);
    });
});

describe("SRV compliance: unavailable service", () => {
    it('accepts a lone target of "."', () => {
        expect(run({ srv: [SRV(".", 0)] })).toEqual([]);
    });

    it('flags a target of "." next to a real one', () => {
        const issues = run({ srv: [SRV(".", 0), SRV("sip.provider.tld.")] });
        expect(ids(issues)).toEqual(["srv.unavailable-with-others"]);
    });
});

describe("SRV compliance: weights", () => {
    it("reports a zero weight sitting next to a non-zero one", () => {
        const issues = run({
            srv: [SRV("a.provider.tld."), SRV("b.provider.tld.", 5061, { Weight: 0 })],
        });
        expect(ids(issues)).toEqual(["srv.zero-weight-mixed"]);
    });

    it("stays quiet when every weight of a priority is zero", () => {
        const issues = run({
            srv: [
                SRV("a.provider.tld.", 5060, { Weight: 0 }),
                SRV("b.provider.tld.", 5061, { Weight: 0 }),
            ],
        });
        expect(issues).toEqual([]);
    });

    it("compares weights within a priority, not across them", () => {
        const issues = run({
            srv: [
                SRV("a.provider.tld."),
                SRV("b.provider.tld.", 5061, { Priority: 20, Weight: 0 }),
            ],
        });
        expect(issues).toEqual([]);
    });
});

describe("SRV compliance: in-zone targets", () => {
    it("flags a target that is an alias", () => {
        const z = zone({ sip: [makeService("svcs.Alias", { record: { Hdr: { Rrtype: 5 } } })] });
        expect(ids(run({ srv: [SRV("sip.example.com.")] }, "svcs.UnknownSRV", z))).toEqual([
            "srv.target-is-cname",
        ]);
    });

    it("warns on a target with no address in the zone", () => {
        expect(ids(run({ srv: [SRV("sip.example.com.")] }, "svcs.UnknownSRV", zone({})))).toEqual([
            "srv.target-no-address",
        ]);
    });
});

describe("SRV compliance: sharing across services", () => {
    it("reads the records field of the abstract services", () => {
        expect(ids(run({ records: [SRV("not a host")] }, "abstract.SIP"))).toEqual([
            "srv.invalid-target",
        ]);
    });

    it("reads every role of a Kerberos realm", () => {
        const issues = run(
            {
                kdc_tcp: [SRV("kdc.example.org.", 88, { Hdr: { Name: "_kerberos._tcp" } })],
                admin: [SRV("not a host", 749, { Hdr: { Name: "_kerberos-adm._tcp" } })],
            },
            "abstract.Kerberos",
        );
        expect(ids(issues)).toEqual(["srv.invalid-target"]);
    });

    it("keeps the record sets of two owner names independent", () => {
        const issues = run(
            {
                records: [
                    SRV(".", 0, { Hdr: { Name: "_sip._tcp" } }),
                    SRV("sip.provider.tld.", 5060, { Hdr: { Name: "_sip._udp" } }),
                ],
            },
            "abstract.SIP",
        );
        expect(issues).toEqual([]);
    });
});
