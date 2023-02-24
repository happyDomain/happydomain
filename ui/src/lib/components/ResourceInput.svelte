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

{#if type.substring(0, 2) === '[]' && type !== '[]byte' && type !== '[]uint8'}
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
