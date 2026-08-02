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
    const type = "abstract.MatrixIM";

    // Initialize records array if needed (treat as array despite type definition)
    if (!(value as any)["records"]) {
        (value as any)["records"] = [];
    }

    // Type-safe accessor for records as array
    const recordsArray = $derived((value as any)["records"] as Array<dnsTypeSRV>);

    // Ensure all records have the correct Hdr.Name
    $effect(() => {
        if (recordsArray) {
            for (const record of recordsArray) {
                if (record?.Hdr && record.Hdr.Name !== "_matrix._tcp") {
                    record.Hdr.Name = "_matrix._tcp";
                }
            }
        }
    });
</script>

<TableRecords
    class="mt-3"
    dn="_matrix._tcp"
    edit
    {origin}
    bind:rrs={(value as any)["records"]}
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
        {#if recordsArray && recordsArray[idx]}
            <RawInput
                edit
                index={field + idx.toString()}
                specs={{
                      id: field,
                      type: field == "Target" ? "string" : "uint16",
                }}
                bind:value={recordsArray[idx][field as keyof dnsTypeSRV]}
            />
        {/if}
    {/snippet}
</TableRecords>
