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
    import type { Domain } from "$lib/model/domain";
    import type { dnsResource, dnsTypeSRV } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "abstract.Kerberos";

    // Each bucket maps to one field on the Go service body. The `key` here
    // matches the JSON tag set on the Kerberos struct (see
    // services/abstract/kerberos.go).
    const buckets = [
        { key: "kdc_tcp", prefix: "_kerberos._tcp", label: "KDC (TCP)" },
        { key: "kdc_udp", prefix: "_kerberos._udp", label: "KDC (UDP)" },
        { key: "master", prefix: "_kerberos-master._tcp", label: "Master KDC" },
        { key: "admin", prefix: "_kerberos-adm._tcp", label: "Admin server (kadmin)" },
        { key: "kpasswd_tcp", prefix: "_kpasswd._tcp", label: "Password change (TCP)" },
        { key: "kpasswd_udp", prefix: "_kpasswd._udp", label: "Password change (UDP)" },
    ];

    // Initialize empty arrays for buckets the server omitted.
    for (const b of buckets) {
        if (!(value as any)[b.key]) {
            (value as any)[b.key] = [];
        }
    }

    // Keep each record's Hdr.Name pinned to its bucket prefix so we don't
    // accidentally write mis-labeled SRV records.
    $effect(() => {
        for (const b of buckets) {
            const arr = (value as any)[b.key] as Array<dnsTypeSRV> | undefined;
            if (!arr) continue;
            for (const record of arr) {
                if (record?.Hdr && record.Hdr.Name !== b.prefix) {
                    record.Hdr.Name = b.prefix;
                }
            }
        }
    });
</script>

{#each buckets as bucket}
    <div class="mb-4">
        <h5 class="pb-1 border-bottom border-1">{bucket.label}</h5>
        <TableRecords
            class="mt-3"
            dn={bucket.prefix}
            edit
            {origin}
            bind:rrs={(value as any)[bucket.key]}
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
                {@const bucketArray = (value as any)[bucket.key] as Array<dnsTypeSRV>}
                {#if bucketArray && bucketArray[idx]}
                    <RawInput
                        edit
                        index={bucket.key + idx.toString()}
                        specs={{
                            id: field,
                            type: field == "Target" ? "string" : "uint16",
                        }}
                        bind:value={bucketArray[idx][field as keyof dnsTypeSRV]}
                    />
                {/if}
            {/snippet}
        </TableRecords>
    </div>
{/each}
