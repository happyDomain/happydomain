<script lang="ts">
 import {
     Button,
     Col,
     Icon,
     Row,
     Spinner,
 } from 'sveltestrap';

 import { getZone } from '$lib/api/zone';
 import SubdomainList from '$lib/components/domains/SubdomainList.svelte';
 import type { DomainInList } from '$lib/model/domain';
 import type { Zone } from '$lib/model/zone';
 import { domains_idx } from '$lib/stores/domains';
 import { servicesSpecs, refreshServicesSpecs } from '$lib/stores/services';
 import { t } from '$lib/translations';

 if (!$servicesSpecs) refreshServicesSpecs();

 export let data: {domain: string; selectedDomain: DomainInList; history: string; zoneId: string; streamed: Object};

 export let newSubdomainModalOpened = false;
</script>

{#if !data.selectedDomain}
    <div class="mt-5 text-center flex-fill">
        <Spinner label="Spinning" />
        <p>{$t('wait.loading')}</p>
    </div>
{:else if !data.selectedDomain.zone_history || data.selectedDomain.zone_history.length == 0}
    <div class="mt-4 text-center flex-fill">
        <Spinner label={$t('common.spinning')} />
        <p>{$t('wait.importing')}</p>
    </div>
{:else}
    {#await data.streamed.zone}
        <div class="mt-4 text-center flex-fill">
            <Spinner label={$t('common.spinning')} />
            <p>{$t('wait.loading')}</p>
        </div>
    {:then zone}
        {#if zone}
            {#await data.streamed.sortedDomains}
                <div class="mt-4 text-center flex-fill">
                    <Spinner label={$t('common.spinning')} />
                    <p>{$t('wait.loading')}</p>
                </div>
            {:then sortedDomains}
                <div style="max-width: 100%;" class="pt-1">
                    <SubdomainList
                        origin={data.selectedDomain}
                        {sortedDomains}
                        {zone}
                        bind:newSubdomainModalOpened={newSubdomainModalOpened}
                        on:update-zone-services={(event) => zone = event.detail}
                    />
                </div>
            {/await}
        {/if}
    {/await}
{/if}
