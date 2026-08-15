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
    it("counts every other record on the name", () => {
        const zone = makeZone({
            services: {
                _dmarc: [at("svcs.DMARC", "v=DMARC1;p=none"), at("svcs.DMARC", "v=DMARC1;p=none")],
            },
        });
        const dup = runWithZone("v=DMARC1;p=reject", zone).find(
            (i) => i.id === "dmarc.duplicate-record",
        );
        expect(dup?.params).toMatchObject({ count: 3 });
    });
    it("recognises a raw TXT whatever the case or the leading blanks", () => {
        for (const value of ["V=DMARC1; p=none", "  v=dmarc1;p=none"]) {
            const zone = makeZone({ services: { _dmarc: [at("svcs.TXT", value)] } });
            expect(ids(runWithZone("v=DMARC1;p=reject", zone))).toContain("dmarc.duplicate-record");
        }
    });
    it("does not take a longer version tag for a DMARC record", () => {
        const zone = makeZone({ services: { _dmarc: [at("svcs.TXT", "v=DMARC10;p=none")] } });
        expect(ids(runWithZone("v=DMARC1;p=reject", zone))).not.toContain("dmarc.duplicate-record");
    });
    it("compares owner names regardless of case and trailing dot", () => {
        const zone = makeZone({
            services: { _dmarc: [at("svcs.DMARC", "v=DMARC1;p=none", "_DMARC.Example.COM")] },
        });
        expect(ids(runWithZone("v=DMARC1;p=reject", zone))).toContain("dmarc.duplicate-record");
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
    it("skips the edited service by id, whatever it holds in the editor", () => {
        const saved = makeService(
            "svcs.DMARC",
            { txt: { Hdr: { Name: "_dmarc.example.com." }, Txt: "v=DMARC1;p=none" } },
            { _id: "svc1" },
        );
        const zone = makeZone({ services: { _dmarc: [saved] } });
        const v = getValidators("svcs.DMARC");
        const edited = { txt: { Hdr: { Name: "_dmarc.example.com." }, Txt: "v=DMARC1;p=reject" } };
        const issues = v!.sync!(edited, buildContext("_dmarc", ORIGIN, zone, "svc1"));
        expect(ids(issues)).not.toContain("dmarc.duplicate-record");
    });
});

