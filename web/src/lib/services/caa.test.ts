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
    CAAPolicy,
    newCAARecord,
    parseCAAIssuer,
    stringifyCAAIssuer,
    parseCAAParameter,
    stringifyCAAParameter,
    parseCAAIodef,
    stringifyCAAIodef,
    type CAAIssuer,
    type CAAParameter,
    type CAAIodef
} from "./caa.svelte";
import type { dnsResource, dnsTypeCAA } from "$lib/dns_rr";

describe("newCAARecord", () => {
    it("should create a CAA record with the specified tag and value", () => {
        const record = newCAARecord("example.com", "issue", "letsencrypt.org");

        expect(record.Tag).toBe("issue");
        expect(record.Value).toBe("letsencrypt.org");
        expect(record.Hdr.Name).toBe("example.com");
    });

    it("should create a CAA record with issuewild tag", () => {
        const record = newCAARecord("example.com", "issuewild", "comodoca.com");

        expect(record.Tag).toBe("issuewild");
        expect(record.Value).toBe("comodoca.com");
    });

    it("should create a CAA record with issuemail tag", () => {
        const record = newCAARecord("example.com", "issuemail", "sectigo.com");

        expect(record.Tag).toBe("issuemail");
        expect(record.Value).toBe("sectigo.com");
    });

    it("should create a CAA record with iodef tag", () => {
        const record = newCAARecord("example.com", "iodef", "mailto:security@example.com");

        expect(record.Tag).toBe("iodef");
        expect(record.Value).toBe("mailto:security@example.com");
    });

    it("should create a disallow record with semicolon value", () => {
        const record = newCAARecord("example.com", "issue", ";");

        expect(record.Tag).toBe("issue");
        expect(record.Value).toBe(";");
    });
});

