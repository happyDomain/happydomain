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
 import { fillUndefinedValues } from '$lib/types';

 const dispatch = createEventDispatcher();

 export let edit = false;
 export let editToolbar = false;
 export let index: string;
 export let readonly = false;
 export let specs: ServiceInfos;
 export let type: string;
 export let value: any;

 let innerSpecs: Array<Field> | undefined = undefined;
 $: {
     getServiceSpec(type).then((ss) => {
         innerSpecs = ss.fields;
     });
 }

 // Initialize unexistant objects and arrays, except standard types.
 $: if (innerSpecs) {
     for (const spec of innerSpecs) {
         fillUndefinedValues(value, spec);
     }
 }

 let editChildren = false;

 let updateServiceInProgress = false;
 function saveObject() {
     updateServiceInProgress = true;
     dispatch("update-this-service");
     editChildren = false;
 }

 let deleteServiceInProgress = false;
 function deleteObject() {
     deleteServiceInProgress = true;
     dispatch("delete-this-service");
 }

 function deleteSubObject(id: string) {
     deleteServiceInProgress = true;
     delete value[id];
     dispatch("update-this-service");
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
                    on:delete-this-service={() => deleteSubObject(spec.id)}
                    on:update-this-service={(event) => dispatch("update-this-service", event.detail)}
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
                <Button
                    type="button"
                    disabled={updateServiceInProgress}
                    size="sm"
                    color="success"
                    on:click={saveObject}
                >
                    {#if updateServiceInProgress}
                        <Spinner size="sm" />
                    {:else}
                        <Icon name="check" />
                    {/if}
                    {$t('domains.save-modifications')}
                </Button>
            {:else}
                <Button type="button" size="sm" color="primary" outline on:click={() => editChildren = true}>
                    <Icon name="pencil" />
                    {$t('common.edit')}
                </Button>
            {/if}
            {#if type !== 'abstract.Origin'}
                <Button
                    type="button"
                    disabled={deleteServiceInProgress}
                    size="sm"
                    color="danger"
                    outline
                    on:click={deleteObject}
                >
                    {#if deleteServiceInProgress}
                        <Spinner size="sm" />
                    {:else}
                        <Icon name="trash" />
                    {/if}
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
