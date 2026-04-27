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
import type { ServiceWithValue } from "$lib/model/service.svelte";
import type { Zone } from "$lib/model/zone";

const ORIGIN = { domain: "example.com." } as unknown as Domain;
const CTX = buildContext("_dmarc", ORIGIN, null);

function run(txt: string, name = "_dmarc.example.com."): ComplianceIssue[] {
    const v = getValidators("svcs.DMARC");
    expect(v?.sync).toBeDefined();
    return v!.sync!({ txt: { Hdr: { Name: name }, Txt: txt } }, CTX);
}

function svc(svctype: string): ServiceWithValue {
    return { _svctype: svctype, Service: {} } as unknown as ServiceWithValue;
}
function makeZone(services: Record<string, ServiceWithValue[]>): Zone {
    return { services } as unknown as Zone;
}
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

describe("DMARC compliance: cross-checks with DKIM / SPF", () => {
    it("does not flag cross-checks when zone is unknown", () => {
        const issues = run("v=DMARC1;p=reject;adkim=s");
        expect(ids(issues)).not.toContain("dmarc.strict-dkim-no-record");
        expect(ids(issues)).not.toContain("dmarc.no-alignment-source");
        expect(ids(issues)).not.toContain("dmarc.no-alignment-source-enforcing");
    });
    it("flags adkim=s with no DKIM record in the zone", () => {
        const zone = makeZone({ "": [svc("svcs.SPF")] });
        const issues = runWithZone("v=DMARC1;p=reject;adkim=s", zone);
        expect(ids(issues)).toContain("dmarc.strict-dkim-no-record");
    });
    it("does not flag adkim=s when a DKIM record is present", () => {
        const zone = makeZone({
            "": [svc("svcs.SPF")],
            "selector1._domainkey": [svc("svcs.DKIMRecord")],
        });
        const issues = runWithZone("v=DMARC1;p=reject;adkim=s", zone);
        expect(ids(issues)).not.toContain("dmarc.strict-dkim-no-record");
    });
    it("flags an enforcing policy with no DKIM and no SPF", () => {
        const zone = makeZone({});
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).toContain("dmarc.no-alignment-source-enforcing");
    });
    it("warns on p=none with no DKIM and no SPF", () => {
        const zone = makeZone({});
        const issues = runWithZone("v=DMARC1;p=none", zone);
        expect(ids(issues)).toContain("dmarc.no-alignment-source");
        expect(ids(issues)).not.toContain("dmarc.no-alignment-source-enforcing");
    });
    it("does not flag missing alignment when SPF is present", () => {
        const zone = makeZone({ "": [svc("svcs.SPF")] });
        const issues = runWithZone("v=DMARC1;p=reject", zone);
        expect(ids(issues)).not.toContain("dmarc.no-alignment-source-enforcing");
        expect(ids(issues)).not.toContain("dmarc.no-alignment-source");
    });
});

describe("DMARC compliance: graceful empty input", () => {
    it("returns no issue on empty TXT", () => {
        expect(run("")).toEqual([]);
    });
});
