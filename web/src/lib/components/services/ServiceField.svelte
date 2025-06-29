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
    import type { Snippet } from 'svelte';

    import SVC from './Service.svelte'
    import SVCField from './ServiceField.svelte'
    import { getServiceSpec } from "$lib/api/service_specs";
    import type { Service } from '$lib/model/service';
    import type { ServiceInfos } from "$lib/model/service_specs";

    export let type: string;
    export let value: any;
    export let aservice: Snippet;
</script>

{#if type.startsWith("[]")}
    {#if value}
        {#each value as row, i}
            <SVCField
                {aservice}
                type={type.substring(2)}
                bind:value={value[i]}
            />
        {/each}
    {:else}
        <!--[]{type.substring(2)} => {value}-->
        {@render aservice(type.substring(2), null)}
    {/if}
{:else if type.startsWith("*")}
    {#if value}
        <SVCField
            {aservice}
            type={type.substring(1)}
            bind:value={value}
        />
    {:else}
        <!--*{type.substring(1)} => {value}-->
        {@render aservice(type.substring(1), null)}
    {/if}
{:else if type.startsWith("dns.") || type == "happydns.Record" || type == "happydns.TXT" || type == "happydns.SPF"}
    <!--{type} => {JSON.stringify(value)}-->
    {@render aservice(type, value)}
{:else}
    {#await getServiceSpec(type) then specs}
        <SVC
            {aservice}
            {type}
            {specs}
            bind:value={value}
        />
    {/await}
{/if}
