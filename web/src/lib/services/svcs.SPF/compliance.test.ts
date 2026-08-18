// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("$lib/api/resolver", () => ({
    flattenSPF: vi.fn(),
}));

import "./compliance";
import { validateSPF, validateSPFRecursive } from "./compliance";
import { buildContext, getValidators, type ComplianceIssue } from "$lib/services/compliance";
import type { ComplianceContext } from "$lib/services/compliance";
import { parseSPF } from "./model";
import { makeDomain } from "$lib/test-utils/fixtures";
import { flattenSPF } from "$lib/api/resolver";
import type { HappydnsSpfFlattenResponse, HappydnsSpfNode } from "$lib/api-base/types.gen";

const ORIGIN = makeDomain({ domain: "example.com." });
const CTX: ComplianceContext = buildContext("@", ORIGIN, null);

const ids = (issues: ComplianceIssue[]) => issues.map((i) => i.id);

// ---------------------------------------------------------------------------
// registerValidators wiring
// ---------------------------------------------------------------------------

describe("svcs.SPF validators registration", () => {
    it("registers both a sync and an async validator", () => {
        const v = getValidators("svcs.SPF");
        expect(v?.sync).toBeDefined();
        expect(v?.async).toBeDefined();
    });

    it("sync entrypoint parses the raw TXT and delegates to validateSPF", () => {
        const v = getValidators("svcs.SPF")!;
        const issues = v.sync!({ txt: { Txt: "v=spf1 ip4:1.2.3.4 -all" } }, CTX);
        expect(issues).toEqual([]);
    });

    it("sync entrypoint tolerates a missing raw.txt.Txt", () => {
        const v = getValidators("svcs.SPF")!;
        const issues = v.sync!({}, CTX);
        expect(ids(issues)).toContain("spf.missing-version");
    });
});

// ---------------------------------------------------------------------------
// validateSPF (sync)
// ---------------------------------------------------------------------------

describe("validateSPF: version", () => {
    it("flags a missing version and stops further checks", () => {
        const issues = validateSPF(parseSPF(""), CTX);
        expect(ids(issues)).toEqual(["spf.missing-version"]);
    });

    it("flags a non-spf1 version and stops further checks", () => {
        const issues = validateSPF(parseSPF("v=spf2.0 -all"), CTX);
        expect(ids(issues)).toEqual(["spf.wrong-version"]);
        expect(issues[0].params).toEqual({ version: "spf2.0" });
    });

    it("accepts a version compared case-insensitively", () => {
        const issues = validateSPF(parseSPF("v=SPF1 -all"), CTX);
        expect(ids(issues)).not.toContain("spf.wrong-version");
    });
});

describe("validateSPF: all mechanism", () => {
    it("accepts a clean record ending in all", () => {
        const issues = validateSPF(parseSPF("v=spf1 ip4:1.2.3.4 -all"), CTX);
        expect(issues).toEqual([]);
    });

    it("warns when there is no all and no redirect", () => {
        const issues = validateSPF(parseSPF("v=spf1 ip4:1.2.3.4"), CTX);
        expect(ids(issues)).toContain("spf.no-all-mechanism");
    });

    it("does not warn about missing all when a redirect is present", () => {
        const issues = validateSPF(parseSPF("v=spf1 redirect=_spf.example.com"), CTX);
        expect(ids(issues)).not.toContain("spf.no-all-mechanism");
    });

    it("flags multiple all mechanisms, pointing at the second occurrence", () => {
        const issues = validateSPF(parseSPF("v=spf1 ~all -all"), CTX);
        const issue = issues.find((i) => i.id === "spf.multiple-all");
        expect(issue).toBeDefined();
        expect(issue!.params).toEqual({ count: 2 });
        expect(issue!.field).toBe("f[1]");
    });

    it("flags three all mechanisms with the correct count", () => {
        const issues = validateSPF(parseSPF("v=spf1 all ~all -all"), CTX);
        const issue = issues.find((i) => i.id === "spf.multiple-all");
        expect(issue!.params).toEqual({ count: 3 });
    });

    it("warns when all is present but not the last term", () => {
        const issues = validateSPF(parseSPF("v=spf1 -all ip4:1.2.3.4"), CTX);
        const issue = issues.find((i) => i.id === "spf.all-not-last");
        expect(issue).toBeDefined();
        expect(issue!.field).toBe("f[0]");
    });

    it("does not warn when all is the last term", () => {
        const issues = validateSPF(parseSPF("v=spf1 ip4:1.2.3.4 -all"), CTX);
        expect(ids(issues)).not.toContain("spf.all-not-last");
    });

    it("does not flag all-not-last when there are multiple all terms", () => {
        // spf.multiple-all already reports the problem; all-not-last only
        // fires when there is exactly one all term.
        const issues = validateSPF(parseSPF("v=spf1 all -all ip4:1.2.3.4"), CTX);
        expect(ids(issues)).not.toContain("spf.all-not-last");
    });
});

