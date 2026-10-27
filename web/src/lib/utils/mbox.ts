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

// A mailbox in the DNS (SOA RNAME, RP) is the address with its "@" turned
// into a dot, and the dots of the local part escaped: john.doe@example.com is
// written john\.doe.example.com. Names not ending with a dot are relative to
// the origin of the zone, as happyDomain stores them.

function trimDot(domain: string): string {
    return domain.replace(/\.$/, "").toLowerCase();
}

/**
 * Turn a DNS mailbox into the e-mail address it stands for.
 * @param mbox Mailbox as stored, relative to origin unless it ends with a dot
 * @param origin Origin of the zone
 * @returns The local part and domain of the address, or null for an empty mailbox
 */
export function mboxToEmail(
    mbox: string,
    origin: string,
): { local: string; domain: string } | null {
    if (mbox === "" || mbox === "@") return null;

    const m = mbox.match(/^((?:[^.\\]|\\.)*)(?:\.(.*))?$/);
    if (!m) return null;

    const local = m[1].replace(/\\(.)/g, "$1");
    const rest = m[2] ?? "";
    const domain = mbox.endsWith(".")
        ? trimDot(rest)
        : [rest, trimDot(origin)].filter((d) => d !== "").join(".");
    return { local, domain };
}

/**
 * Turn what the user typed into a DNS mailbox. Without "@", the address is
 * taken to be in the domain of the zone.
 * @param email Address, or its local part alone
 * @param origin Origin of the zone
 * @returns Mailbox relative to origin when no domain was typed, absolute
 * otherwise, or null when the text is not an address
 */
export function emailToMbox(email: string, origin: string): string | null {
    const text = email.trim();
    if (/\s/.test(text)) return null;

    const at = text.lastIndexOf("@");
    const local = at < 0 ? text : text.slice(0, at);
    const domain = at < 0 ? "" : text.slice(at + 1).replace(/\.$/, "");
    if (local === "" || local.includes("@") || (at >= 0 && domain === "")) return null;
    if (domain !== "" && domain.split(".").some((label) => label === "")) return null;
    if (at < 0 && trimDot(origin) === "") return null;

    const escaped = local.replace(/\\/g, "\\\\").replace(/\./g, "\\.");
    return domain === "" ? escaped : escaped + "." + domain + ".";
}
