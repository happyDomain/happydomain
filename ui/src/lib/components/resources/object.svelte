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

 import {
     Button,
     Icon,
     TabContent,
     TabPane,
     Spinner,
 } from '@sveltestrap/sveltestrap';

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
 $: getServiceSpec(type).then((ss) => {
     innerSpecs = ss.fields;
 });

 // Initialize unexistant objects and arrays, except standard types.
 $: if (innerSpecs) {
     for (const spec of innerSpecs) {
         if (!(specs && specs.tabs) || spec.required) {
             fillUndefinedValues(value, spec);
         }
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
                {#if innerSpecs && value[spec.id] === undefined}
                    <div class="my-3 d-flex justify-content-center">
                        <Button
                            color="primary"
                            type="button"
                            on:click={() => { fillUndefinedValues(value, spec); value = value; }}
                        >
                            <Icon name="plus" />
                            {$t('common.add-object', {thing: spec.label})}
                        </Button>
                    </div>
                {:else}
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
                {/if}
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