describe("validateSPF: redirect modifier", () => {
    it("warns when redirect is combined with an all mechanism", () => {
        const issues = validateSPF(parseSPF("v=spf1 redirect=_spf.example.com -all"), CTX);
        const issue = issues.find((i) => i.id === "spf.redirect-with-all");
        expect(issue).toBeDefined();
        expect(issue!.field).toBe("f[0]");
    });

    it("does not warn about redirect-with-all when there is no all", () => {
        const issues = validateSPF(parseSPF("v=spf1 redirect=_spf.example.com"), CTX);
        expect(ids(issues)).not.toContain("spf.redirect-with-all");
    });

    it("flags a second redirect modifier as an error", () => {
        const issues = validateSPF(parseSPF("v=spf1 redirect=a.com redirect=b.com"), CTX);
        const issue = issues.find((i) => i.id === "spf.multiple-redirect");
        expect(issue).toBeDefined();
        expect(issue!.field).toBe("f[1]");
    });

    it("does not flag a single redirect", () => {
        const issues = validateSPF(parseSPF("v=spf1 redirect=a.com"), CTX);
        expect(ids(issues)).not.toContain("spf.multiple-redirect");
    });
});

describe("validateSPF: ptr mechanism", () => {
    it("warns that ptr is deprecated", () => {
        const issues = validateSPF(parseSPF("v=spf1 ptr -all"), CTX);
        const issue = issues.find((i) => i.id === "spf.ptr-deprecated");
        expect(issue).toBeDefined();
        expect(issue!.field).toBe("f[0]");
    });

    it("does not warn when ptr is absent", () => {
        const issues = validateSPF(parseSPF("v=spf1 -all"), CTX);
        expect(ids(issues)).not.toContain("spf.ptr-deprecated");
    });

    it("only reports the first occurrence's field, even with several ptr terms", () => {
        const issues = validateSPF(parseSPF("v=spf1 ptr ptr -all"), CTX);
        const flagged = issues.filter((i) => i.id === "spf.ptr-deprecated");
        expect(flagged).toHaveLength(1);
        expect(flagged[0].field).toBe("f[0]");
    });
});

describe("validateSPF: per-term checks", () => {
    it("flags an empty term from a trailing separator", () => {
        // A stray trailing ";" survives parseSPF's split (only whitespace is
        // trimmed beforehand), landing as a genuine empty entry in val.f.
        const issues = validateSPF(parseSPF("v=spf1 -all;"), CTX);
        const issue = issues.find((i) => i.id === "spf.empty-term");
        expect(issue).toBeDefined();
        expect(issue!.field).toBe("f[1]");
    });

    it("flags an unknown mechanism", () => {
        const issues = validateSPF(parseSPF("v=spf1 bogus:foo -all"), CTX);
        const issue = issues.find((i) => i.id === "spf.unknown-mechanism");
        expect(issue).toBeDefined();
        expect(issue!.params).toEqual({ mechanism: "bogus:foo" });
        expect(issue!.severity).toBe("error");
    });

    it("flags an unknown modifier as a warning", () => {
        const issues = validateSPF(parseSPF("v=spf1 foo=bar -all"), CTX);
        const issue = issues.find((i) => i.id === "spf.unknown-modifier");
        expect(issue).toBeDefined();
        expect(issue!.params).toEqual({ modifier: "foo" });
        expect(issue!.severity).toBe("warning");
    });

    it.each([
        ["include", "include"],
        ["exists", "exists"],
        ["ptr", "ptr"],
        ["redirect=", "redirect"],
    ])("flags %s without a value", (raw, mech) => {
        const issues = validateSPF(parseSPF(`v=spf1 ${raw} -all`), CTX);
        const issue = issues.find((i) => i.id === "spf.mechanism-missing-value");
        expect(issue).toBeDefined();
        expect(issue!.params).toEqual({ mechanism: mech });
    });

    it("does not flag bare a without a value", () => {
        const issues = validateSPF(parseSPF("v=spf1 a -all"), CTX);
        expect(ids(issues)).not.toContain("spf.mechanism-missing-value");
    });

    it("does not flag bare mx without a value", () => {
        const issues = validateSPF(parseSPF("v=spf1 mx -all"), CTX);
        expect(ids(issues)).not.toContain("spf.mechanism-missing-value");
    });

    it("does not flag a mechanism that has a value", () => {
        const issues = validateSPF(parseSPF("v=spf1 include:_spf.example.com -all"), CTX);
        expect(ids(issues)).not.toContain("spf.mechanism-missing-value");
    });

    it("flags duplicate mechanisms as info, case-insensitively", () => {
        const issues = validateSPF(parseSPF("v=spf1 include:a.com INCLUDE:a.com -all"), CTX);
        const dup = issues.find((i) => i.id === "spf.duplicate-mechanism");
        expect(dup).toBeDefined();
        expect(dup!.severity).toBe("info");
        expect(dup!.field).toBe("f[1]");
    });

    it("does not flag distinct mechanisms as duplicates", () => {
        const issues = validateSPF(parseSPF("v=spf1 include:a.com include:b.com -all"), CTX);
        expect(ids(issues)).not.toContain("spf.duplicate-mechanism");
    });

    it("reports one duplicate issue per repeated occurrence", () => {
        const issues = validateSPF(
            parseSPF("v=spf1 ip4:1.2.3.4 ip4:1.2.3.4 ip4:1.2.3.4 -all"),
            CTX,
        );
        expect(issues.filter((i) => i.id === "spf.duplicate-mechanism")).toHaveLength(2);
    });
});

