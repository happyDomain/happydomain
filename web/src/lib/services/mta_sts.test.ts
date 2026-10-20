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

import { describe, expect, it } from "vitest";

import { buildContext } from "$lib/services/compliance";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";
import {
    fetchFailureIssue,
    getZoneApexMxHosts,
    mtaStsPatternMatches,
    mxHostsFrom,
    policyIssues,
    policyURL,
    stripTrailingDot,
    txtRecordIssues,
} from "./mta_sts";

const URL = "https://mta-sts.example.com/.well-known/mta-sts.txt";

const ids = (issues: { id: string }[]) => issues.map((i) => i.id);

function contextWithMX(...targets: string[]) {
    const mx = makeService("svcs.MXs", {
        mx: targets.map((t, i) => ({ Mx: t, Preference: 10 + i })),
    });
    return buildContext("", makeDomain(), makeZone({ services: { "": [mx] } }));
}

const noZone = buildContext("", makeDomain(), null);

describe("mtaStsPatternMatches", () => {
    it("matches exact names, ignoring case and the final dot", () => {
        expect(mtaStsPatternMatches("mail.example.com", "MAIL.example.com.")).toBe(true);
        expect(mtaStsPatternMatches("mail.example.com.", "mail.example.com")).toBe(true);
        expect(mtaStsPatternMatches("mail.example.com", "mx.example.com")).toBe(false);
    });

    it("lets a wildcard match exactly one label", () => {
        expect(mtaStsPatternMatches("*.example.com", "mail.example.com")).toBe(true);
        expect(mtaStsPatternMatches("*.example.com", "a.mail.example.com")).toBe(false);
        expect(mtaStsPatternMatches("*.example.com", "example.com")).toBe(false);
    });

    it("never matches a wildcard with nothing behind it", () => {
        expect(mtaStsPatternMatches("*.", "mail.example.com")).toBe(false);
    });
});

describe("policyURL and stripTrailingDot", () => {
    it("builds the URL RFC 8461 sec. 3.3 mandates", () => {
        expect(policyURL("example.com")).toBe(URL);
        expect(policyURL("example.com.")).toBe(URL);
    });

    it("strips a single trailing dot only", () => {
        expect(stripTrailingDot("example.com.")).toBe("example.com");
        expect(stripTrailingDot("example.com")).toBe("example.com");
    });
});

describe("mxHostsFrom", () => {
    it("collects the targets of every MX, verbatim", () => {
        const services = [
            {
                _svctype: "svcs.MXs",
                Service: { mx: [{ Mx: "a.example.com." }, { Mx: "B.example.com" }] },
            },
            { _svctype: "svcs.MXs", Service: { mx: [{ Mx: "c.example.com." }] } },
        ];

        expect(mxHostsFrom(services)).toEqual([
            "a.example.com.",
            "B.example.com",
            "c.example.com.",
        ]);
    });

    it("skips services without MX and entries without a target", () => {
        const services = [
            { _svctype: "svcs.MXs" },
            { _svctype: "svcs.MXs", Service: {} },
            { _svctype: "svcs.MXs", Service: { mx: "not a list" } },
            {
                _svctype: "svcs.MXs",
                Service: { mx: [{ Mx: "" }, {}, null, { Mx: "ok.example.com." }] },
            },
        ];

        expect(mxHostsFrom(services)).toEqual(["ok.example.com."]);
    });
});

describe("getZoneApexMxHosts", () => {
    it("reads the MX of the apex", () => {
        expect(getZoneApexMxHosts(contextWithMX("a.example.com.", "b.example.com."))).toEqual([
            "a.example.com.",
            "b.example.com.",
        ]);
    });

    it("is empty without a zone", () => {
        expect(getZoneApexMxHosts(noZone)).toEqual([]);
    });
});

