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
import { parseDMARC, stringifyDMARC, type DMARCValue } from "./dmarc";

describe("parseDMARC", () => {
    it("should parse a minimal DMARC record", () => {
        const result = parseDMARC("v=DMARC1;p=none");
        expect(result).toEqual({
            v: "DMARC1",
            p: "none",
            sp: undefined,
            adkim: undefined,
            aspf: undefined,
            ri: undefined,
            pct: undefined,
            rua: [],
            ruf: [],
            fo: [],
            rf: [],
        });
    });

    it("should parse a DMARC record with all fields", () => {
        const result = parseDMARC(
            "v=DMARC1;p=quarantine;sp=reject;adkim=s;aspf=r;ri=86400;pct=100;rua=mailto:dmarc@example.com;ruf=mailto:forensic@example.com;fo=1;rf=afrf"
        );
        expect(result).toEqual({
            v: "DMARC1",
            p: "quarantine",
            sp: "reject",
            adkim: "s",
            aspf: "r",
            ri: "86400",
            pct: "100",
            rua: ["mailto:dmarc@example.com"],
            ruf: ["mailto:forensic@example.com"],
            fo: ["1"],
            rf: ["afrf"],
        });
    });

    it("should parse DMARC record with spaces after semicolons", () => {
        const result = parseDMARC("v=DMARC1; p=reject; sp=quarantine");
        expect(result).toEqual({
            v: "DMARC1",
            p: "reject",
            sp: "quarantine",
            adkim: undefined,
            aspf: undefined,
            ri: undefined,
            pct: undefined,
            rua: [],
            ruf: [],
            fo: [],
            rf: [],
        });
    });

    it("should parse multiple RUA addresses", () => {
        const result = parseDMARC(
            "v=DMARC1;p=none;rua=mailto:dmarc1@example.com,mailto:dmarc2@example.com,mailto:dmarc3@example.com"
        );
        expect(result.rua).toEqual([
            "mailto:dmarc1@example.com",
            "mailto:dmarc2@example.com",
            "mailto:dmarc3@example.com",
        ]);
    });

    it("should parse multiple RUF addresses", () => {
        const result = parseDMARC(
            "v=DMARC1;p=none;ruf=mailto:forensic1@example.com,mailto:forensic2@example.com"
        );
        expect(result.ruf).toEqual([
            "mailto:forensic1@example.com",
            "mailto:forensic2@example.com",
        ]);
    });

    it("should parse multiple failure reporting options", () => {
        const result = parseDMARC("v=DMARC1;p=none;fo=0,1,d,s");
        expect(result.fo).toEqual(["0", "1", "d", "s"]);
    });

    it("should parse multiple report formats", () => {
        const result = parseDMARC("v=DMARC1;p=none;rf=afrf,iodef");
        expect(result.rf).toEqual(["afrf", "iodef"]);
    });

    it("should handle empty string arrays", () => {
        const result = parseDMARC("v=DMARC1;p=none;rua=;ruf=;fo=;rf=");
        expect(result.rua).toEqual([]);
        expect(result.ruf).toEqual([]);
        expect(result.fo).toEqual([]);
        expect(result.rf).toEqual([]);
    });

    it("should handle missing optional fields", () => {
        const result = parseDMARC("v=DMARC1;p=reject");
        expect(result.sp).toBeUndefined();
        expect(result.adkim).toBeUndefined();
        expect(result.aspf).toBeUndefined();
        expect(result.ri).toBeUndefined();
        expect(result.pct).toBeUndefined();
    });

    it("should parse DMARC record with strict alignment modes", () => {
        const result = parseDMARC("v=DMARC1;p=reject;adkim=s;aspf=s");
        expect(result.adkim).toBe("s");
        expect(result.aspf).toBe("s");
    });

    it("should parse DMARC record with relaxed alignment modes", () => {
        const result = parseDMARC("v=DMARC1;p=reject;adkim=r;aspf=r");
        expect(result.adkim).toBe("r");
        expect(result.aspf).toBe("r");
    });

    it("should parse percentage value", () => {
        const result = parseDMARC("v=DMARC1;p=quarantine;pct=50");
        expect(result.pct).toBe("50");
    });
});

