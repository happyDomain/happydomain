<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
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
    import type { ServiceCombined } from "$lib/model/service.svelte";

    export const controls = {
        Open(service: ServiceCombined): void {},
    };
</script>

<script lang="ts">
    import {
        Button,
        Icon,
        Input,
        Label,
        Offcanvas,
        OffcanvasBody,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { getServiceSpec } from "$lib/api/service_specs";
    import { deleteZoneService, updateZoneService } from "$lib/api/zone";
    import ServiceBadges from "./[[historyid]]/ServiceBadges.svelte";
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import { collectRRs } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import { navigate } from "$lib/stores/config";
    import { servicesSpecs, servicesSpecsLoaded } from "$lib/stores/services";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";

    interface Props {
        domain: Domain;
        selectedHistory?: string;
        isOpen?: boolean;
    }

    let { domain, selectedHistory = "", isOpen = $bindable(false) }: Props = $props();

    let service: ServiceCombined = $state({} as ServiceCombined);
    function Open(svc: ServiceCombined): void {
        isOpen = true;
        service = svc;
    }

    function toggle(): void {
        isOpen = !isOpen;
    }

    controls.Open = Open;

    let deleteInProgress = $state(false);

    function deleteService() {
        if (!service || !$thisZone) return;
        deleteInProgress = true;
        deleteZoneService(domain, $thisZone.id, service).then(
            (z) => {
                thisZone.set(z);
                deleteInProgress = false;
                isOpen = false;
            },
            (err) => {
                deleteInProgress = false;
                throw err;
            },
        );
    }

    let canDelete = $derived(
        !!service?._id &&
            service._svctype !== "abstract.Origin" &&
            service._svctype !== "abstract.NSOnlyOrigin",
    );

    let ttlSaveInProgress = $state(false);

    function saveTtl() {
        if (!service || !$thisZone) return;
        ttlSaveInProgress = true;
        updateZoneService(domain, $thisZone.id, service).then(
            (z) => {
                thisZone.set(z);
                setTimeout(() => {
                    ttlSaveInProgress = false;
                }, 500);
            },
            (err) => {
                ttlSaveInProgress = false;
                throw err;
            },
        );
    }
</script>

<Offcanvas
    header={service._svctype && $servicesSpecsLoaded ? $servicesSpecs[service._svctype].name : ""}
    {isOpen}
    {toggle}
    body={false}
    placement="end"
    class="bg-light"
    style="width: min(max(400px, 35vw), 100vw)"
>
    <OffcanvasBody class="d-flex flex-column pt-0">
        {#if service._svctype && $servicesSpecsLoaded && $servicesSpecs[service._svctype]}
            <p class="text-muted mb-1">
                {$servicesSpecs[service._svctype].description}
            </p>
        {/if}
        <div class="d-flex justify-content-between mb-3">
            {#if service && service._comment}
                <p class="mb-1">
                    {service._comment}
                </p>
            {/if}
            {#if service && service._svctype}
                <ServiceBadges class="mb-2" {service} />
            {/if}
        </div>
        {#if service._svctype && service.Service}
            {#await getServiceSpec(service._svctype) then specs}
                {@const rrs = collectRRs(specs.fields, service.Service)}
                {#each rrs as rr, i}
                    <RecordLine dn={service._domain || ""} origin={domain} bind:rr={rrs[i]} />
                {/each}
            {/await}
        {/if}
        <div class="flex-fill"></div>
        {#if service._id}
            <div class="d-flex align-items-center gap-2 mt-2">
                <Label for="offcanvas_svc_ttl" title={$t("service.ttl-long")} class="mb-0"
                    >{$t("service.default-ttl")}</Label
                >
                <Input
                    id="offcanvas_svc_ttl"
                    bsSize="sm"
                    min="0"
                    type="number"
                    style="width: 8em"
                    title={$t("service.ttl-tip")}
                    bind:value={service._ttl}
                    on:change={(e: any) => {
                        service._ttl = parseInt(e.target.value, 10) || 0;
                        saveTtl();
                    }}
                />
                {#if ttlSaveInProgress}
                    <Spinner size="sm" />
                {/if}
            </div>
        {/if}
        <div class="d-flex flex-column-reverse gap-2 mt-2">
            {#if canDelete}
                <Button
                    size="sm"
                    color="danger"
                    outline
                    disabled={deleteInProgress}
                    on:click={deleteService}
                >
                    {#if deleteInProgress}
                        <Spinner size="sm" />
                    {:else}
                        <Icon name="trash" />
                    {/if}
                    {$t("service.delete")}
                </Button>
            {/if}
            <Button
                size="sm"
                color="info"
                outline
                on:click={() => {
                    isOpen = false;
                    navigate(
                        `/domains/${encodeURIComponent(domain.domain)}/${encodeURIComponent(selectedHistory ?? "")}/${encodeURIComponent(service._domain ? service._domain : "@")}/${encodeURIComponent(service._id!)}`,
                    );
                }}
            >
                <Icon name="pencil" />
                {$t("service.update")}
            </Button>
        </div>
    </OffcanvasBody>
</Offcanvas>
