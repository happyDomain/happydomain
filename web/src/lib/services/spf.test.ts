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
import { countLocalLookups, parseSPF, stringifySPF, validateSPF } from "./spf";
import type { ComplianceContext } from "./compliance";
import type { Domain } from "$lib/model/domain";

const ctx: ComplianceContext = {
    dn: "@",
    origin: { id: "test", domain: "example.com" } as unknown as Domain,
    zone: null,
    findServices: () => [],
};

const ids = (issues: { id: string }[]) => issues.map((i) => i.id);

describe("parseSPF", () => {
    it("parses a minimal record", () => {
        expect(parseSPF("v=spf1 -all")).toEqual({ v: "spf1", f: ["-all"] });
    });

    it("returns no version when missing", () => {
        expect(parseSPF("include:example.com -all")).toEqual({
            v: undefined,
            f: ["include:example.com", "-all"],
        });
    });

    it("trims and collapses whitespace", () => {
        expect(parseSPF("  v=spf1   include:foo  ~all  ")).toEqual({
            v: "spf1",
            f: ["include:foo", "~all"],
        });
    });

    it("handles an empty string", () => {
        expect(parseSPF("")).toEqual({ v: undefined, f: [] });
    });

    it("splits on semicolons so DKIM residue does not stick to a directive", () => {
        const out = parseSPF("v=spf1 -all;k=rsa");
        expect(out.v).toBe("spf1");
        expect(out.f).toEqual(["-all", "k=rsa"]);
    });
});

describe("stringifySPF", () => {
    it("round-trips a parsed record", () => {
        const v = parseSPF("v=spf1 include:_spf.google.com ~all");
        expect(stringifySPF(v)).toBe("v=spf1 include:_spf.google.com ~all");
    });

    it("defaults to spf1 when no version", () => {
        expect(stringifySPF({ v: undefined, f: ["-all"] })).toBe("v=spf1 -all");
    });

    it("works with no directives", () => {
        expect(stringifySPF({ v: "spf1", f: [] })).toBe("v=spf1");
    });
});

describe("countLocalLookups", () => {
    it("counts include / a / mx / exists / ptr / redirect", () => {
        const v = parseSPF(
            "v=spf1 include:a.com a mx exists:_e.example.com ptr redirect=fallback.example.com",
        );
        const b = countLocalLookups(v);
        expect(b.count).toBe(6);
        expect(b.contributors.map((c) => c.mechanism)).toEqual([
            "include",
            "a",
            "mx",
            "exists",
            "ptr",
            "redirect",
        ]);
    });

    it("ignores non-lookup mechanisms", () => {
        const v = parseSPF("v=spf1 ip4:1.2.3.4 ip6:::1 -all");
        expect(countLocalLookups(v).count).toBe(0);
    });
});

describe("validateSPF", () => {
    it("accepts a clean record with no issues", () => {
        const issues = validateSPF(parseSPF("v=spf1 ip4:1.2.3.4 -all"), ctx);
        expect(issues).toEqual([]);
    });

    it("flags missing version", () => {
        const issues = validateSPF(parseSPF("include:foo.com -all"), ctx);
        expect(ids(issues)).toContain("spf.missing-version");
    });

    it("flags wrong version and stops further checks", () => {
        const issues = validateSPF(parseSPF("v=spf2 include:x -all"), ctx);
        expect(ids(issues)).toEqual(["spf.wrong-version"]);
    });

    it("warns when no all and no redirect", () => {
        const issues = validateSPF(parseSPF("v=spf1 ip4:1.2.3.4"), ctx);
        expect(ids(issues)).toContain("spf.no-all-mechanism");
    });

    it("flags multiple all", () => {
        const issues = validateSPF(parseSPF("v=spf1 ~all -all"), ctx);
        expect(ids(issues)).toContain("spf.multiple-all");
    });

    it("warns when all is not last", () => {
        const issues = validateSPF(parseSPF("v=spf1 -all ip4:1.2.3.4"), ctx);
        expect(ids(issues)).toContain("spf.all-not-last");
    });

    it("warns when redirect is combined with all", () => {
        const issues = validateSPF(parseSPF("v=spf1 redirect=foo.com -all"), ctx);
        expect(ids(issues)).toContain("spf.redirect-with-all");
    });

    it("flags multiple redirect modifiers", () => {
        const issues = validateSPF(parseSPF("v=spf1 redirect=a.com redirect=b.com"), ctx);
        expect(ids(issues)).toContain("spf.multiple-redirect");
    });

    it("warns when ptr is used", () => {
        const issues = validateSPF(parseSPF("v=spf1 ptr -all"), ctx);
        expect(ids(issues)).toContain("spf.ptr-deprecated");
    });

    it("delegates lookup-budget reporting to the recursive walk", () => {
        const includes = Array.from({ length: 11 }, (_, i) => `include:i${i}.example.com`).join(" ");
        const issues = validateSPF(parseSPF(`v=spf1 ${includes} -all`), ctx);
        expect(ids(issues)).not.toContain("spf.too-many-lookups");
        expect(ids(issues)).not.toContain("spf.many-lookups");
    });

    it("flags include without value", () => {
        const issues = validateSPF(parseSPF("v=spf1 include -all"), ctx);
        expect(ids(issues)).toContain("spf.mechanism-missing-value");
    });

    it("does not flag bare a or mx", () => {
        const issues = validateSPF(parseSPF("v=spf1 a mx -all"), ctx);
        expect(ids(issues)).not.toContain("spf.mechanism-missing-value");
    });

    it("flags duplicates as info", () => {
        const issues = validateSPF(parseSPF("v=spf1 include:a.com include:a.com -all"), ctx);
        expect(ids(issues)).toContain("spf.duplicate-mechanism");
    });

    it("includes a field path on the offending term", () => {
        const issues = validateSPF(parseSPF("v=spf1 -all ip4:1.2.3.4"), ctx);
        const allNotLast = issues.find((i) => i.id === "spf.all-not-last");
        expect(allNotLast?.field).toBe("f[0]");
    });
});