describe("CAAPolicy", () => {
    describe("constructor", () => {
        it("should initialize with empty records when no CAA records exist", () => {
            const resource: dnsResource = {};
            const policy = new CAAPolicy(resource);

            expect(policy.records).toEqual([]);
            expect(policy.DisallowIssue).toBe(false);
            expect(policy.DisallowWildcardIssue).toBe(false);
            expect(policy.DisallowMailIssue).toBe(false);
        });

        it("should initialize with existing CAA records", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", "letsencrypt.org"),
                newCAARecord("example.com", "issuewild", "comodoca.com"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            expect(policy.records).toEqual(caaRecords);
            expect(policy.records).toHaveLength(2);
        });

        it("should detect disallow issue from existing records", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", ";"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            expect(policy.DisallowIssue).toBe(true);
            expect(policy.DisallowWildcardIssue).toBe(false);
            expect(policy.DisallowMailIssue).toBe(false);
        });

        it("should detect multiple disallow flags", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", ";"),
                newCAARecord("example.com", "issuewild", ";"),
                newCAARecord("example.com", "issuemail", ";"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            expect(policy.DisallowIssue).toBe(true);
            expect(policy.DisallowWildcardIssue).toBe(true);
            expect(policy.DisallowMailIssue).toBe(true);
        });
    });

    describe("hasDisallowIssue", () => {
        it("should return true when a disallow record exists for the tag", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", ";"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            expect(policy.hasDisallowIssue("issue")).toBe(true);
        });

        it("should return false when no disallow record exists for the tag", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", "letsencrypt.org"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            expect(policy.hasDisallowIssue("issue")).toBe(false);
        });

        it("should return false when checking different tag", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", ";"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            expect(policy.hasDisallowIssue("issuewild")).toBe(false);
        });

        it("should handle semicolon with whitespace", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", " ; "),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            expect(policy.hasDisallowIssue("issue")).toBe(true);
        });
    });

    describe("refreshDisallowIssue", () => {
        it("should update all disallow flags based on current records", () => {
            const resource: dnsResource = { caa: [] };
            const policy = new CAAPolicy(resource);

            // Initially all false
            expect(policy.DisallowIssue).toBe(false);
            expect(policy.DisallowWildcardIssue).toBe(false);
            expect(policy.DisallowMailIssue).toBe(false);

            // Add disallow records
            policy.records.push(newCAARecord("example.com", "issue", ";"));
            policy.records.push(newCAARecord("example.com", "issuewild", ";"));
            policy.refreshDisallowIssue();

            expect(policy.DisallowIssue).toBe(true);
            expect(policy.DisallowWildcardIssue).toBe(true);
            expect(policy.DisallowMailIssue).toBe(false);
        });

        it("should update flags when records are removed", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", ";"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            expect(policy.DisallowIssue).toBe(true);

            // Remove the disallow record
            policy.records = [];
            policy.refreshDisallowIssue();

            expect(policy.DisallowIssue).toBe(false);
        });
    });

    describe("changeDisallowIssue", () => {
        it("should add a disallow record when checked", () => {
            const resource: dnsResource = { caa: [] };
            const policy = new CAAPolicy(resource);

            const handler = policy.changeDisallowIssue("example.com", "issue");
            const event = new Event("change");
            Object.defineProperty(event, "target", {
                value: { checked: true },
                writable: false
            });

            handler(event);

            expect(policy.records).toHaveLength(1);
            expect(policy.records[0].Tag).toBe("issue");
            expect(policy.records[0].Value).toBe(";");
            expect(policy.DisallowIssue).toBe(true);
        });

        it("should remove disallow records when unchecked", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", ";"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            expect(policy.DisallowIssue).toBe(true);

            const handler = policy.changeDisallowIssue("example.com", "issue");
            const event = new Event("change");
            Object.defineProperty(event, "target", {
                value: { checked: false },
                writable: false
            });

            handler(event);

            expect(policy.records).toHaveLength(0);
            expect(policy.DisallowIssue).toBe(false);
        });

        it("should remove all disallow records for the tag when unchecked", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", ";"),
                newCAARecord("example.com", "issue", "letsencrypt.org"),
                newCAARecord("example.com", "issue", ";"), // Duplicate disallow
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            const handler = policy.changeDisallowIssue("example.com", "issue");
            const event = new Event("change");
            Object.defineProperty(event, "target", {
                value: { checked: false },
                writable: false
            });

            handler(event);

            // Should remove both disallow records but keep the issuer
            expect(policy.records).toHaveLength(1);
            expect(policy.records[0].Value).toBe("letsencrypt.org");
        });

        it("should not affect records of different tags", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", ";"),
                newCAARecord("example.com", "issuewild", ";"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            const handler = policy.changeDisallowIssue("example.com", "issue");
            const event = new Event("change");
            Object.defineProperty(event, "target", {
                value: { checked: false },
                writable: false
            });

            handler(event);

            expect(policy.records).toHaveLength(1);
            expect(policy.records[0].Tag).toBe("issuewild");
            expect(policy.DisallowWildcardIssue).toBe(true);
        });
    });

    describe("getRecordsByTag", () => {
        it("should return records matching the specified tag", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", "letsencrypt.org"),
                newCAARecord("example.com", "issue", "comodoca.com"),
                newCAARecord("example.com", "issuewild", "digicert.com"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            const issueRecords = policy.getRecordsByTag("issue");

            expect(issueRecords).toHaveLength(2);
            expect(issueRecords[0].Value).toBe("letsencrypt.org");
            expect(issueRecords[1].Value).toBe("comodoca.com");
        });

        it("should return empty array when no records match", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", "letsencrypt.org"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            const issuemailRecords = policy.getRecordsByTag("issuemail");

            expect(issuemailRecords).toEqual([]);
        });
    });

    describe("removeRecord", () => {
        it("should remove the record at the specified index", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", "letsencrypt.org"),
                newCAARecord("example.com", "issue", "comodoca.com"),
                newCAARecord("example.com", "issuewild", "digicert.com"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            policy.removeRecord(1);

            expect(policy.records).toHaveLength(2);
            expect(policy.records[0].Value).toBe("letsencrypt.org");
            expect(policy.records[1].Value).toBe("digicert.com");
        });

        it("should handle removing the first record", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", "letsencrypt.org"),
                newCAARecord("example.com", "issue", "comodoca.com"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            policy.removeRecord(0);

            expect(policy.records).toHaveLength(1);
            expect(policy.records[0].Value).toBe("comodoca.com");
        });

        it("should handle removing the last record", () => {
            const caaRecords: dnsTypeCAA[] = [
                newCAARecord("example.com", "issue", "letsencrypt.org"),
                newCAARecord("example.com", "issue", "comodoca.com"),
            ];
            const resource: dnsResource = { caa: caaRecords };
            const policy = new CAAPolicy(resource);

            policy.removeRecord(1);

            expect(policy.records).toHaveLength(1);
            expect(policy.records[0].Value).toBe("letsencrypt.org");
        });
    });
});

