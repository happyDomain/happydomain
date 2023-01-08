<script lang="ts">
 import {
     Button,
     Icon,
     Table,
     Spinner,
 } from 'sveltestrap';

 import { getServiceSpec } from '$lib/api/service_specs';
 import ResourceInput from '$lib/components/ResourceInput.svelte';
 import type { Field } from '$lib/model/custom_form';

 export let edit = false;
 export let index: string;
 export let noDecorate = false;
 export let readonly = false;
 export let specs: any;
 export let type: string;
 export let value: any;

 let linespecs: Array<Field>|null|undefined = undefined;
 $: {
     getServiceSpec(type).then((ss) => {
         linespecs = ss.fields;
     }, () => {
         linespecs = null;
     })
 }

 function addLine() {
     if (!value) value = [];
     value.push(linespecs?{ }:"");
     value = value;
 }

 function deleteLine(idx: number) {
     value.splice(idx, 1);
     value = value;
 }
</script>

{#if linespecs === undefined}
    <div class="d-flex justify-content-center">
        <Spinner color="primary" />
    </div>
{:else}
    {#if !noDecorate && specs && specs.label}
        <h4 class="mt-1 text-primary pb-1 border-bottom border-1">
            {specs.label}
            {#if specs.description}
                <small class="text-muted">
                    {specs.description}
                </small>
            {/if}
        </h4>
    {/if}
    <Table hover striped>
        <thead>
            <tr>
                {#if linespecs}
                    {#each linespecs as spec}
                        <th>{#if spec.label}{spec.label}{:else}{spec.id}{/if}</th>
                    {/each}
                {:else if specs}
                    <th>{#if specs.label}{specs.label}{:else}{specs.id}{/if}</th>
                {/if}
            </tr>
        </thead>
        <tbody>
          {#if value && value.length}
            {#each value as v, idx}
                <tr>
                    {#if linespecs}
                        {#each linespecs as spec}
                            <td>
                                <ResourceInput
                                    {edit}
                                    noDecorate
                                    index={index + '_' + idx + '_' + spec.id}
                                    {readonly}
                                    specs={spec}
                                    type={spec.type}
                                    bind:value={value[idx][spec.id]}
                                />
                            </td>
                        {/each}
                    {:else}
                        <td>
                            <ResourceInput
                                {edit}
                                noDecorate
                                index={index + '_' + idx}
                                {readonly}
                                type={type}
                                bind:value={value[idx]}
                            />
                        </td>
                    {/if}
                    {#if edit}
                        <td>
                            <Button
                                type="button"
                                color="danger"
                                outline
                                size="sm"
                                on:click={() => deleteLine(idx)}
                            >
                                <Icon name="trash" />
                            </Button>
                        </td>
                    {/if}
                </tr>
            {/each}
          {:else}
            <tr>
                <td colspan={(linespecs?linespecs.length:1)+(edit?1:0)} class="fst-italic text-center">
                    No content
                </td>
            </tr>
          {/if}
        </tbody>
        {#if edit}
            <tfoot>
                <tr>
                    <td colspan={linespecs?linespecs.length:1}>
                        <Button
                            type="button"
                            color="primary"
                            outline
                            size="sm"
                            on:click={addLine}
                        >
                            <Icon name="plus" />
                            New row
                        </Button>
                    </td>
                </tr>
            </tfoot>
        {/if}
    </Table>
{/if}
