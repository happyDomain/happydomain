<script context="module" lang="ts">
 export const controls = { };
</script>

<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import NewSubdomainModal, { controls as ctrlNewSubdomainModal } from '$lib/components/domains/NewSubdomainModal.svelte';
 import { controls as ctrlServicePath } from '$lib/components/NewServicePath.svelte';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ServiceCombined } from '$lib/model/service';
 import type { Zone } from '$lib/model/zone';

 const dispatch = createEventDispatcher();

 export let origin: DomainInList | Domain;

 controls.Open = () => {
     ctrlNewSubdomainModal.Open();
 }
</script>

<NewSubdomainModal
    {origin}
    on:show-next-modal={(event) => ctrlServicePath.Open(event.detail)}
/>
