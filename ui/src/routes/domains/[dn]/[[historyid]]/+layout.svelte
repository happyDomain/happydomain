<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Spinner,
 } from 'sveltestrap';

 import NewServicePath from '$lib/components/NewServicePath.svelte';
 import ServiceModal from '$lib/components/domains/ServiceModal.svelte';
 import { domains_idx } from '$lib/stores/domains';
 import { thisZone } from '$lib/stores/thiszone';
 import { t } from '$lib/translations';

 export let data: {domain: DomainInList; history: string; streamed: Object;};

 let selectedDomain = data.domain.domain;
 let selectedHistory: string | undefined;
 $: selectedHistory = data.history;
 $: if (!data.history && $domains_idx[selectedDomain] && $domains_idx[selectedDomain].zone_history && $domains_idx[selectedDomain].zone_history.length > 0) {
     selectedHistory = $domains_idx[selectedDomain].zone_history[0] as string;
 }
 $: if (selectedHistory && data.history != selectedHistory) {
     goto('/domains/' + encodeURIComponent(selectedDomain) + '/' + encodeURIComponent(selectedHistory));
 }

</script>

{#if data.history == selectedHistory}
    <slot />
{:else}
    <div class="mt-5 text-center flex-fill">
        <Spinner label="Spinning" />
        <p>{$t('wait.loading')}</p>
    </div>
{/if}

{#await data.streamed.zone then zone}
    <NewServicePath
        origin={data.domain}
        zone={zone}
    />
    <ServiceModal
        origin={data.domain}
        zone={zone}
        on:update-zone-services={(event) => thisZone.set(event.detail)}
    />
{/await}