describe("validateSPF: TXT length", () => {
    it("does not flag a short record", () => {
        const issues = validateSPF(parseSPF("v=spf1 -all"), CTX);
        expect(ids(issues)).not.toContain("spf.length-exceeds-txt-string");
    });

    it("flags a record longer than 255 characters", () => {
        const many = Array.from({ length: 30 }, (_, i) => `ip4:10.0.${i}.0/24`).join(" ");
        const issues = validateSPF(parseSPF(`v=spf1 ${many} -all`), CTX);
        const issue = issues.find((i) => i.id === "spf.length-exceeds-txt-string");
        expect(issue).toBeDefined();
        expect(issue!.severity).toBe("info");
        expect(issue!.params?.max).toBe(255);
        expect(typeof issue!.params?.length).toBe("number");
        expect((issue!.params?.length as number) > 255).toBe(true);
    });
});

describe("validateSPF: combined scenario", () => {
    it("reports every applicable issue for a thoroughly broken record", () => {
        const issues = validateSPF(
            parseSPF("v=spf1 bogus -all ~all ptr redirect=a.com redirect=b.com"),
            CTX,
        );
        // With two "all" terms present, spf.all-not-last is suppressed (it
        // only fires when there is exactly one all term); spf.multiple-all
        // already covers that case.
        expect(ids(issues)).toEqual(
            expect.arrayContaining([
                "spf.multiple-all",
                "spf.redirect-with-all",
                "spf.multiple-redirect",
                "spf.ptr-deprecated",
                "spf.unknown-mechanism",
            ]),
        );
        expect(ids(issues)).not.toContain("spf.all-not-last");
    });
});

// ---------------------------------------------------------------------------
// validateSPFRecursive (async)
// ---------------------------------------------------------------------------

function flattenResponse(
    overrides: Partial<HappydnsSpfFlattenResponse> = {},
): HappydnsSpfFlattenResponse {
    return {
        record: "v=spf1 include:_spf.example.com -all",
        lookupCount: 1,
        exceeded: false,
        voidExceeded: false,
        voidLookups: 0,
        truncated: false,
        ...overrides,
    };
}

describe("validateSPFRecursive: short-circuits", () => {
    const signal = new AbortController().signal;

    it("returns no issue and does not call flattenSPF when the version is missing", async () => {
        const issues = await validateSPFRecursive(parseSPF(""), CTX, signal);
        expect(issues).toEqual([]);
        expect(flattenSPF).not.toHaveBeenCalled();
    });

    it("returns no issue and does not call flattenSPF for a wrong version", async () => {
        const issues = await validateSPFRecursive(
            parseSPF("v=spf2 include:a.com -all"),
            CTX,
            signal,
        );
        expect(issues).toEqual([]);
        expect(flattenSPF).not.toHaveBeenCalled();
    });

    it("returns no issue and does not call flattenSPF when there are no lookup mechanisms", async () => {
        const issues = await validateSPFRecursive(parseSPF("v=spf1 ip4:1.2.3.4 -all"), CTX, signal);
        expect(issues).toEqual([]);
        expect(flattenSPF).not.toHaveBeenCalled();
    });

    it("returns no issue when the effective domain cannot be resolved", async () => {
        const ctx = buildContext("@", makeDomain({ domain: "" }), null);
        const issues = await validateSPFRecursive(
            parseSPF("v=spf1 include:a.com -all"),
            ctx,
            signal,
        );
        expect(issues).toEqual([]);
        expect(flattenSPF).not.toHaveBeenCalled();
    });
});

