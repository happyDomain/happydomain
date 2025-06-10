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

export const resolvers: Record<string, Array<{ value: string; text: string }>> = {
    Unfiltered: [
        { value: "local", text: "Local resolver" },
        { value: "1.1.1.1", text: "Cloudflare DNS resolver" },
        { value: "4.2.2.1", text: "Level3 resolver" },
        { value: "8.8.8.8", text: "Google Public DNS resolver" },
        { value: "9.9.9.10", text: "Quad9 DNS resolver without security blocklist" },
        { value: "64.6.64.6", text: "Verisign DNS resolver" },
        { value: "74.82.42.42", text: "Hurricane Electric DNS resolver" },
        { value: "208.67.222.222", text: "OpenDNS resolver" },
        { value: "8.26.56.26", text: "Comodo Secure DNS resolver" },
        { value: "199.85.126.10", text: "Norton ConnectSafe DNS resolver" },
        { value: "198.54.117.10", text: "SafeServe DNS resolver" },
        { value: "84.200.69.80", text: "DNS.WATCH resolver" },
        { value: "185.121.177.177", text: "OpenNIC DNS resolver" },
        { value: "37.235.1.174", text: "FreeDNS resolver" },
        { value: "80.80.80.80", text: "Freenom World DNS resolver" },
        { value: "216.131.65.63", text: "StrongDNS resolver" },
        { value: "94.140.14.140", text: "AdGuard non-filtering DNS resolver" },
        { value: "91.239.100.100", text: "Uncensored DNS resolver" },
        { value: "216.146.35.35", text: "Dyn DNS resolver" },
        { value: "77.88.8.8", text: "Yandex.DNS resolver" },
        { value: "129.250.35.250", text: "NTT DNS resolver" },
        { value: "223.5.5.5", text: "AliDNS resolver" },
        { value: "1.2.4.8", text: "CNNIC SDNS resolver" },
        { value: "119.29.29.29", text: "DNSPod resolver" },
        { value: "114.215.126.16", text: "oneDNS resolver" },
        { value: "124.251.124.251", text: "cloudxns resolver" },
        { value: "114.114.114.114", text: "Baidu DNS resolver" },
        { value: "156.154.70.1", text: "DNS Advantage resolver" },
        { value: "87.118.111.215", text: "FoolDNS resolver" },
        { value: "101.101.101.101", text: "Quad 101 DNS resolver" },
        { value: "114.114.114.114", text: "114DNS resolver" },
        { value: "168.95.1.1", text: "HiNet DNS resolver" },
        { value: "80.67.169.12", text: "French Data Network DNS resolver" },
        { value: "81.218.119.11", text: "GreenTeamDNS resolver" },
        { value: "208.76.50.50", text: "SmartViper DNS resolver" },
        { value: "23.253.163.53", text: "Alternate DNS resolver" },
        { value: "109.69.8.51", text: "puntCAT DNS resolver" },
        { value: "156.154.70.1", text: "Neustar DNS resolver" },
        { value: "101.226.4.6", text: "DNSpai resolver" },
        { value: "185.222.222.222", text: "DNS.SB resolver" },
        { value: "86.54.11.100", text: "DNS4EU resolver" },
        { value: "194.0.5.3", text: "DNS4ALL resolver" },
        // Your open resolver here? Don't hesitate to contribute to the project!
    ],
    Filtered: [
        { value: "1.1.1.2", text: "Cloudflare Malware Blocking Only DNS resolver" },
        {
            value: "1.1.1.3",
            text: "Cloudflare Malware and Adult Content Blocking Only DNS resolver",
        },
        { value: "9.9.9.9", text: "Quad9 DNS resolver" },
        { value: "94.140.14.14", text: "AdGuard default DNS resolver" },
        { value: "94.140.14.15", text: "AdGuard family protection DNS resolver" },
        { value: "77.88.8.2", text: "Yandex.DNS Safe resolver" },
        { value: "77.88.8.3", text: "Yandex.DNS Family resolver" },
        { value: "156.154.70.2", text: "DNS Advantage Threat Protection resolver" },
        { value: "156.154.70.3", text: "DNS Advantage Family Secure resolver" },
        { value: "156.154.70.4", text: "DNS Advantage Business Secure resolver" },
        { value: "185.228.168.168", text: "CleanBrowsing Family Filter DNS resolver" },
        { value: "185.228.168.10", text: "CleanBrowsing Adult Filter DNS resolver" },
        { value: "86.54.11.1", text: "DNS4EU Protective Resolution resolver" },
        { value: "86.54.11.12", text: "DNS4EU Child Protection resolver" },
        { value: "86.54.11.13", text: "DNS4EU Ad blocking resolver" },
        // Your open resolver here? Don't hesitate to contribute to the project!
    ],
};

export function recordsFields(rrtype: number): Array<string> {
    switch (rrtype) {
        case 1:
            return ["A"];
        case 2:
            return ["Ns"];
        case 5:
            return ["Target"];
        case 6:
            return ["Ns", "Mbox", "Serial", "Refresh", "Retry", "Expire", "Minttl"];
        case 12:
            return ["Ptr"];
        case 13:
            return ["Cpu", "Os"];
        case 15:
            return ["Mx", "Preference"];
        case 16:
        case 99:
            return ["Txt"];
        case 28:
            return ["AAAA"];
        case 33:
            return ["Target", "Port", "Priority", "Weight"];
        case 43:
            return ["KeyTag", "Algorithm", "DigestType", "Digest"];
        case 44:
            return ["Algorithm", "Type", "FingerPrint"];
        case 46:
            return [
                "TypeCovered",
                "Algorithm",
                "Labels",
                "OrigTtl",
                "Expiration",
                "Inception",
                "KeyTag",
                "SignerName",
                "Signature",
            ];
        case 52:
            return ["Usage", "Selector", "MatchingType", "Certificate"];
        default:
            console.warn("Unknown RRtype asked fields: ", rrtype);
            return [];
    }
}
