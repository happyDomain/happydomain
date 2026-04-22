<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script lang="ts">
    import TableRecords from "$lib/components/records/TableRecords.svelte";
    import RawInput from "$lib/components/inputs/raw.svelte";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { dnsResource, dnsTypeSRV, dnsTypeTXT } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: Record<string, any>;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "abstract.CardDAV";

    const valueData = value as dnsResource & {
        records?: dnsTypeSRV[];
        paths?: dnsTypeTXT[];
        "carddavs-tcp"?: dnsTypeSRV[];
        "carddav-tcp"?: dnsTypeSRV[];
        [key: string]: any;
    };

    if (!valueData.records) valueData.records = [];
    if (!valueData.paths) valueData.paths = [];

    // Prefix order: TLS (recommended) before plaintext (legacy).
    const services = [
        { key: "carddavs-tcp", prefix: "_carddavs._tcp", label: "CardDAV over TLS" },
        { key: "carddav-tcp", prefix: "_carddav._tcp", label: "CardDAV (plaintext)" },
    ];

    for (const service of services) {
        valueData[service.key] = (valueData.records ?? []).filter(
            (srv) => srv?.Hdr?.Name?.startsWith(service.prefix) || false,
        );
    }

    $effect(() => {
        const allRecords: dnsTypeSRV[] = [];
        for (const service of services) {
            if (valueData[service.key]) {
                allRecords.push(...valueData[service.key]);
            }
        }
        valueData.records = allRecords;
    });

    // One RFC 6764 §4 "path=" TXT per prefix bucket. The UI edits just the
    // path value; we keep the full TXT in valueData.paths so Hdr (name,
    // TTL) round-trips verbatim on save.
    const pathValues: Record<string, string> = $state({});

    for (const service of services) {
        const existing = (valueData.paths ?? []).find(
            (txt) => txt?.Hdr?.Name?.startsWith(service.prefix),
        );
        const raw = existing?.Txt ?? "";
        pathValues[service.key] = raw.startsWith("path=") ? raw.slice(5) : "";
    }

    $effect(() => {
        const next: dnsTypeTXT[] = [];
        for (const txt of valueData.paths ?? []) {
            const known = services.some((s) => txt?.Hdr?.Name?.startsWith(s.prefix));
            if (!known) next.push(txt);
        }
        for (const service of services) {
            const v = pathValues[service.key];
            if (!v) continue;
            const existing = (valueData.paths ?? []).find(
                (txt) => txt?.Hdr?.Name?.startsWith(service.prefix),
            );
            if (existing) {
                existing.Txt = "path=" + v;
                existing.Hdr.Name = service.prefix;
                next.push(existing);
            } else {
                const rec = newRR(service.prefix, getRrtype("TXT")) as dnsTypeTXT;
                rec.Txt = "path=" + v;
                next.push(rec);
            }
        }
        valueData.paths = next;
    });
</script>

{#each services as service}
    <div class="mb-4">
        <h5 class="pb-1 border-bottom border-1">{service.label}</h5>
        <TableRecords
            class="mt-3"
            dn={service.prefix}
            edit
            {origin}
            bind:rrs={valueData[service.key]}
            rrtype="SRV"
        >
            {#snippet header(field: string)}
                {#if field == "Priority"}
                    Priority
                {:else if field == "Weight"}
                    Weight
                {:else if field == "Port"}
                    Port
                {:else if field == "Target"}
                    Target
                {/if}
            {/snippet}
            {#snippet field(idx: number, field: string)}
                {#if valueData[service.key] && valueData[service.key][idx]}
                    <RawInput
                        edit
                        index={service.key + idx.toString()}
                        specs={{
                            id: field,
                            type: field == "Target" ? "string" : "uint16",
                        }}
                        bind:value={valueData[service.key][idx][field]}
                    />
                {/if}
            {/snippet}
        </TableRecords>
        <BasicInput
            class="mt-2"
            edit
            index={service.key + "-path"}
            specs={{
                id: service.key + "-path",
                label: "Context path (RFC 6764 §4)",
                description: "Optional HTTP path advertised via a companion TXT record at this SRV label.",
                type: "string",
                placeholder: "/carddav",
            }}
            bind:value={pathValues[service.key]}
        />
    </div>
{/each}