describe("validateSPFRecursive: calls flattenSPF with the right arguments", () => {
    beforeEach(() => vi.mocked(flattenSPF).mockReset());

    it("passes the resolved domain, stringified record and the abort signal", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(flattenResponse());
        const controller = new AbortController();
        const val = parseSPF("v=spf1 include:_spf.example.com -all");
        await validateSPFRecursive(val, CTX, controller.signal);
        expect(flattenSPF).toHaveBeenCalledWith(
            { domain: "example.com.", record: "v=spf1 include:_spf.example.com -all" },
            controller.signal,
        );
    });

    it("resolves relative subdomains against the origin", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(flattenResponse());
        const ctx = buildContext("mail", ORIGIN, null);
        await validateSPFRecursive(
            parseSPF("v=spf1 include:a.com -all"),
            ctx,
            new AbortController().signal,
        );
        expect(flattenSPF).toHaveBeenCalledWith(
            expect.objectContaining({ domain: "mail.example.com." }),
            expect.anything(),
        );
    });
});

describe("validateSPFRecursive: lookup budget thresholds", () => {
    beforeEach(() => vi.mocked(flattenSPF).mockReset());
    const val = parseSPF("v=spf1 include:a.com -all");
    const signal = new AbortController().signal;

    it("reports no issue when comfortably under the warning threshold", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(
            flattenResponse({ lookupCount: 3, exceeded: false }),
        );
        const issues = await validateSPFRecursive(val, CTX, signal);
        expect(ids(issues)).toEqual([]);
    });

    it("warns at the warning threshold (8)", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(
            flattenResponse({ lookupCount: 8, exceeded: false }),
        );
        const issues = await validateSPFRecursive(val, CTX, signal);
        const issue = issues.find((i) => i.id === "spf.recursive-many-lookups");
        expect(issue).toBeDefined();
        expect(issue!.severity).toBe("warning");
        expect(issue!.params).toEqual({ count: 8, max: 10 });
    });

    it("warns just below the hard limit (9)", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(
            flattenResponse({ lookupCount: 9, exceeded: false }),
        );
        const issues = await validateSPFRecursive(val, CTX, signal);
        expect(ids(issues)).toContain("spf.recursive-many-lookups");
        expect(ids(issues)).not.toContain("spf.recursive-too-many-lookups");
    });

    it("errors when the resolver reports the budget exceeded", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(
            flattenResponse({ lookupCount: 12, exceeded: true }),
        );
        const issues = await validateSPFRecursive(val, CTX, signal);
        const issue = issues.find((i) => i.id === "spf.recursive-too-many-lookups");
        expect(issue).toBeDefined();
        expect(issue!.severity).toBe("error");
        expect(issue!.params).toEqual({ count: 12, max: 10 });
        // exceeded and many-lookups are mutually exclusive branches.
        expect(ids(issues)).not.toContain("spf.recursive-many-lookups");
    });

    it("treats a missing lookupCount as zero", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(
            flattenResponse({ lookupCount: undefined, exceeded: false }),
        );
        const issues = await validateSPFRecursive(val, CTX, signal);
        expect(ids(issues)).not.toContain("spf.recursive-many-lookups");
        expect(ids(issues)).not.toContain("spf.recursive-too-many-lookups");
    });
});

describe("validateSPFRecursive: void lookups", () => {
    beforeEach(() => vi.mocked(flattenSPF).mockReset());
    const val = parseSPF("v=spf1 include:a.com -all");
    const signal = new AbortController().signal;

    it("does not warn when void lookups are under the limit", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(
            flattenResponse({ voidExceeded: false, voidLookups: 1 }),
        );
        const issues = await validateSPFRecursive(val, CTX, signal);
        expect(ids(issues)).not.toContain("spf.too-many-void-lookups");
    });

    it("warns when the resolver reports void lookups exceeded", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(
            flattenResponse({ voidExceeded: true, voidLookups: 3 }),
        );
        const issues = await validateSPFRecursive(val, CTX, signal);
        const issue = issues.find((i) => i.id === "spf.too-many-void-lookups");
        expect(issue).toBeDefined();
        expect(issue!.severity).toBe("warning");
        expect(issue!.params).toEqual({ count: 3, max: 2 });
    });

    it("treats a missing voidLookups as zero in params", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(
            flattenResponse({ voidExceeded: true, voidLookups: undefined }),
        );
        const issues = await validateSPFRecursive(val, CTX, signal);
        const issue = issues.find((i) => i.id === "spf.too-many-void-lookups");
        expect(issue!.params).toEqual({ count: 0, max: 2 });
    });
});

