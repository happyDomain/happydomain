<script lang="ts">
 import {
     Button,
     Icon,
 } from 'sveltestrap';

 import MapEntry from '$lib/components/resources/mapentry.svelte';
 import type { Field } from '$lib/model/custom_form';

 const re = /^map\[(.*)\]\*?(.*)$/;

 export let edit = false;
 export let index: string;
 export let readonly = false;
 export let specs: Field;
 export let type: string;
 export let value: any;

 let keytype: string | undefined;
 let valuetype: string | undefined;
 $: {
     const res = re.exec(type);
     if (res) {
         keytype = res[1];
         valuetype = res[2];
     }
 }
 $: if (valuetype && !value) {
     value = { };
 }

 function renameKey(oldkey: string, newkey: string) {
     value[newkey] = value[oldkey];
     delete value[oldkey];
     value = value;
 }

 function deleteKey(key: string) {
     delete value[key];
     value = value;
 }
</script>

{#if keytype && valuetype}
    {#if value && Object.keys(value).length}
        {#each Object.keys(value) as key}
            {#key key}
                <MapEntry
                    {edit}
                    {key}
                    {keytype}
                    index={index + "_" + key}
                    {readonly}
                    {specs}
                    {valuetype}
                    bind:value={value[key]}
                    on:delete-key={() => deleteKey(key)}
                    on:rename-key={(event) => renameKey(key, event.detail)}
                />
            {/key}
        {/each}
    {:else}
        <div class="my-2 text-center">
            No {specs.label}
        </div>
    {/if}
    {#if !("" in value)}
        <Button
            type="button"
            color="primary"
            on:click={() => value[""] = { }}
        >
            <Icon name="plus" />
            Add {specs.label}
        </Button>
    {/if}
{/if}
