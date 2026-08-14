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

import { describe, expect, it } from "vitest";

import { ID_RE } from "$lib/services/mta_sts";
import { DEFAULT_MAX_AGE, newPolicyId, policyFingerprint, renderPolicy } from "./model";

describe("renderPolicy", () => {
    // The rendering must stay byte-identical to (*abstract.MTASTS).PolicyFile,
    // or the compliance check would report a mismatch against our own server.
    it("renders the RFC 8461 key/value lines with CRLF", () => {
        expect(
            renderPolicy({
                mode: "enforce",
                maxAge: 604800,
                mx: ["mail.example.com.", "  ", "*.example.net"],
            }),
        ).toBe(
            "version: STSv1\r\n" +
                "mode: enforce\r\n" +
                "mx: mail.example.com\r\n" +
                "mx: *.example.net\r\n" +
                "max_age: 604800\r\n",
        );
    });

    it("falls back to the same defaults as the backend", () => {
        expect(renderPolicy({ mx: ["mail.example.com"] })).toBe(
            "version: STSv1\r\nmode: testing\r\nmx: mail.example.com\r\nmax_age: " +
                DEFAULT_MAX_AGE +
                "\r\n",
        );
    });

    it("renders a policy with no MX at all", () => {
        expect(renderPolicy({ mode: "none", maxAge: 86400 })).toBe(
            "version: STSv1\r\nmode: none\r\nmax_age: 86400\r\n",
        );
    });

    // (*abstract.MTASTS).PolicyFile refuses to serve testing/enforce with no
    // authorized MX and returns nil instead: telling senders no MX is
    // authorized would lock mail out of the domain. The preview must show the
    // same "nothing served" outcome rather than a policy that will 404.
    it("renders nothing for testing/enforce with no MX, like the backend", () => {
        expect(renderPolicy({ mode: "testing", maxAge: 604800, mx: [] })).toBe("");
        expect(renderPolicy({ mode: "enforce", maxAge: 604800, mx: ["  "] })).toBe("");
    });
});

describe("newPolicyId", () => {
    it("is a valid RFC 8461 id", () => {
        expect(newPolicyId(new Date("2026-08-14T10:15:00.000Z"))).toBe("20260814T101500Z");
        expect(ID_RE.test(newPolicyId(new Date("2026-08-14T10:15:00.000Z")))).toBe(true);
    });

    it("changes when the policy is updated later", () => {
        const before = newPolicyId(new Date("2026-08-14T10:15:00.000Z"));
        const after = newPolicyId(new Date("2026-08-14T10:15:01.000Z"));
        expect(after).not.toBe(before);
    });
});

describe("policyFingerprint", () => {
    it("ignores whitespace-only MX entries and formatting", () => {
        expect(policyFingerprint({ mode: "enforce", maxAge: 60, mx: [" a ", "", "  "] })).toBe(
            policyFingerprint({ mode: "enforce", maxAge: 60, mx: ["a"] }),
        );
    });

    it("changes when any published field changes", () => {
        const base = policyFingerprint({ mode: "testing", maxAge: 60, mx: ["a"] });
        expect(policyFingerprint({ mode: "enforce", maxAge: 60, mx: ["a"] })).not.toBe(base);
        expect(policyFingerprint({ mode: "testing", maxAge: 61, mx: ["a"] })).not.toBe(base);
        expect(policyFingerprint({ mode: "testing", maxAge: 60, mx: ["b"] })).not.toBe(base);
    });

    // The record fields are not part of the policy file, so touching them must
    // not bump the policy id.
    it("ignores the records themselves", () => {
        const withRecords = policyFingerprint({
            mode: "testing",
            maxAge: 60,
            mx: ["a"],
            txt: { Hdr: { Name: "_mta-sts" }, Txt: "v=STSv1; id=x" } as never,
        });
        expect(withRecords).toBe(policyFingerprint({ mode: "testing", maxAge: 60, mx: ["a"] }));
    });
});
