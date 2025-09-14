<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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
    const type = "abstract.RFC6186";

    // Initialize srv array if needed (treat as array despite type definition)
    if (!(value as any)["srv"]) {
        (value as any)["srv"] = [];
    }

    // Type-safe accessor for srv as array
    const srvArray = $derived((value as any)["srv"] as Array<dnsTypeSRV>);

    // Service type configurations with their prefixes
    const services = [
        { key: "submission", prefix: "_submission._tcp.", label: "Email Submission" },
        { key: "submissions", prefix: "_submissions._tcp.", label: "Email Submission over TLS (RFC 8314)" },
        { key: "imap", prefix: "_imap._tcp.", label: "IMAP" },
        { key: "imaps", prefix: "_imaps._tcp.", label: "IMAP over TLS" },
        { key: "pop3", prefix: "_pop3._tcp.", label: "POP3" },
        { key: "pop3s", prefix: "_pop3s._tcp.", label: "POP3 over TLS" },
    ];

    // Initialize service arrays from srv (one-time, breaks circular dependency)
    let initialized = $state(false);

    // Initialize service arrays on mount
    $effect(() => {
        if (!initialized && srvArray && srvArray.length > 0) {
            for (const service of services) {
                (value as any)[service.key] = srvArray.filter((srv: dnsTypeSRV) =>
                    srv?.Hdr?.Name?.startsWith(service.prefix) || false
                );
            }
            initialized = true;
        }
    });

    // Initialize empty arrays for services with no records
    for (const service of services) {
        if (!(value as any)[service.key]) {
            (value as any)[service.key] = [];
        }
    }

    // When records in service arrays change, sync back to main array (one-way sync)
    $effect(() => {
        if (initialized) {
            const allRecords: Array<dnsTypeSRV> = [];
            for (const service of services) {
                const serviceArray = (value as any)[service.key] as Array<dnsTypeSRV>;
                if (serviceArray) {
                    allRecords.push(...serviceArray);
                }
            }
            (value as any)["srv"] = allRecords;
        }
    });
</script>

{#each services as service}
    <div class="mb-4">
        <h5 class="pb-1 border-bottom border-1">{service.label}</h5>
        <TableRecords
            class="mt-3"
            dn={service.prefix.replace(/\.$/, "")}
            edit
            {origin}
            bind:rrs={(value as any)[service.key]}
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
                {@const serviceArray = (value as any)[service.key] as Array<dnsTypeSRV>}
                {#if serviceArray && serviceArray[idx]}
                    <RawInput
                        edit
                        index={service.key + idx.toString()}
                        specs={{
                              id: field,
                              type: field == "Target" ? "string" : "uint16",
                        }}
                        bind:value={serviceArray[idx][field as keyof dnsTypeSRV]}
                    />
                {/if}
            {/snippet}
        </TableRecords>
    </div>
{/each}
