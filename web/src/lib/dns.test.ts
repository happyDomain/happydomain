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
import { domainCompare, fqdn, fqdnCompare, isReverseZone, nsttl, reverseDomain, unreverseDomain, validateDomain } from "./dns";

describe('fqdn', () => {
  const origin = 'example.com.';
  it('should return the origin if input is "@"', () => {
    expect(fqdn('@', origin)).toBe(origin);
  });

  it('should return the input if it ends with a dot', () => {
    const input = 'test.';
    expect(fqdn(input, origin)).toBe(input);
  });

  it('should return the origin if input is an empty string', () => {
    expect(fqdn('', origin)).toBe(origin);
  });

  it('should concatenate input and origin with a dot if none of the above conditions are met', () => {
    const input = 'subdomain';
    expect(fqdn(input, origin)).toBe(`${input}.${origin}`);
  });
});

describe('domainCompare', () => {
  it('should correctly compare two string domains', () => {
    expect(domainCompare('a.example.com', 'b.example.com')).toBeLessThan(0);
    expect(domainCompare('b.example.com', 'a.example.com')).toBeGreaterThan(0);
    expect(domainCompare('example.com', 'example.com')).toBe(0);
  });

  it('should correctly compare domains with different lengths', () => {
    expect(domainCompare('example.com', 'example.org')).toBeLessThan(0);
    expect(domainCompare('example.org', 'example.com')).toBeGreaterThan(0);
  });

  it('should correctly compare domains with different subdomains', () => {
    expect(domainCompare('sub.example.com', 'example.com')).toBeGreaterThan(0);
    expect(domainCompare('example.com', 'sub.example.com')).toBeLessThan(0);
  });

  it('should correctly compare domains provided as objects', () => {
    const domainA = { domain: 'a.example.com' };
    const domainB = { domain: 'b.example.com' };
    expect(domainCompare(domainA, domainB)).toBeLessThan(0);
    expect(domainCompare(domainB, domainA)).toBeGreaterThan(0);
    expect(domainCompare(domainA, domainA)).toBe(0);
  });

  it('should handle empty strings correctly', () => {
    expect(domainCompare('', 'example.com')).toBeLessThan(0);
    expect(domainCompare('example.com', '')).toBeGreaterThan(0);
    expect(domainCompare('', '')).toBe(0);
  });
});

describe('fqdnCompare', () => {
  it('should return 0 for identical domains', () => {
    expect(fqdnCompare('example.com', 'example.com')).toBe(0);
  });

  it('should return a negative number if the first domain is less than the second', () => {
    expect(fqdnCompare('example.com', 'examples.com')).toBeLessThan(0);
  });

  it('should return a positive number if the first domain is greater than the second', () => {
    expect(fqdnCompare('examples.com', 'example.com')).toBeGreaterThan(0);
  });

  it('should handle domains with different lengths', () => {
    expect(fqdnCompare('sub.example.com', 'example.com')).toBeGreaterThan(0);
    expect(fqdnCompare('example.com', 'sub.example.com')).toBeLessThan(0);
  });

  it('should handle domains with different top-level domains', () => {
    expect(fqdnCompare('example.com', 'example.org')).toBeLessThan(0);
    expect(fqdnCompare('example.org', 'example.com')).toBeGreaterThan(0);
  });

  it('should handle Domain objects', () => {
    const domainA = { domain: 'example.com' };
    const domainB = { domain: 'example.com' };
    expect(fqdnCompare(domainA, domainB)).toBe(0);
  });
});

describe('nsttl', () => {
    it('should return the correct time string for seconds', () => {
        expect(nsttl(45)).toBe('45s');
    });

    it('should return the correct time string for minutes and seconds', () => {
        expect(nsttl(90)).toBe('1m 30s');
    });

    it('should return the correct time string for hours, minutes, and seconds', () => {
        expect(nsttl(3661)).toBe('1h 1m 1s');
    });

    it('should return the correct time string for days, hours, minutes, and seconds', () => {
        expect(nsttl(93781)).toBe('1d 2h 3m 1s');
    });

    it('should handle zero seconds correctly', () => {
        expect(nsttl(0)).toBe('');
    });
});

describe('isReverseZone', () => {
  it('should return true for an IPv4 reverse zone', () => {
    const fqdn = '1.168.192.in-addr.arpa.';
    expect(isReverseZone(fqdn)).toBe(true);
  });

  it('should return true for an IPv6 reverse zone', () => {
    const fqdn = '1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.';
    expect(isReverseZone(fqdn)).toBe(true);
  });

  it('should return false for a regular domain', () => {
    const fqdn = 'example.com.';
    expect(isReverseZone(fqdn)).toBe(false);
  });

  it('should return false for a domain without the reverse zone suffix', () => {
    const fqdn = 'example.arpa.';
    expect(isReverseZone(fqdn)).toBe(false);
  });

  it('should return false for an empty string', () => {
    const fqdn = '';
    expect(isReverseZone(fqdn)).toBe(false);
  });
});

