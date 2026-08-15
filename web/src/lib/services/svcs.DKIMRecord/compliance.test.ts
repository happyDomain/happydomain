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

// Synthetic base64 payloads sized to match the heuristic thresholds.
const KEY_2048 = "A".repeat(360);
const KEY_1024 = "A".repeat(220);
const KEY_TINY = "A".repeat(100);

describe("DKIM compliance: happy paths", () => {
    it("accepts a clean RSA-2048-sized record", () => {
        const issues = run("mail._domainkey", `v=DKIM1;k=rsa;p=${KEY_2048}`);
        expect(ids(issues)).toEqual([]);
    });
    it("accepts a clean ed25519 record", () => {
        const issues = run("ed._domainkey", "v=DKIM1;k=ed25519;p=11qYAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHURo=");
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
    it("does not warn on a distinct selector", () => {
        const raw = dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        const issues = runInZone(raw, [
            makeService("svcs.DKIMRecord", raw),
            makeService("svcs.DKIMRecord", dkimBody("mail2._domainkey", `v=DKIM1;p=${KEY_2048}`)),
        ]);
        expect(ids(issues)).not.toContain("dkim.duplicate-selector");
    });
    it("does not mistake the edited record for a duplicate of itself", () => {
        const raw = dkimBody("mail._domainkey", `v=DKIM1;p=${KEY_2048}`);
        const issues = runInZone(raw, [makeService("svcs.DKIMRecord", raw)]);
        expect(ids(issues)).toEqual([]);
    });
});

describe("DKIM compliance: ed25519 key", () => {
    // 32 raw octets, as RFC 8463 sec. 3 asks for.
    const ED_RAW = "11qYAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHURo=";
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
    it("flags a truncated key", () => {
        const issues = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${ED_RAW.slice(0, 24)}`);
        expect(ids(issues)).toContain("dkim.invalid-ed25519-key-length");
    });
    it("flags an RSA-sized payload announced as ed25519", () => {
        const issues = run("ed._domainkey", `v=DKIM1;k=ed25519;p=${KEY_2048}`);
        expect(ids(issues)).toContain("dkim.invalid-ed25519-key-length");
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
