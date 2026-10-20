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
 * Helpers shared by the service editors for the raw DNS records they carry.
 */

/**
 * Reports whether a record is one of the empty stubs the backend service-spec
 * usecase auto-allocates for pointer-to-DNS fields when it serves a
 * freshly-created service (`Hdr.Name == ""`).
 *
 * Editors drop those before reading anything else, otherwise an unedited form
 * would round-trip a phantom record back to the zone.
 */
export function isStubRecord(r: { Hdr?: { Name?: string } } | null | undefined): boolean {
    return r != null && (!r.Hdr || !r.Hdr.Name);
}

/**
 * Appends a trailing dot to a hostname if it doesn't already have one.
 */
export function ensureTrailingDot(host: string): string {
    if (!host) return "";
    return host.endsWith(".") ? host : host + ".";
}
