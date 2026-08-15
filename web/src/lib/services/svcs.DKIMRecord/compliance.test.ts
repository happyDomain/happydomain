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
import { buildContext, getValidators, type ComplianceIssue } from "$lib/services/compliance";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";
import type { ServiceWithValue } from "$lib/model/service.svelte";

const ORIGIN = makeDomain();
const CTX = buildContext("", ORIGIN, null);

function run(name: string, txt: string): ComplianceIssue[] {
    const v = getValidators("svcs.DKIMRecord");
    expect(v?.sync).toBeDefined();
    return v!.sync!({ txt: { Hdr: { Name: name }, Txt: txt } }, CTX);
}

/** Builds the body of a DKIM service, as the editor and the zone carry it. */
function dkimBody(name: string, txt: string): Record<string, any> {
    return { txt: { Hdr: { Name: name }, Txt: txt } };
}

/** Runs the validator on `raw`, with the given DKIM services published at the apex. */
function runInZone(raw: Record<string, any>, siblings: ServiceWithValue[]): ComplianceIssue[] {
    const zone = makeZone({ services: { "": siblings } });
    const v = getValidators("svcs.DKIMRecord");
    expect(v?.sync).toBeDefined();
    return v!.sync!(raw, buildContext("", ORIGIN, zone));
}

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

// Synthetic base64 payloads sized to match the heuristic thresholds. They are
// deliberately not a repeated character, which the placeholder check catches.
const KEY_2048 =
    "G+fWdGWBsp9cr8o6xKKlAtxA/AILuDPGN2GFjCQAcV+kIyBcCUQBDB52U+y71L7QmoU1nV9q4S2D6CsJ328F6HCKkfxW3AHrxoZCR18TtKg3RDomxmLTH3ykEUo672s3rvnNSBb6F9PxKK5hQeyLX+2QJaHG7On+6T3PV6JDCayuIx0SlBNtbI/BoIIMO/qQLvjLhW1/OXsUlZGmP7AYZ9aqDfo9wOg6O5TWbnOGgZqXC4C0ADVda6lRPcxShr5fPm84ZN8c+f4NAyDSPKCO60/iZCQ7s01X0Lv3vF8bejdEBlSwZMcgkpa3Q12jMBSjqwcAk5xF2FRf+kN1wN/+zvUKr2+aWg7u1uMaqDy2";
const KEY_1024 =
    "MhVUD0sfyKGxcZtMbsEXKCs1WTOUuwMyj8THdkSjAV4IKhoXq/JANB+sloIcwy5ySNIV5oHoQz2rrBxhJOVoArW0F/q6AKMm4iYRHFrZvNThXV7p/djxnlL5PtzB31bJf9DRHF/7SOXE56pnUei9HncUf9dlMHAvYlDG1L/iZCQ7s01XVZYI6rU58qM2u2isF4GJVRh/cD1McMAyTuEvDn/Fq5zy";
const KEY_TINY =
    "TD99kvd1USWfbyakNOrGsmIE35xeuJlTQVmlfdUhDyXDsfy3zaRqqFgNRgfgorOjDIQDrwbuFQ/CcX6h87OlwfipkD2tYndEjWN4";

// 32 raw octets, as RFC 8463 sec. 3 asks for.
const ED_RAW = "11qYAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHURo=";

describe("DKIM compliance: happy paths", () => {
    it("accepts a clean RSA-2048-sized record", () => {
        const issues = run("mail._domainkey", `v=DKIM1;k=rsa;p=${KEY_2048}`);
        expect(ids(issues)).toEqual([]);
    });
    it("accepts a clean ed25519 record", () => {
        const issues = run(
            "ed._domainkey",
            "v=DKIM1;k=ed25519;p=11qYAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHURo=",
        );
        expect(ids(issues)).toEqual([]);
    });
});

