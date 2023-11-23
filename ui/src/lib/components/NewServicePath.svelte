<script context="module" lang="ts">
 export const controls = { };
</script>

<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import ServiceSelectorModal, { controls as ctrlServiceSelector } from '$lib/components/domains/ServiceSelectorModal.svelte';
 import { controls as ctrlService } from '$lib/components/domains/ServiceModal.svelte';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ServiceCombined } from '$lib/model/service';
 import type { Zone } from '$lib/model/zone';

 const dispatch = createEventDispatcher();

 export let origin: DomainInList | Domain;
 export let zone: Zone;

 function Open(domain: string): void {
     ctrlServiceSelector.Open(domain);
 }

 controls.Open = Open;
</script>

<ServiceSelectorModal
    {origin}
    zservices={zone.services}
    on:show-next-modal={(event) => ctrlService.Open(event.detail)}
/>
