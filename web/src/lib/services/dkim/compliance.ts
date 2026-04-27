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
    registerValidators,
} from "$lib/services/compliance";
import { parseDKIM, type DKIMValue } from "$lib/services/dkim.svelte";

const KNOWN_KEY_TYPES = new Set(["rsa", "ed25519"]);
const KNOWN_HASH_ALGS = new Set(["sha1", "sha256"]);
const DEPRECATED_HASH_ALGS = new Set(["sha1"]);
const KNOWN_SERVICE_TYPES = new Set(["email", "*"]);
const KNOWN_FLAGS = new Set(["y", "s"]);
const SELECTOR_LABEL_RE = /^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)*$/;
const BASE64_RE = /^[A-Za-z0-9+/]+={0,2}$/;

function dkimSync(raw: Record<string, any>, _ctx: ComplianceContext): ComplianceIssue[] {
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
            docUrl: "https://www.rfc-editor.org/rfc/rfc6376#section-3.1",
        });
    } else if (!SELECTOR_LABEL_RE.test(selector)) {
        issues.push({
            id: "dkim.invalid-selector",
            severity: "error",
            params: { selector },
            field: "selector",
            docUrl: "https://www.rfc-editor.org/rfc/rfc6376#section-3.1",
        });
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
        // Approximate RSA modulus size from the base64 payload length. The
        // payload encodes a SubjectPublicKeyInfo, so a 1024-bit key sits in
        // the 200-330 char range and a 2048-bit key around 360-400 chars.
        // RFC 8301 forbids RSA keys shorter than 1024 bits, recommends 2048.
        const keyType = val.k ?? "rsa";
        if (keyType === "rsa") {
            const len = val.p.replace(/\s+/g, "").replace(/=+$/, "").length;
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
