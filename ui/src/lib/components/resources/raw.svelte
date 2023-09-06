<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Input,
     InputGroup,
     InputGroupText,
 } from 'sveltestrap';
 import type {
     InputType,
 } from 'sveltestrap/src/Input.d';

 const dispatch = createEventDispatcher();

 export let edit = false;
 export let index: string;
 export let specs: any = { };
 export let value: any;

 let unit: string|null = null;
 $: unit = specs.type === 'time.Duration' || specs.type === 'common.Duration' ? 's' : null

 let inputtype: InputType = "text";
 $: if (specs.type && (specs.type.startsWith("uint") || specs.type.startsWith("int"))) inputtype = "number";

 let inputmin: number | undefined = undefined;
 let inputmax: number | undefined = undefined;
 $: if (specs.type) {
     if (specs.type == "int8" || specs.type == "uint8") inputmax = 255;
     else if (specs.type == "int16" || specs.type == "uint16") inputmax = 65536;
     else if (specs.type == "int" || specs.type == "uint" || specs.type == "int32" || specs.type == "uint32") inputmax = 2147483647;
     else if (specs.type == 'time.Duration' || specs.type == 'common.Duration' || specs.type == "int64" || specs.type == "uint64") inputmax = 9007199254740991;
     else inputmax = undefined;

     if (inputmax) {
         if (specs.type && specs.type.startsWith("uint")) inputmin = 0; else inputmin = -inputmax - 1;
     } else {
         inputmin = undefined;
     }
 }
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
            type={inputtype}
            class="fw-bold"
            min={inputmin}
            max={inputmax}
            placeholder={specs.placeholder}
            plaintext={!edit}
            readonly={!edit}
            required={specs.required}
            style="width: initial"
            bind:value={value}
            on:focus={() => dispatch("focus")}
            on:blur={() => dispatch("blur")}
        />
    {/if}

    {#if unit !== null}
        <InputGroupText>{unit}</InputGroupText>
    {/if}
</InputGroup>
