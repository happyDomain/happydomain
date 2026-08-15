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

import { beforeEach, describe, it, expect, vi } from "vitest";

vi.mock("$lib/api/resolver", () => ({
    checkDMARCReportAuth: vi.fn(),
}));

import "./compliance";
import { buildContext, getValidators, type ComplianceIssue } from "$lib/services/compliance";
import type { Zone } from "$lib/model/zone";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";
import { checkDMARCReportAuth } from "$lib/api/resolver";

const ORIGIN = makeDomain();
const CTX = buildContext("_dmarc", ORIGIN, null);

function run(txt: string, name = "_dmarc.example.com."): ComplianceIssue[] {
    const v = getValidators("svcs.DMARC");
    expect(v?.sync).toBeDefined();
    return v!.sync!({ txt: { Hdr: { Name: name }, Txt: txt } }, CTX);
}

const svc = (svctype: string) => makeService(svctype);

// A DKIM selector of the zone, as the cross-checks read it.
const KEY = "A".repeat(360);
const dkim = (txt: string, name = "selector1._domainkey") =>
    makeService("svcs.DKIMRecord", { txt: { Hdr: { Name: name }, Txt: txt } });

function runWithZone(txt: string, zone: Zone): ComplianceIssue[] {
    const v = getValidators("svcs.DMARC");
    return v!.sync!(
        { txt: { Hdr: { Name: "_dmarc.example.com." }, Txt: txt } },
        buildContext("_dmarc", ORIGIN, zone),
    );
}

const ids = (issues: ComplianceIssue[]) => issues.map((i) => i.id);

describe("DMARC compliance: happy paths", () => {
    it("accepts a minimal reject record", () => {
        const issues = run("v=DMARC1;p=reject");
        expect(ids(issues)).toEqual([]);
    });
    it("accepts a record with rua mailto", () => {
        const issues = run("v=DMARC1;p=quarantine;rua=mailto:dmarc@example.com");
        expect(ids(issues)).toEqual([]);
    });
    it("accepts http rua URIs", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://reports.example.com/dmarc");
        expect(ids(issues)).toEqual([]);
    });
});

describe("DMARC compliance: owner name", () => {
    it("flags wrong owner name", () => {
        const issues = run("v=DMARC1;p=reject", "example.com.");
        expect(ids(issues)).toContain("dmarc.wrong-owner-name");
    });
});

describe("DMARC compliance: version", () => {
    it("flags a missing version", () => {
        const issues = run("p=reject");
        expect(ids(issues)).toContain("dmarc.missing-version");
    });
    it("flags a non-DMARC1 version", () => {
        const issues = run("v=DMARC2;p=reject");
        expect(ids(issues)).toContain("dmarc.invalid-version");
    });
});

describe("DMARC compliance: policy", () => {
    it("flags a missing policy", () => {
        const issues = run("v=DMARC1");
        expect(ids(issues)).toContain("dmarc.missing-policy");
    });
    it("flags an invalid policy", () => {
        const issues = run("v=DMARC1;p=foo");
        expect(ids(issues)).toContain("dmarc.invalid-policy");
    });
    it("infos on monitoring-only (p=none)", () => {
        const issues = run("v=DMARC1;p=none");
        expect(ids(issues)).toContain("dmarc.monitoring-only");
    });
    it("flags an invalid sp", () => {
        const issues = run("v=DMARC1;p=reject;sp=foo");
        expect(ids(issues)).toContain("dmarc.invalid-sp");
    });
});

