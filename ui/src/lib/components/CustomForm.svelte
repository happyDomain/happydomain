<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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
 import ResourceInput from '$lib/components/ResourceInput.svelte';
 import type { CustomForm } from '$lib/model/custom_form';
 import { t } from '$lib/translations';
 import { fillUndefinedValues } from '$lib/types';

 export let form: CustomForm;
 export let value: any;

 $: if (form.fields) {
     if (value === undefined) value = { };
     for (const field of form.fields) {
         fillUndefinedValues(value, field);
     }
 }
</script>

<div {...$$restProps}>
    {#if form.beforeText}
        <p class="lead text-indent">
            {form.beforeText}
        </p>
    {:else}
        <p>
            {$t('domains.please-fill-fields')}
        </p>
    {/if}

    <slot />

    {#if form.fields}
        {#each form.fields as field, index}
            <ResourceInput
                edit
                index={'' + index}
                specs={field}
                type={field.type}
                bind:value={value[field.id]}
            />
        {/each}
    {/if}

    {#if form.afterText}
        <p>
            {form.afterText}
        </p>
    {/if}
</div>