describe("txtRecordIssues", () => {
    const parse = (raw: string) => {
        const fields = Object.fromEntries(
            raw.split(";").map((kv) => kv.trim().split("=") as [string, string]),
        );
        return { v: fields.v, id: fields.id };
    };

    it("accepts a clean record", () => {
        expect(
            txtRecordIssues({ Hdr: { Name: "_mta-sts" }, Txt: "v=STSv1;id=20240101" }, parse),
        ).toEqual([]);
    });

    it("has nothing to say about a missing record or an empty value", () => {
        expect(txtRecordIssues(undefined, parse)).toEqual([]);
        expect(txtRecordIssues(null, parse)).toEqual([]);
        expect(txtRecordIssues({ Hdr: { Name: "_mta-sts" }, Txt: "  " }, parse)).toEqual([]);
    });

    it("flags an owner name that is not _mta-sts", () => {
        const issues = txtRecordIssues({ Hdr: { Name: "mta-sts" }, Txt: "v=STSv1;id=1" }, parse);
        expect(ids(issues)).toEqual(["mta_sts.wrong-owner-name"]);
        expect(issues[0].params).toEqual({ name: "mta-sts" });
    });

    it("flags a value the parser rejects", () => {
        const failing = () => {
            throw new Error("nope");
        };
        expect(ids(txtRecordIssues({ Txt: "garbage" }, failing))).toEqual(["mta_sts.parse-error"]);
    });

    it("flags a missing or unsupported version", () => {
        expect(ids(txtRecordIssues({ Txt: "id=1" }, parse))).toEqual(["mta_sts.missing-version"]);
        expect(ids(txtRecordIssues({ Txt: "v=STSv2;id=1" }, parse))).toEqual([
            "mta_sts.invalid-version",
        ]);
    });

    it("flags a missing or malformed id", () => {
        expect(ids(txtRecordIssues({ Txt: "v=STSv1" }, parse))).toEqual(["mta_sts.missing-id"]);
        expect(ids(txtRecordIssues({ Txt: "v=STSv1;id=not-alphanumeric" }, parse))).toEqual([
            "mta_sts.invalid-id",
        ]);
        expect(ids(txtRecordIssues({ Txt: "v=STSv1;id=" + "a".repeat(33) }, parse))).toEqual([
            "mta_sts.invalid-id",
        ]);
    });
});