describe("DMARC compliance: one record per name", () => {
    const at = (svctype: string, txt: string, name = "_dmarc.example.com.") =>
        makeService(svctype, { txt: { Hdr: { Name: name }, Txt: txt } });

    it("flags a second DMARC record on the same name", () => {
        const zone = makeZone({
            services: { _dmarc: [at("svcs.DMARC", "v=DMARC1;p=none")] },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        const dup = issues.find((i) => i.id === "dmarc.duplicate-record");
        expect(dup?.params).toMatchObject({ count: 2 });
    });
    it("counts a raw TXT holding a DMARC record", () => {
        const zone = makeZone({
            services: { _dmarc: [at("svcs.TXT", "v=DMARC1;p=none")] },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).toContain("dmarc.duplicate-record");
    });
    it("leaves an unrelated TXT alone", () => {
        const zone = makeZone({
            services: { _dmarc: [at("svcs.TXT", "some-verification-token")] },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.duplicate-record");
    });
    it("does not count a record published on another name", () => {
        const zone = makeZone({
            services: {
                _dmarc: [at("svcs.DMARC", "v=DMARC1;p=none", "_dmarc.sub.example.com.")],
            },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.duplicate-record");
    });
    it("says nothing when the zone is unknown", () => {
        const issues = run("v=DMARC1;p=reject");
        expect(ids(issues)).not.toContain("dmarc.duplicate-record");
    });
});

describe("DMARC compliance: subdomain policy inheritance", () => {
    const dmarc = (txt: string) =>
        makeService("svcs.DMARC", { txt: { Hdr: { Name: "_dmarc" }, Txt: txt } });

    function runOnSubdomain(txt: string, zone: Zone | null = null): ComplianceIssue[] {
        const v = getValidators("svcs.DMARC");
        return v!.sync!(
            { txt: { Hdr: { Name: "_dmarc.sub.example.com." }, Txt: txt } },
            buildContext("_dmarc.sub", ORIGIN, zone),
        );
    }

    it("warns when sp is weaker than p", () => {
        const issues = run("v=DMARC1;p=reject;sp=none");
        expect(ids(issues)).toContain("dmarc.sp-weaker-than-p");
    });
    it("accepts an sp stronger than p", () => {
        const issues = run("v=DMARC1;p=quarantine;sp=reject");
        expect(ids(issues)).not.toContain("dmarc.sp-weaker-than-p");
    });
    it("infos when sp restates p", () => {
        const issues = run("v=DMARC1;p=reject;sp=reject");
        expect(ids(issues)).toContain("dmarc.sp-same-as-p");
    });
    it("says nothing about inheritance without sp", () => {
        const issues = run("v=DMARC1;p=reject");
        expect(ids(issues)).not.toContain("dmarc.sp-same-as-p");
        expect(ids(issues)).not.toContain("dmarc.sp-weaker-than-p");
        expect(ids(issues)).not.toContain("dmarc.sp-on-subdomain");
    });
    it("stays quiet on an invalid sp, already reported", () => {
        const issues = run("v=DMARC1;p=reject;sp=foo");
        expect(ids(issues)).not.toContain("dmarc.sp-weaker-than-p");
    });
    it("infos on an sp published on a subdomain record", () => {
        const issues = runOnSubdomain("v=DMARC1;p=reject;sp=reject");
        const sub = issues.find((i) => i.id === "dmarc.sp-on-subdomain");
        expect(sub?.params).toMatchObject({ name: "sub" });
    });
    it("warns when a subdomain record weakens the apex policy", () => {
        const zone = makeZone({ services: { _dmarc: [dmarc("v=DMARC1;p=reject")] } });
        const issues = runOnSubdomain("v=DMARC1;p=none", zone);
        const weaken = issues.find((i) => i.id === "dmarc.subdomain-weakens-parent");
        expect(weaken?.params).toMatchObject({ policy: "none", inherited: "reject" });
    });
    it("compares against the sp of the apex when it sets one", () => {
        const zone = makeZone({ services: { _dmarc: [dmarc("v=DMARC1;p=reject;sp=none")] } });
        const issues = runOnSubdomain("v=DMARC1;p=none", zone);
        expect(ids(issues)).not.toContain("dmarc.subdomain-weakens-parent");
    });
    it("stays quiet when the subdomain matches what it inherits", () => {
        const zone = makeZone({ services: { _dmarc: [dmarc("v=DMARC1;p=reject")] } });
        const issues = runOnSubdomain("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.subdomain-weakens-parent");
    });
    it("does not compare an apex record with itself", () => {
        const zone = makeZone({ services: { _dmarc: [dmarc("v=DMARC1;p=reject")] } });
        const issues = runWithZone("v=DMARC1;p=none", zone);
        expect(ids(issues)).not.toContain("dmarc.subdomain-weakens-parent");
    });
});

describe("DMARC compliance: alignment", () => {
    it("flags invalid adkim", () => {
        const issues = run("v=DMARC1;p=reject;adkim=x");
        expect(ids(issues)).toContain("dmarc.invalid-alignment");
    });
    it("flags invalid aspf", () => {
        const issues = run("v=DMARC1;p=reject;aspf=loose");
        expect(ids(issues)).toContain("dmarc.invalid-alignment");
    });
    it("accepts strict alignment", () => {
        const issues = run("v=DMARC1;p=reject;adkim=s;aspf=s");
        expect(ids(issues)).not.toContain("dmarc.invalid-alignment");
    });
});

describe("DMARC compliance: pct & ri", () => {
    it("flags out-of-range pct", () => {
        const issues = run("v=DMARC1;p=reject;pct=150");
        expect(ids(issues)).toContain("dmarc.invalid-pct");
    });
    it("flags negative pct", () => {
        const issues = run("v=DMARC1;p=reject;pct=-5");
        expect(ids(issues)).toContain("dmarc.invalid-pct");
    });
    it("infos on partial deployment (pct < 100)", () => {
        const issues = run("v=DMARC1;p=reject;pct=25");
        expect(ids(issues)).toContain("dmarc.partial-deployment");
    });
    it("flags invalid ri", () => {
        const issues = run("v=DMARC1;p=reject;ri=abc");
        expect(ids(issues)).toContain("dmarc.invalid-ri");
    });
    it("flags zero ri", () => {
        const issues = run("v=DMARC1;p=reject;ri=0");
        expect(ids(issues)).toContain("dmarc.invalid-ri");
    });
});

describe("DMARC compliance: fo / rf", () => {
    it("warns on unknown fo", () => {
        const issues = run("v=DMARC1;p=reject;fo=z");
        expect(ids(issues)).toContain("dmarc.invalid-fo");
    });
    it("accepts fo=d:s", () => {
        const issues = run("v=DMARC1;p=reject;fo=d,s");
        expect(ids(issues)).not.toContain("dmarc.invalid-fo");
    });
    it("warns on unknown rf", () => {
        const issues = run("v=DMARC1;p=reject;rf=iodef");
        expect(ids(issues)).toContain("dmarc.unknown-rf");
    });
});

describe("DMARC compliance: rua / ruf", () => {
    it("flags non-mailto/http URI", () => {
        const issues = run("v=DMARC1;p=reject;rua=ftp://example.com");
        expect(ids(issues)).toContain("dmarc.invalid-uri-scheme");
    });
    it("flags malformed mailto", () => {
        const issues = run("v=DMARC1;p=reject;rua=mailto:not-an-email");
        expect(ids(issues)).toContain("dmarc.invalid-mailto");
    });
    it("accepts mailto with !size suffix", () => {
        const issues = run("v=DMARC1;p=reject;rua=mailto:dmarc@example.com!10m");
        expect(ids(issues)).not.toContain("dmarc.invalid-mailto");
    });
});

describe("DMARC compliance: http(s) report URI", () => {
    it("accepts an https URI with a !size suffix", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://reports.example.com/dmarc!10m");
        expect(ids(issues)).toEqual([]);
    });
    it("flags an http URI that does not parse", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://");
        expect(ids(issues)).toContain("dmarc.invalid-http-uri");
    });
    it("warns on a plain http destination", () => {
        const issues = run("v=DMARC1;p=reject;rua=http://reports.example.com/dmarc");
        expect(ids(issues)).toContain("dmarc.report-uri-insecure");
    });
    it("warns on an IPv4 literal host", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://192.0.2.1/dmarc");
        expect(ids(issues)).toContain("dmarc.report-host-ip-literal");
    });
    it("warns on an IPv6 literal host", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://[2001:db8::1]/dmarc");
        expect(ids(issues)).toContain("dmarc.report-host-ip-literal");
    });
    it("flags a host that is not a valid name", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://-reports-.example.com/dmarc");
        expect(ids(issues)).toContain("dmarc.invalid-report-host");
    });
    it("warns on a single-label host", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://localhost/dmarc");
        expect(ids(issues)).toContain("dmarc.report-host-single-label");
    });
    it("checks ruf destinations too", () => {
        const issues = run("v=DMARC1;p=reject;ruf=http://localhost/dmarc");
        expect(ids(issues)).toContain("dmarc.report-uri-insecure");
        expect(ids(issues)).toContain("dmarc.report-host-single-label");
    });
});

describe("DMARC compliance: tags receivers ignore", () => {
    it("says nothing about a record made of known tags", () => {
        const issues = run("v=DMARC1;p=reject;adkim=s;fo=1;ri=3600;pct=100");
        expect(ids(issues)).toEqual([]);
    });
    it("warns on an unknown tag", () => {
        const issues = run("v=DMARC1;p=reject;widget=bar");
        const unknown = issues.find((i) => i.id === "dmarc.unknown-tag");
        expect(unknown?.params).toMatchObject({ tag: "widget" });
    });
    it("suggests the tag a typo was aiming at", () => {
        const issues = run("v=DMARC1;p=reject;adkin=s");
        const unknown = issues.find((i) => i.id === "dmarc.unknown-tag-suggestion");
        expect(unknown?.params).toMatchObject({ tag: "adkin", suggestion: "adkim" });
    });
    it("accepts a known tag spelled in upper case", () => {
        const issues = run("v=DMARC1;p=reject;RUA=mailto:d@example.com");
        expect(ids(issues)).not.toContain("dmarc.unknown-tag");
        expect(ids(issues)).not.toContain("dmarc.unknown-tag-suggestion");
    });
    it("infos on a tag defined by later work on DMARC", () => {
        const issues = run("v=DMARC1;p=reject;np=reject");
        const later = issues.find((i) => i.id === "dmarc.later-tag");
        expect(later?.params).toMatchObject({ tag: "np" });
    });
    it("warns on a repeated tag", () => {
        const issues = run("v=DMARC1;p=reject;p=none");
        expect(ids(issues)).toContain("dmarc.duplicate-tag");
    });
    it("warns on a chunk carrying no value", () => {
        const issues = run("v=DMARC1;p=reject;junk");
        const malformed = issues.find((i) => i.id === "dmarc.malformed-pair");
        expect(malformed?.params).toMatchObject({ pair: "junk" });
    });
    it("warns on a tag left empty", () => {
        const issues = run("v=DMARC1;p=reject;pct=");
        expect(ids(issues)).toContain("dmarc.empty-tag-value");
    });
    it("leaves an empty p= to the missing-policy report", () => {
        const issues = run("v=DMARC1;p=");
        expect(ids(issues)).toContain("dmarc.missing-policy");
        expect(ids(issues)).not.toContain("dmarc.empty-tag-value");
    });
    it("tolerates a trailing semicolon", () => {
        const issues = run("v=DMARC1;p=reject;");
        expect(ids(issues)).toEqual([]);
    });
});

describe("DMARC compliance: cross-checks with DKIM / SPF", () => {
    it("does not flag cross-checks when zone is unknown", () => {
        const issues = run("v=DMARC1;p=reject;adkim=s");
        expect(ids(issues)).not.toContain("dmarc.strict-dkim-no-record");
        expect(ids(issues)).not.toContain("dmarc.no-alignment-source");
        expect(ids(issues)).not.toContain("dmarc.no-alignment-source-enforcing");
    });
    it("flags adkim=s with no DKIM record in the zone", () => {
        const zone = makeZone({ services: { "": [svc("svcs.SPF")] } });
        const issues = runWithZone("v=DMARC1;p=reject;adkim=s", zone);
        expect(ids(issues)).toContain("dmarc.strict-dkim-no-record");
    });
    it("does not flag adkim=s when a DKIM record is present", () => {
        const zone = makeZone({
            services: {
                "": [svc("svcs.SPF")],
                "selector1._domainkey": [svc("svcs.DKIMRecord")],
            },
        });
        const issues = runWithZone("v=DMARC1;p=reject;adkim=s", zone);
        expect(ids(issues)).not.toContain("dmarc.strict-dkim-no-record");
    });
    it("flags an enforcing policy with no DKIM and no SPF", () => {
        const zone = makeZone();
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).toContain("dmarc.no-alignment-source-enforcing");
    });
    it("warns on p=none with no DKIM and no SPF", () => {
        const zone = makeZone();
        const issues = runWithZone("v=DMARC1;p=none", zone);
        expect(ids(issues)).toContain("dmarc.no-alignment-source");
        expect(ids(issues)).not.toContain("dmarc.no-alignment-source-enforcing");
    });
    it("does not flag missing alignment when SPF is present", () => {
        const zone = makeZone({ services: { "": [svc("svcs.SPF")] } });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.no-alignment-source-enforcing");
        expect(ids(issues)).not.toContain("dmarc.no-alignment-source");
    });
    it("warns when the zone relies on SPF only", () => {
        const zone = makeZone({ services: { "": [svc("svcs.SPF")] } });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).toContain("dmarc.no-dkim-record");
    });
    it("does not warn about DKIM when a selector is published", () => {
        const zone = makeZone({
            services: { "": [svc("svcs.SPF"), dkim("v=DKIM1;p=" + KEY)] },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.no-dkim-record");
        expect(ids(issues)).not.toContain("dmarc.all-dkim-revoked");
        expect(ids(issues)).not.toContain("dmarc.all-dkim-testing");
    });
    it("warns when every DKIM selector is revoked", () => {
        const zone = makeZone({
            services: { "": [dkim("v=DKIM1;p="), dkim("v=DKIM1;p=")] },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).toContain("dmarc.all-dkim-revoked");
    });
    it("warns when every DKIM selector is in testing mode", () => {
        const zone = makeZone({
            services: { "": [dkim("v=DKIM1;t=y;p=" + KEY), dkim("v=DKIM1;p=")] },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).toContain("dmarc.all-dkim-testing");
        expect(ids(issues)).not.toContain("dmarc.all-dkim-revoked");
    });
    it("stays quiet when one selector among several is usable", () => {
        const zone = makeZone({
            services: { "": [dkim("v=DKIM1;t=y;p=" + KEY), dkim("v=DKIM1;p=" + KEY)] },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.all-dkim-testing");
        expect(ids(issues)).not.toContain("dmarc.all-dkim-revoked");
    });
});

describe("DMARC compliance: external reporting (sync hint)", () => {
    it("flags a rua to a domain different from the protected one", () => {
        const issues = run("v=DMARC1;p=reject;rua=mailto:dmarc@thirdparty.tld");
        const ext = issues.find((i) => i.id === "dmarc.external-reporting");
        expect(ext).toBeDefined();
        expect(ext?.params).toMatchObject({ domain: "thirdparty.tld" });
    });
    it("does not flag a rua pointing at the protected domain itself", () => {
        const issues = run("v=DMARC1;p=reject;rua=mailto:dmarc@example.com");
        expect(ids(issues)).not.toContain("dmarc.external-reporting");
    });
    it("does not flag a rua pointing at a subdomain of the protected domain", () => {
        // Same Organizational Domain: sec. 7.1 asks for nothing there.
        const issues = run("v=DMARC1;p=reject;rua=mailto:dmarc@reports.example.com");
        expect(ids(issues)).not.toContain("dmarc.external-reporting");
    });
    it("dedupes external destinations", () => {
        const issues = run(
            "v=DMARC1;p=reject;rua=mailto:a@thirdparty.tld;ruf=mailto:b@thirdparty.tld",
        );
        expect(issues.filter((i) => i.id === "dmarc.external-reporting")).toHaveLength(1);
    });
    it("flags an http rua pointing outside the protected domain", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://thirdparty.tld/collect");
        const ext = issues.find((i) => i.id === "dmarc.external-reporting");
        expect(ext?.params).toMatchObject({ domain: "thirdparty.tld" });
    });
    it("does not flag an http rua hosted inside the protected domain", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://reports.example.com/collect");
        expect(ids(issues)).not.toContain("dmarc.external-reporting");
    });
    it("counts a mailto and an http destination on one host only once", () => {
        const issues = run(
            "v=DMARC1;p=reject;rua=mailto:a@thirdparty.tld,https://thirdparty.tld/collect",
        );
        expect(issues.filter((i) => i.id === "dmarc.external-reporting")).toHaveLength(1);
    });
});

describe("DMARC compliance: external reporting authorization (async)", () => {
    beforeEach(() => {
        vi.mocked(checkDMARCReportAuth).mockReset();
    });

    function runAsync(txt: string): Promise<ComplianceIssue[]> {
        const v = getValidators("svcs.DMARC");
        expect(v?.async).toBeDefined();
        return v!.async!(
            { txt: { Hdr: { Name: "_dmarc.example.com." }, Txt: txt } },
            CTX,
            new AbortController().signal,
        );
    }

    it("looks up the authorization of an http destination", async () => {
        vi.mocked(checkDMARCReportAuth).mockResolvedValueOnce({
            status: "not-found",
            queriedName: "example.com._report._dmarc.thirdparty.tld",
        });
        const issues = await runAsync("v=DMARC1;p=reject;rua=https://thirdparty.tld/collect");
        expect(checkDMARCReportAuth).toHaveBeenCalledWith(
            { owner: "example.com", externalDomain: "thirdparty.tld" },
            expect.anything(),
        );
        const missing = issues.find((i) => i.id === "dmarc.report-auth-missing");
        expect(missing?.params).toMatchObject({
            domain: "thirdparty.tld",
            destination: "https://thirdparty.tld/collect",
        });
    });
    it("stays quiet on an authorized destination", async () => {
        vi.mocked(checkDMARCReportAuth).mockResolvedValueOnce({
            status: "ok",
            queriedName: "example.com._report._dmarc.thirdparty.tld",
        });
        expect(await runAsync("v=DMARC1;p=reject;rua=mailto:d@thirdparty.tld")).toEqual([]);
    });
    it("does not look up a destination of the same organizational domain", async () => {
        const issues = await runAsync(
            "v=DMARC1;p=reject;rua=mailto:d@reports.example.com,https://collect.example.com/d",
        );
        expect(checkDMARCReportAuth).not.toHaveBeenCalled();
        expect(issues).toEqual([]);
    });
    it("does not look up an address literal, which has no domain to ask", async () => {
        const issues = await runAsync("v=DMARC1;p=reject;rua=https://192.0.2.1/collect");
        expect(checkDMARCReportAuth).not.toHaveBeenCalled();
        expect(issues).toEqual([]);
    });
    it("warns when the resolver cannot answer", async () => {
        vi.mocked(checkDMARCReportAuth).mockResolvedValueOnce({
            status: "resolver-error",
            queriedName: "example.com._report._dmarc.thirdparty.tld",
            errorMsg: "timeout",
        });
        const issues = await runAsync("v=DMARC1;p=reject;ruf=mailto:d@thirdparty.tld");
        expect(ids(issues)).toContain("dmarc.report-auth-resolver-error");
    });
});

describe("DMARC compliance: graceful empty input", () => {
    it("returns no issue on empty TXT", () => {
        expect(run("")).toEqual([]);
    });
});