describe("DMARC compliance: subdomain policy inheritance", () => {
    it("treats an empty dn (generator) as the apex", () => {
        const v = getValidators("svcs.DMARC");
        const issues = v!.sync!(
            { txt: { Hdr: { Name: "" }, Txt: "v=DMARC1;p=reject;sp=reject" } },
            buildContext("", ORIGIN, null),
        );
        expect(ids(issues)).not.toContain("dmarc.sp-on-subdomain");
    });
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
    it("ranks quarantine between none and reject", () => {
        expect(ids(run("v=DMARC1;p=reject;sp=quarantine"))).toContain("dmarc.sp-weaker-than-p");
        expect(ids(run("v=DMARC1;p=quarantine;sp=none"))).toContain("dmarc.sp-weaker-than-p");
        expect(ids(run("v=DMARC1;p=quarantine;sp=reject"))).not.toContain("dmarc.sp-weaker-than-p");
    });
    it("warns when a subdomain record is a notch below the apex", () => {
        const zone = makeZone({ services: { _dmarc: [dmarc("v=DMARC1;p=reject")] } });
        const issues = runOnSubdomain("v=DMARC1;p=quarantine", zone);
        expect(ids(issues)).toContain("dmarc.subdomain-weakens-parent");
    });
    it("accepts a subdomain record stronger than the apex", () => {
        const zone = makeZone({ services: { _dmarc: [dmarc("v=DMARC1;p=none")] } });
        const issues = runOnSubdomain("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.subdomain-weakens-parent");
    });
    it("reports a weak sp and a weak subdomain record side by side", () => {
        const zone = makeZone({ services: { _dmarc: [dmarc("v=DMARC1;p=reject")] } });
        const issues = runOnSubdomain("v=DMARC1;p=quarantine;sp=none", zone);
        expect(ids(issues)).toEqual(
            expect.arrayContaining([
                "dmarc.sp-weaker-than-p",
                "dmarc.sp-on-subdomain",
                "dmarc.subdomain-weakens-parent",
            ]),
        );
    });
    it("falls back on the apex whatever the depth of the subdomain", () => {
        const zone = makeZone({ services: { _dmarc: [dmarc("v=DMARC1;p=reject")] } });
        const v = getValidators("svcs.DMARC");
        const issues = v!.sync!(
            { txt: { Hdr: { Name: "_dmarc.a.sub.example.com." }, Txt: "v=DMARC1;p=none" } },
            buildContext("_dmarc.a.sub", ORIGIN, zone),
        );
        expect(ids(issues)).toContain("dmarc.subdomain-weakens-parent");
    });
    it("says nothing when the apex record is missing or unreadable", () => {
        for (const services of [
            {},
            { _dmarc: [dmarc("")] },
            { _dmarc: [dmarc("v=DMARC1;p=bogus")] },
        ]) {
            const issues = runOnSubdomain("v=DMARC1;p=none", makeZone({ services }));
            expect(ids(issues)).not.toContain("dmarc.subdomain-weakens-parent");
        }
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
    it("accepts a !size suffix in any unit and case", () => {
        for (const size of ["!10m", "!10M", "!5k", "!2g", "!1T", "!10"]) {
            const issues = run(`v=DMARC1;p=reject;rua=https://reports.example.com/dmarc${size}`);
            expect(ids(issues)).toEqual([]);
        }
    });
    it("keeps checking a plain http URI that carries a !size suffix", () => {
        const issues = run("v=DMARC1;p=reject;rua=http://reports.example.com/dmarc!10m");
        expect(ids(issues)).toEqual(["dmarc.report-uri-insecure"]);
    });
    it("reports nothing else about a plain http URI with a valid host", () => {
        const issues = run("v=DMARC1;p=reject;rua=http://reports.example.com/dmarc");
        expect(ids(issues)).toEqual(["dmarc.report-uri-insecure"]);
    });
    it("reports an insecure address literal once for each problem", () => {
        const issues = run("v=DMARC1;p=reject;rua=http://192.0.2.1/dmarc");
        expect(ids(issues)).toEqual(
            expect.arrayContaining(["dmarc.report-uri-insecure", "dmarc.report-host-ip-literal"]),
        );
    });
    it("flags a dotted quad out of range as a URI that does not parse", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://256.1.1.1/dmarc");
        expect(ids(issues)).toContain("dmarc.invalid-http-uri");
    });
    it("flags an unterminated IPv6 literal as a URI that does not parse", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://[::1/dmarc");
        expect(ids(issues)).toContain("dmarc.invalid-http-uri");
    });
    it("reads the host past a user name and a port", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://user:pw@reports.example.com:8443/dmarc");
        expect(ids(issues)).toEqual([]);
    });
    it("lower cases the host and accepts a trailing dot", () => {
        expect(ids(run("v=DMARC1;p=reject;rua=https://Reports.EXAMPLE.com/dmarc"))).toEqual([]);
        expect(ids(run("v=DMARC1;p=reject;rua=https://reports.example.com./dmarc"))).toEqual([]);
    });
    it("accepts a punycode host", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://xn--rapports-d1a.example.com/dmarc");
        expect(ids(issues)).toEqual([]);
    });
    it("still turns away a scheme that is neither mailto nor http(s)", () => {
        const issues = run("v=DMARC1;p=reject;rua=ftp://reports.example.com/dmarc");
        expect(ids(issues)).toContain("dmarc.invalid-uri-scheme");
    });
    it("checks ruf destinations too", () => {
        const issues = run("v=DMARC1;p=reject;ruf=http://localhost/dmarc");
        expect(ids(issues)).toContain("dmarc.report-uri-insecure");
        expect(ids(issues)).toContain("dmarc.report-host-single-label");
    });
});

