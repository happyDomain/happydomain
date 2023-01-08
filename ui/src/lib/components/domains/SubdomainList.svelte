<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import AliasModal from '$lib/components/domains/AliasModal.svelte';
 import NewSubdomainModal from '$lib/components/domains/NewSubdomainModal.svelte';
 import ServiceModal from '$lib/components/domains/ServiceModal.svelte';
 import ServiceSelectorModal from '$lib/components/domains/ServiceSelectorModal.svelte';
 import SubdomainItem from '$lib/components/domains/SubdomainItem.svelte';
 import type { Domain } from '$lib/model/domain';
 import type { ServiceCombined } from '$lib/model/service';
 import type { Zone } from '$lib/model/zone';

 const dispatch = createEventDispatcher();

 export let origin: Domain;
 export let showSubdomainsList: boolean;
 export let sortedDomains: Array<string>;
 export let zone: Zone;

 let aliases: Record<string, Array<string>>;
 $: {
     const tmp: Record<string, Array<string>> = { };

     for (const dn of sortedDomains) {
         if (!zone.services[dn]) continue;

         zone.services[dn].forEach(function (svc) {
             if (svc._svctype === 'svcs.CNAME') {
                 if (!tmp[svc.Service.Target]) {
                     tmp[svc.Service.Target] = []
                 }
                 tmp[svc.Service.Target].push(dn)
             }
         })
     }

     aliases = tmp;
 }

 let newSubdomainModalOpened = false;
 let subdomainModal = "";
 let newAliasModalOpened = false;

 let serviceSelectorModalOpened = false;
 let serviceSelectedModal: string | null = null;
 function showServiceSelectorModal(subdomain: string) {
     subdomainModal = subdomain;
     serviceSelectorModalOpened = true;
 }

 let serviceModalOpened = false;
 let serviceModalService: ServiceCombined | null = null;
 function showServiceModal(event: CustomEvent<ServiceCombined>) {
     serviceModalService = event.detail;
     serviceModalOpened = true;
 }
</script>

{#each sortedDomains as dn}
    <SubdomainItem
        aliases={aliases[dn]?aliases[dn]:[]}
        {dn}
        {origin}
        {showSubdomainsList}
        zoneId={zone.id}
        services={zone.services[dn]?zone.services[dn]:[]}
        on:new-alias={() => {subdomainModal = dn; newAliasModalOpened = true;}}
        on:new-service={() => showServiceSelectorModal(dn)}
        on:new-subdomain={() => newSubdomainModalOpened = true}
        on:show-service={showServiceModal}
        on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
    />
{/each}

<NewSubdomainModal
    bind:isOpen={newSubdomainModalOpened}
    {origin}
    bind:value={subdomainModal}
    on:show-next-modal={(event) => showServiceSelectorModal(event.detail)}
/>
<ServiceSelectorModal
    bind:isOpen={serviceSelectorModalOpened}
    dn={subdomainModal}
    {origin}
    bind:value={serviceSelectedModal}
    zservices={zone.services}
    on:show-next-modal={showServiceModal}
/>
{#if serviceModalService}
    <ServiceModal
        bind:isOpen={serviceModalOpened}
        {origin}
        service={serviceModalService}
        {zone}
        on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
    />
{/if}
<AliasModal
    bind:isOpen={newAliasModalOpened}
    dn={subdomainModal}
    {origin}
    {zone}
    on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
/>
