// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
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

import { nsrrtype, rdatatostr } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";

export { nsrrtype, rdatatostr };

export const dns_common_types: Array<string> = [
    "ANY",
    "A",
    "AAAA",
    "NS",
    "SRV",
    "MX",
    "TXT",
    "SOA",
];

export function fqdn(input: string, origin: string) {
    if (input === "@") {
        return origin;
    } else if (input[-1] === ".") {
        return input;
    } else if (input === "") {
        return origin;
    } else {
        return input + "." + origin;
    }
}

export function domainCompare(a: string | Domain, b: string | Domain) {
    // Convert to string if Domain
    let domainA = typeof a === "object" ? a.domain : a;
    let domainB = typeof b === "object" ? b.domain : b;

    const as = domainA.split(".").reverse();
    const bs = domainB.split(".").reverse();

    // Remove first item if empty
    if (!as[0].length) as.shift();
    if (!bs[0].length) bs.shift();

    const maxDepth = Math.min(as.length, bs.length);
    for (let i = 0; i < maxDepth; i++) {
        const cmp = as[i].localeCompare(bs[i]);
        if (cmp !== 0) {
            return cmp;
        }
    }

    return as.length - bs.length;
}

export function fqdnCompare(a: string | Domain, b: string | Domain) {
    // Convert to string if Domain
    let domainA = typeof a === "object" ? a.domain : a;
    let domainB = typeof b === "object" ? b.domain : b;

    const as = domainA.split(".").reverse();
    const bs = domainB.split(".").reverse();

    // Remove first item if empty
    if (!as[0].length) as.shift();
    if (!bs[0].length) bs.shift();

    const maxDepth = Math.min(as.length, bs.length);
    for (let i = Math.min(maxDepth, 1); i < maxDepth; i++) {
        const cmp = as[i].localeCompare(bs[i]);
        if (cmp !== 0) {
            return cmp;
        } else if (i == 1) {
            const cmp = as[0].localeCompare(bs[0]);
            if (cmp !== 0) {
                return cmp;
            }
        }
    }

    return as.length - bs.length;
}

export function nsclass(input: number): string {
    switch (input) {
        case 1:
            return "IN";
        case 3:
            return "CH";
        case 4:
            return "HS";
        case 254:
            return "NONE";
        default:
            return "##";
    }
}

export function nsttl(input: number): string {
    let ret = "";

    if (input / 86400 >= 1) {
        ret = Math.floor(input / 86400) + "d ";
        input = input % 86400;
    }
    if (input / 3600 >= 1) {
        ret = Math.floor(input / 3600) + "h ";
        input = input % 3600;
    }
    if (input / 60 >= 1) {
        ret = Math.floor(input / 60) + "m ";
        input = input % 60;
    }
    if (input >= 1) {
        ret = Math.floor(input) + "s";
    }

    return ret;
}

export function validateDomain(
    dn: string,
    origin: string = "",
    hostname: boolean = false,
): boolean | undefined {
    let ret: boolean | undefined = undefined;
    if (dn.length !== 0) {
        dn = fqdn(dn, origin);
        if (!dn.endsWith(origin)) {
            return false;
        }

        ret = dn.length >= 1 && dn.length <= 254;

        if (ret) {
            const domains = dn.split(".");

            // Remove the last . if any, it's ok
            if (domains[domains.length - 1] === "") {
                domains.pop();
            }

            let newDomainState: boolean = ret;
            domains.forEach(function (domain) {
                newDomainState = newDomainState && domain.length >= 1 && domain.length <= 63;
                newDomainState =
                    newDomainState &&
                    (!hostname || /^(\*|_?[a-zA-Z0-9]([a-zA-Z0-9-]?[a-zA-Z0-9])*)$/.test(domain));
            });
            ret = newDomainState;
        }
    }

    return ret;
}

export function isReverseZone(fqdn: string) {
    return fqdn.endsWith("in-addr.arpa.") || fqdn.endsWith("ip6.arpa.");
}

export function reverseDomain(ip: string) {
    let suffix = "in-addr.arpa.";

    let fields: Array<string>;
    if (ip.indexOf(":") > 0) {
        suffix = "ip6.arpa.";

        fields = ip.split(":");

        fields = fields.map((e) => {
            let exp_len = 4;
            if (e.length == 0) {
                exp_len = 4 * (7 - fields.length);
            }
            while (e.length < exp_len) {
                e = "0" + e;
            }
            return e;
        });
    } else {
        fields = ip.split(".");
        while (fields.length < 4) {
            const last = fields.pop()!;
            fields.push("0", last);
        }
    }

    return fields.reduce(
        (a, v) => v.replace(/^0*(0|[^0].*)$/, "$1") + "." + a,
        suffix,
    );
}

export function unreverseDomain(dn: string) {
    let split_char = ".";
    let group = 1;

    if (dn.endsWith("ip6.arpa.")) {
        split_char = ":";
        group = 4;
        dn = dn.substring(0, dn.indexOf(".ip6.arpa."));
    } else {
        dn = dn.substring(0, dn.indexOf(".in-addr.arpa."));
    }

    const fields = dn.split(".");
    let ip = fields.reduce((a, v, i) => v + (i % group == 0 ? split_char : "") + a, "");
    ip = ip.substring(0, ip.length - 1);
    return ip.replace(/:(0000:)+/, "::").replace(/:0+/g, ":");
}
