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

export interface DMARCValue {
    v?: string;
    p?: string;
    sp?: string;
    adkim?: string;
    aspf?: string;
    fo: string[];
    rf: string[];
    ri?: string;
    rua: string[];
    ruf: string[];
    pct?: string | number;
}

export function parseDMARC(val: string): DMARCValue {
    const parsed = parseKeyValueTxt(val);

    return {
        v: parsed.v,
        p: parsed.p,
        sp: parsed.sp,
        adkim: parsed.adkim,
        aspf: parsed.aspf,
        ri: parsed.ri,
        pct: parsed.pct,
        rua: parsed.rua && parsed.rua.length ? parsed.rua.split(",") : [],
        ruf: parsed.ruf && parsed.ruf.length ? parsed.ruf.split(",") : [],
        fo: parsed.fo && parsed.fo.length ? parsed.fo.split(",") : [],
        rf: parsed.rf && parsed.rf.length ? parsed.rf.split(",") : [],
    };
}

export function stringifyDMARC(val: DMARCValue, existingTxt: string = ""): string {
    const sep = (existingTxt.indexOf("; ") >= 0 ? "; " : ";");

    return "v=" + (val.v || "DMARCv1") +
              (val.p ? sep + "p=" + val.p : "") +
              (val.sp ? sep + "sp=" + val.sp : "") +
              (val.adkim ? sep + "adkim=" + val.adkim : "") +
              (val.aspf ? sep + "aspf=" + val.aspf : "") +
              (val.fo.length ? sep + "fo=" + val.fo.join(",") : "") +
              (val.rf.length ? sep + "rf=" + val.rf.join(",") : "") +
              (val.ri ? sep + "ri=" + val.ri : "") +
              (val.rua.length ? sep + "rua=" + val.rua.join(",") : "") +
              (val.ruf.length ? sep + "ruf=" + val.ruf.join(",") : "") +
              (val.pct ? sep + "pct=" + val.pct : "");
}
