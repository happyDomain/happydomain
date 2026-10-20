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

import { describe, it, expect, vi } from "vitest";
import { createHash } from "node:crypto";
import {
    certificateProblems,
    contentName,
    countCertificates,
    decodeBase64,
    encodeBase64,
    generatedRecord,
    logoProblems,
    selectorOf,
    serviceDomain,
} from "./model";

const LOGO = `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>Example</title></svg>`;
const CERT = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n";

describe("base64", () => {
    it("round-trips arbitrary bytes", () => {
        const bytes = new Uint8Array([0, 1, 127, 128, 255]);
        expect(decodeBase64(encodeBase64(bytes))).toEqual(bytes);
    });
    it("rejects what is not base64", () => {
        expect(() => decodeBase64("not base64!")).toThrow();
    });
    it("decodes nothing to nothing", () => {
        expect(decodeBase64(undefined)).toEqual(new Uint8Array());
    });
});

describe("contentName", () => {
    it("matches the backend naming", async () => {
        const bytes = new TextEncoder().encode(LOGO);
        const want = createHash("sha256").update(bytes).digest("hex").slice(0, 16) + ".svg";
        expect(await contentName(bytes, ".svg")).toBe(want);
    });
    it("gives up without WebCrypto, as outside a secure context", async () => {
        vi.stubGlobal("crypto", {});
        try {
            expect(await contentName(new TextEncoder().encode(LOGO), ".svg")).toBeNull();
        } finally {
            vi.unstubAllGlobals();
        }
    });
    it("names nothing for an empty file", async () => {
        expect(await contentName(new Uint8Array(), ".pem")).toBe("");
    });
});

describe("generatedRecord", () => {
    it("matches the backend record, with and without certificate", () => {
        expect(generatedRecord("example.com", "aa.svg", "")).toBe(
            "v=BIMI1;l=https://bimi.example.com/bimi/aa.svg",
        );
        expect(generatedRecord("example.com", "aa.svg", "bb.pem")).toBe(
            "v=BIMI1;l=https://bimi.example.com/bimi/aa.svg;a=https://bimi.example.com/bimi/bb.pem",
        );
    });
});

describe("serviceDomain and selectorOf", () => {
    it("resolves the domain of the service", () => {
        expect(serviceDomain("", "example.com.")).toBe("example.com");
        expect(serviceDomain("sub", "example.com.")).toBe("sub.example.com");
    });
    it("extracts the selector", () => {
        expect(selectorOf("default._bimi")).toBe("default");
        expect(selectorOf(undefined)).toBe("");
        expect(selectorOf("default")).toBe("default");
        expect(selectorOf("")).toBe("");
    });
});

describe("logoProblems", () => {
    it("accepts a tiny-ps logo", () => {
        expect(logoProblems(LOGO, LOGO.length)).toEqual([]);
    });
    it("flags what the backend rejects", () => {
        expect(logoProblems("<html></html>", 13)).toEqual(["logo-not-svg"]);
        expect(logoProblems(`<svg baseProfile="tiny">`, 10)).toEqual(["logo-not-tiny-ps"]);
        expect(logoProblems(LOGO.replace("<title>", "<script>x</script><title>"), 10)).toEqual([
            "logo-active-content",
        ]);
        expect(logoProblems(LOGO.replace("<svg ", `<svg onload="x" `), 10)).toEqual([
            "logo-active-content",
        ]);
        expect(
            logoProblems(LOGO.replace("<title>", `<image href="https://x/y.png"/><title>`), 10),
        ).toEqual(["logo-external-reference"]);
        expect(logoProblems(LOGO.replace("<title>", `<!ENTITY x "y"><title>`), 10)).toEqual([
            "logo-active-content",
        ]);
        expect(logoProblems(LOGO.replace("<title>", `<foreignObject/><title>`), 10)).toEqual([
            "logo-active-content",
        ]);
        expect(
            logoProblems(LOGO.replace("<title>", `<use xlink:href="https://x/y"/><title>`), 10),
        ).toEqual(["logo-external-reference"]);
        expect(logoProblems(LOGO.replace("tiny-ps", "tiny"), 10)).toEqual(["logo-not-tiny-ps"]);
        expect(
            logoProblems(`<svg baseProfile="tiny"><script>x</script></svg>`, 10),
        ).toEqual(["logo-not-tiny-ps", "logo-active-content"]);
    });
    it("limits the size inclusively", () => {
        expect(logoProblems(LOGO, 32 * 1024)).toEqual([]);
        expect(logoProblems(LOGO, 32 * 1024 + 1)).toEqual(["logo-too-large"]);
    });
    it("accepts internal references", () => {
        expect(logoProblems(LOGO.replace("<title>", `<use href="#a"/><title>`), 10)).toEqual([]);
    });
});

describe("certificateProblems", () => {
    it("accepts a certificate chain", () => {
        expect(certificateProblems(CERT + CERT, 100)).toEqual([]);
        expect(countCertificates(CERT + CERT)).toBe(2);
    });
    it("flags what the backend rejects", () => {
        expect(certificateProblems("junk", 4)).toEqual(["certificate-not-pem"]);
        expect(
            certificateProblems(
                CERT + "-----BEGIN PRIVATE KEY-----\nx\n-----END PRIVATE KEY-----\n",
                100,
            ),
        ).toEqual(["certificate-not-only-certificates"]);
        expect(certificateProblems(CERT, 70000)).toEqual(["certificate-too-large"]);
        expect(certificateProblems("", 0)).toEqual(["certificate-not-pem"]);
        expect(
            certificateProblems("-----BEGIN CERTIFICATE REQUEST-----\nx\n-----END CERTIFICATE REQUEST-----", 10),
        ).toEqual(["certificate-not-only-certificates"]);
    });
    it("limits the size inclusively", () => {
        expect(certificateProblems(CERT, 64 * 1024)).toEqual([]);
    });
    it("counts no certificate in an empty file", () => {
        expect(countCertificates("")).toBe(0);
    });
});
