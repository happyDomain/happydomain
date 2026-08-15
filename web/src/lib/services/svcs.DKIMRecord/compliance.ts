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

import {
    type ComplianceContext,
    type ComplianceIssue,
    isValidHostname,
    recordFqdn,
    registerValidators,
} from "$lib/services/compliance";
import { parseDKIM, type DKIMValue } from "./model.svelte";

const KNOWN_KEY_TYPES = new Set(["rsa", "ed25519"]);
const KNOWN_HASH_ALGS = new Set(["sha1", "sha256"]);
const DEPRECATED_HASH_ALGS = new Set(["sha1"]);
const KNOWN_SERVICE_TYPES = new Set(["email", "*"]);
const KNOWN_FLAGS = new Set(["y", "s"]);
const SELECTOR_LABEL_RE = /^[A-Za-z0-9_-]+$/;
const BASE64_RE = /^[A-Za-z0-9+/]+={0,2}$/;

const RFC6376_SELECTOR = "https://www.rfc-editor.org/rfc/rfc6376#section-3.1";
// RFC 1035 sec. 2.3.4: 63 octets per label, 255 for the whole name once the
// length octets and the root are counted, hence 253 characters written down.
const MAX_LABEL_LENGTH = 63;
const MAX_NAME_LENGTH = 253;

/**
 * Checks the DNS name structure of a selector. RFC 6376 sec. 3.1 defines it as
 * one or more RFC 5321 sub-domains, so a dotted selector is fine but the usual
 * DNS length limits apply, and a label is expected to be letters, digits and
 * hyphens only.
 */
function selectorIssues(
    selector: string,
    hdrName: string,
    ctx: ComplianceContext,
): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const labels = selector.split(".");

    // An empty label ("foo..bar") or a character outside the LDH-plus-underscore
    // set makes the whole name unusable: report it once and stop there.
    if (labels.some((label) => !SELECTOR_LABEL_RE.test(label))) {
        return [
            {
                id: "dkim.invalid-selector",
                severity: "error",
                params: { selector },
                field: "selector",
                docUrl: RFC6376_SELECTOR,
            },
        ];
    }

    for (const label of labels) {
        if (label.length > MAX_LABEL_LENGTH) {
            issues.push({
                id: "dkim.selector-label-too-long",
                severity: "error",
                params: { label, length: label.length },
                field: "selector",
                docUrl: "https://www.rfc-editor.org/rfc/rfc1035#section-2.3.4",
            });
        } else if (!isValidHostname(label)) {
            // Underscores and leading or trailing hyphens still resolve, but
            // they are outside the sub-domain grammar the RFC points at.
            issues.push({
                id: "dkim.selector-non-ldh",
                severity: "warning",
                params: { label },
                field: "selector",
                docUrl: RFC6376_SELECTOR,
            });
        }
    }

    const owner = recordFqdn(hdrName, ctx);
    if (owner.length > MAX_NAME_LENGTH) {
        issues.push({
            id: "dkim.selector-name-too-long",
            severity: "error",
            params: { length: owner.length },
            field: "selector",
            docUrl: "https://www.rfc-editor.org/rfc/rfc1035#section-2.3.4",
        });
    }

    return issues;
}

// RFC 8463 sec. 3: an Ed25519 public key is 256 bits, published as the base64
// encoding of the 32 raw octets.
const ED25519_KEY_LENGTH = 32;
// The same key wrapped in a SubjectPublicKeyInfo is 44 octets, always prefixed
// by this fixed header (SEQUENCE, AlgorithmIdentifier id-Ed25519 1.3.101.112,
// then the BIT STRING holding the 32 octets).
const ED25519_SPKI_PREFIX = [
    0x30, 0x2a, 0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, 0x70, 0x03, 0x21, 0x00,
];
const ED25519_SPKI_LENGTH = ED25519_SPKI_PREFIX.length + ED25519_KEY_LENGTH;

