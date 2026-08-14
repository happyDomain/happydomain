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
    algorithm: 4,
    type: 2,
    fingerprint: SHA256,
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
        expect(ids(run([SSHFP({ algorithm: 9 })]))).toEqual(["sshfp.unknown-algorithm"]);
    });

    it("flags an unknown fingerprint type", () => {
        expect(ids(run([SSHFP({ type: 3 })]))).toEqual(["sshfp.unknown-type"]);
    });

    it("flags a fingerprint that is not hexadecimal", () => {
        expect(ids(run([SSHFP({ fingerprint: "z".repeat(64) })]))).toEqual([
            "sshfp.invalid-fingerprint",
        ]);
    });

    it("flags a fingerprint whose length does not match its type", () => {
        expect(ids(run([SSHFP({ fingerprint: SHA1 })]))).toEqual(["sshfp.fingerprint-length"]);
    });

    it("warns on a duplicate", () => {
        expect(ids(run([SSHFP(), SSHFP()]))).toEqual(["sshfp.duplicate"]);
    });
});

describe("SSHFP compliance: algorithms", () => {
    it("warns when a key is only published with SHA-1", () => {
        expect(ids(run([SSHFP({ type: 1, fingerprint: SHA1 })]))).toEqual(["sshfp.sha1-only"]);
    });

    it("stays quiet when SHA-256 accompanies SHA-1", () => {
        const issues = run([SSHFP(), SSHFP({ type: 1, fingerprint: SHA1 })]);
        expect(issues).toEqual([]);
    });

    it("looks at each algorithm on its own", () => {
        const issues = run([SSHFP(), SSHFP({ algorithm: 3, type: 1, fingerprint: SHA1 })]);
        expect(ids(issues)).toEqual(["sshfp.sha1-only"]);
    });

    it("warns on DSA", () => {
        expect(ids(run([SSHFP({ algorithm: 2 })]))).toEqual(["sshfp.dsa-deprecated"]);
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