describe("parseCAAIssuer", () => {
    it("should parse issuer domain name without parameters", () => {
        const result = parseCAAIssuer("letsencrypt.org");

        expect(result).toEqual({
            IssuerDomainName: "letsencrypt.org",
            Parameters: [],
        });
    });

    it("should parse issuer domain name with one parameter", () => {
        const result = parseCAAIssuer("letsencrypt.org;accounturi=https://acme.example.com/account/123");

        expect(result).toEqual({
            IssuerDomainName: "letsencrypt.org",
            Parameters: ["accounturi=https://acme.example.com/account/123"],
        });
    });

    it("should parse issuer domain name with multiple parameters", () => {
        const result = parseCAAIssuer("letsencrypt.org;accounturi=https://acme.example.com/account/123;validationmethods=dns-01");

        expect(result).toEqual({
            IssuerDomainName: "letsencrypt.org",
            Parameters: ["accounturi=https://acme.example.com/account/123", "validationmethods=dns-01"],
        });
    });

    it("should return undefined domain name for empty string when newone is true", () => {
        const result = parseCAAIssuer("", true);

        expect(result).toEqual({
            IssuerDomainName: undefined,
            Parameters: [],
        });
    });

    it("should return empty string as domain name when newone is false", () => {
        const result = parseCAAIssuer("", false);

        expect(result).toEqual({
            IssuerDomainName: "",
            Parameters: [],
        });
    });

    it("should handle trailing semicolons", () => {
        const result = parseCAAIssuer("letsencrypt.org;");

        expect(result.IssuerDomainName).toBe("letsencrypt.org");
        expect(result.Parameters).toEqual([""]);
    });
});

describe("stringifyCAAIssuer", () => {
    it("should stringify issuer without parameters", () => {
        const issuer: CAAIssuer = {
            IssuerDomainName: "letsencrypt.org",
            Parameters: [],
        };

        const result = stringifyCAAIssuer(issuer);

        expect(result).toBe("letsencrypt.org");
    });

    it("should stringify issuer with parameters using semicolon", () => {
        const issuer: CAAIssuer = {
            IssuerDomainName: "letsencrypt.org",
            Parameters: ["accounturi=https://acme.example.com/account/123"],
        };

        const result = stringifyCAAIssuer(issuer);

        expect(result).toBe("letsencrypt.org;accounturi=https://acme.example.com/account/123");
    });

    it("should stringify issuer with parameters using semicolon and space", () => {
        const issuer: CAAIssuer = {
            IssuerDomainName: "letsencrypt.org",
            Parameters: ["accounturi=https://acme.example.com/account/123"],
        };

        const result = stringifyCAAIssuer(issuer, "letsencrypt.org; accounturi=https://acme.example.com/account/123");

        expect(result).toBe("letsencrypt.org; accounturi=https://acme.example.com/account/123");
    });

    it("should stringify issuer with multiple parameters", () => {
        const issuer: CAAIssuer = {
            IssuerDomainName: "letsencrypt.org",
            Parameters: ["accounturi=https://acme.example.com/account/123", "validationmethods=dns-01"],
        };

        const result = stringifyCAAIssuer(issuer);

        expect(result).toBe("letsencrypt.org;accounturi=https://acme.example.com/account/123;validationmethods=dns-01");
    });

    it("should return empty string when domain name is undefined", () => {
        const issuer: CAAIssuer = {
            IssuerDomainName: undefined,
            Parameters: [],
        };

        const result = stringifyCAAIssuer(issuer);

        expect(result).toBe("");
    });
});

describe("parseCAAParameter", () => {
    it("should parse parameter with tag and value", () => {
        const result = parseCAAParameter("accounturi=https://acme.example.com/account/123");

        expect(result).toEqual({
            Tag: "accounturi",
            Value: "https://acme.example.com/account/123",
        });
    });

    it("should handle value with equals signs", () => {
        const result = parseCAAParameter("key=value=with=equals");

        expect(result).toEqual({
            Tag: "key",
            Value: "value=with=equals",
        });
    });

    it("should handle parameter with tag only", () => {
        const result = parseCAAParameter("validationmethods");

        expect(result).toEqual({
            Tag: "validationmethods",
            Value: "",
        });
    });

    it("should handle empty string", () => {
        const result = parseCAAParameter("");

        expect(result).toEqual({
            Tag: "",
            Value: "",
        });
    });

    it("should handle parameter with empty value", () => {
        const result = parseCAAParameter("key=");

        expect(result).toEqual({
            Tag: "key",
            Value: "",
        });
    });
});

