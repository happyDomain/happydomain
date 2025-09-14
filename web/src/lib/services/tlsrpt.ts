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
import type { dnsTypeTXT } from "$lib/dns_rr";

export interface TLSRPTRecord {
    v?: string;
    rua: string[];
}

export function parseTLSRPT(val: string): TLSRPTRecord {
    const kv = parseKeyValueTxt(val);

    return {
        v: kv.v,
        rua: kv.rua ? kv.rua.split(",") : [],
    };
}

export function stringifyTLSRPT(val: TLSRPTRecord, existingValue: string = ""): string {
    const sep = (existingValue.indexOf("; ") >= 0 ? "; " : ";");

    return "v=" + (val.v ? val.v : "TLSRPTv1") + (val.rua && val.rua.length ? sep + "rua=" + val.rua.join(",") : "");
}

export class TLSRPTPolicy {
    private txtRecord: dnsTypeTXT;
    private parsedValue: TLSRPTRecord;

    constructor(txtRecord: dnsTypeTXT) {
        this.txtRecord = txtRecord;
        this.parsedValue = parseTLSRPT(txtRecord.Txt);
    }

    get v(): string | undefined {
        return this.parsedValue.v;
    }

    set v(value: string | undefined) {
        this.parsedValue.v = value;
        this.updateTxtRecord();
    }

    get rua(): string[] {
        return this.parsedValue.rua;
    }

    set rua(value: string[]) {
        this.parsedValue.rua = value;
        this.updateTxtRecord();
    }

    private updateTxtRecord(): void {
        this.txtRecord.Txt = stringifyTLSRPT(this.parsedValue, this.txtRecord.Txt);
    }

    addRua(uri: string): void {
        this.parsedValue.rua.push(uri);
        this.updateTxtRecord();
    }

    removeRua(index: number): void {
        this.parsedValue.rua.splice(index, 1);
        this.updateTxtRecord();
    }

    updateRua(index: number, uri: string): void {
        this.parsedValue.rua[index] = uri;
        this.updateTxtRecord();
    }
}
