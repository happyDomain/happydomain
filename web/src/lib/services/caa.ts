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

import { getRrtype, newRR, type dnsResource, type dnsTypeCAA } from "$lib/dns_rr";

export type CAATag = "issue" | "issuewild" | "issuemail" | "iodef";

// CAA Issuer types
export interface CAAIssuer {
    IssuerDomainName?: string;
    Parameters: string[];
}

export interface CAAParameter {
    Tag: string;
    Value: string;
}

export interface CAAIodef {
    kind: string;
    url: string;
}

export function newCAARecord(dn: string, tag: CAATag, value: string): dnsTypeCAA {
    const rr = newRR(dn, getRrtype("CAA")) as dnsTypeCAA;
    rr.Tag = tag;
    rr.Value = value;
    return rr;
}

export class CAAPolicy {
    records: Array<dnsTypeCAA>;
    DisallowIssue: boolean;
    DisallowWildcardIssue: boolean;
    DisallowMailIssue: boolean;

    constructor(records: dnsResource) {
        if (records["caa"]) {
            this.records = records["caa"];
        } else {
            this.records = [];
        }
        this.DisallowIssue = false;
        this.DisallowWildcardIssue = false;
        this.DisallowMailIssue = false;
        this.refreshDisallowIssue();
    }

    hasDisallowIssue(tag: CAATag): boolean {
        for (const record of this.records) {
            if (record.Tag == tag && record.Value.trim() == ";") {
                return true;
            }
        }
        return false;
    }

    refreshDisallowIssue(): void {
        this.DisallowIssue = this.hasDisallowIssue("issue");
        this.DisallowWildcardIssue = this.hasDisallowIssue("issuewild");
        this.DisallowMailIssue = this.hasDisallowIssue("issuemail");
    }

    changeDisallowIssue(dn: string, tag: CAATag): (e: Event) => void {
        return (e: Event) => {
            const target = e.target as HTMLInputElement;
            if (target && target.checked) {
                this.records.push(newCAARecord(dn, tag, ";"));
                this.refreshDisallowIssue();
            } else {
                for (let i = this.records.length - 1; i >= 0; i--) {
                    const r = this.records[i];
                    if (r.Tag == tag && r.Value.trim() == ";") {
                        this.records.splice(i, 1);
                    }
                }
                this.refreshDisallowIssue();
            }
        };
    }

    getRecordsByTag(tag: CAATag): Array<dnsTypeCAA> {
        return this.records.filter((r) => r.Tag === tag);
    }

    removeRecord(index: number): void {
        this.records.splice(index, 1);
    }
}

// CAA Issuer parsing/stringifying
export function parseCAAIssuer(val: string, newone: boolean = false): CAAIssuer {
    const fields = val.split(";");

    return {
        IssuerDomainName: !fields[0] && newone ? undefined : fields[0],
        Parameters: fields.length > 1 ? fields.slice(1) : [],
    };
}

export function stringifyCAAIssuer(val: CAAIssuer, existingValue: string = ""): string {
    const sep = (existingValue && existingValue.indexOf("; ") >= 0 ? "; " : ";");

    return val.IssuerDomainName === undefined ? "" : (val.IssuerDomainName + (val.Parameters.length ? sep + val.Parameters.join(sep) : ""));
}

// CAA Parameter parsing/stringifying
export function parseCAAParameter(val: string): CAAParameter {
    const fields = val.split("=");

    return {
        Tag: fields[0],
        Value: fields.length > 1 ? fields.slice(1).join("=") : "",
    };
}

export function stringifyCAAParameter(val: CAAParameter): string {
    if (val.Tag === "" && val.Value === "") return "";
    return val.Tag + "=" + val.Value;
}

// CAA Iodef parsing/stringifying
export function parseCAAIodef(val: string): CAAIodef {
    const fields = val.split(":");

    return {
        kind: fields[0].replace(/s$/, ""),
        url: fields[0] === "mailto" ? fields.slice(1).join(":") : fields.join(":"),
    };
}

export function stringifyCAAIodef(val: CAAIodef): string {
    return val.kind === "mailto" ? (val.kind + ":" + val.url) : val.url;
}
