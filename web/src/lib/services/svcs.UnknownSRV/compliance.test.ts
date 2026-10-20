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
import "../abstract.RFC6186/compliance";
import "../abstract.LibravatarServer/compliance";
import "../abstract.XMPP/compliance";
import "../abstract.LDAP/compliance";
import "../abstract.MatrixIM/compliance";
import "../abstract.CalDAV/compliance";
import "../abstract.CardDAV/compliance";
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

describe("SRV compliance: registrations", () => {
    it.each([
        ["svcs.UnknownSRV", ["srv"]],
        ["abstract.RFC6186", ["srv"]],
        ["abstract.LibravatarServer", ["srv"]],
        ["abstract.SIP", ["records"]],
        ["abstract.XMPP", ["records"]],
        ["abstract.LDAP", ["records"]],
        ["abstract.MatrixIM", ["records"]],
        ["abstract.CalDAV", ["records"]],
        ["abstract.CardDAV", ["records"]],
        [
            "abstract.Kerberos",
            ["kdc_tcp", "kdc_udp", "master", "admin", "kpasswd_tcp", "kpasswd_udp"],
        ],
    ])("%s reads %j", (svctype, fields) => {
        for (const f of fields) {
            expect(ids(run({ [f]: [SRV("not a host")] }, svctype))).toEqual(["srv.invalid-target"]);
        }
    });
});

describe("SRV compliance: locating issues", () => {
    it("locates each issue by field and index", () => {
        const issues = run({ srv: [SRV("a.provider.tld."), SRV("not a host", 5061)] });
        expect(issues.map((i) => i.field)).toEqual(["srv[1]"]);
    });

    it("keeps the real index when an entry is dropped", () => {
        const issues = run({ srv: [null, SRV("not a host")] });
        expect(issues.map((i) => i.field)).toEqual(["srv[1]"]);
    });

    it("puts set-level issues on the bare field", () => {
        const issues = run({ srv: [SRV("."), SRV("a.provider.tld.")] });
        expect(issues[0].field).toBe("srv");
    });
});

describe("SRV compliance: edge cases", () => {
    it("skips the owner check when Hdr.Name is empty or Hdr is absent", () => {
        expect(run({ srv: [SRV("a.provider.tld.", 5060, { Hdr: undefined })] })).toEqual([]);
        expect(run({ srv: [SRV("a.provider.tld.", 5060, { Hdr: { Name: "" } })] })).toEqual([]);
    });

    it("accepts a fully-qualified owner", () => {
        const record = SRV("a.provider.tld.", 5060, { Hdr: { Name: "_sip._tcp.example.com." } });
        expect(run({ srv: [record] })).toEqual([]);
    });

    it("flags an empty target", () => {
        expect(ids(run({ srv: [SRV("")] }))).toEqual(["srv.invalid-target"]);
    });

    it("does not flag the same host on different ports", () => {
        const records = [SRV("a.provider.tld.", 5060), SRV("a.provider.tld.", 5061)];
        expect(run({ srv: records })).toEqual([]);
    });

    it("accepts several unavailable records", () => {
        expect(run({ srv: [SRV(".", 0), SRV(".", 0)] })).toEqual([]);
    });

    it("still validates the fields of an unavailable record", () => {
        expect(ids(run({ srv: [SRV(".", 70000)] }))).toEqual(["srv.invalid-port"]);
    });

    it("leaves a lone zero weight, and an out-of-range one, out of the comparison", () => {
        expect(run({ srv: [SRV("a.provider.tld.", 5060, { Weight: 0 })] })).toEqual([]);
        const issues = run({
            srv: [SRV("a.provider.tld."), SRV("b.provider.tld.", 5061, { Weight: 70000 })],
        });
        expect(ids(issues)).toEqual(["srv.invalid-weight"]);
    });

    it("groups owners case-insensitively", () => {
        const issues = run({
            srv: [
                SRV(".", 0, { Hdr: { Name: "_SIP._tcp" } }),
                SRV("a.provider.tld.", 5060, { Hdr: { Name: "_sip._tcp" } }),
            ],
        });
        expect(ids(issues)).toEqual(["srv.unavailable-with-others"]);
    });

    it("reports an invalid owner once per owner set", () => {
        const issues = run({
            srv: [
                SRV("a.provider.tld.", 5060, { Hdr: { Name: "sip" } }),
                SRV("b.provider.tld.", 5060, { Hdr: { Name: "web" } }),
            ],
        });
        expect(ids(issues)).toEqual(["srv.invalid-owner", "srv.invalid-owner"]);
    });

    it("tolerates a missing, null or non-array field and non-object entries", () => {
        expect(run({})).toEqual([]);
        expect(run({ srv: null })).toEqual([]);
        expect(run({ srv: [null, "x", SRV("a.provider.tld.")] })).toEqual([]);
    });

    it("does not mix the same owner across Kerberos roles", () => {
        const issues = run(
            {
                kdc_tcp: [SRV(".", 0, { Hdr: { Name: "_kerberos._tcp" } })],
                kdc_udp: [SRV("kdc.example.org.", 88, { Hdr: { Name: "_kerberos._tcp" } })],
            },
            "abstract.Kerberos",
        );
        expect(issues).toEqual([]);
    });

    it("does not check out-of-zone targets, and accepts an in-zone one with an address", () => {
        expect(run({ srv: [SRV("sip.other.org.")] }, "svcs.UnknownSRV", zone({}))).toEqual([]);
        const z = zone({ sip: [makeService("abstract.Server", {})] });
        expect(run({ srv: [SRV("sip.example.com.")] }, "svcs.UnknownSRV", z)).toEqual([]);
    });
});