describe("DMARC compliance: cross-checks with DKIM / SPF", () => {
    it("counts a CNAME delegated DKIM selector as a DKIM record", () => {
        const zone = makeZone({
            services: {
                "": [svc("svcs.SPF")],
                "selector1._domainkey": [svc("svcs.Alias")],
            },
        });
        const issues = runWithZone("v=DMARC1;p=reject;adkim=s", zone);
        expect(ids(issues)).not.toContain("dmarc.no-dkim-record");
        expect(ids(issues)).not.toContain("dmarc.strict-dkim-no-record");
    });
    it("does not call DKIM revoked when a delegated selector sits next to a revoked one", () => {
        const zone = makeZone({
            services: {
                "": [svc("svcs.SPF")],
                "old._domainkey": [dkim("v=DKIM1;p=", "old._domainkey")],
                "selector1._domainkey": [svc("svcs.Alias")],
            },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.all-dkim-revoked");
    });
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
        const issue = issues.find((i) => i.id === "dmarc.no-alignment-source-enforcing");
        expect(issue?.params).toEqual({ policy: "reject" });
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
    it("warns about SPF only whether or not the policy is enforcing", () => {
        const zone = makeZone({ services: { "": [svc("svcs.SPF")] } });
        for (const policy of ["none", "quarantine", "reject"]) {
            const issues = runWithZone(`v=DMARC1;p=${policy}`, zone);
            expect(ids(issues)).toContain("dmarc.no-dkim-record");
        }
    });
    it("reports strict DKIM alignment too when the zone relies on SPF only", () => {
        const zone = makeZone({ services: { "": [svc("svcs.SPF")] } });
        const issues = runWithZone("v=DMARC1;p=reject;adkim=s", zone);
        expect(ids(issues)).toContain("dmarc.no-dkim-record");
        expect(ids(issues)).toContain("dmarc.strict-dkim-no-record");
    });
    it("does not add the SPF only warning when nothing aligns at all", () => {
        const zone = makeZone({ services: { "": [] } });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.no-dkim-record");
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
    it("leaves a selector whose record cannot be read out of the verdict", () => {
        const zone = makeZone({
            services: { "": [dkim("v=DKIM1;p="), dkim(""), dkim("not a dkim record")] },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).toContain("dmarc.all-dkim-revoked");
    });
    it("stays quiet when no selector can be read at all", () => {
        const zone = makeZone({ services: { "": [dkim(""), dkim("   ")] } });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.all-dkim-revoked");
        expect(ids(issues)).not.toContain("dmarc.all-dkim-testing");
    });
    it("takes t=y among several flags for testing mode", () => {
        const zone = makeZone({ services: { "": [dkim("v=DKIM1;t=y:s;p=" + KEY)] } });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).toContain("dmarc.all-dkim-testing");
    });
    it("counts the selectors published under a subdomain", () => {
        const zone = makeZone({
            services: { "": [svc("svcs.SPF")], mail: [dkim("v=DKIM1;p=")] },
        });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).toContain("dmarc.all-dkim-revoked");
    });
    it("stays quiet when only the s flag is set", () => {
        const zone = makeZone({ services: { "": [dkim("v=DKIM1;t=s;p=" + KEY)] } });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.all-dkim-testing");
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
    it("does not take a name merely ending like the protected domain for a subdomain", () => {
        const issues = run("v=DMARC1;p=reject;rua=mailto:dmarc@notexample.com");
        expect(ids(issues)).toContain("dmarc.external-reporting");
    });
    it("still flags a parent of the protected domain, which may be a public suffix", () => {
        const v = getValidators("svcs.DMARC");
        const issues = v!.sync!(
            {
                txt: {
                    Hdr: { Name: "_dmarc.sub" },
                    Txt: "v=DMARC1;p=reject;rua=mailto:d@example.com",
                },
            },
            buildContext("_dmarc.sub", ORIGIN, null),
        );
        expect(ids(issues)).toContain("dmarc.external-reporting");
    });
    it("compares the destination without regard to case", () => {
        const issues = run("v=DMARC1;p=reject;rua=mailto:dmarc@Reports.EXAMPLE.com");
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
    it("flags an address literal rua as nothing to authorize", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://192.0.2.1/collect");
        expect(ids(issues)).not.toContain("dmarc.external-reporting");
    });
    it("tags the external reporting issue with the tag that referenced it", () => {
        const issues = run("v=DMARC1;p=reject;ruf=https://thirdparty.tld/collect");
        expect(issues.find((i) => i.id === "dmarc.external-reporting")?.field).toBe("ruf");
    });
    it("lower cases the host of an http destination", () => {
        const issues = run("v=DMARC1;p=reject;rua=https://ThirdParty.TLD/collect");
        const ext = issues.find((i) => i.id === "dmarc.external-reporting");
        expect(ext?.params).toMatchObject({ domain: "thirdparty.tld" });
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
    it("looks up a destination only once, quoting the first URI that named it", async () => {
        vi.mocked(checkDMARCReportAuth).mockResolvedValue({
            status: "not-found",
            queriedName: "example.com._report._dmarc.thirdparty.tld",
        });
        const issues = await runAsync(
            "v=DMARC1;p=reject;rua=mailto:a@thirdparty.tld,https://thirdparty.tld/c!10m;ruf=mailto:b@thirdparty.tld",
        );
        expect(checkDMARCReportAuth).toHaveBeenCalledTimes(1);
        expect(issues).toHaveLength(1);
        expect(issues[0].params).toMatchObject({ destination: "a@thirdparty.tld", tag: "rua" });
    });
    it("quotes an http destination without its !size suffix", async () => {
        vi.mocked(checkDMARCReportAuth).mockResolvedValueOnce({
            status: "not-found",
            queriedName: "example.com._report._dmarc.thirdparty.tld",
        });
        const issues = await runAsync("v=DMARC1;p=reject;rua=https://thirdparty.tld/c!10m");
        expect(issues[0].params).toMatchObject({ destination: "https://thirdparty.tld/c" });
    });
    it("reports a domain with no DMARC record at all", async () => {
        vi.mocked(checkDMARCReportAuth).mockResolvedValueOnce({
            status: "no-dmarc-record",
            queriedName: "_dmarc.thirdparty.tld",
        });
        const issues = await runAsync("v=DMARC1;p=reject;ruf=mailto:d@thirdparty.tld");
        expect(ids(issues)).toContain("dmarc.report-auth-no-dmarc");
        expect(issues[0].field).toBe("ruf");
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