describe("validateSPFRecursive: tree walk of include errors", () => {
    beforeEach(() => vi.mocked(flattenSPF).mockReset());
    const val = parseSPF("v=spf1 include:a.com -all");
    const signal = new AbortController().signal;

    async function walkWith(tree: HappydnsSpfNode): Promise<ComplianceIssue[]> {
        vi.mocked(flattenSPF).mockResolvedValueOnce(flattenResponse({ tree }));
        return validateSPFRecursive(val, CTX, signal);
    }

    it("reports no include issue when the tree has no errors", async () => {
        const issues = await walkWith({
            domain: "a.com",
            mechanism: "include:a.com",
            children: [],
        });
        expect(ids(issues)).toEqual([]);
    });

    it.each([
        ["loop", "spf.include-loop", "warning"],
        ["no-spf", "spf.include-no-spf", "warning"],
        ["nxdomain", "spf.include-no-spf", "warning"],
        ["timeout", "spf.include-resolver-error", "info"],
        ["resolver", "spf.include-resolver-error", "info"],
        ["syntax", "spf.include-error", "warning"],
    ])("maps node error %s to %s (%s)", async (err, expectedId, expectedSeverity) => {
        const issues = await walkWith({
            domain: "broken.example.com",
            mechanism: `include:broken.example.com`,
            error: err,
            children: [],
        });
        const issue = issues.find((i) => i.id === expectedId);
        expect(issue).toBeDefined();
        expect(issue!.severity).toBe(expectedSeverity);
        expect(issue!.params).toEqual({
            domain: "broken.example.com",
            mechanism: "include:broken.example.com",
        });
    });

    it("ignores budget and depth node errors (already reported at the top level)", async () => {
        const issues = await walkWith({
            domain: "deep.example.com",
            mechanism: "include:deep.example.com",
            error: "budget",
            children: [
                {
                    domain: "deeper.example.com",
                    mechanism: "include:x",
                    error: "depth",
                    children: [],
                },
            ],
        });
        expect(ids(issues)).toEqual([]);
    });

    it("recurses into children and reports each erroring node", async () => {
        const issues = await walkWith({
            domain: "root.example.com",
            mechanism: "include:root.example.com",
            children: [
                { domain: "ok.example.com", mechanism: "include:ok.example.com", children: [] },
                {
                    domain: "loop.example.com",
                    mechanism: "include:loop.example.com",
                    error: "loop",
                    children: [
                        {
                            domain: "grandchild.example.com",
                            mechanism: "include:grandchild.example.com",
                            error: "no-spf",
                            children: [],
                        },
                    ],
                },
            ],
        });
        expect(ids(issues)).toEqual(
            expect.arrayContaining(["spf.include-loop", "spf.include-no-spf"]),
        );
        expect(issues).toHaveLength(2);
    });

    it("handles a missing tree gracefully", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(flattenResponse({ tree: undefined }));
        const issues = await validateSPFRecursive(val, CTX, signal);
        expect(issues).toEqual([]);
    });

    it("defaults domain/mechanism params to empty strings when absent on the node", async () => {
        const issues = await walkWith({ error: "loop", children: [] });
        const issue = issues.find((i) => i.id === "spf.include-loop");
        expect(issue!.params).toEqual({ domain: "", mechanism: "" });
    });
});

describe("validateSPFRecursive: combined result", () => {
    beforeEach(() => vi.mocked(flattenSPF).mockReset());

    it("can report a budget error, a void-lookup warning and an include error together", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(
            flattenResponse({
                lookupCount: 14,
                exceeded: true,
                voidExceeded: true,
                voidLookups: 4,
                tree: {
                    domain: "example.com.",
                    children: [
                        {
                            domain: "broken.example.com",
                            mechanism: "include:broken.example.com",
                            error: "nxdomain",
                            children: [],
                        },
                    ],
                },
            }),
        );
        const issues = await validateSPFRecursive(
            parseSPF("v=spf1 include:a.com include:broken.example.com -all"),
            CTX,
            new AbortController().signal,
        );
        expect(ids(issues)).toEqual(
            expect.arrayContaining([
                "spf.recursive-too-many-lookups",
                "spf.too-many-void-lookups",
                "spf.include-no-spf",
            ]),
        );
        expect(issues).toHaveLength(3);
    });
});
