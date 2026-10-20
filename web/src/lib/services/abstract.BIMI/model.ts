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

import { fqdn } from "$lib/dns";
import { stringifyBIMI } from "../svcs.BIMI/model";

/** Kept in step with abstract.BIMIHostPrefix and abstract.BIMIPathPrefix. */
export const HOST_PREFIX = "bimi.";
export const PATH_PREFIX = "/bimi/";

/** Kept in step with abstract.BIMIMaxLogoSize. */
export const MAX_LOGO_SIZE = 32 * 1024;
/** Kept in step with abstract.BIMIMaxCertificateSize. */
export const MAX_CERTIFICATE_SIZE = 64 * 1024;

/** The bimi. CNAME and TXT owner names, relative to the service domain. */
export const HOST_CNAME_NAME = "bimi";
export const RECORD_SUFFIX = "._bimi";

/** The JSON encoding of a Go []byte is standard base64. */
export function encodeBase64(bytes: Uint8Array): string {
    let binary = "";
    for (const b of bytes) binary += String.fromCharCode(b);
    return btoa(binary);
}

export function decodeBase64(b64: string | undefined): Uint8Array<ArrayBuffer> {
    if (!b64) return new Uint8Array();
    const binary = atob(b64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
    return bytes;
}

/**
 * Names a file after its content, as (*abstract.BIMI).LogoName does: the first
 * 16 hex digits of its SHA-256, plus the extension.
 *
 * Returns null when hashing is unavailable (WebCrypto needs a secure context):
 * the backend generates the record anyway, only the preview is lost.
 */
export async function contentName(
    content: Uint8Array<ArrayBuffer>,
    ext: string,
): Promise<string | null> {
    if (content.length === 0) return "";
    if (!globalThis.crypto?.subtle) return null;

    const digest = new Uint8Array(await globalThis.crypto.subtle.digest("SHA-256", content));
    let hex = "";
    for (const b of digest.slice(0, 8)) hex += b.toString(16).padStart(2, "0");
    return hex + ext;
}

export function serviceDomain(dn: string, origin: string): string {
    return fqdn(dn, origin).replace(/\.$/, "");
}

export function fileURL(domain: string, name: string): string {
    return "https://" + HOST_PREFIX + domain + PATH_PREFIX + name;
}

/**
 * The TXT record the backend generates for the given files, as
 * (*abstract.BIMI).GetRecords does.
 */
export function generatedRecord(domain: string, logoName: string, certificateName: string): string {
    return stringifyBIMI({
        v: "BIMI1",
        l: fileURL(domain, logoName),
        a: certificateName ? fileURL(domain, certificateName) : undefined,
    });
}

export function selectorOf(name: string | undefined): string {
    if (!name) return "";
    return name.endsWith(RECORD_SUFFIX) ? name.slice(0, -RECORD_SUFFIX.length) : name;
}

/**
 * Problems with an SVG logo, as translation keys under services.bimi-hosted.
 *
 * A quick, approximate mirror of abstract.ValidateLogo, for instant feedback:
 * it does not parse the XML, and the backend still has the last word when the
 * zone is saved.
 */
export function logoProblems(svg: string, size: number): string[] {
    const problems: string[] = [];

    if (size > MAX_LOGO_SIZE) problems.push("logo-too-large");
    if (!/<svg[\s>]/i.test(svg)) {
        problems.push("logo-not-svg");
        return problems;
    }
    if (!/\bbaseProfile\s*=\s*["']tiny-ps["']/.test(svg)) problems.push("logo-not-tiny-ps");
    if (
        /<(script|foreignObject)[\s>/]/i.test(svg) ||
        /\son\w+\s*=/i.test(svg) ||
        /<!ENTITY/.test(svg)
    ) {
        problems.push("logo-active-content");
    }
    if (/\bhref\s*=\s*["']\s*[^"'#\s]/i.test(svg)) problems.push("logo-external-reference");

    return problems;
}

/**
 * Problems with a certificate file, as translation keys under
 * services.bimi-hosted. Mirrors the checks of abstract.ValidateCertificate
 * that do not need an X.509 parser.
 */
export function certificateProblems(pem: string, size: number): string[] {
    const problems: string[] = [];

    if (size > MAX_CERTIFICATE_SIZE) problems.push("certificate-too-large");
    const blocks = [...pem.matchAll(/-----BEGIN ([A-Z0-9 ]+)-----/g)].map((m) => m[1]);
    if (blocks.length === 0) problems.push("certificate-not-pem");
    else if (blocks.some((b) => b !== "CERTIFICATE"))
        problems.push("certificate-not-only-certificates");

    return problems;
}

export function countCertificates(pem: string): number {
    return (pem.match(/-----BEGIN CERTIFICATE-----/g) ?? []).length;
}
