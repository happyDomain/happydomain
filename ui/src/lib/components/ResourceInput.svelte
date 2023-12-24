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
 import { createEventDispatcher } from 'svelte';

 import BasicInput from '$lib/components/resources/basic.svelte';
 import CAAForm from '$lib/components/resources/CAA.svelte';
 import MapInput from '$lib/components/resources/map.svelte';
 import ObjectInput from '$lib/components/resources/object.svelte';
 import RawInput from '$lib/components/resources/raw.svelte';
 import TableInput from '$lib/components/resources/table.svelte';

 const dispatch = createEventDispatcher();

 export let edit = false;
 export let editToolbar = false;
 export let index = "";
 export let noDecorate = false;
 export let readonly = false;
 export let showDescription = true;
 export let specs: any = undefined;
 export let type: string;
 export let value: any;

 function sanitizeType(t: string) {
     if (t.substring(0, 2) === '[]') t = t.substring(2);
     if (t.substring(0, 1) === '*') t = t.substring(1);
     return t;
 }
</script>

{#if specs && specs.hide}
    <!-- hidden input -->
{:else if type.substring(0, 2) === '[]' && type !== '[]byte' && type !== '[]uint8'}
    <TableInput
        edit={edit || editToolbar}
        {index}
        {noDecorate}
        {readonly}
        {specs}
        type={sanitizeType(type)}
        bind:value={value}
    />
{:else if type.substring(0, 3) === 'map'}
    <MapInput
        edit={edit || editToolbar}
        {index}
        {readonly}
        {specs}
        type={sanitizeType(type)}
        bind:value={value}
    />
{:else if type == "svcs.CAA"}
    <CAAForm
        edit={edit || editToolbar}
        {index}
        {readonly}
        {specs}
        bind:value={value}
        on:delete-this-service={(event) => dispatch("delete-this-service", event.detail)}
        on:update-this-service={(event) => dispatch("update-this-service", event.detail)}
    />
{:else if typeof value === 'object' || Array.isArray(specs)}
    <ObjectInput
        {edit}
        {editToolbar}
        {index}
        {readonly}
        {specs}
        type={sanitizeType(type)}
        bind:value={value}
        on:delete-this-service={(event) => dispatch("delete-this-service", event.detail)}
        on:update-this-service={(event) => dispatch("update-this-service", event.detail)}
    />
{:else if noDecorate}
    <RawInput
        edit={edit || editToolbar}
        {index}
        {readonly}
        {specs}
        type={sanitizeType(type)}
        bind:value={value}
    />
{:else}
    <BasicInput
        edit={edit || editToolbar}
        {index}
        {readonly}
        {showDescription}
        {specs}
        type={sanitizeType(type)}
        bind:value={value}
    />
{/if}
