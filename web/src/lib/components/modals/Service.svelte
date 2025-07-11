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
        Open(svc: ServiceCombined): void { },
    };
</script>

<script lang="ts">
    // @ts-ignore
    import { escape } from "html-escaper";

    import { createEventDispatcher } from "svelte";

    import { Modal, ModalHeader, ModalBody, Spinner } from "@sveltestrap/sveltestrap";

    import { getServiceSpec } from "$lib/api/service_specs";
    import { addZoneService, deleteZoneService, updateZoneService } from "$lib/api/zone";
    import ModalFooter from "$lib/components/modals/Footer.svelte";
    import ResourceInput from "$lib/components/inputs/Resource.svelte";
    import ServiceEditor from "$lib/components/services/ServiceEditor.svelte";
    import { fqdn } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service";
    import { servicesSpecs } from "$lib/stores/services";
    import type { Zone } from "$lib/model/zone";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    const toggle = () => (isOpen = !isOpen);

    interface Props {
        isOpen?: boolean;
        origin: Domain;
        zone: Zone;
    }

    let { isOpen = $bindable(false), origin, zone }: Props = $props();

    let service: ServiceCombined | undefined = $state(undefined);

    let addServiceInProgress = $state(false);
    let deleteServiceInProgress = $state(false);

    function deleteService() {
        if (!service) return;

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
            },
        );
    }

    function submitServiceForm(e: FormDataEvent) {
        e.preventDefault();

        if (!service) return;

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
            },
        );
    }

    function Open(svc: ServiceCombined): void {
        service = svc;
        isOpen = true;
    }

    controls.Open = Open;
</script>

{#if service && service._domain !== undefined}
    <Modal {isOpen} scrollable size="lg" {toggle}>
        <ModalHeader
            class={service._id != undefined ? "bg-warning-subtle" : "bg-primary-subtle"}
            {toggle}
        >
            {#if service._id != undefined}
                {#if $servicesSpecs}
                    {$t("common.update-what", { what: $servicesSpecs[service._svctype].name })}
                {:else}
                    {$t("service.update")}
                {/if}
            {:else}
                {@html $t("service.form-new", {
                    domain: `<span class="font-monospace">${escape(fqdn(service._domain, origin.domain))}</span>`,
                  })}
            {/if}
        </ModalHeader>
        <ModalBody class="pt-0">
            <form class="mt-2" id="addSvcForm" onsubmit={submitServiceForm}>
                {#if $servicesSpecs == null}
                    <div class="d-flex justify-content-center">
                        <Spinner />
                    </div>
                {:else}
                    <ServiceEditor
                        dn={service._domain}
                        {origin}
                        type={service._svctype}
                        bind:value={service.Service}
                    />
                    <!--ResourceInput
                        edit
                        specs={$servicesSpecs[service._svctype]}
                        type={service._svctype}
                        bind:value={service.Service}
                        on:delete-this-service={(event) => dispatch("delete-this-service", event.detail)}
                        on:update-this-service={(event) => dispatch("update-this-service", event.detail)}
                    /-->
                {/if}
            </form>
        </ModalBody>
        {#if zone}
            <ModalFooter
                step={2}
                {addServiceInProgress}
                canDelete={service._svctype !== "abstract.Origin" &&
                    service._svctype !== "abstract.NSOnlyOrigin"}
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
