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
import {
    parseTLSRPT,
    stringifyTLSRPT,
    TLSRPTPolicy,
    TLSRPTRecord
} from "./tlsrpt.svelte";
import { newRR, getRrtype, type dnsTypeTXT } from "$lib/dns_rr";

function createTxtRecord(txt: string): dnsTypeTXT {
    const rr = newRR("_smtp._tls.example.com", getRrtype("TXT")) as dnsTypeTXT;
    rr.Txt = txt;
    return rr;
}

describe("parseTLSRPT", () => {
    it("should parse basic TLS-RPT record with version and single RUA", () => {
        const result = parseTLSRPT("v=TLSRPTv1;rua=mailto:reports@example.com");

        expect(result).toEqual({
            v: "TLSRPTv1",
            rua: ["mailto:reports@example.com"],
        });
    });

    it("should parse TLS-RPT record with multiple RUA addresses", () => {
        const result = parseTLSRPT("v=TLSRPTv1;rua=mailto:reports@example.com,https://example.com/reports");

        expect(result).toEqual({
            v: "TLSRPTv1",
            rua: ["mailto:reports@example.com", "https://example.com/reports"],
        });
    });

    it("should parse TLS-RPT record with spaces after semicolons", () => {
        const result = parseTLSRPT("v=TLSRPTv1; rua=mailto:reports@example.com");

        expect(result).toEqual({
            v: "TLSRPTv1",
            rua: ["mailto:reports@example.com"],
        });
    });

    it("should handle TLS-RPT record with version only", () => {
        const result = parseTLSRPT("v=TLSRPTv1");

        expect(result).toEqual({
            v: "TLSRPTv1",
            rua: [],
        });
    });

    it("should handle empty RUA field", () => {
        const result = parseTLSRPT("v=TLSRPTv1;rua=");

        expect(result).toEqual({
            v: "TLSRPTv1",
            rua: [],
        });
    });

    it("should handle missing version field", () => {
        const result = parseTLSRPT("rua=mailto:reports@example.com");

        expect(result).toEqual({
            v: undefined,
            rua: ["mailto:reports@example.com"],
        });
    });

    it("should handle empty string", () => {
        const result = parseTLSRPT("");

        expect(result).toEqual({
            v: undefined,
            rua: [],
        });
    });
});

describe("stringifyTLSRPT", () => {
    it("should stringify TLS-RPT record with version and single RUA", () => {
        const record = new TLSRPTRecord("TLSRPTv1", ["mailto:reports@example.com"]);

        const result = stringifyTLSRPT(record);

        expect(result).toBe("v=TLSRPTv1;rua=mailto:reports@example.com");
    });

    it("should stringify TLS-RPT record with multiple RUA addresses", () => {
        const record = new TLSRPTRecord("TLSRPTv1", ["mailto:reports@example.com", "https://example.com/reports"]);

        const result = stringifyTLSRPT(record);

        expect(result).toBe("v=TLSRPTv1;rua=mailto:reports@example.com,https://example.com/reports");
    });

    it("should use semicolon with space when existing value has spaces", () => {
        const record = new TLSRPTRecord("TLSRPTv1", ["mailto:reports@example.com"]);

        const result = stringifyTLSRPT(record, "v=TLSRPTv1; rua=mailto:old@example.com");

        expect(result).toBe("v=TLSRPTv1; rua=mailto:reports@example.com");
    });

    it("should default version to TLSRPTv1 when not provided", () => {
        const record = new TLSRPTRecord(undefined, ["mailto:reports@example.com"]);

        const result = stringifyTLSRPT(record);

        expect(result).toBe("v=TLSRPTv1;rua=mailto:reports@example.com");
    });

    it("should omit RUA when empty", () => {
        const record = new TLSRPTRecord("TLSRPTv1", []);

        const result = stringifyTLSRPT(record);

        expect(result).toBe("v=TLSRPTv1");
    });

    it("should handle undefined RUA", () => {
        const record = new TLSRPTRecord("TLSRPTv1", []);

        const result = stringifyTLSRPT(record);

        expect(result).toBe("v=TLSRPTv1");
    });

    it("should use custom version when provided", () => {
        const record = new TLSRPTRecord("TLSRPTv2", ["mailto:reports@example.com"]);

        const result = stringifyTLSRPT(record);

        expect(result).toBe("v=TLSRPTv2;rua=mailto:reports@example.com");
    });
});

