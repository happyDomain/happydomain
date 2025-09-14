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

import { parseKeyValueTxt } from "$lib/dns";

export interface DKIMValue {
    v?: string;
    g?: string;
    h?: string[];
    k?: string;
    n?: string;
    p?: string;
    s?: string[];
    t?: string[];
    f?: string[];
    [key: string]: any;
}

export function parseDKIM(val: string): DKIMValue {
    const kv = parseKeyValueTxt(val) as any;

    return {
        ...kv,
        h: kv.h ? kv.h.split(":") : [],
        s: kv.s ? kv.s.split(":") : [],
        t: kv.t ? kv.t.split(":") : [],
        f: kv.f ? kv.f.split(":") : [],
    };
}

export function stringifyDKIM(val: DKIMValue, existingValue: string = ""): string {
    const sep = (existingValue.indexOf("; ") >= 0 ? "; " : ";");

    return "v=" + (val.v || "DKIM1") +
              (val.g ? sep + "g=" + val.g : "") +
              (val.h?.length ? sep + "h=" + val.h.join(":") : "") +
              (val.k ? sep + "k=" + val.k : "") +
              (val.n ? sep + "n=" + val.n : "") +
              (val.p ? sep + "p=" + val.p : "") +
              (val.s?.length ? sep + "s=" + val.s.join(":") : "") +
              (val.t?.length ? sep + "t=" + val.t.join(":") : "") +
              (val.f?.length ? sep + "f=" + val.f.join(":") : "");
}
