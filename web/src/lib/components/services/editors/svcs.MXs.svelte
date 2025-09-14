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
    import type { dnsResource, dnsTypeMX } from "$lib/dns_rr";
    import { servicesSpecs } from "$lib/stores/services";

    interface Props {
        dn: string;
        origin: Domain;
        value: dnsResource;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    const type = "svcs.MXs";

    // Ensure mx is always an array at runtime
    $effect(() => {
        if (value["mx"] && !Array.isArray(value["mx"])) {
            value["mx"] = [value["mx"]];
        }
    });
</script>

{#if $servicesSpecs && $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">EMail Servers (MX records)</h4>
    {#if value["mx"]}
    <TableRecords
        class="mt-3"
        {dn}
        edit
        {origin}
        rrs={value["mx"] as dnsTypeMX[]}
        rrtype="MX"
    >
        {#snippet header(field: string)}
            {#if field == "Mx"}
                Target
            {:else if field == "Preference"}
                Preference
            {/if}
        {/snippet}
        {#snippet field(idx: number, field: string)}
            {#if value["mx"] && (value["mx"] as dnsTypeMX[])[idx]}
                {#if field == "Preference"}
                    <RawInput
                        edit
                        index={field + idx.toString()}
                        specs={{
                              id: "preference",
                              type: "uint",
                        }}
                        bind:value={(value["mx"] as dnsTypeMX[])[idx].Preference}
                    />
                {:else if field == "Mx"}
                    <RawInput
                        edit
                        index={field + idx.toString()}
                        bind:value={(value["mx"] as dnsTypeMX[])[idx].Mx}
                    />
                {/if}
            {/if}
        {/snippet}
    </TableRecords>
    {/if}
</div>
