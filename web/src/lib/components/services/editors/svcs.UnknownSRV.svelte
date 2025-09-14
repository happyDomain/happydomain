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
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { dnsResource, dnsTypeSRV, dnsRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "svcs.UnknownSRV";

    // Initialize srv array if needed (treat as array despite type definition)
    if (!value["srv"]) {
        value["srv"] = [] as any;
    }

    // Type-safe accessor for srv as array
    const srvArray = $derived(value["srv"] as any as Array<dnsTypeSRV>);

    // Extract service name and protocol from first record's domain name
    let serviceName = $state<string>("http");
    let protocol = $state<string>("tcp");

    // Initialize serviceName and protocol from first SRV record
    $effect(() => {
        if (srvArray?.[0]?.Hdr?.Name) {
            const match = srvArray[0].Hdr.Name.match(/^_([^.]+)\._(\w+)\./);
            if (match) {
                serviceName = match[1];
                protocol = match[2];
            }
        }
    });

    // Construct the full DN with service and protocol prefix
    let fullDn = $derived(`_${serviceName}._${protocol}`);

    // Sync the DN to all SRV records
    $effect(() => {
        if (srvArray) {
            for (const record of srvArray) {
                if (record?.Hdr) {
                    record.Hdr.Name = fullDn;
                }
            }
        }
    });
</script>

<div>
    <BasicInput
        edit
        index="service"
        specs={{
            id: "service",
            label: "Service Name",
            description: "The symbolic name of the desired service",
            type: "string",
            placeholder: "http",
        }}
        bind:value={serviceName}
    />

    <BasicInput
        edit
        index="protocol"
        specs={{
            id: "protocol",
            label: "Protocol",
            description: "Protocol used to establish the connection",
            type: "string",
            choices: ["tcp", "udp"],
        }}
        bind:value={protocol}
    />

    <TableRecords
        class="mt-3"
        dn={fullDn}
        edit
        {origin}
        bind:rrs={value["srv"] as any as dnsRR[]}
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
            {#if srvArray && srvArray[idx]}
                <RawInput
                    edit
                    index={idx.toString()}
                    specs={{
                          id: field,
                          type: field == "Target" ? "string" : "uint16",
                    }}
                    bind:value={srvArray[idx][field as keyof dnsTypeSRV]}
                />
            {/if}
        {/snippet}
    </TableRecords>
</div>
