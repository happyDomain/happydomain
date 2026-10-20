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
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";

const ORIGIN = makeDomain();
const SHA256 = "a".repeat(64);
const SHA1 = "b".repeat(40);

const SSHFP = (overrides: Record<string, unknown> = {}) => ({
    Hdr: { Name: "", Rrtype: 44, Class: 1, Ttl: 3600 },
    Algorithm: 4,
    Type: 2,
    FingerPrint: SHA256,
    ...overrides,
});

function run(value: Record<string, unknown>, z: Zone | null = makeZone({})): ComplianceIssue[] {
    const ctx = buildContext("srv", ORIGIN, z);
    const v = getValidators("abstract.Server");
    expect(v?.sync).toBeDefined();
    return v!.sync!(value, ctx);
}

const ids = (issues: ComplianceIssue[]) => issues.map((i) => i.id);

describe("Server compliance: SSHFP", () => {
    it("stays quiet on a server publishing no fingerprint", () => {
        expect(run({ A: { A: "192.0.2.1" } })).toEqual([]);
        expect(run({ A: { A: "192.0.2.1" }, SSHFP: [] })).toEqual([]);
    });

    it("accepts a well-formed record", () => {
        expect(run({ SSHFP: [SSHFP()] })).toEqual([]);
    });

    it("reads the fields of dns.SSHFP records", () => {
        expect(ids(run({ SSHFP: [SSHFP({ Algorithm: 9 })] }))).toEqual(["sshfp.unknown-algorithm"]);
        expect(ids(run({ SSHFP: [SSHFP({ FingerPrint: SHA1 })] }))).toEqual([
            "sshfp.fingerprint-length",
        ]);
        expect(ids(run({ SSHFP: [SSHFP({ Type: 1, FingerPrint: SHA1 })] }))).toEqual([
            "sshfp.sha1-only",
        ]);
    });

    it("points at the row in the SSHFP field", () => {
        const issues = run({ SSHFP: [SSHFP(), SSHFP({ FingerPrint: "zz" })] });
        expect(issues.map((i) => i.field)).toEqual(["SSHFP[1]"]);
    });

    it("does not ask for an address it publishes itself", () => {
        expect(run({ SSHFP: [SSHFP()] }, makeZone({}))).toEqual([]);
    });

    it("flags fingerprints published on an alias", () => {
        const z = makeZone({
            services: { srv: [makeService("svcs.Alias", { record: { Hdr: { Rrtype: 5 } } })] },
        });
        expect(ids(run({ SSHFP: [SSHFP()] }, z))).toEqual(["sshfp.owner-is-cname"]);
    });
});
