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
const SHA1 = "b".repeat(40);

const zone = (services: Record<string, ServiceWithValue[]>): Zone => makeZone({ services });

// A zone where the edited name carries its addresses, so the cross-record
// checks stay quiet and only the record itself is under test.
const HOSTED = zone({ srv: [makeService("abstract.Server", {})] });

const SSHFP = (overrides: Record<string, unknown> = {}) => ({
    Hdr: { Name: "", Rrtype: 44, Class: 1, Ttl: 3600 },
    Algorithm: 4,
    Type: 2,
    FingerPrint: SHA256,
    ...overrides,
});

function run(SSHFPs: unknown, dn = "srv", z: Zone | null = HOSTED): ComplianceIssue[] {
    const ctx = buildContext(dn, ORIGIN, z);
    const v = getValidators("svcs.SSHFPs");
    expect(v?.sync).toBeDefined();
    return v!.sync!({ SSHFP: SSHFPs }, ctx);
}

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

describe("SSHFP compliance: fields", () => {
    it("accepts a well-formed Ed25519 SHA-256 fingerprint", () => {
        expect(run([SSHFP()])).toEqual([]);
    });

    it("reports an empty service", () => {
        expect(ids(run([]))).toEqual(["sshfp.no-record"]);
    });

    it("flags an unknown key algorithm", () => {
        expect(ids(run([SSHFP({ Algorithm: 9 })]))).toEqual(["sshfp.unknown-algorithm"]);
    });

    it("flags an unknown fingerprint type", () => {
        expect(ids(run([SSHFP({ Type: 3 })]))).toEqual(["sshfp.unknown-type"]);
    });

    it("flags a fingerprint that is not hexadecimal", () => {
        expect(ids(run([SSHFP({ FingerPrint: "z".repeat(64) })]))).toEqual([
            "sshfp.invalid-fingerprint",
        ]);
    });

    it("flags a fingerprint whose length does not match its type", () => {
        expect(ids(run([SSHFP({ FingerPrint: SHA1 })]))).toEqual(["sshfp.fingerprint-length"]);
    });

    it("warns on a duplicate", () => {
        expect(ids(run([SSHFP(), SSHFP()]))).toEqual(["sshfp.duplicate"]);
    });
});

describe("SSHFP compliance: algorithms", () => {
    it("warns when a key is only published with SHA-1", () => {
        expect(ids(run([SSHFP({ Type: 1, FingerPrint: SHA1 })]))).toEqual(["sshfp.sha1-only"]);
    });

    it("stays quiet when SHA-256 accompanies SHA-1", () => {
        const issues = run([SSHFP(), SSHFP({ Type: 1, FingerPrint: SHA1 })]);
        expect(issues).toEqual([]);
    });

    it("looks at each algorithm on its own", () => {
        const issues = run([SSHFP(), SSHFP({ Algorithm: 3, Type: 1, FingerPrint: SHA1 })]);
        expect(ids(issues)).toEqual(["sshfp.sha1-only"]);
    });

    it("warns on DSA", () => {
        expect(ids(run([SSHFP({ Algorithm: 2 })]))).toEqual(["sshfp.dsa-deprecated"]);
    });
});

describe("SSHFP compliance: owner name", () => {
    it("flags fingerprints published on an alias", () => {
        const z = zone({ srv: [makeService("svcs.Alias", { record: { Hdr: { Rrtype: 5 } } })] });
        expect(ids(run([SSHFP()], "srv", z))).toEqual(["sshfp.owner-is-cname"]);
    });

    it("warns when the name has no address in the zone", () => {
        expect(ids(run([SSHFP()], "srv", zone({})))).toEqual(["sshfp.no-address"]);
    });

    it("says nothing about the name when the zone is unknown", () => {
        expect(run([SSHFP()], "srv", null)).toEqual([]);
    });
});

