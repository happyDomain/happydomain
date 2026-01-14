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
    import { getServiceSpec } from "$lib/api/service_specs";
    import type { ServiceSpec } from "$lib/model/service_specs.svelte";
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import RecordEditor from "$lib/components/records/Editor.svelte";
    import type { Domain } from "$lib/model/domain";
    import { newRecord } from "$lib/model/service_specs.svelte";
    import { servicesSpecs } from "$lib/stores/services";
    import type { dnsResource } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        type?: string;
        value: dnsResource;
    }

    let {
        dn,
        origin,
        readonly = false,
        type = "svcs.Orphan",
        value = $bindable({ }),
    }: Props = $props();

    let sspecs: ServiceSpec = {} as ServiceSpec;

    $effect(() => {
        getServiceSpec(type).then((res) => sspecs = res);
    });
</script>

{#if $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
{#each Object.keys(value) as key}
    {@const valueKey = (value as any)[key]}
    {#if valueKey instanceof Array}
        {#each valueKey as v, i}
            {#if i > 0}
                <hr>
            {/if}
            <RecordEditor
                bind:dn={dn}
                {origin}
                bind:record={valueKey[i]}
            />
        {/each}
        <button type="button" class="btn btn-primary" aria-label="Add new record" onclick={() => sspecs.fields && valueKey.push(newRecord(sspecs.fields.filter((field: any) => field.id == key)[0]))}>
            <i class="bi bi-plus"></i>
        </button>
    {:else}
        <RecordEditor
            bind:dn={dn}
            {origin}
            bind:record={(value as any)[key]}
        />
    {/if}
{/each}
