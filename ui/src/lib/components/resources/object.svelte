<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     Icon,
     TabContent,
     TabPane,
     Spinner,
 } from 'sveltestrap';

 import { getServiceSpec } from '$lib/api/service_specs';
 import ResourceInput from '$lib/components/ResourceInput.svelte';
 import type { Field } from '$lib/model/custom_form';
 import type { ServiceInfos } from '$lib/model/service_specs';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let edit = false;
 export let editToolbar = false;
 export let index: string;
 export let readonly = false;
 export let specs: ServiceInfos;
 export let type: string;
 export let value: any;

 let innerSpecs: Array<Field>|undefined = undefined;
 $: {
     getServiceSpec(type).then((ss) => {
         innerSpecs = ss.fields;
     });
 }

 let editChildren = false;

 function saveObject() {
     editChildren = false
 }

 function deleteObject() {

 }
</script>

{#if !innerSpecs}
    <div class="d-flex justify-content-center">
        <Spinner color="primary" />
    </div>
{:else if specs && specs.tabs}
    <TabContent>
        {#each innerSpecs as spec, i}
            <TabPane
                tabId={spec.id}
                tab={spec.label}
                active={i == 0}
            >
                <ResourceInput
                    {edit}
                    {editToolbar}
                    index={index + '_' + spec.id}
                    noDecorate
                    {readonly}
                    specs={spec}
                    type={spec.type}
                    bind:value={value[spec.id]}
                />
            </TabPane>
        {/each}
    </TabContent>
{:else if Array.isArray(innerSpecs)}
    {#if !readonly && editToolbar}
        <div
            class="d-flex justify-content-end mb-2 gap-1"
        >
            {#if editChildren}
                <Button type="button" size="sm" color="success" on:click={saveObject}>
                    <Icon name="check" />
                    {$t('domains.save-modifications')}
                </Button>
            {:else}
                <Button type="button" size="sm" color="primary" outline on:click={() => editChildren = true}>
                    <Icon name="pencil" />
                    {$t('common.edit')}
                </Button>
            {/if}
            {#if type !== 'abstract.Origin'}
                <Button type="button" size="sm" color="danger" outline on:click={deleteObject}>
                    <Icon name="trash" />
                    {$t('common.delete')}
                </Button>
            {/if}
        </div>
    {/if}
    {#each innerSpecs as spec}
        <ResourceInput
            edit={edit || editChildren}
            index={index + '_' + spec.id}
            {readonly}
            specs={spec}
            type={spec.type}
            bind:value={value[spec.id]}
        />
    {/each}
{/if}
