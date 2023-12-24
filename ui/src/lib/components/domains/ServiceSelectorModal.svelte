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
 export const controls = { };
</script>

<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Modal,
     ModalBody,
 } from 'sveltestrap';

 import ModalFooter from '$lib/components/domains/ModalFooter.svelte';
 import ModalHeader from '$lib/components/domains/ModalHeader.svelte';
 import ServiceSelector from '$lib/components/ServiceSelector.svelte';
 import { fqdn } from '$lib/dns';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ServiceCombined } from '$lib/model/service';

 const dispatch = createEventDispatcher();

 export let isOpen = false;
 const toggle = () => (isOpen = !isOpen);

 export let dn: string;
 export let origin: Domain | DomainInList;
 export let value: string | null = null;
 export let zservices: Record<string, Array<ServiceCombined>>;

 function submitSelectorForm() {
     if (value !== null) {
         toggle();
         dispatch("show-next-modal", {_svctype: value, _domain: dn, Service: { }});
     }
 }

 function Open(domain: string): void {
     dn = domain;
     isOpen = true;
     value = '';
 }

 controls.Open = Open;
</script>

<Modal
    {isOpen}
    scrollable
    {toggle}
>
    <ModalHeader {toggle} dn={fqdn(dn, origin.domain)} />
    <ModalBody class="pt-0">
        <form
            id="selectServiceForm"
            on:submit|preventDefault={submitSelectorForm}
        >
            <ServiceSelector
                {dn}
                {origin}
                bind:value={value}
                {zservices}
            />
        </form>
    </ModalBody>
    <ModalFooter
        canContinue={value !== null}
        form="selectServiceForm"
        step={1}
        {toggle}
    />
</Modal>
