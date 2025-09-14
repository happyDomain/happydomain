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
import { parseMTASTS, stringifyMTASTS, type MTASTSValue } from "./mta_sts";

describe("parseMTASTS", () => {
    it("should parse a minimal MTA-STS record", () => {
        const result = parseMTASTS("v=STSv1;id=20160831085700Z");
        expect(result).toEqual({
            v: "STSv1",
            id: "20160831085700Z",
        });
    });

    it("should parse an MTA-STS record with spaces after semicolons", () => {
        const result = parseMTASTS("v=STSv1; id=20160831085700Z");
        expect(result).toEqual({
            v: "STSv1",
            id: "20160831085700Z",
        });
    });

    it("should parse MTA-STS record with only version", () => {
        const result = parseMTASTS("v=STSv1");
        expect(result).toEqual({
            v: "STSv1",
            id: undefined,
        });
    });

    it("should parse MTA-STS record with custom policy ID", () => {
        const result = parseMTASTS("v=STSv1;id=custom-policy-123");
        expect(result).toEqual({
            v: "STSv1",
            id: "custom-policy-123",
        });
    });

    it("should handle missing optional id field", () => {
        const result = parseMTASTS("v=STSv1");
        expect(result.id).toBeUndefined();
    });

    it("should handle empty string", () => {
        const result = parseMTASTS("");
        expect(result).toEqual({
            v: undefined,
            id: undefined,
        });
    });

    it("should parse MTA-STS record with numeric-like id", () => {
        const result = parseMTASTS("v=STSv1;id=1234567890");
        expect(result).toEqual({
            v: "STSv1",
            id: "1234567890",
        });
    });

    it("should parse MTA-STS record with timestamp format id", () => {
        const result = parseMTASTS("v=STSv1;id=20231215T123045Z");
        expect(result).toEqual({
            v: "STSv1",
            id: "20231215T123045Z",
        });
    });
});

describe("stringifyMTASTS", () => {
    it("should stringify a minimal MTA-STS record with semicolons", () => {
        const mtasts: MTASTSValue = {
            v: "STSv1",
            id: "20160831085700Z",
        };
        const result = stringifyMTASTS(mtasts);
        expect(result).toBe("v=STSv1;id=20160831085700Z");
    });

    it("should stringify a minimal MTA-STS record with spaces after semicolons", () => {
        const mtasts: MTASTSValue = {
            v: "STSv1",
            id: "20160831085700Z",
        };
        const result = stringifyMTASTS(mtasts, "v=STSv1; id=20160831085700Z");
        expect(result).toBe("v=STSv1; id=20160831085700Z");
    });

    it("should use default version STSv1 when v is not provided", () => {
        const mtasts: MTASTSValue = {
            id: "20160831085700Z",
        };
        const result = stringifyMTASTS(mtasts);
        expect(result).toBe("v=STSv1;id=20160831085700Z");
    });

    it("should respect custom version when provided", () => {
        const mtasts: MTASTSValue = {
            v: "STSv2",
            id: "20160831085700Z",
        };
        const result = stringifyMTASTS(mtasts);
        expect(result).toBe("v=STSv2;id=20160831085700Z");
    });

    it("should omit id when not provided", () => {
        const mtasts: MTASTSValue = {
            v: "STSv1",
        };
        const result = stringifyMTASTS(mtasts);
        expect(result).toBe("v=STSv1");
        expect(result).not.toContain("id=");
    });

    it("should handle empty MTASTSValue object", () => {
        const mtasts: MTASTSValue = {};
        const result = stringifyMTASTS(mtasts);
        expect(result).toBe("v=STSv1");
    });

    it("should handle custom policy ID", () => {
        const mtasts: MTASTSValue = {
            v: "STSv1",
            id: "custom-policy-123",
        };
        const result = stringifyMTASTS(mtasts);
        expect(result).toBe("v=STSv1;id=custom-policy-123");
    });

    it("should handle numeric-like id", () => {
        const mtasts: MTASTSValue = {
            v: "STSv1",
            id: "1234567890",
        };
        const result = stringifyMTASTS(mtasts);
        expect(result).toBe("v=STSv1;id=1234567890");
    });

    it("should handle timestamp format id", () => {
        const mtasts: MTASTSValue = {
            v: "STSv1",
            id: "20231215T123045Z",
        };
        const result = stringifyMTASTS(mtasts);
        expect(result).toBe("v=STSv1;id=20231215T123045Z");
    });
});

describe("parseMTASTS and stringifyMTASTS roundtrip", () => {
    it("should maintain data through parse and stringify cycle", () => {
        const original = "v=STSv1;id=20160831085700Z";
        const parsed = parseMTASTS(original);
        const stringified = stringifyMTASTS(parsed, original);

        expect(stringified).toBe(original);
    });

    it("should maintain data through parse and stringify cycle with custom id", () => {
        const original = "v=STSv1;id=custom-policy-456";
        const parsed = parseMTASTS(original);
        const stringified = stringifyMTASTS(parsed, original);

        expect(stringified).toBe(original);
    });

    it("should maintain separator style through roundtrip", () => {
        const withSpaces = "v=STSv1; id=20160831085700Z";
        const parsed = parseMTASTS(withSpaces);
        const stringified = stringifyMTASTS(parsed, withSpaces);

        expect(stringified).toContain("; ");
        expect(stringified).toBe(withSpaces);
    });

    it("should maintain separator style without spaces through roundtrip", () => {
        const withoutSpaces = "v=STSv1;id=20160831085700Z";
        const parsed = parseMTASTS(withoutSpaces);
        const stringified = stringifyMTASTS(parsed, withoutSpaces);

        expect(stringified).not.toContain("; ");
        expect(stringified).toContain(";id=");
        expect(stringified).toBe(withoutSpaces);
    });

    it("should handle roundtrip with only version", () => {
        const original = "v=STSv1";
        const parsed = parseMTASTS(original);
        const stringified = stringifyMTASTS(parsed, original);

        const parsedRoundtrip = parseMTASTS(stringified);
        expect(parsedRoundtrip.v).toBe("STSv1");
        expect(parsedRoundtrip.id).toBeUndefined();
    });

    it("should maintain custom version through roundtrip", () => {
        const original = "v=STSv2;id=20231215T123045Z";
        const parsed = parseMTASTS(original);
        const stringified = stringifyMTASTS(parsed, original);

        const parsedRoundtrip = parseMTASTS(stringified);
        expect(parsedRoundtrip.v).toBe("STSv2");
        expect(parsedRoundtrip.id).toBe("20231215T123045Z");
    });
});