describe('reverseDomain', () => {
  it('should correctly reverse an IPv4 address', () => {
    expect(reverseDomain('192.168.1.1')).toBe('1.1.168.192.in-addr.arpa.');
    expect(reverseDomain('10.0.0.1')).toBe('1.0.0.10.in-addr.arpa.');
  });

  it('should correctly reverse an IPv4 address with less than 4 fields', () => {
    expect(reverseDomain('192.168.1')).toBe('1.0.168.192.in-addr.arpa.');
    expect(reverseDomain('192.1')).toBe('1.0.0.192.in-addr.arpa.');
  });

  it('should correctly reverse an IPv6 address', () => {
    expect(reverseDomain('2001:0db8:85a3:0000:0000:8a2e:0370:7334')).toBe(
      '4.3.3.7.0.7.3.0.e.2.a.8.0.0.0.0.0.0.0.0.3.a.5.8.8.b.d.0.1.0.0.2.ip6.arpa.'
    );
  });

  it('should correctly reverse a compressed IPv6 address', () => {
    expect(reverseDomain('2001:db8::8a2e:370:7334')).toBe(
      '4.3.3.7.0.7.3.0.e.2.a.8.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.'
    );
  });

  it('should correctly handle an IPv6 address with an empty field', () => {
    expect(reverseDomain('2001:db8::0:7334')).toBe(
      '4.3.3.7.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.'
    );
  });
});

describe('unreverseDomain', () => {
  it('should correctly convert an IPv4 reverse DNS domain to an IP address', () => {
    const dn = '1.168.192.in-addr.arpa.';
    const ip = unreverseDomain(dn);
    expect(ip).toBe('192.168.1');
  });

  it('should correctly convert an IPv6 reverse DNS domain to an IP address', () => {
    const dn = '1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.';
    const ip = unreverseDomain(dn);
    expect(ip).toBe('2001:db8::1');
  });

  it('should handle IPv6 compression correctly', () => {
    const dn = '0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa.';
    const ip = unreverseDomain(dn);
    expect(ip).toBe('::');
  });

  it('should handle IPv6 compression correctly', () => {
    const dn = '1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa.';
    const ip = unreverseDomain(dn);
    expect(ip).toBe('::1');
  });

  it('should handle IPv6 compression correctly', () => {
    const dn = '1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.2.0.0.0.ip6.arpa.';
    const ip = unreverseDomain(dn);
    expect(ip).toBe('2::1');
  });

  it('should handle IPv6 compression correctly', () => {
    const dn = '1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.3.0.0.0.2.0.0.0.ip6.arpa.';
    const ip = unreverseDomain(dn);
    expect(ip).toBe('2:3::1');
  });

  it('should handle IPv6 compression correctly', () => {
    const dn = '1.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.0.0.0.0.3.0.0.0.2.0.0.0.ip6.arpa.';
    const ip = unreverseDomain(dn);
    expect(ip).toBe('2:3::4:0:0:1');
  });

  it('should handle leading zeros in IPv6 correctly', () => {
    const dn = '1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.';
    const ip = unreverseDomain(dn);
    expect(ip).toBe('2001:db8::1');
  });
});

describe("validateDomain", () => {
  it("validates a simple domain", () => {
    expect(validateDomain("example.com")).toBe(true);
  });

  it("validates a domain with multiple subdomains", () => {
    expect(validateDomain("a.b.c.d.example.com")).toBe(true);
  });

  it("rejects domains that are too long", () => {
    const label = "a".repeat(63);
    const tooLong = `${label}.${label}.${label}.${label}.com`;
    expect(tooLong.length).toBeGreaterThan(254);
    expect(validateDomain(tooLong)).toBe(false);
  });

  it("rejects labels that are too long", () => {
    const invalid = "a".repeat(64) + ".com";
    expect(validateDomain(invalid)).toBe(false);
  });

  it("rejects labels with hyphens at the start or end", () => {
    expect(validateDomain("-example.com")).toBe(false);
    expect(validateDomain("example-.com")).toBe(false);
  });

  it("rejects empty labels", () => {
    expect(validateDomain("example..com")).toBe(false);
  });

  it("validates a domain with a wildcard at the start", () => {
    expect(validateDomain("*.example.com")).toBe(true);
  });

  it("rejects wildcards in the middle of a domain", () => {
    expect(validateDomain("www.*.example.com")).toBe(false);
  });

  it("rejects domains with invalid characters", () => {
    expect(validateDomain("exa$mple.com")).toBe(false);
    expect(validateDomain("examp!e.com")).toBe(false);
    expect(validateDomain("exam ple.com")).toBe(false);
    expect(validateDomain("exam_ple.com")).toBe(false);
  });

  it("validates labels with underscores for special DNS records", () => {
    expect(validateDomain("_dmarc.example.com")).toBe(true);
    expect(validateDomain("_tcp.mail.example.com")).toBe(true);
    expect(validateDomain("_ssh._tcp.example.com")).toBe(true);
  });

  it("validates domains with a relative origin", () => {
    expect(validateDomain("www", "example.com.")).toBe(true);  // www.example.com
    expect(validateDomain("*.www", "example.com.")).toBe(true); // *.www.example.com
  });

  it("validates absolute domains", () => {
    expect(validateDomain("www.example.com.", "example.com.")).toBe(true);
    expect(validateDomain("www.example.net.", "example.com.")).toBe(false);
  });

  it("rejects empty domain strings", () => {
    expect(validateDomain("")).toBe(undefined);
  });

  it("accepts labels starting with a digit (per RFC allowance)", () => {
    expect(validateDomain("3example.com")).toBe(true);
  });
});
