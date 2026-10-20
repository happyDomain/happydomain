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
const SHA256 = "a".repeat(64);

const zone = (services: Record<string, ServiceWithValue[]>): Zone => makeZone({ services });

const NS = (Ns: string) => ({ Ns, Hdr: { Name: "" } });
const DS = (overrides: Record<string, unknown> = {}) => ({
    KeyTag: 12345,
    Algorithm: 13,
    DigestType: 2,
    Digest: SHA256,
    ...overrides,
});

function run(raw: Record<string, unknown>, dn = "sub", z: Zone | null = null): ComplianceIssue[] {
    const ctx = buildContext(dn, ORIGIN, z);
    const v = getValidators("abstract.Delegation");
    expect(v?.sync).toBeDefined();
    return v!.sync!(raw, ctx);
}

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

describe("Delegation compliance: name servers", () => {
    it("accepts two external name servers", () => {
        expect(run({ ns: [NS("ns1.provider.tld."), NS("ns2.provider.tld.")] })).toEqual([]);
    });

    it("flags a delegation without any name server", () => {
        expect(ids(run({ ns: [] }))).toEqual(["ns.no-nameserver"]);
    });

    it("warns on a single name server", () => {
        expect(ids(run({ ns: [NS("ns1.provider.tld.")] }))).toEqual(["ns.single-nameserver"]);
    });

    it("flags an invalid target", () => {
        const issues = run({ ns: [NS("not a host"), NS("ns2.provider.tld.")] });
        expect(ids(issues)).toEqual(["ns.invalid-target"]);
    });

    it("warns on a duplicate target, whatever its spelling", () => {
        const issues = run({ ns: [NS("ns1.provider.tld."), NS("NS1.provider.tld")] });
        expect(ids(issues)).toContain("ns.duplicate-target");
    });
});

describe("Delegation compliance: name servers, more", () => {
    it("flags a missing or malformed payload as having no name server", () => {
        expect(ids(run({}))).toEqual(["ns.no-nameserver"]);
        expect(ids(run({ ns: [null, "x"] as any }))).toEqual(["ns.no-nameserver"]);
    });

    it("flags an empty target", () => {
        expect(ids(run({ ns: [NS(""), NS("ns2.provider.tld.")] }))).toEqual(["ns.invalid-target"]);
    });

    it("reports an invalid target each time, not as a duplicate", () => {
        expect(ids(run({ ns: [NS("bad host"), NS("bad host")] }))).toEqual([
            "ns.invalid-target",
            "ns.invalid-target",
        ]);
    });

    it("names the delegated subdomain when there is a single name server", () => {
        const [i] = run({ ns: [NS("ns1.provider.tld.")] });
        expect(i.params).toEqual({ subdomain: "sub.example.com" });
    });
});

describe("Delegation compliance: glue", () => {
    it("does not confuse a sibling label with the delegated subtree", () => {
        const issues = run(
            { ns: [NS("ns1.notsub.example.com."), NS("ns2.provider.tld.")] },
            "sub",
            zone({}),
        );
        expect(ids(issues)).toEqual(["ns.target-no-address"]);
    });

    it("requires glue for the delegated apex itself used as name server", () => {
        const issues = run(
            { ns: [NS("sub.example.com."), NS("ns2.provider.tld.")] },
            "sub",
            zone({}),
        );
        expect(ids(issues)).toContain("ns.missing-glue");
    });

    it("flags a name server that is a SubAlias owner", () => {
        const z = zone({
            ns1: [makeService("svcs.SpecialCNAME", { cname: { Hdr: { Name: "" } } })],
        });
        const issues = run({ ns: [NS("ns1.example.com."), NS("ns2.provider.tld.")] }, "sub", z);
        expect(ids(issues)).toEqual(["ns.target-is-cname"]);
    });

    it("warns instead of erroring when no zone is loaded", () => {
        const issues = run({ ns: [NS("ns1.example.com."), NS("ns2.provider.tld.")] }, "sub", null);
        expect(ids(issues)).toEqual(["ns.target-no-address"]);
    });

    it("requires glue for a name server inside the delegated subtree", () => {
        const issues = run({
            ns: [NS("ns1.sub.example.com."), NS("ns2.provider.tld.")],
        });
        expect(ids(issues)).toContain("ns.missing-glue");
    });

    it("is happy once the glue is published in this zone", () => {
        const z = zone({ "ns1.sub": [makeService("abstract.Server", {})] });
        const issues = run({ ns: [NS("ns1.sub.example.com."), NS("ns2.provider.tld.")] }, "sub", z);
        expect(ids(issues)).toEqual([]);
    });

    it("only warns for an in-zone name server outside the delegated subtree", () => {
        const issues = run(
            { ns: [NS("ns1.example.com."), NS("ns2.provider.tld.")] },
            "sub",
            zone({}),
        );
        expect(ids(issues)).toEqual(["ns.target-no-address"]);
    });

    it("flags a name server that is an alias", () => {
        const z = zone({
            ns1: [makeService("svcs.Alias", { record: { Hdr: { Rrtype: 5 } } })],
        });
        const issues = run({ ns: [NS("ns1.example.com."), NS("ns2.provider.tld.")] }, "sub", z);
        expect(ids(issues)).toEqual(["ns.target-is-cname"]);
    });
});

