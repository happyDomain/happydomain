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
    import type { dnsResource } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "abstract.Delegation";
</script>

<div class="mb-4">
    <h5 class="pb-1 border-bottom border-1">Name Servers</h5>
    <TableRecords
        class="mt-3"
        {dn}
        edit
        {origin}
        bind:rrs={(value["ns"] as any)}
        rrtype="NS"
    >
        {#snippet header(field: string)}
            {#if field == "Ns"}
                Name Server
            {/if}
        {/snippet}
        {#snippet field(idx: number, field: string)}
            {#if value["ns"] && Array.isArray(value["ns"]) && value["ns"][idx]}
                <RawInput
                    edit
                    index={field + idx.toString()}
                    bind:value={value["ns"][idx].Ns}
                />
            {/if}
        {/snippet}
    </TableRecords>
</div>

{#if value["ds"] && Array.isArray(value["ds"]) && value["ds"].length > 0}
    <div>
        <h5 class="pb-1 border-bottom border-1">Delegation Signers (DNSSEC)</h5>
        <TableRecords
            class="mt-3"
            {dn}
            edit
            {origin}
            bind:rrs={(value["ds"] as any)}
            rrtype="DS"
        >
            {#snippet header(field: string)}
                {#if field == "KeyTag"}
                    Key Tag
                {:else if field == "Algorithm"}
                    Algorithm
                {:else if field == "DigestType"}
                    Digest Type
                {:else if field == "Digest"}
                    Digest
                {/if}
            {/snippet}
            {#snippet field(idx: number, field: string)}
                {#if value["ds"] && Array.isArray(value["ds"]) && value["ds"][idx]}
                    <RawInput
                        edit
                        index={field + idx.toString()}
                        bind:value={(value["ds"][idx] as any)[field]}
                    />
                {/if}
            {/snippet}
        </TableRecords>
    </div>
{/if}