describe("stringifyCAAParameter", () => {
    it("should stringify parameter with tag and value", () => {
        const param: CAAParameter = {
            Tag: "accounturi",
            Value: "https://acme.example.com/account/123",
        };

        const result = stringifyCAAParameter(param);

        expect(result).toBe("accounturi=https://acme.example.com/account/123");
    });

    it("should return empty string when both tag and value are empty", () => {
        const param: CAAParameter = {
            Tag: "",
            Value: "",
        };

        const result = stringifyCAAParameter(param);

        expect(result).toBe("");
    });

    it("should stringify parameter with tag and empty value", () => {
        const param: CAAParameter = {
            Tag: "key",
            Value: "",
        };

        const result = stringifyCAAParameter(param);

        expect(result).toBe("key=");
    });

    it("should handle value with equals signs", () => {
        const param: CAAParameter = {
            Tag: "key",
            Value: "value=with=equals",
        };

        const result = stringifyCAAParameter(param);

        expect(result).toBe("key=value=with=equals");
    });
});

describe("parseCAAIodef", () => {
    it("should parse mailto URL", () => {
        const result = parseCAAIodef("mailto:security@example.com");

        expect(result).toEqual({
            kind: "mailto",
            url: "security@example.com",
        });
    });

    it("should parse http URL", () => {
        const result = parseCAAIodef("http://example.com/report");

        expect(result).toEqual({
            kind: "http",
            url: "http://example.com/report",
        });
    });

    it("should parse https URL", () => {
        const result = parseCAAIodef("https://example.com/report");

        expect(result).toEqual({
            kind: "http",
            url: "https://example.com/report",
        });
    });

    it("should handle mailto with colon in email", () => {
        const result = parseCAAIodef("mailto:user:tag@example.com");

        expect(result).toEqual({
            kind: "mailto",
            url: "user:tag@example.com",
        });
    });

    it("should strip trailing 's' from https", () => {
        const result = parseCAAIodef("https://example.com");

        expect(result.kind).toBe("http");
    });

    it("should handle plain URL without protocol prefix", () => {
        const result = parseCAAIodef("example.com/report");

        expect(result).toEqual({
            kind: "example.com/report",
            url: "example.com/report",
        });
    });
});

describe("stringifyCAAIodef", () => {
    it("should stringify mailto URL", () => {
        const iodef: CAAIodef = {
            kind: "mailto",
            url: "security@example.com",
        };

        const result = stringifyCAAIodef(iodef);

        expect(result).toBe("mailto:security@example.com");
    });

    it("should stringify http URL", () => {
        const iodef: CAAIodef = {
            kind: "http",
            url: "http://example.com/report",
        };

        const result = stringifyCAAIodef(iodef);

        expect(result).toBe("http://example.com/report");
    });

    it("should stringify https URL", () => {
        const iodef: CAAIodef = {
            kind: "http",
            url: "https://example.com/report",
        };

        const result = stringifyCAAIodef(iodef);

        expect(result).toBe("https://example.com/report");
    });
});

describe("CAA parsing roundtrip tests", () => {
    it("should maintain issuer data through parse and stringify", () => {
        const original = "letsencrypt.org;accounturi=https://acme.example.com/account/123;validationmethods=dns-01";
        const parsed = parseCAAIssuer(original);
        const stringified = stringifyCAAIssuer(parsed, original);

        expect(stringified).toBe(original);
    });

    it("should maintain parameter data through parse and stringify", () => {
        const original = "accounturi=https://acme.example.com/account/123";
        const parsed = parseCAAParameter(original);
        const stringified = stringifyCAAParameter(parsed);

        expect(stringified).toBe(original);
    });

    it("should maintain iodef data through parse and stringify", () => {
        const original = "mailto:security@example.com";
        const parsed = parseCAAIodef(original);
        const stringified = stringifyCAAIodef(parsed);

        expect(stringified).toBe(original);
    });
});
