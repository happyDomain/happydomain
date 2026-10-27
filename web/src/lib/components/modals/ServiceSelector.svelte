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
        Open(_domain: string, _previousSteps?: number): void { },
    };
</script>

<script lang="ts">
    import { createEventDispatcher } from "svelte";

    import { ModalBody } from "@sveltestrap/sveltestrap";

    import Modal from "$lib/components/Modal.svelte";

    import { getProviderSpec } from "$lib/api/provider_specs";
    import { initializeService } from "$lib/api/service_specs";
    import ModalFooter from "$lib/components/modals/Footer.svelte";
    import ModalHeader from "$lib/components/modals/Header.svelte";
    import FilterServiceSelectorInput from "$lib/components/services/FilterServiceSelectorInput.svelte";
    import ServiceSelector from "$lib/components/services/ServiceSelector.svelte";
    import { filterServices } from "$lib/components/services/service-filter";
    import { serviceName } from "$lib/services/infos";
    import { t } from "$lib/translations";
    import type { Domain } from "$lib/model/domain";
    import type { HappydnsService } from "$lib/api-base/types.gen";
    import { providers_idx } from "$lib/stores/providers";
    import { servicesSpecsList, servicesSpecsLoaded } from "$lib/stores/services";
    import { filteredName } from "$lib/stores/serviceSelector";

    const dispatch = createEventDispatcher();

    const toggle = () => (isOpen = !isOpen);

    let dn: string = $state("");
    // Steps taken before this modal, as choosing the subdomain.
    let previousSteps = $state(0);

    interface Props {
        isOpen?: boolean;
        origin: Domain;
        value?: string | null;
        zservices: Record<string, Array<HappydnsService>>;
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

        // Get provider specs and find the first available matching service;
        // an origin without provider, as a zone template, has none.
        const provider = origin.id_provider ? $providers_idx[origin.id_provider] : undefined;
        if (origin.id_provider && !provider) return;
        (provider ? getProviderSpec(provider._srctype) : Promise.resolve(null)).then((prvdspecs) => {
            if ((provider && !prvdspecs) || !$servicesSpecsLoaded) return;

            // Use the shared filter function to get available services
            const { available } = filterServices(
                $servicesSpecsList,
                prvdspecs,
                zservices,
                dn,
                $filteredName,
                null,
                $t,
            );

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

    function Open(domain: string, previous = 0): void {
        $filteredName = "";
        dn = domain;
        previousSteps = previous;
        isOpen = true;
        value = null;
    }

    controls.Open = Open;

    let selectedName = $derived.by(() => {
        const svc = $servicesSpecsList.find((svc) => svc._svctype === value);
        return svc ? $serviceName(svc) : null;
    });
</script>

<Modal {isOpen} scrollable {toggle}>
    <ModalHeader
        {toggle}
        title={$t("service.new-on")}
        {dn}
        origin={origin.domain}
        step={previousSteps + 1}
        steps={previousSteps + 2}
    >
        <form onsubmit={submitFilter}>
            <FilterServiceSelectorInput />
        </form>
    </ModalHeader>
    <ModalBody class="pt-0">
        <form id="selectServiceForm" onsubmit={submitSelectorForm}>
            <ServiceSelector {dn} {origin} bind:value {zservices} />
        </form>
    </ModalBody>
    <ModalFooter
        canContinue={value !== null}
        continueLabel={selectedName ? $t("service.continue-with", { service: selectedName }) : undefined}
        form="selectServiceForm"
        step={1}
        {toggle}
    >
        {#snippet hints()}
            <span
                ><kbd class="px-1 py-0 border bg-transparent text-reset">↑</kbd>
                <kbd class="px-1 py-0 border bg-transparent text-reset">↓</kbd>
                {$t("service.keys-browse")}</span
            >
            <span><kbd class="px-1 py-0 border bg-transparent text-reset">↵</kbd> {$t("service.keys-continue")}</span>
        {/snippet}
    </ModalFooter>
</Modal>
