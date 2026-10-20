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

import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("$lib/api/resolver", () => ({
    fetchMTAStsPolicy: vi.fn(),
}));

import "./compliance";
import { fetchMTAStsPolicy } from "$lib/api/resolver";
import { buildContext, getValidators, type ComplianceIssue } from "$lib/services/compliance";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";

const ORIGIN = makeDomain();
const POLICY_URL = "https://mta-sts.example.com/.well-known/mta-sts.txt";

const ids = (issues: ComplianceIssue[]) => issues.map((i) => i.id);

const validators = () => getValidators("abstract.MTASTS")!;

function ctxWithMX(...targets: string[]) {
    const mx = makeService("svcs.MXs", {
        mx: targets.map((t, i) => ({ Mx: t, Preference: 10 + i })),
    });
    return buildContext("", ORIGIN, makeZone({ services: { "": [mx] } }));
}

// A hosted service whose policy is configured as given.
function hosted(overrides: Record<string, any> = {}): Record<string, any> {
    return {
        mode: "enforce",
        maxAge: 604800,
        mx: ["mail.example.com"],
        txt: { Hdr: { Name: "_mta-sts" }, Txt: "v=STSv1;id=20240101" },
        policyCNAME: { Hdr: { Name: "mta-sts" }, Target: "happydomain.example.com." },
        ...overrides,
    };
}

describe("abstract.MTASTS sync compliance", () => {
    const run = (raw: Record<string, any>, ctx = ctxWithMX("mail.example.com.")) =>
        validators().sync!(raw, ctx);

    it("accepts a clean hosted policy", () => {
        expect(run(hosted())).toEqual([]);
    });

    it("warns that the policy is not served when the CNAME is missing", () => {
        const issues = run(hosted({ policyCNAME: undefined }));

        expect(ids(issues)).toEqual(["mta_sts.hosted-no-policy-host"]);
        expect(issues[0].severity).toBe("warning");
    });

    it("flags a CNAME that is not on the mta-sts host", () => {
        const issues = run(hosted({ policyCNAME: { Hdr: { Name: "www" } } }));

        expect(ids(issues)).toEqual(["mta_sts.hosted-wrong-policy-host"]);
        expect(issues[0].params).toEqual({ name: "www" });
    });

    it("accepts the mta-sts host of a subdomain", () => {
        expect(ids(run(hosted({ policyCNAME: { Hdr: { Name: "mta-sts.sub" } } })))).toEqual([]);
    });

    it("checks the TXT record like the plain service does", () => {
        const issues = run(hosted({ txt: { Hdr: { Name: "_mta-sts" }, Txt: "v=STSv1;id=" } }));

        expect(ids(issues)).toContain("mta_sts.missing-id");
    });

    // The editor fills these in, so the policy is validated as it will be served.
    it("applies the same defaults as the policy file", () => {
        const issues = run(hosted({ mode: "", maxAge: 0 }));

        expect(ids(issues)).toEqual(["mta_sts.policy-mode-testing"]);
    });

    it("ignores blank MX entries", () => {
        expect(ids(run(hosted({ mx: ["mail.example.com", "  ", ""] })))).toEqual([]);
        expect(ids(run(hosted({ mx: ["  "] })))).toContain("mta_sts.policy-missing-mx");
    });

    it("cross-checks the policy with the MX records of the zone", () => {
        const issues = run(hosted(), ctxWithMX("mail.example.com.", "other.example.com."));

        expect(ids(issues)).toEqual(["mta_sts.zone-mx-not-covered"]);
    });
});

describe("abstract.MTASTS async compliance", () => {
    beforeEach(() => {
        vi.mocked(fetchMTAStsPolicy).mockReset();
    });

    function serve(policy: Record<string, any>) {
        vi.mocked(fetchMTAStsPolicy).mockResolvedValueOnce({
            status: "ok",
            url: POLICY_URL,
            version: "STSv1",
            mode: "enforce",
            maxAge: 604800,
            mx: ["mail.example.com"],
            ...policy,
        });
    }

    const run = (raw: Record<string, any>) =>
        validators().async!(raw, ctxWithMX("mail.example.com."), new AbortController().signal);

    it("has no policy to compare with until the CNAME is published", async () => {
        expect(await run(hosted({ policyCNAME: undefined }))).toEqual([]);
        expect(fetchMTAStsPolicy).not.toHaveBeenCalled();
    });

    it("does not fetch anything without a domain", async () => {
        const nameless = buildContext("", makeDomain({ domain: "" }), null);

        expect(await validators().async!(hosted(), nameless, new AbortController().signal)).toEqual(
            [],
        );
        expect(fetchMTAStsPolicy).not.toHaveBeenCalled();
    });

    it("accepts a served policy equal to the configured one", async () => {
        serve({});

        expect(await run(hosted())).toEqual([]);
        expect(fetchMTAStsPolicy).toHaveBeenCalledWith(
            { domain: ORIGIN.domain.replace(/\.$/, "") },
            expect.anything(),
        );
    });

    it("reports a failed fetch and nothing else", async () => {
        serve({ status: "tls-error", errorMsg: "expired", mode: "testing" });

        expect(ids(await run(hosted()))).toEqual(["mta_sts.policy-tls-error"]);
    });

    it("reports a served mode that is not the configured one", async () => {
        serve({ mode: "testing" });

        const issues = await run(hosted());
        expect(ids(issues)).toEqual(["mta_sts.hosted-mode-mismatch"]);
        expect(issues[0].params).toEqual({
            url: POLICY_URL,
            served: "testing",
            configured: "enforce",
        });
    });

    it("compares the default mode when none is configured", async () => {
        serve({ mode: "testing" });

        expect(await run(hosted({ mode: "" }))).toEqual([]);
    });

    it("reports served MX hosts that are not the configured ones", async () => {
        serve({ mx: ["mail.example.com", "old.example.com"] });

        const issues = await run(hosted());
        expect(ids(issues)).toEqual(["mta_sts.hosted-mx-mismatch"]);
        expect(issues[0].params).toMatchObject({
            served: "mail.example.com, old.example.com",
            configured: "mail.example.com",
        });
    });

    it("compares MX hosts regardless of case, order and final dot", async () => {
        serve({ mx: ["B.example.com.", "mail.example.com"] });

        expect(await run(hosted({ mx: ["mail.example.com.", "b.EXAMPLE.com"] }))).toEqual([]);
    });

    // A duplicated entry on one side must not hide a difference.
    it("does not let a duplicated entry hide a missing one", async () => {
        serve({ mx: ["a.example.com", "a.example.com"] });

        expect(ids(await run(hosted({ mx: ["a.example.com", "b.example.com"] })))).toEqual([
            "mta_sts.hosted-mx-mismatch",
        ]);
    });

    it("does not report duplicates alone as a mismatch", async () => {
        serve({ mx: ["a.example.com", "a.example.com"] });

        expect(await run(hosted({ mx: ["a.example.com"] }))).toEqual([]);
    });
});
