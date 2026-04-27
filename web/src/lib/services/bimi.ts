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

import { parseKeyValueTxt } from "$lib/dns";

export interface BIMIValue {
    v?: string;
    l?: string;
    a?: string;
    e?: string;
}

export function parseBIMI(val: string): BIMIValue {
    const parsed = parseKeyValueTxt(val);
    return {
        v: parsed.v,
        l: parsed.l,
        a: parsed.a,
        e: parsed.e,
    };
}

/**
 * Detects a BIMI declination record. Per the BIMI draft, a domain that does
 * not wish to participate publishes a record with v=BIMI1 and an explicitly
 * empty l= tag.
 */
export function isBIMIDeclination(val: string): boolean {
    if (!/(?:^|;)\s*v\s*=\s*BIMI\d+/i.test(val)) return false;
    return /(?:^|;)\s*l\s*=\s*(?:;|$)/i.test(val);
}

export function stringifyBIMIDeclination(version: string = "BIMI1"): string {
    return `v=${version};l=`;
}

export function stringifyBIMI(val: BIMIValue, existingTxt: string = ""): string {
    const sep = existingTxt.indexOf("; ") >= 0 ? "; " : ";";

    return (
        "v=" + (val.v || "BIMI1") +
        (val.l ? sep + "l=" + val.l : "") +
        (val.a ? sep + "a=" + val.a : "") +
        (val.e ? sep + "e=" + val.e : "")
    );
}
