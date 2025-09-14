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

    const type = "svcs.MXs";
</script>

{#if $servicesSpecs && $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">EMail Servers (MX records)</h4>
    <TableRecords
        class="mt-3"
        {dn}
        edit
        rrs={value["mx"]}
        rrtype="MX"
    >
        {#snippet header(field)}
            {#if field == "Mx"}
                Target
            {:else if field == "Preference"}
                Preference
            {/if}
        {/snippet}
        {#snippet field(idx, field)}
            {#if field == "Preference"}
                <RawInput
                    edit
                    index={idx}
                    specs={{
                          type: "uint",
                    }}
                    bind:value={value["mx"][idx][field]}
                />
            {:else}
                <RawInput
                    edit
                    index={idx}
                    bind:value={value["mx"][idx][field]}
                />
            {/if}
        {/snippet}
    </TableRecords>
</div>
