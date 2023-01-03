<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     Spinner,
 } from 'sveltestrap';

 import type { CustomForm } from '$lib/model/custom_form';

 const dispatch = createEventDispatcher();

 export let edit = false;
 export let form: CustomForm|null = null;
 export let nextInProgress = false;
 export let previousInProgress = false;
 export let submitForm: string|null = null;

 let disabled = false;
 $: disabled = nextInProgress || previousInProgress;
</script>

<div {...$$restProps}>
    {#if form}
        {#if (!form.previousEditButtonText || !edit) && form.previousButtonText}
            <Button
                type="button"
                class="mx-1"
                color="secondary"
                outline
                {disabled}
                on:click={() => dispatch('previous-state')}
            >
                {#if previousInProgress}
                    <Spinner label="Spinning" size="sm" />
                {/if}
                {form.previousButtonText}
            </Button>
        {/if}
        {#if (!form.nextEditButtonText || !edit) && form.nextButtonText}
            <Button
                type="submit"
                class="mx-1"
                color="primary"
                {disabled}
                form={submitForm}
            >
                {#if nextInProgress}
                    <Spinner label="Spinning" size="sm" />
                {/if}
                {form.nextButtonText}
            </Button>
        {/if}
        {#if edit && form.previousEditButtonText}
            <Button
                type="button"
                class="mx-1"
                color="secondary"
                outline
                {disabled}
                on:click={() => dispatch('previous-state')}
            >
                {#if previousInProgress}
                    <Spinner label="Spinning" size="sm" />
                {/if}
                {form.previousEditButtonText}
            </Button>
        {/if}
        {#if edit && form.nextEditButtonText}
            <Button
                type="submit"
                class="mx-1"
                color="primary"
                {disabled}
                form={submitForm}
            >
                {#if nextInProgress}
                    <Spinner label="Spinning" size="sm" />
                {/if}
                {form.nextEditButtonText}
            </Button>
        {/if}
    {:else}
        <Button
            type="button"
            class="mx-1"
            color="secondary"
            outline
            {disabled}
            on:click={() => dispatch('previous-state')}
        >
            {#if previousInProgress}
                <Spinner label="Spinning" size="sm" />
            {/if}
            Cancel
        </Button>
        <Button
            type="submit"
            class="mx-1"
            color="primary"
            {disabled}
            form={submitForm}
        >
            {#if nextInProgress}
                <Spinner label="Spinning" size="sm" />
            {/if}
            Next &gt;
        </Button>
    {/if}
</div>