describe("TLSRPTPolicy", () => {
    describe("constructor", () => {
        it("should initialize from TXT record", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            expect(policy.v).toBe("TLSRPTv1");
            expect(policy.rua).toEqual(["mailto:reports@example.com"]);
        });

        it("should initialize from TXT record with multiple RUA addresses", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com,https://example.com/reports");
            const policy = new TLSRPTPolicy(txtRecord);

            expect(policy.v).toBe("TLSRPTv1");
            expect(policy.rua).toEqual(["mailto:reports@example.com", "https://example.com/reports"]);
        });

        it("should handle empty TXT record", () => {
            const txtRecord = createTxtRecord("");
            const policy = new TLSRPTPolicy(txtRecord);

            expect(policy.v).toBeUndefined();
            expect(policy.rua).toEqual([]);
        });
    });

    describe("version getter and setter", () => {
        it("should get version from parsed value", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            expect(policy.v).toBe("TLSRPTv1");
        });

        it("should update version and TXT record", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.v = "TLSRPTv2";

            expect(policy.v).toBe("TLSRPTv2");
            expect(txtRecord.Txt).toBe("v=TLSRPTv2;rua=mailto:reports@example.com");
        });

        it("should handle setting version to undefined", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.v = undefined;

            expect(policy.v).toBeUndefined();
            expect(txtRecord.Txt).toBe("v=TLSRPTv1;rua=mailto:reports@example.com"); // Defaults to TLSRPTv1
        });
    });

    describe("rua getter and setter", () => {
        it("should get RUA array from parsed value", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com,https://example.com/reports");
            const policy = new TLSRPTPolicy(txtRecord);

            expect(policy.rua).toEqual(["mailto:reports@example.com", "https://example.com/reports"]);
        });

        it("should update RUA array and TXT record", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.rua = ["mailto:new@example.com"];

            expect(policy.rua).toEqual(["mailto:new@example.com"]);
            expect(txtRecord.Txt).toBe("v=TLSRPTv1;rua=mailto:new@example.com");
        });

        it("should handle setting empty RUA array", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.rua = [];

            expect(policy.rua).toEqual([]);
            expect(txtRecord.Txt).toBe("v=TLSRPTv1");
        });
    });

    describe("addRua", () => {
        it("should add RUA address to empty list", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.addRua("mailto:reports@example.com");

            expect(policy.rua).toEqual(["mailto:reports@example.com"]);
            expect(txtRecord.Txt).toBe("v=TLSRPTv1;rua=mailto:reports@example.com");
        });

        it("should add RUA address to existing list", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.addRua("https://example.com/reports");

            expect(policy.rua).toEqual(["mailto:reports@example.com", "https://example.com/reports"]);
            expect(txtRecord.Txt).toBe("v=TLSRPTv1;rua=mailto:reports@example.com,https://example.com/reports");
        });

        it("should handle adding empty string", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.addRua("");

            expect(policy.rua).toEqual([""]);
            expect(txtRecord.Txt).toContain("rua=");
        });
    });

    describe("removeRua", () => {
        it("should remove RUA address by index", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com,https://example.com/reports");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.removeRua(0);

            expect(policy.rua).toEqual(["https://example.com/reports"]);
            expect(txtRecord.Txt).toBe("v=TLSRPTv1;rua=https://example.com/reports");
        });

        it("should remove last RUA address", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com,https://example.com/reports");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.removeRua(1);

            expect(policy.rua).toEqual(["mailto:reports@example.com"]);
            expect(txtRecord.Txt).toBe("v=TLSRPTv1;rua=mailto:reports@example.com");
        });

        it("should remove only RUA address", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.removeRua(0);

            expect(policy.rua).toEqual([]);
            expect(txtRecord.Txt).toBe("v=TLSRPTv1");
        });

        it("should handle removing from middle of list", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:a@example.com,mailto:b@example.com,mailto:c@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.removeRua(1);

            expect(policy.rua).toEqual(["mailto:a@example.com", "mailto:c@example.com"]);
            expect(txtRecord.Txt).toBe("v=TLSRPTv1;rua=mailto:a@example.com,mailto:c@example.com");
        });
    });

    describe("updateRua", () => {
        it("should update RUA address by index", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:old@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.updateRua(0, "mailto:new@example.com");

            expect(policy.rua).toEqual(["mailto:new@example.com"]);
            expect(txtRecord.Txt).toBe("v=TLSRPTv1;rua=mailto:new@example.com");
        });

        it("should update RUA address in middle of list", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:a@example.com,mailto:b@example.com,mailto:c@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.updateRua(1, "mailto:updated@example.com");

            expect(policy.rua).toEqual(["mailto:a@example.com", "mailto:updated@example.com", "mailto:c@example.com"]);
            expect(txtRecord.Txt).toBe("v=TLSRPTv1;rua=mailto:a@example.com,mailto:updated@example.com,mailto:c@example.com");
        });

        it("should handle updating to empty string", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.updateRua(0, "");

            expect(policy.rua).toEqual([""]);
        });
    });

    describe("TXT record updates", () => {
        it("should preserve separator style from original", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1; rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.addRua("https://example.com/reports");

            expect(txtRecord.Txt).toContain("; rua=");
        });

        it("should use semicolon without space by default", () => {
            const txtRecord = createTxtRecord("v=TLSRPTv1;rua=mailto:reports@example.com");
            const policy = new TLSRPTPolicy(txtRecord);

            policy.addRua("https://example.com/reports");

            expect(txtRecord.Txt).toBe("v=TLSRPTv1;rua=mailto:reports@example.com,https://example.com/reports");
        });
    });
});

describe("TLS-RPT parsing roundtrip tests", () => {
    it("should maintain data through parse and stringify", () => {
        const original = "v=TLSRPTv1;rua=mailto:reports@example.com,https://example.com/reports";
        const parsed = parseTLSRPT(original);
        const stringified = stringifyTLSRPT(parsed, original);

        expect(stringified).toBe(original);
    });

    it("should maintain data with spaces through parse and stringify", () => {
        const original = "v=TLSRPTv1; rua=mailto:reports@example.com";
        const parsed = parseTLSRPT(original);
        const stringified = stringifyTLSRPT(parsed, original);

        expect(stringified).toBe(original);
    });

    it("should maintain version-only record through parse and stringify", () => {
        const original = "v=TLSRPTv1";
        const parsed = parseTLSRPT(original);
        const stringified = stringifyTLSRPT(parsed, original);

        expect(stringified).toBe(original);
    });

    it("should maintain single RUA through parse and stringify", () => {
        const original = "v=TLSRPTv1;rua=mailto:reports@example.com";
        const parsed = parseTLSRPT(original);
        const stringified = stringifyTLSRPT(parsed, original);

        expect(stringified).toBe(original);
    });
});
