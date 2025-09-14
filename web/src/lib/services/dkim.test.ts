// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2025 happyDomain
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
import { parseDKIM, stringifyDKIM, type DKIMValue } from "./dkim.svelte";

describe("parseDKIM", () => {
    it("should parse a minimal DKIM record", () => {
        const result = parseDKIM("v=DKIM1;p=MIGfMA0GCSqGSIb3DQEBAQUAA");
        expect(result).toEqual({
            v: "DKIM1",
            p: "MIGfMA0GCSqGSIb3DQEBAQUAA",
            h: [],
            s: [],
            t: [],
            f: [],
        });
    });

    it("should parse a DKIM record with all fields", () => {
        const result = parseDKIM(
            "v=DKIM1;g=*;h=sha256:sha512;k=rsa;n=test note;p=MIGfMA0GCS;s=email:*;t=y:s;f=s"
        );
        expect(result).toEqual({
            v: "DKIM1",
            g: "*",
            h: ["sha256", "sha512"],
            k: "rsa",
            n: "test note",
            p: "MIGfMA0GCS",
            s: ["email", "*"],
            t: ["y", "s"],
            f: ["s"],
        });
    });

    it("should parse DKIM record with spaces after semicolons", () => {
        const result = parseDKIM("v=DKIM1; k=rsa; p=MIGfMA0GCS");
        expect(result).toEqual({
            v: "DKIM1",
            k: "rsa",
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
            f: [],
        });
    });

    it("should parse hash algorithms field", () => {
        const result = parseDKIM("v=DKIM1;h=sha1:sha256;p=MIGfMA0GCS");
        expect(result.h).toEqual(["sha1", "sha256"]);
    });

    it("should parse single hash algorithm", () => {
        const result = parseDKIM("v=DKIM1;h=sha256;p=MIGfMA0GCS");
        expect(result.h).toEqual(["sha256"]);
    });

    it("should parse service types field", () => {
        const result = parseDKIM("v=DKIM1;s=email:*;p=MIGfMA0GCS");
        expect(result.s).toEqual(["email", "*"]);
    });

    it("should parse single service type", () => {
        const result = parseDKIM("v=DKIM1;s=email;p=MIGfMA0GCS");
        expect(result.s).toEqual(["email"]);
    });

    it("should parse flags field", () => {
        const result = parseDKIM("v=DKIM1;t=y:s;p=MIGfMA0GCS");
        expect(result.t).toEqual(["y", "s"]);
    });

    it("should parse single flag", () => {
        const result = parseDKIM("v=DKIM1;t=y;p=MIGfMA0GCS");
        expect(result.t).toEqual(["y"]);
    });

    it("should parse f field", () => {
        const result = parseDKIM("v=DKIM1;f=s;p=MIGfMA0GCS");
        expect(result.f).toEqual(["s"]);
    });

    it("should parse multiple f values", () => {
        const result = parseDKIM("v=DKIM1;f=s:t;p=MIGfMA0GCS");
        expect(result.f).toEqual(["s", "t"]);
    });

    it("should handle empty arrays when fields are missing", () => {
        const result = parseDKIM("v=DKIM1;p=MIGfMA0GCS");
        expect(result.h).toEqual([]);
        expect(result.s).toEqual([]);
        expect(result.t).toEqual([]);
        expect(result.f).toEqual([]);
    });

    it("should handle empty array values", () => {
        const result = parseDKIM("v=DKIM1;h=;s=;t=;f=;p=MIGfMA0GCS");
        expect(result.h).toEqual([]);
        expect(result.s).toEqual([]);
        expect(result.t).toEqual([]);
        expect(result.f).toEqual([]);
    });

    it("should parse key type", () => {
        const result = parseDKIM("v=DKIM1;k=ed25519;p=MIGfMA0GCS");
        expect(result.k).toBe("ed25519");
    });

    it("should parse granularity field", () => {
        const result = parseDKIM("v=DKIM1;g=user@example.com;p=MIGfMA0GCS");
        expect(result.g).toBe("user@example.com");
    });

    it("should parse notes field", () => {
        const result = parseDKIM("v=DKIM1;n=This is a test key;p=MIGfMA0GCS");
        expect(result.n).toBe("This is a test key");
    });

    it("should handle missing optional fields", () => {
        const result = parseDKIM("v=DKIM1;p=MIGfMA0GCS");
        expect(result.g).toBeUndefined();
        expect(result.k).toBeUndefined();
        expect(result.n).toBeUndefined();
    });

    it("should parse real-world DKIM record", () => {
        const result = parseDKIM(
            "v=DKIM1;k=rsa;p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAviPGBk4ZB64UfSqWyAicdR7lodhytae+EYRQVtKDhM+1mXjEqRtP/pDT3sBhazkmA48n2k5NJUyMEoO8nc2r6sUA+/Dom5jRBZp6qDKJOwjJ5R/OpHamlRG+YRJQqRtqEgSiJWG7h7efGYWmh4FAgDPYVqtDPU0B4s3S8sQ8qNbhPQJ62qhQgBkGULRxFSQqyxK5OZfCTMNiWS+5EqLi0JWUjCpXkdBZgYt/PABMDPMGcP91PmJhNrEO7K+Vgmq+6gAJQ0JYJKfxNRPH3L9LKL0gVcGmCGcYEgQRPaQtpvNmUUZ7aRFLVrUhFqWGrPQGkQgRHNCGt6vCL1s5gQIDAQAB"
        );
        expect(result.v).toBe("DKIM1");
        expect(result.k).toBe("rsa");
        expect(result.p).toBe(
            "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAviPGBk4ZB64UfSqWyAicdR7lodhytae+EYRQVtKDhM+1mXjEqRtP/pDT3sBhazkmA48n2k5NJUyMEoO8nc2r6sUA+/Dom5jRBZp6qDKJOwjJ5R/OpHamlRG+YRJQqRtqEgSiJWG7h7efGYWmh4FAgDPYVqtDPU0B4s3S8sQ8qNbhPQJ62qhQgBkGULRxFSQqyxK5OZfCTMNiWS+5EqLi0JWUjCpXkdBZgYt/PABMDPMGcP91PmJhNrEO7K+Vgmq+6gAJQ0JYJKfxNRPH3L9LKL0gVcGmCGcYEgQRPaQtpvNmUUZ7aRFLVrUhFqWGrPQGkQgRHNCGt6vCL1s5gQIDAQAB"
        );
    });
});

