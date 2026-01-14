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

<script module lang="ts">
    export const controls = {
        Open(domain: string): void { },
    };
</script>

<script lang="ts">
    import { createEventDispatcher } from "svelte";

    import { Input, Modal, ModalBody } from "@sveltestrap/sveltestrap";

    import { getProviderSpec } from "$lib/api/provider_specs";
    import { initializeService } from "$lib/api/service_specs";
    import { getRrtype, newRR } from "$lib/dns_rr";
    import ModalFooter from "$lib/components/modals/Footer.svelte";
    import ModalHeader from "$lib/components/modals/Header.svelte";
    import FilterServiceSelectorInput from "$lib/components/services/FilterServiceSelectorInput.svelte";
    import ServiceSelector from "$lib/components/services/ServiceSelector.svelte";
    import { filterServices } from "$lib/components/services/service-filter";
    import { fqdn } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service.svelte";
    import { providers_idx } from "$lib/stores/providers";
    import { servicesSpecsList, servicesSpecsLoaded } from "$lib/stores/services";
    import { filteredName } from "$lib/stores/serviceSelector";

    const dispatch = createEventDispatcher();

    const toggle = () => (isOpen = !isOpen);

    let dn: string = $state("");

    interface Props {
        isOpen?: boolean;
        origin: Domain;
        value?: string | null;
        zservices: Record<string, Array<ServiceCombined>>;
    }
    let {
        isOpen = $bindable(false),
        origin,
        value = $bindable(null),
        zservices
    }: Props = $props();

    function submitSelectorForm(e: SubmitEvent) {
        e.preventDefault();

        if (value !== null) {
            toggle();
            initializeService(value).then((svc) => {
                dispatch("show-next-modal", { _svctype: value, _domain: dn, Service: svc });
            });
            $filteredName = "";
        }
    }

    function submitFilter(e: SubmitEvent) {
        e.preventDefault();

        // Get provider specs and find the first available matching service
        getProviderSpec($providers_idx[origin.id_provider]._srctype).then((prvdspecs) => {
            if (!prvdspecs || !$servicesSpecsLoaded) return;

            // Use the shared filter function to get available services
            const { available } = filterServices($servicesSpecsList, prvdspecs, zservices, dn, $filteredName);

            // Select the first available service and submit
            if (available.length > 0) {
                value = available[0]._svctype;

                // Submit the selector form
                const form = document.getElementById('selectServiceForm') as HTMLFormElement;
                if (form) {
                    form.requestSubmit();
                }
            }
        });
    }

    function Open(domain: string): void {
        $filteredName = "";
        dn = domain;
        isOpen = true;
        value = null;
    }

    controls.Open = Open;
</script>

<Modal {isOpen} scrollable {toggle}>
    <ModalHeader {toggle} dn={fqdn(dn, origin.domain)} />
    <ModalBody class="pt-0">
        <form onsubmit={submitFilter}>
            <FilterServiceSelectorInput class="my-2" />
        </form>
        <form id="selectServiceForm" onsubmit={submitSelectorForm}>
            <ServiceSelector {dn} {origin} bind:value {zservices} />
        </form>
    </ModalBody>
    <ModalFooter canContinue={value !== null} form="selectServiceForm" step={1} {toggle} />
</Modal>
