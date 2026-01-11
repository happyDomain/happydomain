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

    import SVCField from './ServiceField.svelte'
    import type { ServiceSpec } from "$lib/model/service_specs.svelte";

    const { MODE } = import.meta.env;

    interface Props {
        specs: ServiceSpec;
        value: any;
        aservice: Snippet<[string, any]>;
    }

    let { specs, value = $bindable(), aservice }: Props = $props();
</script>

{#if specs.fields}
    {#each specs.fields as field}
        <SVCField
            {aservice}
            type={field.type}
            bind:value={value[field.id]}
        />
    {/each}
{:else if MODE != "production"}
    NO FIELD
{/if}
