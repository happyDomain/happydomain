<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Modal,
     ModalBody,
     Spinner,
 } from 'sveltestrap';

 import { addZoneService, deleteZoneService, updateZoneService } from '$lib/api/zone';
 import ModalFooter from '$lib/components/domains/ModalFooter.svelte';
 import ModalHeader from '$lib/components/domains/ModalHeader.svelte';
 import ResourceInput from '$lib/components/ResourceInput.svelte';
 import { fqdn } from '$lib/dns';
 import type { Domain } from '$lib/model/domain';
 import type { ServiceCombined } from '$lib/model/service';
 import { servicesSpecs } from '$lib/stores/services';
 import type { Zone } from '$lib/model/zone';

 const dispatch = createEventDispatcher();

 export let isOpen = false;
 const toggle = () => (isOpen = !isOpen);

 export let origin: Domain;
 export let service: ServiceCombined;
 export let zone: Zone;

 let addServiceInProgress = false;
 let deleteServiceInProgress = false;

 function deleteService() {
     deleteServiceInProgress = true;
     deleteZoneService(origin, zone.id, service).then(
         (z) => {
             dispatch("update-zone-services", z);
             deleteServiceInProgress = false;
             toggle();
         },
         (err) => {
             deleteServiceInProgress = false;
             throw err;
         }
     );
 }

 function submitServiceForm() {
     addServiceInProgress = true;

     let action = addZoneService;
     if (service._id) {
         action = updateZoneService;
     }

     action(origin, zone.id, service).then(
         (z) => {
             dispatch("update-zone-services", z);
             addServiceInProgress = false;
             toggle();
         },
         (err) => {
             addServiceInProgress = false;
             throw err;
         }
     );
 }
</script>

<Modal
    {isOpen}
    {toggle}
    scrollable
    size="lg"
>
    <ModalHeader
        {toggle}
        dn={fqdn(service._domain, origin.domain)}
        update={service._id != undefined}
    />
    <ModalBody>
        <form
            id="addSvcForm"
            on:submit|preventDefault={submitServiceForm}
        >
            {#if $servicesSpecs == null}
                <div class="d-flex justify-content-center">
                    <Spinner />
                </div>
            {:else}
                <ResourceInput
                    edit
                    specs={$servicesSpecs[service._svctype]}
                    type={service._svctype}
                    bind:value={service.Service}
                    update-my-services="$emit('update-my-services', $event)"
                />
            {/if}
        </form>
    </ModalBody>
    <ModalFooter
        step={2}
        {addServiceInProgress}
        canDelete={service._svctype !== 'abstract.Origin'}
        {deleteServiceInProgress}
        {service}
        {toggle}
        update={service._id != undefined}
        on:delete-service={deleteService}
    />
</Modal>
