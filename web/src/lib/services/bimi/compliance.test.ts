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
import {
    buildContext,
    getValidators,
    type ComplianceIssue,
} from "$lib/services/compliance";
import type { Domain } from "$lib/model/domain";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import type { Zone } from "$lib/model/zone";

const ORIGIN = { domain: "example.com." } as unknown as Domain;

function dmarcService(txt: string): ServiceWithValue {
    return {
        _svctype: "svcs.DMARC",
        Service: { txt: { Txt: txt, Hdr: { Name: "_dmarc" } } },
    } as unknown as ServiceWithValue;
}

function zoneWith(...services: ServiceWithValue[]): Zone {
    return { services: { "": services } } as unknown as Zone;
}

function run(
    name: string,
    txt: string,
    zone: Zone | null = null,
): ComplianceIssue[] {
    const v = getValidators("svcs.BIMI");
    expect(v?.sync).toBeDefined();
    const ctx = buildContext("", ORIGIN, zone);
    return v!.sync!({ txt: { Hdr: { Name: name }, Txt: txt } }, ctx);
}

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

describe("BIMI compliance: happy paths", () => {
    it("accepts a complete record with VMC and DMARC reject", () => {
        const zone = zoneWith(dmarcService("v=DMARC1; p=reject"));
        const issues = run(
            "default._bimi",
            "v=BIMI1; l=https://example.com/logo.svg; a=https://example.com/vmc.pem",
            zone,
        );
        expect(ids(issues)).toEqual([]);
    });
    it("accepts a minimal record but flags the missing VMC as info", () => {
        const issues = run("default._bimi", "v=BIMI1;l=https://example.com/logo.svg");
        expect(ids(issues)).toContain("bimi.missing-vmc");
    });
});

describe("BIMI compliance: selector", () => {
    it("flags an empty selector", () => {
        const issues = run("_bimi", "v=BIMI1;l=https://example.com/logo.svg");
        expect(ids(issues)).toContain("bimi.missing-selector");
    });
    it("flags an invalid selector", () => {
        const issues = run("bad selector._bimi", "v=BIMI1;l=https://example.com/logo.svg");
        expect(ids(issues)).toContain("bimi.invalid-selector");
    });
    it("flags an owner name that does not end in ._bimi", () => {
        const issues = run("default", "v=BIMI1;l=https://example.com/logo.svg");
        expect(ids(issues)).toContain("bimi.wrong-owner-name");
    });
    it("accepts a custom selector", () => {
        const issues = run("brand._bimi", "v=BIMI1;l=https://example.com/logo.svg");
        expect(ids(issues)).not.toContain("bimi.invalid-selector");
        expect(ids(issues)).not.toContain("bimi.missing-selector");
    });
});

describe("BIMI compliance: version & location", () => {
    it("flags a non-BIMI1 version", () => {
        const issues = run("default._bimi", "v=BIMI2;l=https://example.com/logo.svg");
        expect(ids(issues)).toContain("bimi.invalid-version");
    });
    it("flags a missing location", () => {
        const issues = run("default._bimi", "v=BIMI1");
        expect(ids(issues)).toContain("bimi.missing-location");
    });
    it("flags a non-HTTPS location", () => {
        const issues = run("default._bimi", "v=BIMI1;l=http://example.com/logo.svg");
        expect(ids(issues)).toContain("bimi.location-not-https");
    });
    it("warns on a location not ending in .svg", () => {
        const issues = run("default._bimi", "v=BIMI1;l=https://example.com/logo.png");
        expect(ids(issues)).toContain("bimi.location-not-svg");
    });
    it("accepts a location with query string", () => {
        const issues = run(
            "default._bimi",
            "v=BIMI1;l=https://example.com/logo.svg?v=1",
        );
        expect(ids(issues)).not.toContain("bimi.location-not-svg");
    });
});

describe("BIMI compliance: authority & evidence", () => {
    it("flags a non-HTTPS VMC", () => {
        const issues = run(
            "default._bimi",
            "v=BIMI1;l=https://example.com/logo.svg;a=http://example.com/vmc.pem",
        );
        expect(ids(issues)).toContain("bimi.authority-not-https");
    });
    it("infos when authority does not end in .pem", () => {
        const issues = run(
            "default._bimi",
            "v=BIMI1;l=https://example.com/logo.svg;a=https://example.com/vmc",
        );
        expect(ids(issues)).toContain("bimi.authority-not-pem");
    });
    it("warns on a non-HTTPS evidence URL", () => {
        const issues = run(
            "default._bimi",
            "v=BIMI1;l=https://example.com/logo.svg;a=https://example.com/vmc.pem;e=http://example.com/ev",
        );
        expect(ids(issues)).toContain("bimi.evidence-not-https");
    });
});

describe("BIMI compliance: DMARC cross-checks", () => {
    it("warns when no DMARC is published", () => {
        const zone = zoneWith();
        const issues = run(
            "default._bimi",
            "v=BIMI1;l=https://example.com/logo.svg;a=https://example.com/vmc.pem",
            zone,
        );
        expect(ids(issues)).toContain("bimi.no-dmarc");
    });
    it("warns when DMARC is p=none", () => {
        const zone = zoneWith(dmarcService("v=DMARC1; p=none"));
        const issues = run(
            "default._bimi",
            "v=BIMI1;l=https://example.com/logo.svg;a=https://example.com/vmc.pem",
            zone,
        );
        expect(ids(issues)).toContain("bimi.weak-dmarc-policy");
    });
    it("does not warn when DMARC is p=quarantine", () => {
        const zone = zoneWith(dmarcService("v=DMARC1; p=quarantine"));
        const issues = run(
            "default._bimi",
            "v=BIMI1;l=https://example.com/logo.svg;a=https://example.com/vmc.pem",
            zone,
        );
        expect(ids(issues)).not.toContain("bimi.no-dmarc");
        expect(ids(issues)).not.toContain("bimi.weak-dmarc-policy");
    });
    it("skips cross-checks when zone is null", () => {
        const issues = run("default._bimi", "v=BIMI1;l=https://example.com/logo.svg;a=https://example.com/vmc.pem");
        expect(ids(issues)).not.toContain("bimi.no-dmarc");
        expect(ids(issues)).not.toContain("bimi.weak-dmarc-policy");
    });
});