describe("policyIssues", () => {
    const good = { version: "STSv1", mode: "enforce", mx: ["mail.example.com"], maxAge: 604800 };

    it("accepts a clean policy", () => {
        expect(policyIssues(good, noZone, URL)).toEqual([]);
    });

    it("flags a missing or unsupported version", () => {
        expect(ids(policyIssues({ ...good, version: undefined }, noZone, URL))).toEqual([
            "mta_sts.policy-missing-version",
        ]);
        expect(ids(policyIssues({ ...good, version: "STSv2" }, noZone, URL))).toEqual([
            "mta_sts.policy-invalid-version",
        ]);
    });

    it("flags a missing or unknown mode", () => {
        expect(ids(policyIssues({ ...good, mode: undefined }, noZone, URL))).toContain(
            "mta_sts.policy-missing-mode",
        );
        expect(ids(policyIssues({ ...good, mode: "strict" }, noZone, URL))).toContain(
            "mta_sts.policy-invalid-mode",
        );
    });

    it("reports the modes that do not protect anything", () => {
        const none = policyIssues({ ...good, mode: "none", mx: [] }, noZone, URL);
        expect(none.map((i) => [i.id, i.severity])).toEqual([
            ["mta_sts.policy-mode-none", "warning"],
        ]);

        const testing = policyIssues({ ...good, mode: "testing" }, noZone, URL);
        expect(testing.map((i) => [i.id, i.severity])).toEqual([
            ["mta_sts.policy-mode-testing", "info"],
        ]);
    });

    it("requires an mx unless the policy is withdrawn", () => {
        expect(ids(policyIssues({ ...good, mx: [] }, noZone, URL))).toEqual([
            "mta_sts.policy-missing-mx",
        ]);
        expect(ids(policyIssues({ ...good, mx: undefined }, noZone, URL))).toEqual([
            "mta_sts.policy-missing-mx",
        ]);
        expect(ids(policyIssues({ ...good, mode: "none", mx: [] }, noZone, URL))).not.toContain(
            "mta_sts.policy-missing-mx",
        );
    });

    it("checks max_age", () => {
        expect(ids(policyIssues({ ...good, maxAge: 0 }, noZone, URL))).toEqual([
            "mta_sts.policy-missing-max-age",
        ]);
        expect(ids(policyIssues({ ...good, maxAge: 31557601 }, noZone, URL))).toEqual([
            "mta_sts.policy-invalid-max-age",
        ]);
        expect(ids(policyIssues({ ...good, maxAge: 31557600 }, noZone, URL))).toEqual([]);
        expect(ids(policyIssues({ ...good, maxAge: 3600 }, noZone, URL))).toEqual([
            "mta_sts.policy-short-max-age",
        ]);
        expect(ids(policyIssues({ ...good, maxAge: 86400 }, noZone, URL))).toEqual([]);
    });

    describe("against the MX records of the zone", () => {
        it("accepts a policy covering them, by name or wildcard", () => {
            const ctx = contextWithMX("mail.example.com.", "mx.example.net.");
            const policy = { ...good, mx: ["mail.example.com", "*.example.net"] };

            expect(policyIssues(policy, ctx, URL)).toEqual([]);
        });

        it("flags an MX the policy does not cover, as an error when enforcing", () => {
            const ctx = contextWithMX("mail.example.com.", "other.example.com.");

            const enforce = policyIssues(good, ctx, URL).find(
                (i) => i.id === "mta_sts.zone-mx-not-covered",
            );
            expect(enforce?.severity).toBe("error");
            expect(enforce?.params).toMatchObject({ host: "other.example.com." });

            const testing = policyIssues({ ...good, mode: "testing" }, ctx, URL);
            expect(testing.find((i) => i.id === "mta_sts.zone-mx-not-covered")?.severity).toBe(
                "warning",
            );
        });

        it("flags a pattern matching no MX of the zone", () => {
            const ctx = contextWithMX("mail.example.com.");
            const policy = { ...good, mx: ["mail.example.com", "old.example.com"] };

            const issues = policyIssues(policy, ctx, URL);
            expect(ids(issues)).toEqual(["mta_sts.policy-mx-unused"]);
            expect(issues[0].params).toMatchObject({ pattern: "old.example.com" });
        });

        it("warns when the zone has no MX at all", () => {
            const ctx = buildContext("", makeDomain(), makeZone({ services: { "": [] } }));

            expect(ids(policyIssues(good, ctx, URL))).toEqual(["mta_sts.zone-no-mx"]);
        });

        it("does not cross-check a withdrawn policy", () => {
            const ctx = contextWithMX("mail.example.com.");

            expect(
                ids(policyIssues({ ...good, mode: "none", mx: ["old.example.com"] }, ctx, URL)),
            ).toEqual(["mta_sts.policy-mode-none"]);
        });
    });
});

describe("fetchFailureIssue", () => {
    it("has no failure to report for a successful fetch or an unknown status", () => {
        expect(fetchFailureIssue({ status: "ok" })).toBeNull();
        expect(fetchFailureIssue({ status: "something-new" })).toBeNull();
        expect(fetchFailureIssue({})).toBeNull();
    });

    it.each([
        ["dns-error", "mta_sts.policy-dns-error", "error"],
        ["tls-error", "mta_sts.policy-tls-error", "error"],
        ["not-found", "mta_sts.policy-not-found", "error"],
        ["fetch-error", "mta_sts.policy-fetch-error", "warning"],
        ["too-large", "mta_sts.policy-too-large", "error"],
    ])("maps %s to %s", (status, id, severity) => {
        const issue = fetchFailureIssue({ status, url: URL, errorMsg: "boom" });

        expect(issue?.id).toBe(id);
        expect(issue?.severity).toBe(severity);
        expect(issue?.params).toMatchObject({ url: URL });
    });

    it("tells a redirect from another HTTP error", () => {
        expect(fetchFailureIssue({ status: "http-error", url: URL, httpCode: 500 })?.id).toBe(
            "mta_sts.policy-http-error",
        );

        const redirect = fetchFailureIssue({
            status: "http-error",
            url: URL,
            httpCode: 301,
            redirected: true,
        });
        expect(redirect?.id).toBe("mta_sts.policy-redirect");
        expect(redirect?.params).toMatchObject({ code: 301 });
    });

    it("passes the error message of the failures that have one", () => {
        expect(
            fetchFailureIssue({ status: "tls-error", url: URL, errorMsg: "expired" })?.params,
        ).toEqual({
            url: URL,
            error: "expired",
        });
    });
});
