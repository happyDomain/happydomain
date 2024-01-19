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

import type { Domain } from '$lib/model/domain';

export const dns_common_types: Array<string> = ['ANY', 'A', 'AAAA', 'NS', 'SRV', 'MX', 'TXT', 'SOA'];

export function fqdn(input: string, origin: string) {
    if (input[-1] === '.') {
        return input
    } else if (input === '') {
        return origin
    } else {
        return input + '.' + origin
    }
}

export function domainCompare (a: string | Domain, b: string | Domain) {
    // Convert to string if Domain
    if (typeof a === "object" && a.domain) a = a.domain;
    if (typeof b === "object" && b.domain) b = b.domain;

    const as = a.split('.').reverse();
    const bs = b.split('.').reverse();

    // Remove first item if empty
    if (!as[0].length) as.shift();
    if (!bs[0].length) bs.shift();

    const maxDepth = Math.min(as.length, bs.length)
    for (let i = 0; i < maxDepth; i++) {
        const cmp = as[i].localeCompare(bs[i])
        if (cmp !== 0) {
            return cmp;
        }
    }

    return as.length - bs.length
}

export function fqdnCompare (a: string | Domain, b: string | Domain) {
    // Convert to string if Domain
    if (typeof a === "object" && a.domain) a = a.domain;
    if (typeof b === "object" && b.domain) b = b.domain;

    const as = a.split('.').reverse();
    const bs = b.split('.').reverse();

    // Remove first item if empty
    if (!as[0].length) as.shift();
    if (!bs[0].length) bs.shift();

    const maxDepth = Math.min(as.length, bs.length)
    for (let i = Math.min(maxDepth, 1); i < maxDepth; i++) {
        const cmp = as[i].localeCompare(bs[i])
        if (cmp !== 0) {
            return cmp;
        } else if (i == 1) {
            const cmp = as[0].localeCompare(bs[0]);
            if (cmp !== 0) {
                return cmp;
            }
        }
    }

    return as.length - bs.length
}

export function nsclass(input: number): string {
  switch (input) {
    case 1:
      return 'IN'
    case 3:
      return 'CH'
    case 4:
      return 'HS'
    case 254:
      return 'NONE'
    default:
      return '##'
  }
}

export function nsttl(input: number): string {
    let ret = '';

    if (input / 86400 >= 1) {
        ret = Math.floor(input / 86400) + 'd '
        input = input % 86400
    }
    if (input / 3600 >= 1) {
        ret = Math.floor(input / 3600) + 'h '
        input = input % 3600
    }
    if (input / 60 >= 1) {
        ret = Math.floor(input / 60) + 'm '
        input = input % 60
    }
    if (input >= 1) {
        ret = Math.floor(input) + 's'
    }

    return ret
}

export function nsrrtype(input: number | string): string {
  switch (input) {
    case '1': case 1: return 'A'
    case '2': case 2: return 'NS'
    case '3': case 3: return 'MD'
    case '4': case 4: return 'MF'
    case '5': case 5: return 'CNAME'
    case '6': case 6: return 'SOA'
    case '7': case 7: return 'MB'
    case '8': case 8: return 'MG'
    case '9': case 9: return 'MR'
    case '10': case 10: return 'NULL'
    case '11': case 11: return 'WKS'
    case '12': case 12: return 'PTR'
    case '13': case 13: return 'HINFO'
    case '14': case 14: return 'MINFO'
    case '15': case 15: return 'MX'
    case '16': case 16: return 'TXT'
    case '17': case 17: return 'RP'
    case '18': case 18: return 'AFSDB'
    case '19': case 19: return 'X25'
    case '20': case 20: return 'ISDN'
    case '21': case 21: return 'RT'
    case '22': case 22: return 'NSAP'
    case '23': case 23: return 'NSAP-PTR'
    case '24': case 24: return 'SIG'
    case '25': case 25: return 'KEY'
    case '26': case 26: return 'PX'
    case '27': case 27: return 'GPOS'
    case '28': case 28: return 'AAAA'
    case '29': case 29: return 'LOC'
    case '30': case 30: return 'NXT'
    case '31': case 31: return 'EID'
    case '32': case 32: return 'NIMLOC'
    case '33': case 33: return 'SRV'
    case '34': case 34: return 'ATMA'
    case '35': case 35: return 'NAPTR'
    case '36': case 36: return 'KX'
    case '37': case 37: return 'CERT'
    case '38': case 38: return 'A6'
    case '39': case 39: return 'DNAME'
    case '40': case 40: return 'SINK'
    case '41': case 41: return 'OPT'
    case '42': case 42: return 'APL'
    case '43': case 43: return 'DS'
    case '44': case 44: return 'SSHFP'
    case '45': case 45: return 'IPSECKEY'
    case '46': case 46: return 'RRSIG'
    case '47': case 47: return 'NSEC'
    case '48': case 48: return 'DNSKEY'
    case '49': case 49: return 'DHCID'
    case '50': case 50: return 'NSEC3'
    case '51': case 51: return 'NSEC3PARAM'
    case '52': case 52: return 'TLSA'
    case '53': case 53: return 'SMIMEA'
    case '55': case 55: return 'HIP'
    case '56': case 56: return 'NINFO'
    case '57': case 57: return 'RKEY'
    case '58': case 58: return 'TALINK'
    case '59': case 59: return 'CDS'
    case '60': case 60: return 'CDNSKEY'
    case '61': case 61: return 'OPENPGPKEY'
    case '62': case 62: return 'CSYNC'
    case '63': case 63: return 'ZONEMD'
    case '99': case 99: return 'SPF'
    case '100': case 100: return 'UINFO'
    case '101': case 101: return 'UID'
    case '102': case 102: return 'GID'
    case '103': case 103: return 'UNSPEC'
    case '104': case 104: return 'NID'
    case '105': case 105: return 'L32'
    case '106': case 106: return 'L64'
    case '107': case 107: return 'LP'
    case '108': case 108: return 'EUI48'
    case '109': case 109: return 'EUI64'
    case '249': case 249: return 'TKEY'
    case '250': case 250: return 'TSIG'
    case '251': case 251: return 'IXFR'
    case '252': case 252: return 'AXFR'
    case '253': case 253: return 'MAILB'
    case '254': case 254: return 'MAILA'
    case '256': case 256: return 'URI'
    case '257': case 257: return 'CAA'
    case '258': case 258: return 'AVC'
    case '259': case 259: return 'DOA'
    case '260': case 260: return 'AMTRELAY'
    case '32768': case 32768: return 'TA'
    case '32769': case 32769: return 'DLV'
    default: return '#'
  }
}

export function validateDomain(dn: string, origin: string = "", hostname: boolean = false): boolean | undefined {
    let ret: boolean | undefined = undefined;
    if (dn.length !== 0) {
        dn = fqdn(dn, origin);
        if (!dn.endsWith(origin)) {
            return false;
        }

        ret = dn.length >= 1 && dn.length <= 254;

        if (ret) {
            const domains = dn.split('.');

            // Remove the last . if any, it's ok
            if (domains[domains.length - 1] === '') {
                domains.pop();
            }

            let newDomainState: boolean = ret
            domains.forEach(function (domain) {
                newDomainState = newDomainState && domain.length >= 1 && domain.length <= 63;
                newDomainState = newDomainState && (!hostname || /^(\*|_?[a-zA-Z0-9]([a-zA-Z0-9-]?[a-zA-Z0-9])*)$/.test(domain));
            })
            ret = newDomainState;
        }
    }

    return ret;
}
