<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import AliasModal, { controls as ctrlAlias } from '$lib/components/domains/AliasModal.svelte';
 import { controls as ctrlNewService } from '$lib/components/NewServicePath.svelte';
 import { controls as ctrlService } from '$lib/components/domains/ServiceModal.svelte';
 import SubdomainItem from '$lib/components/domains/SubdomainItem.svelte';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ServiceCombined } from '$lib/model/service';
 import type { Zone } from '$lib/model/zone';

 const dispatch = createEventDispatcher();

 export let origin: DomainInList | Domain;
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
     if (tmp['@']) tmp[""] = tmp["@"];

     aliases = tmp;
 }

 export let newSubdomainModalOpened = false;
 $: if (newSubdomainModalOpened) {
     ctrlNewSubdomain.Open();
 }

 function showServiceModal(event: CustomEvent<ServiceCombined>) {
     ctrlService.Open(event.detail);
 }
</script>

{#each sortedDomains as dn}
    <SubdomainItem
        aliases={aliases[dn]?aliases[dn]:[]}
        {dn}
        {origin}
        zoneId={zone.id}
        services={zone.services[dn]?zone.services[dn]:[]}
        on:new-alias={() => ctrlAlias.Open(dn)}
        on:new-service={() => ctrlNewService.Open(dn)}
        on:show-service={showServiceModal}
        on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
    />
{/each}

<AliasModal
    {origin}
    {zone}
    on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
/>
