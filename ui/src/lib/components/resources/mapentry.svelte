<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     Icon,
     Input,
     InputGroup,
 } from 'sveltestrap';

 import ResourceInput from '$lib/components/ResourceInput.svelte';
 import type { Field } from '$lib/model/custom_form';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let edit = false;
 export let index: string;
 export let isNew = false;
 export let key: string;
 export let keytype: string;
 export let readonly = false;
 export let specs: Field;
 export let valuetype: string;
 export let value: any;

 let editKey = false;

 function rename() {
     editKey = false;
 }
</script>

<h3>
    {#if editKey}
        <InputGroup>
            <Input
                type="text"
                placeholder={specs.placeholder}
                bind:value={key}
            />
            <Button
                type="button"
                size="sm"
                color="primary"
                on:click={rename}
            >
                <Icon name="check" />
                {#if isNew}
                    {$t('domains.create-new-key', { id: specs.id })}
                {:else}
                    {$t('common.rename')}
                {/if}
            </Button>
        </InputGroup>
    {:else}
        {key}
        <Button type="button" size="sm" color="link" on:click={() => editKey = true}>
            <Icon name="pencil" />
        </Button>
    {/if}
</h3>
<ResourceInput
    {edit}
    index={index}
    {readonly}
    type={valuetype}
    bind:value={value}
/>
