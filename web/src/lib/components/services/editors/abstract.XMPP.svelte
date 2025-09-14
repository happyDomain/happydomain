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
    const type = "abstract.XMPP";

    // Type-safe wrapper for dynamic access to value
    const valueData = value as dnsResource & {
        records?: dnsTypeSRV[];
        "xmpp-client"?: dnsTypeSRV[];
        "xmpp-server"?: dnsTypeSRV[];
        "jabber"?: dnsTypeSRV[];
        [key: string]: any;
    };

    // Initialize records array if needed
    if (!valueData.records) {
        valueData.records = [];
    }

    // Service type configurations with their prefixes
    const services = [
        { key: "xmpp-client", prefix: "_xmpp-client._tcp.", label: "XMPP Client" },
        { key: "xmpp-server", prefix: "_xmpp-server._tcp.", label: "XMPP Server" },
        { key: "jabber", prefix: "_jabber._tcp.", label: "Jabber" },
    ];

    // Initialize service arrays from records (one-time, breaks circular dependency)
    let initialized = $state(false);

    $effect(() => {
        if (!initialized && valueData.records) {
            for (const service of services) {
                valueData[service.key] = valueData.records.filter((srv) =>
                    srv?.Hdr?.Name?.startsWith(service.prefix) || false
                );
            }
            initialized = true;
        }
    });

    // Initialize empty arrays for services with no records
    for (const service of services) {
        if (!valueData[service.key]) {
            valueData[service.key] = [] as dnsTypeSRV[];
        }
    }

    // When records in service arrays change, sync back to main array (one-way sync)
    $effect(() => {
        if (initialized) {
            const allRecords: dnsTypeSRV[] = [];
            for (const service of services) {
                if (valueData[service.key]) {
                    allRecords.push(...valueData[service.key]);
                }
            }
            valueData.records = allRecords;
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
    </div>
{/each}
