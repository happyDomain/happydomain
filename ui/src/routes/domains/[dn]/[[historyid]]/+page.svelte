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
 import { domainCompare, fqdn } from '$lib/dns';
 import type { DomainInList } from '$lib/model/domain';
 import type { Zone } from '$lib/model/zone';
 import { domains_idx } from '$lib/stores/domains';
 import { servicesSpecs, refreshServicesSpecs } from '$lib/stores/services';
 import { t } from '$lib/translations';

 if (!$servicesSpecs) refreshServicesSpecs();

 export let data: {domain: string; history: string;};

 let domain: DomainInList | null = null;
 $: if ($domains_idx[data.domain]) {
     domain = $domains_idx[data.domain];
     zoneId = null;
 }

 let zoneId: string | null = null;
 $: if (domain && domain.zone_history && domain.zone_history.length > 0) {
     let zhidx = 0;
     if (data.history) {
         zhidx = domain.zone_history.indexOf(data.history);
     }
     if (zhidx >= 0) {
         zoneId = domain.zone_history[zhidx];
     } else {
         // TODO: Not found
     }
 }

 let zone: Zone | null = null;
 async function refreshZone(domain: DomainInList, zoneId: string) {
     zone = await getZone(domain, zoneId);
 }
 $: if (domain && zoneId) refreshZone(domain, zoneId);

 let sortedDomains: Array<string> = [];
 $: if (zone && zone.services) {
     const domains = Object.keys(zone.services);
     domains.sort(domainCompare);
     sortedDomains = domains;
 } else {
     sortedDomains = [];
 }

 let showSubdomainsList = false;

 let newSubdomainModalOpened = false;
 function addSubdomain() {
     newSubdomainModalOpened = true;
 }
</script>

{#if !domain}
    <div class="mt-5 text-center flex-fill">
        <Spinner label="Spinning" />
        <p>{$t('wait.loading')}</p>
    </div>
{:else if !zone || !domain.zone_history || domain.zone_history.length == 0}
    <div class="mt-4 text-center flex-fill">
        <Spinner label={$t('common.spinning')} />
        <p>{$t('wait.importing')}</p>
    </div>
{:else}
    <Row class="pt-3 flex-fill" style="max-width: 100%">
        <Col class="mb-5">
            {#if !showSubdomainsList}
                <Button
                    class="float-end"
                    color="secondary"
                    outline
                    on:click={() => showSubdomainsList = !showSubdomainsList}
                    style="position: relative; z-index: 2"
                >
                    <Icon name="list" aria-hidden="true" />
                </Button>
            {/if}

            <SubdomainList
                origin={domain}
                {showSubdomainsList}
                {sortedDomains}
                {zone}
                bind:newSubdomainModalOpened={newSubdomainModalOpened}
                on:update-zone-services={(event) => zone = event.detail}
            />
        </Col>
        {#if showSubdomainsList}
        <Col
            sm={3}
            class="sticky-top bg-light"
            style="margin-top: -10px; overflow-y: auto; max-height: 100vh"
        >
            <div class="d-flex gap-2 pb-2 sticky-top bg-light" style="padding-top: 10px">
                <Button
                    type="button"
                    color="secondary"
                    outline
                    size="sm"
                    class="ml-2 w-100"
                    on:click={addSubdomain}
                >
                    <Icon name="server" />
                    {$t('domains.add-a-subdomain')}
                </Button>
                <Button
                    color="secondary"
                    on:click={() => showSubdomainsList = !showSubdomainsList}
                >
                    <Icon name="list" aria-hidden="true" /><br>
                </Button>
            </div>
            {#each sortedDomains as dn}
                <a
                    href={'#' + (dn?dn:'@')}
                    title={fqdn(dn, domain.domain)}
                    class="d-block text-truncate font-monospace text-muted text-decoration-none"
                    style={'max-width: none; padding-left: ' + (dn === '' ? 0 : (dn.split('.').length * 10)) + 'px'}
                >
                    {fqdn(dn, domain.domain)}
                </a>
            {/each}
        </Col>
        {/if}
    </Row>
{/if}