describe("DKIM compliance: selector", () => {
    it("flags a missing selector", () => {
        const issues = run("._domainkey", `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.missing-selector");
    });
    it("flags an invalid selector", () => {
        const issues = run("bad selector._domainkey", `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.invalid-selector");
    });
    it("accepts dotted selectors (RFC 6376 sec. 3.1)", () => {
        const issues = run("foo.bar._domainkey", `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).toEqual([]);
    });
    it("flags an empty label", () => {
        const issues = run("foo..bar._domainkey", `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.invalid-selector");
    });
    it("flags a label longer than 63 octets", () => {
        const issues = run(`${"a".repeat(64)}._domainkey`, `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.selector-label-too-long");
    });
    it("flags an owner name longer than 253 octets", () => {
        const selector = Array.from({ length: 4 }, () => "a".repeat(60)).join(".");
        const issues = run(`${selector}._domainkey`, `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.selector-name-too-long");
    });
    it("accepts a label of exactly 63 octets", () => {
        const issues = run(`${"a".repeat(63)}._domainkey`, `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).toEqual([]);
    });
    it("reports the label and its length", () => {
        const issue = run(`${"a".repeat(64)}._domainkey`, `v=DKIM1;p=${KEY_2048}`).find(
            (i) => i.id === "dkim.selector-label-too-long",
        );
        expect(issue?.params).toEqual({ label: "a".repeat(64), length: 64 });
    });
    it("draws the owner name limit at 253 octets, trailing dot excluded", () => {
        // Three labels of 60 octets, then one sized so the name reaches the limit.
        const at = (last: number) => {
            const selector = [..."abc", "d"].map((c) =>
                c === "d" ? "d".repeat(last) : c.repeat(60),
            );
            return run(`${selector.join(".")}._domainkey`, `v=DKIM1;p=${KEY_2048}`);
        };
        // 3 * 60 + 3 dots + 47 + "._domainkey.example.com" (23) = 253
        expect(ids(at(47))).not.toContain("dkim.selector-name-too-long");
        expect(ids(at(48))).toContain("dkim.selector-name-too-long");
    });
    it("reports a character outside the grammar before any length", () => {
        const issues = run(`${"a".repeat(64)}$._domainkey`, `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).toEqual(["dkim.invalid-selector"]);
    });
    it("flags a leading, trailing or lone dot in the selector", () => {
        for (const name of [".mail._domainkey", "mail.._domainkey", ".._domainkey"]) {
            expect(ids(run(name, `v=DKIM1;p=${KEY_2048}`))).toContain("dkim.invalid-selector");
        }
    });
    it("flags a space or a dollar sign in the selector", () => {
        for (const name of ["a b._domainkey", "a$b._domainkey"]) {
            expect(ids(run(name, `v=DKIM1;p=${KEY_2048}`))).toContain("dkim.invalid-selector");
        }
    });
    it("accepts a hyphen inside a label and a numeric selector", () => {
        expect(ids(run("my-key._domainkey", `v=DKIM1;p=${KEY_2048}`))).toEqual([]);
        expect(ids(run("20240101._domainkey", `v=DKIM1;p=${KEY_2048}`))).toEqual([]);
    });
    it("warns on a selector label outside the LDH grammar", () => {
        const issues = run("_mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.selector-non-ldh");
    });
    it("warns on a selector label ending with a hyphen", () => {
        const issues = run("mail-._domainkey", `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.selector-non-ldh");
    });
});

describe("DKIM compliance: version & key", () => {
    it("flags a non-DKIM1 version", () => {
        const issues = run("s._domainkey", `v=DKIM2;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.invalid-version");
    });
    it("flags a missing key", () => {
        const issues = run("s._domainkey", "v=DKIM1;k=rsa");
        expect(ids(issues)).toContain("dkim.missing-key");
    });
    it("warns on revoked (empty) key", () => {
        const issues = run("s._domainkey", "v=DKIM1;p=");
        expect(ids(issues)).toContain("dkim.revoked-key");
        expect(ids(issues)).not.toContain("dkim.missing-key");
    });
    it("treats a whitespace-only key as a revocation", () => {
        const issues = run("s._domainkey", "v=DKIM1;p=   ;t=y");
        expect(ids(issues)).toContain("dkim.revoked-key");
        expect(ids(issues)).not.toContain("dkim.missing-key");
    });
    it("flags the placeholder key of the editor", () => {
        const issues = run("s._domainkey", "v=DKIM1;p=a0b1c2d3e4f5==");
        expect(ids(issues)).toContain("dkim.placeholder-key");
    });
    it("flags a payload made of a single repeated character", () => {
        const issues = run("s._domainkey", `v=DKIM1;p=${"A".repeat(360)}`);
        expect(ids(issues)).toContain("dkim.placeholder-key");
        expect(ids(issues)).not.toContain("dkim.short-rsa-key");
    });
    it("takes an ed25519 key shorter than 32 octets for a placeholder", () => {
        const issues = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${ED_RAW.slice(0, 24)}`);
        expect(ids(issues)).toEqual(["dkim.placeholder-key"]);
    });
    it("draws the placeholder limit between 31 and 32 octets", () => {
        const thirtyOne = "11qYAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHUQ==";
        expect(ids(run("s._domainkey", `v=DKIM1;k=rsa;p=${thirtyOne}`))).toContain(
            "dkim.placeholder-key",
        );
        expect(ids(run("s._domainkey", `v=DKIM1;k=rsa;p=${ED_RAW}`))).not.toContain(
            "dkim.placeholder-key",
        );
    });
    it("estimates the size of a payload that does not decode", () => {
        // Five characters cannot be base64, yet they are plainly too few for a key.
        const issues = run("s._domainkey", "v=DKIM1;p=abcde");
        expect(ids(issues)).toContain("dkim.placeholder-key");
    });
    it("takes a padded repeated character for a placeholder", () => {
        const issues = run("s._domainkey", "v=DKIM1;p=AAAA==");
        expect(ids(issues)).toContain("dkim.placeholder-key");
    });
    it("quotes the start of a long placeholder", () => {
        const issue = run("s._domainkey", `v=DKIM1;p=${"A".repeat(360)}`).find(
            (i) => i.id === "dkim.placeholder-key",
        );
        expect(issue?.params).toEqual({ value: "A".repeat(24) + "…" });
    });
    it("does not take a real key for a placeholder", () => {
        const issues = run("s._domainkey", `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).not.toContain("dkim.placeholder-key");
    });
    it("flags non-base64 key payload", () => {
        const issues = run("s._domainkey", "v=DKIM1;p=!!!not-base64!!!");
        expect(ids(issues)).toContain("dkim.invalid-base64");
    });
});

describe("DKIM compliance: RSA key length", () => {
    it("flags a too-short RSA key as an error", () => {
        const issues = run("s._domainkey", `v=DKIM1;k=rsa;p=${KEY_TINY}`);
        expect(ids(issues)).toContain("dkim.weak-rsa-key");
    });
    it("warns on a 1024-bit-sized RSA key", () => {
        const issues = run("s._domainkey", `v=DKIM1;k=rsa;p=${KEY_1024}`);
        expect(ids(issues)).toContain("dkim.short-rsa-key");
    });
    it("does not flag a 2048-bit-sized RSA key", () => {
        const issues = run("s._domainkey", `v=DKIM1;k=rsa;p=${KEY_2048}`);
        expect(ids(issues)).not.toContain("dkim.short-rsa-key");
        expect(ids(issues)).not.toContain("dkim.weak-rsa-key");
    });
});

describe("DKIM compliance: selectors sharing a name", () => {
    it("warns when another record publishes the same selector", () => {
        const raw = dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        const issues = runInZone(raw, [
            makeService("svcs.DKIMRecord", raw),
            makeService("svcs.DKIMRecord", dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_1024}`)),
        ]);
        expect(ids(issues)).toContain("dkim.duplicate-selector");
    });
    it("skips the edited service by id, even when its body differs", () => {
        const saved = makeService(
            "svcs.DKIMRecord",
            dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_1024}`),
            { _id: "svc1" },
        );
        const zone = makeZone({ services: { "": [saved] } });
        const raw = dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        const v = getValidators("svcs.DKIMRecord");
        const issues = v!.sync!(raw, buildContext("", ORIGIN, zone, "svc1"));
        expect(ids(issues)).not.toContain("dkim.duplicate-selector");
    });
    it("does not warn on a distinct selector", () => {
        const raw = dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        const issues = runInZone(raw, [
            makeService("svcs.DKIMRecord", raw),
            makeService("svcs.DKIMRecord", dkimBody("mail2._domainkey", `v=DKIM1;p=${KEY_2048}`)),
        ]);
        expect(ids(issues)).not.toContain("dkim.duplicate-selector");
    });
    it("counts every record sharing the selector", () => {
        const raw = dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        const issues = runInZone(raw, [
            makeService("svcs.DKIMRecord", raw),
            makeService("svcs.DKIMRecord", dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_1024}`)),
            makeService("svcs.DKIMRecord", dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_TINY}`)),
        ]);
        const issue = issues.find((i) => i.id === "dkim.duplicate-selector");
        expect(issue?.params).toEqual({ selector: "mail", count: 3 });
    });
    it("recognises a duplicate written at the apex with the subdomain in its name", () => {
        const raw = dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        const zone = makeZone({
            services: {
                corp: [makeService("svcs.DKIMRecord", raw)],
                "": [
                    makeService(
                        "svcs.DKIMRecord",
                        dkimBody("mail._domainkey.corp", `v=DKIM1;p=${KEY_1024}`),
                    ),
                ],
            },
        });
        const v = getValidators("svcs.DKIMRecord");
        const issues = v!.sync!(raw, buildContext("corp", ORIGIN, zone));
        expect(ids(issues)).toContain("dkim.duplicate-selector");
    });
    it("compares names regardless of case", () => {
        const raw = dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        const issues = runInZone(raw, [
            makeService("svcs.DKIMRecord", raw),
            makeService("svcs.DKIMRecord", dkimBody("MAIL._DomainKey", `v=DKIM1;p=${KEY_1024}`)),
        ]);
        expect(ids(issues)).toContain("dkim.duplicate-selector");
    });
    it("ignores a zone service carrying no TXT", () => {
        const raw = dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        const issues = runInZone(raw, [makeService("svcs.DKIMRecord", {})]);
        expect(ids(issues)).not.toContain("dkim.duplicate-selector");
    });
    it("stays quiet when the zone is unknown", () => {
        const issues = run("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        expect(ids(issues)).not.toContain("dkim.duplicate-selector");
    });
    it("does not mistake the edited record for a duplicate of itself", () => {
        const raw = dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        const issues = runInZone(raw, [makeService("svcs.DKIMRecord", raw)]);
        expect(ids(issues)).toEqual([]);
    });
});

describe("DKIM compliance: ed25519 key", () => {
    // The same key wrapped in a SubjectPublicKeyInfo (44 octets).
    const ED_SPKI = "MCowBQYDK2VwAyEA11qYAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHURo=";

    it("accepts the 32 raw octets", () => {
        const issues = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${ED_RAW}`);
        expect(ids(issues)).toEqual([]);
    });
    it("warns on a SubjectPublicKeyInfo-wrapped key", () => {
        const issues = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${ED_SPKI}`);
        expect(ids(issues)).toContain("dkim.ed25519-spki-key");
    });
    it("flags a key that is not 32 octets long", () => {
        // The same key with 8 octets of junk appended.
        const ED_40 = "11qYAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHURoqKioqKioqKg==";
        const issues = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${ED_40}`);
        expect(ids(issues)).toContain("dkim.invalid-ed25519-key-length");
    });
    it("flags an RSA-sized payload announced as ed25519", () => {
        const issues = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.invalid-ed25519-key-length");
    });
    it("reports the length of a key that is not 32 octets", () => {
        const ED_40 = "11qYAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHURoqKioqKioqKg==";
        const issue = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${ED_40}`).find(
            (i) => i.id === "dkim.invalid-ed25519-key-length",
        );
        expect(issue?.params).toEqual({ length: 40 });
    });
    it("flags 44 octets that do not start with the SubjectPublicKeyInfo prefix", () => {
        const issues = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${"ABCD".repeat(14)}ABC=`);
        expect(ids(issues)).toContain("dkim.invalid-ed25519-key-length");
        expect(ids(issues)).not.toContain("dkim.ed25519-spki-key");
    });
    it("flags an Ed25519 payload whose length cannot be base64", () => {
        const issues = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${"AbCd".repeat(11)}x`);
        expect(ids(issues)).toContain("dkim.invalid-base64");
    });
    it("leaves a payload that is not base64 to the generic check", () => {
        const issues = run("ed._domainkey", "v=DKIM1;k=ed25519;p=not*base64!");
        expect(ids(issues)).toContain("dkim.invalid-base64");
        expect(ids(issues)).not.toContain("dkim.invalid-ed25519-key-length");
    });
    it("ignores whitespace inside the key, as the TXT chunks may add some", () => {
        const spaced = `${ED_RAW.slice(0, 20)} ${ED_RAW.slice(20)}`;
        const issues = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${spaced}`);
        expect(ids(issues)).toEqual([]);
    });
    it("takes a missing k= for rsa, and says so of a 32 octet key", () => {
        const issues = run("s._domainkey", `v=DKIM1;p=${ED_RAW}`);
        expect(ids(issues)).toContain("dkim.key-type-mismatch");
    });
    it("does not guess the algorithm of an unknown key type", () => {
        const issues = run("s._domainkey", `v=DKIM1;k=foo;p=${ED_RAW}`);
        expect(ids(issues)).not.toContain("dkim.key-type-mismatch");
        expect(ids(issues)).not.toContain("dkim.invalid-ed25519-key-length");
    });
    it("flags an ed25519 key announced as rsa", () => {
        const issues = run("s._domainkey", `v=DKIM1;k=rsa;p=${ED_RAW}`);
        expect(ids(issues)).toContain("dkim.key-type-mismatch");
        expect(ids(issues)).not.toContain("dkim.weak-rsa-key");
    });
});

describe("DKIM compliance: algorithms & flags", () => {
    it("warns on sha1 hash", () => {
        const issues = run("s._domainkey", `v=DKIM1;h=sha1;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.deprecated-hash");
    });
    it("warns on unknown hash", () => {
        const issues = run("s._domainkey", `v=DKIM1;h=md5;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.unknown-hash");
    });
    it("warns on unknown key type", () => {
        const issues = run("s._domainkey", `v=DKIM1;k=foo;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.unknown-key-type");
    });
    it("infos on unknown service type", () => {
        const issues = run("s._domainkey", `v=DKIM1;s=foo;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.unknown-service-type");
    });
    it("warns on unknown flag", () => {
        const issues = run("s._domainkey", `v=DKIM1;t=q;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.unknown-flag");
    });
    it("infos on testing mode (t=y)", () => {
        const issues = run("s._domainkey", `v=DKIM1;t=y;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.testing-mode");
    });
    it("infos on deprecated g= when not '*'", () => {
        const issues = run("s._domainkey", `v=DKIM1;g=user;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.deprecated-granularity");
    });
    it("does not flag g=*", () => {
        const issues = run("s._domainkey", `v=DKIM1;g=*;p=${KEY_2048}`);
        expect(ids(issues)).not.toContain("dkim.deprecated-granularity");
    });
});

describe("DKIM compliance: graceful empty input", () => {
    it("returns no issues when txt and selector are present and empty TXT", () => {
        const issues = run("mail._domainkey", "");
        expect(issues).toEqual([]);
    });
});
