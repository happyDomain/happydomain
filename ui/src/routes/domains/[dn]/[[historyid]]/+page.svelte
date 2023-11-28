<script lang="ts">
 import {
     Button,
     Col,
     Icon,
     Row,
     Spinner,
 } from 'sveltestrap';

 import SubdomainList from '$lib/components/domains/SubdomainList.svelte';
 import type { DomainInList } from '$lib/model/domain';
 import type { Zone } from '$lib/model/zone';
 import { domains_idx } from '$lib/stores/domains';
 import { servicesSpecs, refreshServicesSpecs } from '$lib/stores/services';
 import { retrieveZone, sortedDomains, thisZone } from '$lib/stores/thiszone';
 import { t } from '$lib/translations';

 if (!$servicesSpecs) refreshServicesSpecs();

 export let data: {domain: DomainInList; history: string; zoneId: string; streamed: Object};
</script>

{#if !data.domain}
    <div class="mt-5 text-center flex-fill">
        <Spinner label="Spinning" />
        <p>{$t('wait.loading')}</p>
    </div>
{:else if !data.domain.zone_history || data.domain.zone_history.length == 0}
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
        {#if zone && $sortedDomains}
            <div style="max-width: 100%;" class="pt-1">
                <SubdomainList
                    origin={data.domain}
                    sortedDomains={$sortedDomains}
                    zone={$thisZone}
                    on:update-zone-services={(event) => thisZone.set(event.detail)}
                />
            </div>
        {/if}
    {/await}
{/if}