describe("Delegation compliance: DS", () => {
    const withNs = (ds: unknown[]) => ({
        ns: [NS("ns1.provider.tld."), NS("ns2.provider.tld.")],
        ds,
    });

    it("accepts a well-formed SHA-256 DS", () => {
        expect(run(withNs([DS()]))).toEqual([]);
    });

    it("flags a DS published without any name server", () => {
        expect(ids(run({ ns: [], ds: [DS()] }))).toEqual(["ns.ds-without-ns"]);
    });

    it("flags an out-of-range key tag", () => {
        expect(ids(run(withNs([DS({ KeyTag: 70000 })])))).toContain("ns.ds-invalid-keytag");
    });

    it("flags an unknown digest type", () => {
        expect(ids(run(withNs([DS({ DigestType: 5 })])))).toEqual(["ns.ds-unknown-digest-type"]);
    });

    it("flags a digest whose length does not match its type", () => {
        expect(ids(run(withNs([DS({ Digest: "abcdef" })])))).toEqual(["ns.ds-digest-length"]);
    });

    it("flags a digest carrying something else than hexadecimal", () => {
        expect(ids(run(withNs([DS({ Digest: "z".repeat(64) })])))).toEqual(["ns.ds-digest-length"]);
    });

    it("warns on SHA-1 and on the deprecated algorithms", () => {
        const issues = run(withNs([DS({ Algorithm: 5, DigestType: 1, Digest: "a".repeat(40) })]));
        expect(ids(issues)).toEqual(["ns.ds-deprecated-algorithm", "ns.ds-sha1"]);
    });

    it("warns on a duplicate DS", () => {
        expect(ids(run(withNs([DS(), DS()])))).toEqual(["ns.ds-duplicate"]);
    });
});

describe("Delegation compliance: DS, more", () => {
    const withNs = (ds: unknown[]) => ({
        ns: [NS("ns1.provider.tld."), NS("ns2.provider.tld.")],
        ds,
    });

    it("still validates the DS records when there is no name server", () => {
        expect(ids(run({ ns: [], ds: [DS({ KeyTag: 70000 })] }))).toEqual([
            "ns.ds-without-ns",
            "ns.ds-invalid-keytag",
        ]);
    });

    it("accepts the key tag bounds and rejects the rest", () => {
        expect(run(withNs([DS({ KeyTag: 0 }), DS({ KeyTag: 65535 })]))).toEqual([]);
        for (const KeyTag of [-1, 1.5, "1", undefined]) {
            expect(ids(run(withNs([DS({ KeyTag })])))).toEqual(["ns.ds-invalid-keytag"]);
        }
    });

    it("accepts the GOST and SHA-384 digest types at their length", () => {
        expect(
            run(withNs([DS({ DigestType: 3 }), DS({ DigestType: 4, Digest: "a".repeat(96) })])),
        ).toEqual([]);
    });

    it("warns on each deprecated algorithm, and only on those", () => {
        for (const Algorithm of [1, 3, 5, 6, 7, 12]) {
            expect(ids(run(withNs([DS({ Algorithm })])))).toEqual(["ns.ds-deprecated-algorithm"]);
        }
        expect(run(withNs([DS({ Algorithm: 15 })]))).toEqual([]);
    });

    it("reports both the SHA-1 warning and the length error", () => {
        expect(ids(run(withNs([DS({ DigestType: 1, Digest: "ab" })])))).toEqual([
            "ns.ds-sha1",
            "ns.ds-digest-length",
        ]);
    });

    it("reports neither SHA-1 nor a length for an unknown digest type", () => {
        expect(ids(run(withNs([DS({ DigestType: 0, Digest: "" })])))).toEqual([
            "ns.ds-unknown-digest-type",
        ]);
    });

    it("trims the digest and detects duplicates whatever its case", () => {
        const issues = run(
            withNs([DS({ Digest: "A".repeat(64) }), DS({ Digest: ` ${"a".repeat(64)} ` })]),
        );
        expect(ids(issues)).toEqual(["ns.ds-duplicate"]);
    });

    it("does not call two DS with a different key tag duplicates", () => {
        expect(run(withNs([DS(), DS({ KeyTag: 54321 })]))).toEqual([]);
    });

    it("reports an empty digest as a length error", () => {
        expect(ids(run(withNs([DS({ Digest: "" })])))).toEqual(["ns.ds-digest-length"]);
    });
});
