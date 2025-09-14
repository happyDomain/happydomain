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
    import type { dnsResource, dnsRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "svcs.TLSAs";

    // Initialize tlsa array if needed - cast to array type
    if (!value["tlsa"]) {
        value["tlsa"] = [] as any;
    }

    // Extract port and protocol from first record's domain name
    let port = $state<number>(443);
    let protocol = $state<string>("tcp");

    // Type-safe accessor for tlsa records as array
    const getTlsaArray = (): dnsRR[] => (value["tlsa"] as any) as dnsRR[];

    if (getTlsaArray()?.[0]?.Hdr?.Name) {
        const match = getTlsaArray()[0].Hdr.Name.match(/^_(\d+)\._(\w+)\./);
        if (match) {
            port = parseInt(match[1], 10);
            protocol = match[2];
        }
    }

    // Construct the full DN with port and protocol prefix
    let fullDn = $derived(`_${port}._${protocol}`);

    // Sync the DN to all TLSA records
    $effect(() => {
        const records = getTlsaArray();
        if (records) {
            for (const record of records) {
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
        index="port"
        specs={{
            id: "port",
            label: "Service Port",
            description: "Port number where people will establish the connection",
            type: "uint16",
            placeholder: "443",
        }}
        bind:value={port}
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
        bind:rrs={(value["tlsa"] as any)}
        rrtype="TLSA"
    >
    {#snippet header(field: string)}
        {#if field == "Usage"}
            Certificate Usage
        {:else if field == "Selector"}
            Selector
        {:else if field == "MatchingType"}
            Matching Type
        {:else if field == "Certificate"}
            Certificate
        {/if}
    {/snippet}
    {#snippet field(idx: number, field: string)}
        {@const tlsaArray = (value["tlsa"] as any) as dnsRR[]}
        {#if tlsaArray && tlsaArray[idx]}
            <RawInput
                edit
                index={idx.toString()}
                specs={{
                       id: field,
                       type: field == "Certificate" ? "string" : "uint16",
                 }}
                 bind:value={tlsaArray[idx][field]}
            />
        {/if}
    {/snippet}
</TableRecords>
</div>
