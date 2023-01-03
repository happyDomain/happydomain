<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Col,
     Row,
 } from 'sveltestrap';

 import ResourceRawInput from '$lib/components/resources/raw.svelte';

 const dispatch = createEventDispatcher();

 export let alwaysShow = false;
 export let edit = false;
 export let index: string;
 export let showDescription = true;
 export let specs: any;
 export let value: any;
</script>

{#if alwaysShow || edit || value != null}
    <Row {...$$restProps}>
        <label for={'spec-' + index + '-' + specs.id} title={specs.label} class="col-md-4 col-form-label text-truncate text-md-right text-primary">
            {#if specs.label}
                {specs.label}
            {:else}
                {specs.id}
            {/if}
        </label>
        <Col md="8">
            <ResourceRawInput
                {edit}
                {index}
                {specs}
                bind:value={value}
                on:focus={() => dispatch("focus")}
                on:blur={() => dispatch("blur")}
            />
            {#if specs.description && (showDescription || (specs.choices && specs.choices.length > 0))}
                <p class="text-justify" style="line-height: 1.1">
                    <small class="text-muted">{specs.description}</small>
                </p>
            {/if}
        </Col>
    </Row>
{/if}
