<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import MapEntry from '$lib/components/resources/mapentry.svelte';
 import type { Field } from '$lib/model/custom_form';

 const dispatch = createEventDispatcher();
 const re = /^map\[(.*)\]\*?(.*)$/;

 export let edit = false;
 export let index: string;
 export let readonly = false;
 export let specs: Field;
 export let type: string;
 export let value: any;

 let keytype: string|undefined;
 let valuetype: string|undefined;
 $: {
     const res = re.exec(type);
     if (res) {
         keytype = res[1];
         valuetype = res[2];
     }
 }
</script>

{#if keytype && valuetype}
    {#each Object.keys(value) as key}
        <MapEntry
            {edit}
            {key}
            {keytype}
            index={index + "_" + key}
            {readonly}
            {specs}
            {valuetype}
            bind:value={value[key]}
        />
    {/each}
{/if}