describe("stringifyDKIM", () => {
    it("should stringify a minimal DKIM record with semicolons", () => {
        const dkim: DKIMValue = {
            p: "MIGfMA0GCSqGSIb3DQEBAQUAA",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toBe("v=DKIM1;p=MIGfMA0GCSqGSIb3DQEBAQUAA");
    });

    it("should stringify a minimal DKIM record with spaces after semicolons", () => {
        const dkim: DKIMValue = {
            p: "MIGfMA0GCSqGSIb3DQEBAQUAA",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim, "v=DKIM1; p=MIGfMA0GCSqGSIb3DQEBAQUAA");
        expect(result).toBe("v=DKIM1; p=MIGfMA0GCSqGSIb3DQEBAQUAA");
    });

    it("should stringify a DKIM record with all fields", () => {
        const dkim: DKIMValue = {
            v: "DKIM1",
            g: "*",
            h: ["sha256", "sha512"],
            k: "rsa",
            n: "test note",
            p: "MIGfMA0GCS",
            s: ["email", "*"],
            t: ["y", "s"],
            f: ["s"],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toBe(
            "v=DKIM1;g=*;h=sha256:sha512;k=rsa;n=test note;p=MIGfMA0GCS;s=email:*;t=y:s;f=s"
        );
    });

    it("should use default version DKIM1 when v is not provided", () => {
        const dkim: DKIMValue = {
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toMatch(/^v=DKIM1/);
    });

    it("should respect custom version when provided", () => {
        const dkim: DKIMValue = {
            v: "DKIM2",
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toMatch(/^v=DKIM2/);
    });

    it("should stringify hash algorithms", () => {
        const dkim: DKIMValue = {
            h: ["sha1", "sha256", "sha512"],
            p: "MIGfMA0GCS",
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toContain("h=sha1:sha256:sha512");
    });

    it("should stringify single hash algorithm", () => {
        const dkim: DKIMValue = {
            h: ["sha256"],
            p: "MIGfMA0GCS",
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toContain("h=sha256");
    });

    it("should stringify service types", () => {
        const dkim: DKIMValue = {
            s: ["email", "*"],
            p: "MIGfMA0GCS",
            h: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toContain("s=email:*");
    });

    it("should stringify flags", () => {
        const dkim: DKIMValue = {
            t: ["y", "s"],
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toContain("t=y:s");
    });

    it("should stringify f field", () => {
        const dkim: DKIMValue = {
            f: ["s", "t"],
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toContain("f=s:t");
    });

    it("should omit empty arrays", () => {
        const dkim: DKIMValue = {
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).not.toContain("h=");
        expect(result).not.toContain("s=");
        expect(result).not.toContain("t=");
        expect(result).not.toContain("f=");
    });

    it("should omit undefined optional fields", () => {
        const dkim: DKIMValue = {
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).not.toContain("g=");
        expect(result).not.toContain("k=");
        expect(result).not.toContain("n=");
    });

    it("should include optional fields when provided", () => {
        const dkim: DKIMValue = {
            g: "*",
            k: "ed25519",
            n: "Test key",
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toContain("g=*");
        expect(result).toContain("k=ed25519");
        expect(result).toContain("n=Test key");
    });

    it("should stringify key type rsa", () => {
        const dkim: DKIMValue = {
            k: "rsa",
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toContain("k=rsa");
    });

    it("should stringify key type ed25519", () => {
        const dkim: DKIMValue = {
            k: "ed25519",
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toContain("k=ed25519");
    });

    it("should stringify granularity field", () => {
        const dkim: DKIMValue = {
            g: "user@example.com",
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toContain("g=user@example.com");
    });

    it("should stringify notes field", () => {
        const dkim: DKIMValue = {
            n: "Production DKIM key",
            p: "MIGfMA0GCS",
            h: [],
            s: [],
            t: [],
            f: [],
        };
        const result = stringifyDKIM(dkim);
        expect(result).toContain("n=Production DKIM key");
    });

    it("should maintain field order consistently", () => {
        const dkim: DKIMValue = {
            v: "DKIM1",
            g: "*",
            h: ["sha256"],
            k: "rsa",
            n: "note",
            p: "key",
            s: ["email"],
            t: ["y"],
            f: ["s"],
        };
        const result = stringifyDKIM(dkim);
        // Field order: v, g, h, k, n, p, s, t, f
        expect(result).toBe("v=DKIM1;g=*;h=sha256;k=rsa;n=note;p=key;s=email;t=y;f=s");
    });
});

describe("parseDKIM and stringifyDKIM roundtrip", () => {
    it("should maintain data through parse and stringify cycle", () => {
        const original = "v=DKIM1;g=*;h=sha256:sha512;k=rsa;n=test;p=MIGfMA0GCS;s=email;t=y;f=s";
        const parsed = parseDKIM(original);
        const stringified = stringifyDKIM(parsed, original);

        // Parse both to compare, since field order might differ
        const parsedOriginal = parseDKIM(original);
        const parsedRoundtrip = parseDKIM(stringified);

        expect(parsedRoundtrip).toEqual(parsedOriginal);
    });

    it("should maintain minimal record through roundtrip", () => {
        const original = "v=DKIM1;p=MIGfMA0GCSqGSIb3DQEBAQUAA";
        const parsed = parseDKIM(original);
        const stringified = stringifyDKIM(parsed, original);

        const parsedRoundtrip = parseDKIM(stringified);
        expect(parsedRoundtrip.v).toBe("DKIM1");
        expect(parsedRoundtrip.p).toBe("MIGfMA0GCSqGSIb3DQEBAQUAA");
    });

    it("should maintain array fields through roundtrip", () => {
        const original = "v=DKIM1;h=sha1:sha256;s=email:*;t=y:s;f=s:t;p=key";
        const parsed = parseDKIM(original);
        const stringified = stringifyDKIM(parsed, original);

        const parsedRoundtrip = parseDKIM(stringified);
        expect(parsedRoundtrip.h).toEqual(["sha1", "sha256"]);
        expect(parsedRoundtrip.s).toEqual(["email", "*"]);
        expect(parsedRoundtrip.t).toEqual(["y", "s"]);
        expect(parsedRoundtrip.f).toEqual(["s", "t"]);
    });

    it("should maintain separator style through roundtrip", () => {
        const withSpaces = "v=DKIM1; k=rsa; p=MIGfMA0GCS";
        const parsed = parseDKIM(withSpaces);
        const stringified = stringifyDKIM(parsed, withSpaces);

        expect(stringified).toContain("; ");
    });

    it("should maintain separator style without spaces through roundtrip", () => {
        const withoutSpaces = "v=DKIM1;k=rsa;p=MIGfMA0GCS";
        const parsed = parseDKIM(withoutSpaces);
        const stringified = stringifyDKIM(parsed, withoutSpaces);

        expect(stringified).not.toContain("; ");
        expect(stringified).toContain(";k=");
    });

    it("should handle real-world DKIM record roundtrip", () => {
        const original =
            "v=DKIM1;k=rsa;p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAviPGBk4ZB64UfSqWyAicdR7lodhytae+EYRQVtKDhM+1mXjEqRtP/pDT3sBhazkmA48n2k5NJUyMEoO8nc2r6sUA+/Dom5jRBZp6qDKJOwjJ5R/OpHamlRG+YRJQqRtqEgSiJWG7h7efGYWmh4FAgDPYVqtDPU0B4s3S8sQ8qNbhPQJ62qhQgBkGULRxFSQqyxK5OZfCTMNiWS+5EqLi0JWUjCpXkdBZgYt/PABMDPMGcP91PmJhNrEO7K+Vgmq+6gAJQ0JYJKfxNRPH3L9LKL0gVcGmCGcYEgQRPaQtpvNmUUZ7aRFLVrUhFqWGrPQGkQgRHNCGt6vCL1s5gQIDAQAB";
        const parsed = parseDKIM(original);
        const stringified = stringifyDKIM(parsed, original);

        const parsedRoundtrip = parseDKIM(stringified);
        expect(parsedRoundtrip.v).toBe("DKIM1");
        expect(parsedRoundtrip.k).toBe("rsa");
        expect(parsedRoundtrip.p).toBe(parsed.p);
    });

    it("should handle empty arrays through roundtrip", () => {
        const original = "v=DKIM1;p=key";
        const parsed = parseDKIM(original);
        const stringified = stringifyDKIM(parsed, original);

        const parsedRoundtrip = parseDKIM(stringified);
        expect(parsedRoundtrip.h).toEqual([]);
        expect(parsedRoundtrip.s).toEqual([]);
        expect(parsedRoundtrip.t).toEqual([]);
        expect(parsedRoundtrip.f).toEqual([]);
    });

    it("should preserve all optional fields through roundtrip", () => {
        const original = "v=DKIM1;g=user@example.com;h=sha256;k=ed25519;n=My Key;p=key123;s=email;t=y;f=s";
        const parsed = parseDKIM(original);
        const stringified = stringifyDKIM(parsed, original);

        const parsedRoundtrip = parseDKIM(stringified);
        expect(parsedRoundtrip.g).toBe("user@example.com");
        expect(parsedRoundtrip.h).toEqual(["sha256"]);
        expect(parsedRoundtrip.k).toBe("ed25519");
        expect(parsedRoundtrip.n).toBe("My Key");
        expect(parsedRoundtrip.p).toBe("key123");
        expect(parsedRoundtrip.s).toEqual(["email"]);
        expect(parsedRoundtrip.t).toEqual(["y"]);
        expect(parsedRoundtrip.f).toEqual(["s"]);
    });
});
