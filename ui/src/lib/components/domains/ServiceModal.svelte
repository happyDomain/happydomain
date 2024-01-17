<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script context="module" lang="ts">
 import type { ModalController } from '$lib/model/modal_controller';

 export const controls: ModalController = { };
</script>

<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Modal,
     ModalBody,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import { addZoneService, deleteZoneService, updateZoneService } from '$lib/api/zone';
 import ModalFooter from '$lib/components/domains/ModalFooter.svelte';
 import ModalHeader from '$lib/components/domains/ModalHeader.svelte';
 import ResourceInput from '$lib/components/ResourceInput.svelte';
 import { fqdn } from '$lib/dns';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ServiceCombined } from '$lib/model/service';
 import { servicesSpecs } from '$lib/stores/services';
 import type { Zone } from '$lib/model/zone';

 const dispatch = createEventDispatcher();

 export let isOpen = false;
 const toggle = () => (isOpen = !isOpen);

 export let origin: Domain | DomainInList;
 export let zone: Zone;

 let service: ServiceCombined | undefined = undefined;

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

 function Open(svc: ServiceCombined): void {
     service = svc;
     isOpen = true;
 }

 controls.Open = Open;
</script>

{#if service && service._domain !== undefined}
<Modal
    {isOpen}
    scrollable
    size="lg"
    {toggle}
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
                    update-this-services="$emit('update-this-services', $event)"
                />
            {/if}
        </form>
    </ModalBody>
    {#if zone}
    <ModalFooter
        step={2}
        {addServiceInProgress}
        canDelete={service._svctype !== 'abstract.Origin' && service._svctype !== 'abstract.NSOnlyOrigin'}
        {deleteServiceInProgress}
        {origin}
        {service}
        {toggle}
        update={service._id != undefined}
        zoneId={zone.id}
        on:delete-service={deleteService}
    />
    {/if}
</Modal>
{/if}