/** Decodes a base64 payload, returning null when it is not decodable. */
function decodeBase64(payload: string): Uint8Array | null {
    try {
        const binary = atob(payload);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
        return bytes;
    } catch {
        return null;
    }
}

function hasEd25519SpkiPrefix(bytes: Uint8Array): boolean {
    return ED25519_SPKI_PREFIX.every((b, i) => bytes[i] === b);
}

/**
 * Checks the shape of an Ed25519 public key (RFC 8463 sec. 3). The curve is
 * fixed, so the octet count is the whole verification: 32 raw octets, or the
 * 44 octets of a SubjectPublicKeyInfo that some tools emit instead.
 */
function ed25519KeyIssues(bytes: Uint8Array | null): ComplianceIssue[] {
    // An undecodable payload is already reported as invalid base64.
    if (!bytes) return [];

    if (bytes.length === ED25519_KEY_LENGTH) return [];

    if (bytes.length === ED25519_SPKI_LENGTH && hasEd25519SpkiPrefix(bytes)) {
        return [
            {
                id: "dkim.ed25519-spki-key",
                severity: "warning",
                field: "p",
                docUrl: "https://www.rfc-editor.org/rfc/rfc8463#section-3",
            },
        ];
    }

    return [
        {
            id: "dkim.invalid-ed25519-key-length",
            severity: "error",
            params: { length: bytes.length },
            field: "p",
            docUrl: "https://www.rfc-editor.org/rfc/rfc8463#section-3",
        },
    ];
}

function dkimSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const txt = raw?.txt;
    if (!txt) return issues;

    const txtValue: string = typeof txt.Txt === "string" ? txt.Txt : "";
    const name: string = typeof txt.Hdr?.Name === "string" ? txt.Hdr.Name : "";

    // Selector: owner name must be "<selector>._domainkey".
    const selector = name.endsWith("._domainkey")
        ? name.slice(0, -"._domainkey".length)
        : "";
    if (!selector) {
        issues.push({
            id: "dkim.missing-selector",
            severity: "error",
            field: "selector",
            docUrl: RFC6376_SELECTOR,
        });
    } else {
        issues.push(...selectorIssues(selector, name, ctx));
    }

    if (!txtValue.trim()) {
        // Nothing yet to validate beyond the selector: the user is starting.
        return issues;
    }

    let val: DKIMValue;
    try {
        val = parseDKIM(txtValue);
    } catch {
        issues.push({
            id: "dkim.parse-error",
            severity: "error",
            field: "txt",
        });
        return issues;
    }

    // v= must be DKIM1 when present (RFC 6376 §3.6.1).
    if (val.v !== undefined && val.v !== "" && val.v !== "DKIM1") {
        issues.push({
            id: "dkim.invalid-version",
            severity: "error",
            params: { version: val.v },
            field: "v",
            docUrl: "https://www.rfc-editor.org/rfc/rfc6376#section-3.6.1",
        });
    }

    // p= is mandatory. parseKeyValueTxt drops empty values, so check the raw
    // string to tell "no p tag" from "p=" (the latter being a key revocation
    // per RFC 6376 §3.6.1).
    const hasPTag = /(?:^|;)\s*p\s*=/i.test(txtValue);
    if (!hasPTag) {
        issues.push({
            id: "dkim.missing-key",
            severity: "error",
            field: "p",
            docUrl: "https://www.rfc-editor.org/rfc/rfc6376#section-3.6.1",
        });
    } else if (!val.p) {
        issues.push({
            id: "dkim.revoked-key",
            severity: "warning",
            field: "p",
            docUrl: "https://www.rfc-editor.org/rfc/rfc6376#section-3.6.1",
        });
    } else if (!BASE64_RE.test(val.p.replace(/\s+/g, ""))) {
        issues.push({
            id: "dkim.invalid-base64",
            severity: "error",
            field: "p",
        });
    } else {
        const payload = val.p.replace(/\s+/g, "");
        const bytes = decodeBase64(payload);
        const keyType = val.k ?? "rsa";

        if (keyType === "ed25519") {
            issues.push(...ed25519KeyIssues(bytes));
        } else if (keyType === "rsa" && bytes && bytes.length === ED25519_KEY_LENGTH) {
            // 32 octets is an Ed25519 key, never an RSA one: the k= tag and the
            // key disagree. Reporting a weak RSA key here would send the user
            // regenerating a key that is perfectly fine.
            issues.push({
                id: "dkim.key-type-mismatch",
                severity: "error",
                params: { type: keyType, expected: "ed25519" },
                field: "p",
                docUrl: "https://www.rfc-editor.org/rfc/rfc8463#section-3",
            });
        } else if (keyType === "rsa") {
            // Approximate RSA modulus size from the base64 payload length. The
            // payload encodes a SubjectPublicKeyInfo, so a 1024-bit key sits in
            // the 200-330 char range and a 2048-bit key around 360-400 chars.
            // RFC 8301 forbids RSA keys shorter than 1024 bits, recommends 2048.
            const len = payload.replace(/=+$/, "").length;
            if (len < 200) {
                issues.push({
                    id: "dkim.weak-rsa-key",
                    severity: "error",
                    field: "p",
                    docUrl: "https://www.rfc-editor.org/rfc/rfc8301#section-3.2",
                });
            } else if (len < 330) {
                issues.push({
                    id: "dkim.short-rsa-key",
                    severity: "warning",
                    field: "p",
                    docUrl: "https://www.rfc-editor.org/rfc/rfc8301#section-3.2",
                });
            }
        }
    }

    // k= key type.
    if (val.k !== undefined && val.k !== "" && !KNOWN_KEY_TYPES.has(val.k)) {
        issues.push({
            id: "dkim.unknown-key-type",
            severity: "warning",
            params: { type: val.k },
            field: "k",
        });
    }

    // h= hash algorithms.
    for (const h of val.h ?? []) {
        if (!h) continue;
        if (DEPRECATED_HASH_ALGS.has(h)) {
            issues.push({
                id: "dkim.deprecated-hash",
                severity: "warning",
                params: { hash: h },
                field: "h",
                docUrl: "https://www.rfc-editor.org/rfc/rfc8301#section-3.1",
            });
        } else if (!KNOWN_HASH_ALGS.has(h)) {
            issues.push({
                id: "dkim.unknown-hash",
                severity: "warning",
                params: { hash: h },
                field: "h",
            });
        }
    }

    // s= service types.
    for (const s of val.s ?? []) {
        if (!s) continue;
        if (!KNOWN_SERVICE_TYPES.has(s)) {
            issues.push({
                id: "dkim.unknown-service-type",
                severity: "info",
                params: { type: s },
                field: "s",
            });
        }
    }

    // t= flags.
    for (const flag of val.t ?? []) {
        if (!flag) continue;
        if (!KNOWN_FLAGS.has(flag)) {
            issues.push({
                id: "dkim.unknown-flag",
                severity: "warning",
                params: { flag },
                field: "t",
            });
        } else if (flag === "y") {
            issues.push({
                id: "dkim.testing-mode",
                severity: "info",
                field: "t",
                docUrl: "https://www.rfc-editor.org/rfc/rfc6376#section-3.6.1",
            });
        }
    }

    // g= granularity is deprecated (was in RFC 4871, removed by RFC 6376).
    if (val.g !== undefined && val.g !== "" && val.g !== "*") {
        issues.push({
            id: "dkim.deprecated-granularity",
            severity: "info",
            field: "g",
            docUrl: "https://www.rfc-editor.org/rfc/rfc6376#appendix-C",
        });
    }

    return issues;
}

registerValidators("svcs.DKIMRecord", { sync: dkimSync });
