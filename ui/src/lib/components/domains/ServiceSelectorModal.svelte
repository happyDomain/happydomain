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
 import type { Domain } from '$lib/model/domain';
 import type { ServiceCombined } from '$lib/model/service';

 const dispatch = createEventDispatcher();

 export let isOpen = false;
 const toggle = () => (isOpen = !isOpen);

 $: {
     if (isOpen) {
         value = "";
     }
 }

 export let dn: string;
 export let origin: Domain;
 export let value: string | null = null;
 export let zservices: Record<string, Array<ServiceCombined>>;

 function submitSelectorForm() {
     if (value !== null) {
         toggle();
         dispatch("show-next-modal", {_svctype: value, _domain: dn, Service: { }});
     }
 }
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
                class="mb-2"
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
