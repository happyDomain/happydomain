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

/**
 * Whether `crypto.subtle` is usable.
 *
 * Browsers only expose it in a secure context (HTTPS, or localhost/127.0.0.1),
 * so a self-hosted happyDomain reached over plain HTTP on a LAN has none. The
 * few features relying on it can't work there and should tell the user why
 * rather than fail silently.
 */
export function hasSubtleCrypto(): boolean {
    return (
        typeof crypto !== "undefined" &&
        typeof crypto.subtle !== "undefined" &&
        typeof crypto.subtle.digest === "function"
    );
}

/**
 * Lowercase hexadecimal representation of `bytes`.
 */
export function toHex(bytes: ArrayLike<number>): string {
    return Array.from(bytes)
        .map((b) => b.toString(16).padStart(2, "0"))
        .join("");
}

/**
 * Hex-encoded SHA digest of `data`, or `undefined` outside a secure context.
 */
export async function digestHex(
    algorithm: "SHA-256" | "SHA-512",
    data: BufferSource,
    bytes?: number,
): Promise<string | undefined> {
    if (!hasSubtleCrypto()) return undefined;

    const digest = new Uint8Array(await crypto.subtle.digest(algorithm, data));

    // `slice(0, undefined)` already returns the whole array.
    return toHex(digest.slice(0, bytes));
}