describe("stringifyDMARC", () => {
    it("should stringify a minimal DMARC record with semicolons", () => {
        const dmarc: DMARCValue = {
            p: "none",
            rua: [],
            ruf: [],
            fo: [],
            rf: [],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).toBe("v=DMARCv1;p=none");
    });

    it("should stringify a minimal DMARC record with spaces after semicolons", () => {
        const dmarc: DMARCValue = {
            p: "none",
            rua: [],
            ruf: [],
            fo: [],
            rf: [],
        };
        const result = stringifyDMARC(dmarc, "v=DMARC1; p=none");
        expect(result).toBe("v=DMARCv1; p=none");
    });

    it("should stringify a DMARC record with all fields", () => {
        const dmarc: DMARCValue = {
            v: "DMARC1",
            p: "quarantine",
            sp: "reject",
            adkim: "s",
            aspf: "r",
            ri: "86400",
            pct: "100",
            rua: ["mailto:dmarc@example.com"],
            ruf: ["mailto:forensic@example.com"],
            fo: ["1"],
            rf: ["afrf"],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).toBe(
            "v=DMARC1;p=quarantine;sp=reject;adkim=s;aspf=r;fo=1;rf=afrf;ri=86400;rua=mailto:dmarc@example.com;ruf=mailto:forensic@example.com;pct=100"
        );
    });

    it("should stringify multiple RUA addresses", () => {
        const dmarc: DMARCValue = {
            p: "none",
            rua: [
                "mailto:dmarc1@example.com",
                "mailto:dmarc2@example.com",
                "mailto:dmarc3@example.com",
            ],
            ruf: [],
            fo: [],
            rf: [],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).toContain(
            "rua=mailto:dmarc1@example.com,mailto:dmarc2@example.com,mailto:dmarc3@example.com"
        );
    });

    it("should stringify multiple RUF addresses", () => {
        const dmarc: DMARCValue = {
            p: "none",
            rua: [],
            ruf: ["mailto:forensic1@example.com", "mailto:forensic2@example.com"],
            fo: [],
            rf: [],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).toContain("ruf=mailto:forensic1@example.com,mailto:forensic2@example.com");
    });

    it("should stringify multiple failure reporting options", () => {
        const dmarc: DMARCValue = {
            p: "none",
            rua: [],
            ruf: [],
            fo: ["0", "1", "d", "s"],
            rf: [],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).toContain("fo=0,1,d,s");
    });

    it("should stringify multiple report formats", () => {
        const dmarc: DMARCValue = {
            p: "none",
            rua: [],
            ruf: [],
            fo: [],
            rf: ["afrf", "iodef"],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).toContain("rf=afrf,iodef");
    });

    it("should omit empty arrays", () => {
        const dmarc: DMARCValue = {
            p: "reject",
            rua: [],
            ruf: [],
            fo: [],
            rf: [],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).not.toContain("rua=");
        expect(result).not.toContain("ruf=");
        expect(result).not.toContain("fo=");
        expect(result).not.toContain("rf=");
    });

    it("should omit undefined optional fields", () => {
        const dmarc: DMARCValue = {
            p: "quarantine",
            rua: [],
            ruf: [],
            fo: [],
            rf: [],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).not.toContain("sp=");
        expect(result).not.toContain("adkim=");
        expect(result).not.toContain("aspf=");
        expect(result).not.toContain("ri=");
        expect(result).not.toContain("pct=");
    });

    it("should use default version DMARCv1 when v is not provided", () => {
        const dmarc: DMARCValue = {
            p: "none",
            rua: [],
            ruf: [],
            fo: [],
            rf: [],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).toMatch(/^v=DMARCv1/);
    });

    it("should respect custom version when provided", () => {
        const dmarc: DMARCValue = {
            v: "DMARC1",
            p: "none",
            rua: [],
            ruf: [],
            fo: [],
            rf: [],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).toMatch(/^v=DMARC1/);
    });

    it("should handle numeric pct value", () => {
        const dmarc: DMARCValue = {
            p: "quarantine",
            pct: 75,
            rua: [],
            ruf: [],
            fo: [],
            rf: [],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).toContain("pct=75");
    });

    it("should handle string pct value", () => {
        const dmarc: DMARCValue = {
            p: "quarantine",
            pct: "50",
            rua: [],
            ruf: [],
            fo: [],
            rf: [],
        };
        const result = stringifyDMARC(dmarc);
        expect(result).toContain("pct=50");
    });
});

describe("parseDMARC and stringifyDMARC roundtrip", () => {
    it("should maintain data through parse and stringify cycle", () => {
        const original =
            "v=DMARC1;p=quarantine;sp=reject;adkim=s;aspf=r;fo=1;rf=afrf;ri=86400;rua=mailto:dmarc@example.com;ruf=mailto:forensic@example.com;pct=100";
        const parsed = parseDMARC(original);
        const stringified = stringifyDMARC(parsed, original);

        // Parse both to compare, since field order might differ
        const parsedOriginal = parseDMARC(original);
        const parsedRoundtrip = parseDMARC(stringified);

        expect(parsedRoundtrip).toEqual(parsedOriginal);
    });

    it("should maintain multiple addresses through roundtrip", () => {
        const original =
            "v=DMARC1;p=none;rua=mailto:a@example.com,mailto:b@example.com;ruf=mailto:c@example.com,mailto:d@example.com";
        const parsed = parseDMARC(original);
        const stringified = stringifyDMARC(parsed, original);

        const parsedRoundtrip = parseDMARC(stringified);
        expect(parsedRoundtrip.rua).toEqual(["mailto:a@example.com", "mailto:b@example.com"]);
        expect(parsedRoundtrip.ruf).toEqual(["mailto:c@example.com", "mailto:d@example.com"]);
    });

    it("should maintain separator style through roundtrip", () => {
        const withSpaces = "v=DMARC1; p=reject; sp=quarantine";
        const parsed = parseDMARC(withSpaces);
        const stringified = stringifyDMARC(parsed, withSpaces);

        expect(stringified).toContain("; ");
    });

    it("should maintain separator style without spaces through roundtrip", () => {
        const withoutSpaces = "v=DMARC1;p=reject;sp=quarantine";
        const parsed = parseDMARC(withoutSpaces);
        const stringified = stringifyDMARC(parsed, withoutSpaces);

        expect(stringified).not.toContain("; ");
        expect(stringified).toContain(";p=");
    });
});
