// This file is part of the happyDomain (R) project.
// Copyright (c) 2026 happyDomain
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

const zone = (services: Record<string, ServiceWithValue[]>): Zone => makeZone({ services });

const CNAME = () => makeService("svcs.Alias", { record: { Hdr: { Rrtype: 5 } } });

// Timers in use at a large provider, sound by every measure checked here.
const SOA = (overrides: Record<string, unknown> = {}) => ({
    Hdr: { Name: "" },
    Ns: "ns1.provider.tld.",
    Mbox: "hostmaster",
    Serial: 2026102701,
    Refresh: 10800,
    Retry: 3600,
    Expire: 604800,
    Minttl: 3600,
    ...overrides,
});

function run(soa: unknown, z: Zone | null = null): ComplianceIssue[] {
    const ctx = buildContext("", ORIGIN, z);
    const v = getValidators("abstract.Origin");
    expect(v?.sync).toBeDefined();
    return v!.sync!({ soa, ns: [] }, ctx);
}

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

describe("Origin compliance: primary name server", () => {
    it("accepts a sound SOA", () => {
        expect(run(SOA())).toEqual([]);
    });

    it("accepts a name server relative to the zone", () => {
        expect(run(SOA({ Ns: "ns0" }))).toEqual([]);
    });

    it("flags an empty name server", () => {
        expect(ids(run(SOA({ Ns: "  " })))).toEqual(["soa.empty-mname"]);
        expect(ids(run(SOA({ Ns: undefined })))).toEqual(["soa.empty-mname"]);
    });

    it.each(["not a host", "a..b.", "_ns.provider.tld."])("flags %s as invalid", (Ns) => {
        expect(run(SOA({ Ns }))).toMatchObject([
            { id: "soa.invalid-mname", field: "mname", params: { mname: Ns } },
        ]);
    });

    it("flags a relative name server that is an alias", () => {
        const z = zone({ ns0: [CNAME()] });
        expect(run(SOA({ Ns: "ns0" }), z)).toMatchObject([
            { id: "soa.mname-is-cname", params: { mname: "ns0.example.com" } },
        ]);
    });

    it("flags an absolute name server of the zone that is an alias", () => {
        const z = zone({ ns0: [CNAME()] });
        expect(ids(run(SOA({ Ns: "NS0.example.com." }), z))).toEqual(["soa.mname-is-cname"]);
    });

    it("does not look for aliases outside of the zone", () => {
        const z = zone({ ns0: [CNAME()] });
        expect(run(SOA({ Ns: "ns0.provider.tld." }), z)).toEqual([]);
    });
});

describe("Origin compliance: contact address", () => {
    it.each(["hostmaster.example.com.", "john\\.doe", "@"])("accepts %s", (Mbox) => {
        expect(run(SOA({ Mbox }))).toEqual([]);
    });

    it("flags an empty contact", () => {
        expect(ids(run(SOA({ Mbox: "" })))).toEqual(["soa.empty-rname"]);
    });

    it("flags an email address typed as is, and suggests its DNS spelling", () => {
        expect(run(SOA({ Mbox: "hostmaster@example.com." }))).toMatchObject([
            {
                id: "soa.rname-has-at",
                field: "rname",
                params: { suggestion: "hostmaster.example.com." },
            },
        ]);
    });

    it("escapes the dots of the local part in the suggestion", () => {
        expect(run(SOA({ Mbox: "john.doe@example.com." }))[0].params).toMatchObject({
            suggestion: "john\\.doe.example.com.",
        });
    });

    it.each(["com.", "hostmaster.bad_domain.tld.", "hostmaster..tld."])(
        "flags %s as invalid",
        (Mbox) => {
            expect(ids(run(SOA({ Mbox })))).toEqual(["soa.invalid-rname"]);
        },
    );
});

describe("Origin compliance: timers", () => {
    it("warns about a refresh shorter than 20 minutes", () => {
        expect(run(SOA({ Refresh: 600, Retry: 300 }))).toMatchObject([
            { id: "soa.refresh-too-short", field: "refresh", params: { refresh: 600 } },
        ]);
    });

    it("accepts a refresh of exactly 20 minutes", () => {
        expect(run(SOA({ Refresh: 1200, Retry: 600 }))).toEqual([]);
    });

    it("warns about a retry not shorter than the refresh", () => {
        expect(ids(run(SOA({ Retry: 10800 })))).toEqual(["soa.retry-not-below-refresh"]);
    });

    it("warns about an expire shorter than a refresh and a retry", () => {
        expect(run(SOA({ Expire: 14000 }))).toMatchObject([
            { id: "soa.expire-too-short", field: "expire" },
        ]);
    });

    it("accepts an expire of exactly a refresh and a retry", () => {
        expect(run(SOA({ Expire: 14400 }))).toEqual([]);
    });

    it("warns about a negative caching time longer than a day", () => {
        expect(run(SOA({ Minttl: 86401 }))).toMatchObject([
            { id: "soa.negative-ttl-too-long", field: "nxttl" },
        ]);
        expect(run(SOA({ Minttl: 86400 }))).toEqual([]);
    });

    it.each([-1, 1.5, 2 ** 32, "3600", undefined])("flags %s as an invalid timer", (Retry) => {
        expect(run(SOA({ Retry }))).toMatchObject([
            { id: "soa.invalid-timer", field: "retry", params: { value: String(Retry) } },
        ]);
    });

    it("does not compare timers while one of them is invalid", () => {
        expect(ids(run(SOA({ Refresh: 0, Expire: -1 })))).toEqual(["soa.invalid-timer"]);
    });
});

describe("Origin compliance: edge cases", () => {
    it("ignores an origin without SOA", () => {
        expect(run(undefined)).toEqual([]);
        expect(run(null)).toEqual([]);
    });

    it("keeps the issues of every field together", () => {
        expect(ids(run(SOA({ Ns: "", Mbox: "a@b.c", Retry: 99999 })))).toEqual([
            "soa.empty-mname",
            "soa.rname-has-at",
            "soa.retry-not-below-refresh",
        ]);
    });
});
