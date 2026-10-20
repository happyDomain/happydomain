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

import { describe, it, expect } from "vitest";
import "./compliance";
import { buildContext, getValidators, type ComplianceIssue } from "$lib/services/compliance";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";

const LOGO = btoa(
    `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>Example</title></svg>`,
);

function body(overrides: Record<string, any> = {}): Record<string, any> {
    return {
        logo: LOGO,
        vmc: btoa("-----BEGIN CERTIFICATE-----"),
        txt: { Hdr: { Name: "default._bimi" }, Txt: "" },
        hostCNAME: { Hdr: { Name: "bimi" }, Target: "happydomain.example.com." },
        ...overrides,
    };
}

function run(raw: Record<string, any>, dmarc: string | null = "reject"): string[] {
    const v = getValidators("abstract.BIMI");
    expect(v?.sync).toBeDefined();
    const services = dmarc
        ? [
              makeService("svcs.DMARC", {
                  txt: { Txt: "v=DMARC1; p=" + dmarc, Hdr: { Name: "_dmarc" } },
              }),
          ]
        : [];
    const ctx = buildContext("", makeDomain(), makeZone({ services: { "": services } }));
    return (v!.sync!(raw, ctx) as ComplianceIssue[]).map((i) => i.id);
}

describe("hosted BIMI compliance", () => {
    it("accepts a complete service", () => {
        expect(run(body())).toEqual([]);
    });
    it("flags a missing certificate as info", () => {
        expect(run(body({ vmc: undefined }))).toEqual(["bimi.missing-vmc"]);
    });
    it("flags a missing logo, as after an import", () => {
        expect(run(body({ logo: undefined }))).toEqual(["bimi.hosted-no-logo"]);
    });
    it("flags a logo without title", () => {
        expect(run(body({ logo: btoa(`<svg baseProfile="tiny-ps"></svg>`) }))).toEqual([
            "bimi.hosted-logo-no-title",
        ]);
    });
    it("flags a missing bimi. CNAME", () => {
        expect(run(body({ hostCNAME: undefined }))).toEqual(["bimi.hosted-no-host"]);
    });
    it("flags a bimi. CNAME under another name", () => {
        expect(run(body({ hostCNAME: { Hdr: { Name: "www" }, Target: "x." } }))).toEqual([
            "bimi.hosted-no-host",
        ]);
    });
    it("treats a logo that is not base64 as no logo", () => {
        expect(run(body({ logo: "not base64!" }))).toEqual(["bimi.hosted-no-logo"]);
    });
    it("flags a zone without DMARC", () => {
        expect(run(body(), null)).toEqual(["bimi.no-dmarc"]);
    });
    it("reports several problems at once", () => {
        expect(run(body({ logo: undefined, hostCNAME: undefined, vmc: undefined }))).toEqual([
            "bimi.hosted-no-host",
            "bimi.hosted-no-logo",
            "bimi.missing-vmc",
        ]);
    });
    it("checks the selector and the DMARC policy", () => {
        expect(run(body({ txt: { Hdr: { Name: "a.b._bimi" }, Txt: "" } }), "none")).toEqual([
            "bimi.invalid-selector",
            "bimi.weak-dmarc-policy",
        ]);
    });
});
