// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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
import type { Domain } from "$lib/model/domain";

const ORIGIN = { domain: "example.com." } as unknown as Domain;
const CTX = buildContext("_smtp._tls", ORIGIN, null);

function run(txt: string, name = "_smtp._tls.example.com."): ComplianceIssue[] {
    return getValidators("svcs.TLS_RPT")!.sync!({
        txt: { Hdr: { Name: name }, Txt: txt },
    }, CTX);
}
const ids = (issues: ComplianceIssue[]) => issues.map((i) => i.id);

describe("TLS-RPT compliance", () => {
    it("accepts a clean record with mailto", () => {
        expect(ids(run("v=TLSRPTv1;rua=mailto:tlsrpt@example.com"))).toEqual([]);
    });
    it("accepts a clean record with https", () => {
        expect(ids(run("v=TLSRPTv1;rua=https://reports.example.com/tlsrpt"))).toEqual([]);
    });
    it("flags wrong owner name", () => {
        expect(ids(run("v=TLSRPTv1;rua=mailto:t@example.com", "example.com.")))
            .toContain("tlsrpt.wrong-owner-name");
    });
    it("flags missing version", () => {
        expect(ids(run("rua=mailto:t@example.com"))).toContain("tlsrpt.missing-version");
    });
    it("flags non-TLSRPTv1 version", () => {
        expect(ids(run("v=TLSRPTv2;rua=mailto:t@example.com"))).toContain("tlsrpt.invalid-version");
    });
    it("flags missing rua", () => {
        expect(ids(run("v=TLSRPTv1"))).toContain("tlsrpt.missing-rua");
    });
    it("flags non-mailto/http URI", () => {
        expect(ids(run("v=TLSRPTv1;rua=ftp://example.com"))).toContain("tlsrpt.invalid-rua-scheme");
    });
    it("flags malformed mailto", () => {
        expect(ids(run("v=TLSRPTv1;rua=mailto:not-an-email"))).toContain("tlsrpt.invalid-mailto");
    });
    it("returns no issue on empty TXT", () => {
        expect(run("")).toEqual([]);
    });
});
