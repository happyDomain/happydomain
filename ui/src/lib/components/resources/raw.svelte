<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Input,
     InputGroup,
     InputGroupText,
 } from 'sveltestrap';

 const dispatch = createEventDispatcher();

 export let edit = false;
 export let index: string;
 export let specs: any = { };
 export let value: any;

 let unit: string|null = null;

 $: unit = specs.type === 'time.Duration' ? 's' : null
</script>

<InputGroup size="sm" {...$$restProps}>
    {#if edit && specs.choices && specs.choices.length > 0}
        <Input
            id={'spec-' + index + '-' + specs.id}
            type="select"
            required={specs.required}
            bind:value={value}
            on:focus={() => dispatch("focus")}
            on:blur={() => dispatch("blur")}
        >
            {#each specs.choices as opt}
                <option value={opt}>{opt}</option>
            {/each}
        </Input>
    {:else}
        <Input
            id={'spec-' + index + '-' + specs.id}
            class="fw-bold"
            required={specs.required}
            placeholder={specs.placeholder}
            plaintext={!edit}
            bind:value={value}
            on:focus={() => dispatch("focus")}
            on:blur={() => dispatch("blur")}
        />
    {/if}

    {#if unit !== null}
        <InputGroupText>{unit}</InputGroupText>
    {/if}
</InputGroup>
