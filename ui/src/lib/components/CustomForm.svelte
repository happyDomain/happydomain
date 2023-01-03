<script lang="ts">
 import ResourceInput from '$lib/components/ResourceInput.svelte';
 import type { CustomForm } from '$lib/model/custom_form';
 import { t } from '$lib/translations';

 export let form: CustomForm;
 export let value: any;
</script>

<div {...$$restProps}>
    {#if form.beforeText}
        <p class="lead text-indent">
            {form.beforeText}
        </p>
    {:else}
        <p>
            {$t('domains.please-fill-fields')}
        </p>
    {/if}

    <slot />

    {#if form.fields}
        {#each form.fields as field, index}
            <ResourceInput
                edit
                index={'' + index}
                specs={field}
                type={field.type}
                bind:value={value[field.id]}
            />
        {/each}
    {/if}

    {#if form.afterText}
        <p>
            {form.afterText}
        </p>
    {/if}
</div>
