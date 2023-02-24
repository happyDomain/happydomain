<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Badge,
     Button,
     FormGroup,
     Icon,
     Input,
 } from 'sveltestrap';

 import { issuers, rev_issuers } from './CAA-issuers';

 const dispatch = createEventDispatcher();

 export let edit = false;
 export let index: string;
 export let newone = false;
 export let readonly = false;
 export let value: any;

 $: if (!value) value = { };
</script>

<div class="d-flex gap-2 mb-2">
    {#if (newone && value.IssuerDomainName == "") || rev_issuers[value.IssuerDomainName]}
        <Input type="select" name="select" id="exampleSelect" readonly={readonly} bind:value={value.IssuerDomainName}>
            {#each Object.keys(issuers) as issuer}
                <option value={issuers[issuer][0]}>{issuer}</option>
            {/each}
            <option value={" "}>Autre</option>
        </Input>
    {:else}
        <Input type="text" bind:value={value.IssuerDomainName} />
    {/if}
    {#if !newone}
        <Button
            type="button"
            color="danger"
            outline
            on:click={() => dispatch("delete-issuer")}
        >
            <Icon name="trash" />
        </Button>
    {:else}
        <Button
            color="success"
            outline
            type="button"
            disabled={!value}
            on:click={() => {dispatch("add-issuer", value); value = { }}}
        >
            <Icon name="plus" />
        </Button>
    {/if}
</div>
{#if !newone}
    <div class="d-flex align-items-center">
        {#if value.Parameters}
            {#each value.Parameters as parameter, k}
                <Badge color="info" class="me-1">
                    {#if parameter.edit}
                        <form
                            class="d-flex align-items-center gap-1"
                            on:submit|preventDefault={() => parameter.edit = false}
                        >
                            <Input size="sm" placeholder="key" bind:value={parameter.Tag} />
                            =
                            <Input size="sm" placeholder="value" bind:value={parameter.Value} />
                            <Button
                                type="submit"
                                color="success"
                                size="sm"
                            >
                                <Icon
                                    name="check"
                                />
                            </Button>
                        </form>
                    {:else}
                        <span
                            on:dblclick={() => parameter.edit = true}
                        >
                            {parameter.Tag}={parameter.Value}
                        </span>
                        <span
                            role="button"
                            on:click={() => {value.Parameters.splice(k, 1); value = value;}}
                        >
                            <Icon
                                name="x-circle-fill"
                            />
                        </span>
                    {/if}
                </Badge>
            {/each}
        {/if}
        <span
            class="badge bg-primary"
            role="button"
            on:click={() => {if (value.Parameters == null) value.Parameters = []; value.Parameters.push({Tag:"", Value: "", edit: true}); value = value;}}
        >
            <Icon name="plus" /> Add parameter
        </span>
    </div>
{/if}
