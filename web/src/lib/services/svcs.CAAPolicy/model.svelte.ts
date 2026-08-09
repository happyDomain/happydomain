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

export type CAATag = "issue" | "issuewild" | "issuemail" | "issuevmc" | "iodef" | "contactemail" | "contactphone";

/** The tags that name a kind of certificate, in the order the editor shows them. */
export const CAA_ISSUE_TAGS = ["issue", "issuewild", "issuemail", "issuevmc"] as const;

export type CAAIssueTag = (typeof CAA_ISSUE_TAGS)[number];

/**
 * What the policy currently says about one kind of certificate:
 *  - "any": nothing is published, so every authority may issue. For issuewild,
 *    this means the rule set for regular certificates applies instead.
 *  - "restricted": only the listed authorities may issue.
 *  - "none": the ";" pseudo-issuer forbids everyone.
 */
export type CAAMode = "any" | "restricted" | "none";

/** A record of the RRset, keeping a link back to its index. */
export interface CAAEntry {
    index: number;
    record: dnsTypeCAA;
}

/** The ";" value: a syntactically valid issuer name that no CA can match. */
const DENY_ALL = ";";

// CAA Issuer types
export class CAAIssuer {
    IssuerDomainName = $state<string | undefined>(undefined);
    Parameters = $state<string[]>([]);

    constructor(issuerDomainName?: string, parameters: string[] = []) {
        this.IssuerDomainName = issuerDomainName;
        this.Parameters = parameters;
    }
}

export class CAAParameter {
    Tag = $state<string>("");
    Value = $state<string>("");

    constructor(tag: string = "", value: string = "") {
        this.Tag = tag;
        this.Value = value;
    }
}

export class CAAIodef {
    kind = $state<string>("");
    url = $state<string>("");

    constructor(kind: string = "", url: string = "") {
        this.kind = kind;
        this.url = url;
    }
}

export function newCAARecord(dn: string, tag: CAATag, value: string): dnsTypeCAA {
    const rr = newRR(dn, getRrtype("CAA")) as dnsTypeCAA;
    rr.Tag = tag;
    rr.Value = value;
    return rr;
}

/**
 * Editable view over the CAA RRset. The records array is the source of truth:
 * the editor mutates it in place, exactly like ForSaleService.
 */
export class CAAPolicy {
    records = $state<Array<dnsTypeCAA>>([]);
    dn: string;

    constructor(records: dnsResource, dn: string = "") {
        if (records["caa"]) {
            this.records = Array.isArray(records["caa"]) ? records["caa"] : [records["caa"]];
        } else {
            this.records = [];
        }
        this.dn = dn;
    }

    /** Every record carrying that tag, deny marker included. */
    entries(tag: CAATag): Array<CAAEntry> {
        return this.records
            .map((record, index) => ({ index, record }))
            .filter((e) => e.record.Tag === tag);
    }

    /** The authorities the user picked, so the deny marker is left out. */
    issuers(tag: CAATag): Array<CAAEntry> {
        return this.entries(tag).filter((e) => e.record.Value.trim() !== DENY_ALL);
    }

    isDenied(tag: CAATag): boolean {
        return this.entries(tag).some((e) => e.record.Value.trim() === DENY_ALL);
    }

    mode(tag: CAAIssueTag): CAAMode {
        if (this.isDenied(tag)) return "none";
        if (this.issuers(tag).length) return "restricted";
        return "any";
    }

    /**
     * Rewrite the records of a tag to express the requested mode. Switching to
     * "restricted" keeps the authorities already listed, so the user can toggle
     * a kind off and back on without losing their choices.
     */
    setMode(tag: CAAIssueTag, mode: CAAMode): void {
        if (mode === "restricted") {
            this.removeAll(this.entries(tag).filter((e) => e.record.Value.trim() === DENY_ALL));
            return;
        }

        this.removeAll(this.entries(tag));
        if (mode === "none") this.add(tag, DENY_ALL);
    }

    add(tag: CAATag, value: string): void {
        this.records.push(newCAARecord(this.dn, tag, value));
    }

    remove(index: number): void {
        this.records.splice(index, 1);
    }

    /** Drop several records at once, from the last one, so indexes stay valid. */
    private removeAll(entries: Array<CAAEntry>): void {
        for (const entry of [...entries].reverse()) {
            this.remove(entry.index);
        }
    }
}

// CAA Issuer parsing/stringifying
export function parseCAAIssuer(val: string, newone: boolean = false): CAAIssuer {
    const fields = val.split(";");

    return new CAAIssuer(
        !fields[0] && newone ? undefined : fields[0],
        fields.length > 1 ? fields.slice(1) : []
    );
}

export function stringifyCAAIssuer(val: CAAIssuer, existingValue: string = ""): string {
    const sep = (existingValue && existingValue.indexOf("; ") >= 0 ? "; " : ";");

    return val.IssuerDomainName === undefined ? "" : (val.IssuerDomainName + (val.Parameters.length ? sep + val.Parameters.join(sep) : ""));
}

// CAA Parameter parsing/stringifying
export function parseCAAParameter(val: string): CAAParameter {
    const fields = val.split("=");

    return new CAAParameter(
        fields[0],
        fields.length > 1 ? fields.slice(1).join("=") : ""
    );
}

export function stringifyCAAParameter(val: CAAParameter): string {
    if (val.Tag === "" && val.Value === "") return "";
    return val.Tag + "=" + val.Value;
}

// CAA Iodef parsing/stringifying
export function parseCAAIodef(val: string): CAAIodef {
    const fields = val.split(":");

    return new CAAIodef(
        fields[0].replace(/s$/, ""),
        fields[0] === "mailto" ? fields.slice(1).join(":") : fields.join(":")
    );
}

export function stringifyCAAIodef(val: CAAIodef): string {
    return val.kind === "mailto" ? (val.kind + ":" + val.url) : val.url;
}
