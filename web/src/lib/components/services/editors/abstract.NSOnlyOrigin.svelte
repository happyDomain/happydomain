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
    import type { Domain } from "$lib/model/domain";
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import TableRecords from "$lib/components/records/TableRecords.svelte";
    import RawInput from "$lib/components/inputs/raw.svelte";
    import { servicesSpecs } from "$lib/stores/services";

    interface Props {
        dn: string;
        origin: Domain;
        value: any;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();
    const type = "abstract.Origin";
</script>

{#if $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">Zone's Name Servers  (NS records)</h4>
    <!--RecordsLines {dn} {origin} bind:rrs={value["ns"]} /-->
    <TableRecords
        class="mt-3"
        {dn}
        edit
        {origin}
        rrs={value["ns"]}
        rrtype="NS"
    >
        {#snippet header(field: string)}
            {#if field == "Ns"}
                Name Servers
            {/if}
        {/snippet}
        {#snippet field(idx: number, field: string)}
            <RawInput
                edit
                index={idx.toString()}
                bind:value={value["ns"][idx][field]}
            />
        {/snippet}
    </TableRecords>
</div>
