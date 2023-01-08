<script lang="ts">
 import {
     Button,
     Icon,
 } from 'sveltestrap';

 import ProviderForm from '$lib/components/providers/Form.svelte';
 import type { Provider } from '$lib/model/provider';
 import type { ProviderSettingsState } from '$lib/model/provider_settings';
 import { t } from '$lib/translations';

 export let data: {provider: Provider; provider_id: string;};

 let value: ProviderSettingsState = {...data.provider, state: 0};
</script>

<h1 class="text-center my-2">
    <Button
        type="button"
        class="fw-bolder"
        color="link"
        on:click={() => history.go(-1)}
    >
        <Icon name="chevron-left" />
    </Button>
    {$t('wait.updating')}
    {#if value}
        <em>{value._comment}</em>
    {/if}
</h1>
<hr class="mt-0 mb-0">
<ProviderForm
    edit
    ptype={data.provider._srctype}
    state={0}
    providerId={data.provider_id}
    bind:value={value}
/>
