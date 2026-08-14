// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

/**
 * RFC 2782 checks, shared by every service built on SRV records.
 *
 * They all describe the same thing, a host and a port reachable for a given
 * service, but each one names its own field: svcs.UnknownSRV publishes them
 * under `srv`, most of the abstract services under `records`, and Kerberos
 * spreads them over six keys, one per role. Hence a validator taking the field
 * names to read, and a single set of `srv.*` messages for all of them.
 */

import {
    asArray,
    type ComplianceContext,
    type ComplianceIssue,
    hasAddress,
    inZoneSubdomain,
    isCnameOwner,
    isUint16,
    isValidHostname,
    normalizeFqdn,
    originFqdn,
    registerValidators,
} from "$lib/services/compliance";

interface SRV {
    Priority: number;
    Weight: number;
    Port: number;
    Target: string;
    Hdr?: { Name?: string };
}

const RFC2782 = "https://www.rfc-editor.org/rfc/rfc2782";

// RFC 2782: the owner of an SRV is _service._proto followed by the name it
// serves. The protocol is not restricted to tcp and udp, so only the shape is
// checked here.
const SRV_OWNER_RE = /^_[^.]+\._[^.]+(\.|$)/;

/** An SRV record, kept alongside its position in the field it came from. */
interface SRVEntry {
    record: SRV;
    index: number;
}

/** Checks one set of SRV records sharing the same owner name. */
export function srvIssues(
    entries: SRVEntry[],
    ctx: ComplianceContext,
    field: string,
): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    if (entries.length === 0) return issues;

    const records = entries.map((e) => e.record);
    const origin = originFqdn(ctx);
    const owner = (records[0].Hdr?.Name ?? "").trim();

    if (owner !== "" && !SRV_OWNER_RE.test(owner)) {
        issues.push({
            id: "srv.invalid-owner",
            severity: "error",
            params: { name: owner },
            field,
            docUrl: RFC2782,
        });
    }

    // RFC 2782: a target of "." says the service is decidedly not available
    // here. It only means that when it stands alone.
    const unavailable = records.filter((r) => (r.Target ?? "").trim() === ".");
    if (unavailable.length > 0 && unavailable.length !== records.length) {
        issues.push({
            id: "srv.unavailable-with-others",
            severity: "error",
            field,
            docUrl: RFC2782,
        });
    }

    const seen = new Set<string>();
    const weightsByPriority = new Map<number, number[]>();

    entries.forEach(({ record: r, index }) => {
        const target = (r.Target ?? "").trim();
        const at = `${field}[${index}]`;

        for (const [id, value] of [
            ["srv.invalid-priority", r.Priority],
            ["srv.invalid-weight", r.Weight],
            ["srv.invalid-port", r.Port],
        ] as const) {
            if (!isUint16(value)) {
                issues.push({ id, severity: "error", params: { value: String(value) }, field: at });
            }
        }

        if (target === ".") return;

        if (isUint16(r.Priority) && isUint16(r.Weight)) {
            const weights = weightsByPriority.get(r.Priority) ?? [];
            weights.push(r.Weight);
            weightsByPriority.set(r.Priority, weights);
        }

        const norm = normalizeFqdn(target);
        if (norm === "" || !isValidHostname(norm)) {
            issues.push({
                id: "srv.invalid-target",
                severity: "error",
                params: { target },
                field: at,
            });
            return;
        }

        // Port 0 is how a target of "." is spelled out, it makes no sense on a
        // real host.
        if (r.Port === 0) {
            issues.push({ id: "srv.port-zero", severity: "error", field: at, docUrl: RFC2782 });
        }

        const key = `${norm}:${r.Port}`;
        if (seen.has(key)) {
            issues.push({
                id: "srv.duplicate",
                severity: "warning",
                params: { target: norm, port: r.Port },
                field: at,
            });
        }
        seen.add(key);

        const sub = inZoneSubdomain(norm, origin);
        if (sub === null) return;

        if (isCnameOwner(ctx, sub)) {
            issues.push({
                id: "srv.target-is-cname",
                severity: "error",
                params: { target: norm },
                field: at,
                docUrl: RFC2782,
            });
        } else if (!hasAddress(ctx, sub)) {
            issues.push({
                id: "srv.target-no-address",
                severity: "warning",
                params: { target: norm },
                field: at,
            });
        }
    });

    // Within a priority level, the load is shared in proportion to the weights.
    // A zero next to non-zero weights means that record is only ever picked
    // when every other one is unreachable, which is rarely the intent.
    for (const [priority, weights] of weightsByPriority) {
        if (weights.length > 1 && weights.some((w) => w === 0) && weights.some((w) => w > 0)) {
            issues.push({
                id: "srv.zero-weight-mixed",
                severity: "info",
                params: { priority },
                field,
                docUrl: RFC2782,
            });
        }
    }

    return issues;
}

/**
 * Registers the RFC 2782 checks for a service, reading its records from the
 * given body fields. Records are grouped by owner name: two SRV sets of the
 * same service (_sip._tcp and _sip._udp) are independent of one another.
 */
export function registerSrvValidators(svctype: string, ...fields: string[]): void {
    registerValidators(svctype, {
        sync(raw, ctx) {
            const issues: ComplianceIssue[] = [];

            for (const field of fields) {
                const records = asArray<SRV>(raw?.[field]).filter(
                    (r) => r && typeof r === "object",
                );

                const byOwner = new Map<string, SRVEntry[]>();
                records.forEach((record, index) => {
                    const owner = (record.Hdr?.Name ?? "").trim().toLowerCase();
                    const set = byOwner.get(owner);
                    if (set) {
                        set.push({ record, index });
                    } else {
                        byOwner.set(owner, [{ record, index }]);
                    }
                });

                for (const set of byOwner.values()) {
                    issues.push(...srvIssues(set, ctx, field));
                }
            }

            return issues;
        },
    });
}