describe("SSHFP compliance: normalisation and edge cases", () => {
    it("accepts an upper-case or padded fingerprint", () => {
        expect(run([SSHFP({ FingerPrint: `  ${"A".repeat(64)}\n` })])).toEqual([]);
    });

    it("detects a duplicate that only differs by case", () => {
        const issues = run([SSHFP(), SSHFP({ FingerPrint: SHA256.toUpperCase() })]);
        expect(ids(issues)).toEqual(["sshfp.duplicate"]);
        expect(issues[0].field).toBe("SSHFP[1]");
    });

    it("reports each extra copy of a triplicate", () => {
        expect(ids(run([SSHFP(), SSHFP(), SSHFP()]))).toEqual([
            "sshfp.duplicate",
            "sshfp.duplicate",
        ]);
    });

    it("does not call two records duplicates when algorithm or type differ", () => {
        const issues = run([
            SSHFP(),
            SSHFP({ Algorithm: 6 }),
            SSHFP({ Type: 1, FingerPrint: SHA1 }),
        ]);
        expect(ids(issues)).toEqual([]);
    });

    it("flags an empty or colon separated fingerprint as non hexadecimal", () => {
        expect(ids(run([SSHFP({ FingerPrint: "" })]))).toEqual(["sshfp.invalid-fingerprint"]);
        expect(ids(run([SSHFP({ FingerPrint: "aa:".repeat(32) })]))).toEqual([
            "sshfp.invalid-fingerprint",
        ]);
    });

    it("reports an unknown type before looking at the fingerprint", () => {
        expect(ids(run([SSHFP({ Type: 3, FingerPrint: "zz" })]))).toEqual(["sshfp.unknown-type"]);
    });

    it("keeps an unknown algorithm out of the SHA-1 only check", () => {
        const issues = run([SSHFP({ Algorithm: 9, Type: 1, FingerPrint: SHA1 })]);
        expect(ids(issues)).toEqual(["sshfp.unknown-algorithm"]);
    });

    it("cumulates dsa-deprecated and sha1-only on a SHA-1 DSA key", () => {
        const issues = run([SSHFP({ Algorithm: 2, Type: 1, FingerPrint: SHA1 })]);
        expect(ids(issues).sort()).toEqual(["sshfp.dsa-deprecated", "sshfp.sha1-only"]);
    });

    it("fills the message parameters, field and doc link", () => {
        const [len] = run([SSHFP({ FingerPrint: SHA1 })]);
        expect(len.params).toEqual({ hash: "SHA-256", expected: 64, got: 40 });
        expect(len.field).toBe("SSHFP[0]");

        const [alg] = run([SSHFP({ Algorithm: 9 })]);
        expect(alg.params).toEqual({ algorithm: "9" });
        expect(alg.docUrl).toContain("rfc4255");

        const [typ] = run([SSHFP({ Type: 7 })]);
        expect(typ.params).toEqual({ type: "7" });
        expect(typ.docUrl).toContain("rfc6594");

        const [sha1] = run([SSHFP({ Algorithm: 3, Type: 1, FingerPrint: SHA1 })]);
        expect(sha1.params).toEqual({ algorithm: "ECDSA" });
    });

    it("tolerates a missing, single-object or malformed SSHFP field", () => {
        expect(ids(run(undefined))).toEqual(["sshfp.no-record"]);
        expect(ids(run(null))).toEqual(["sshfp.no-record"]);
        expect(run(SSHFP())).toEqual([]);
        expect(ids(run([null, "x"]))).toEqual(["sshfp.no-record"]);
    });

    it("keeps the real index when an entry is dropped", () => {
        const issues = run([null, SSHFP({ Algorithm: 9 })]);
        expect(issues[0].field).toBe("SSHFP[1]");
    });

    it("does not throw on a fingerprint that is not a string", () => {
        expect(ids(run([SSHFP({ FingerPrint: 12 })]))).toEqual(["sshfp.fingerprint-length"]);
    });
});

describe("SSHFP compliance: owner name, more", () => {
    const CNAME = () => makeService("svcs.Alias", { record: { Hdr: { Rrtype: 5 } } });

    it("names the apex as @ in the messages", () => {
        const [i] = run([SSHFP()], "", zone({ "": [CNAME()] }));
        expect(i.id).toBe("sshfp.owner-is-cname");
        expect(i.params).toEqual({ subdomain: "@" });

        const [j] = run([SSHFP()], "", zone({}));
        expect(j.id).toBe("sshfp.no-address");
        expect(j.params).toEqual({ subdomain: "@" });
    });

    it("flags fingerprints published on a SubAlias", () => {
        const z = zone({
            srv: [makeService("svcs.SpecialCNAME", { cname: { Hdr: { Name: "" } } })],
        });
        expect(ids(run([SSHFP()], "srv", z))).toEqual(["sshfp.owner-is-cname"]);
    });

    it("does not add no-address on top of owner-is-cname", () => {
        expect(ids(run([SSHFP()], "srv", zone({ srv: [CNAME()] })))).toEqual([
            "sshfp.owner-is-cname",
        ]);
    });

    it("counts a non-CNAME alias as an address", () => {
        const alias = makeService("svcs.Alias", { record: { Hdr: { Rrtype: 1 } } });
        expect(run([SSHFP()], "srv", zone({ srv: [alias] }))).toEqual([]);
    });

    it("stays silent about the address when no zone is loaded", () => {
        expect(run([SSHFP()], "srv", null)).toEqual([]);
    });
});
