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

<script lang="ts">
    import {
        Badge,
        Card,
        CardBody,
        CardText,
        CardTitle,
        CardSubtitle,
        Icon,
        ListGroup,
        ListGroupItem,
        Table,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { deleteZoneService, updateZoneService } from '$lib/api/zone';
    import ResourceInput from '$lib/components/forms/ResourceInput.svelte';
    import TableRecords from '$lib/components/domains/TableRecords.svelte';
    import { controls as ctrlRecord } from '$lib/components/domains/RecordModal.svelte';
    import { controls as ctrlService } from "$lib/components/services/ServiceModal.svelte";
    import { nsrrtype } from '$lib/dns';
    import type { Domain } from '$lib/model/domain';
    import type { ServiceCombined } from '$lib/model/service';
    import { ZoneViewGrid, ZoneViewList, ZoneViewRecords } from '$lib/model/usersettings';
    import type { ServiceRecord } from '$lib/model/zone';
    import { servicesSpecs } from '$lib/stores/services';
    import { thisZone } from "$lib/stores/thiszone";
    import { userSession } from '$lib/stores/usersession';
    import { t } from '$lib/translations';

    export let origin: Domain;
    export let service: ServiceCombined | null = null;
    export let zoneId: string;

    // FIXME: find which type is Card & ListGroup
    let component: any = Card;
    $: if ($userSession && $userSession.settings.zoneview !== ZoneViewGrid) {
        component = ListGroup;
    } else {
        component = Card;
    }

    let showDetails = false;
    function toggleDetails() {
        if (service) {
            if ($userSession && ($userSession.settings.zoneview === ZoneViewGrid || $userSession.settings.zoneview === ZoneViewRecords)) {
                ctrlService.Open(service);
            } else {
                showDetails = !showDetails;
            }
        }
    }

    function deleteService() {
        if (service == null) return;

        deleteZoneService(origin, zoneId, service).then((z) => {
            thisZone.set(z);
        });
    }

    function saveService() {
        if (service == null) return;

        updateZoneService(origin, zoneId, service).then((z) => {
            thisZone.set(z);
        });
    }
</script>

<svelte:component
    this={component}
    class={$userSession && $userSession.settings.zoneview !== ZoneViewList ? "card-hover" : ""}
    style={"cursor: pointer;" +
        (!service ? "border-style: dashed; " : "") +
        ($userSession && $userSession.settings.zoneview === ZoneViewGrid
            ? "width: 32%; min-width: 225px; margin-bottom: 1em; cursor: pointer;"
            : $userSession && $userSession.settings.zoneview === ZoneViewRecords
              ? "margin-bottom: .5em; cursor: pointer;"
              : "")}
    on:click={toggleDetails}
>
    {#if !$userSession || !$servicesSpecs}
        <div class="d-flex justify-content-center">
            <Spinner color="primary" />
        </div>
    {:else if $userSession.settings.zoneview === ZoneViewGrid}
        <CardBody title={service ? $servicesSpecs[service._svctype].name : undefined}>
            <div class="d-flex justify-content-between gap-1 mb-2">
                <CardTitle class="text-truncate mb-0">
                    {#if service}
                        {$servicesSpecs[service._svctype].name}
                    {:else}
                        <Icon name="plus" /> {$t("service.new")}
                    {/if}
                </CardTitle>
                {#if service && $servicesSpecs[service._svctype].categories && $servicesSpecs[service._svctype].categories.length && !$userSession.settings.showrrtypes}
                    <div class="d-flex align-items-center gap-1">
                        {#each $servicesSpecs[service._svctype].categories as category}
                            <Badge color="secondary">
                                {category}
                            </Badge>
                        {/each}
                    </div>
                {:else if $userSession.settings.showrrtypes && service && $servicesSpecs[service._svctype].record_types && $servicesSpecs[service._svctype].record_types.length}
                    <div class="d-flex align-items-center gap-1">
                        {#each $servicesSpecs[service._svctype].record_types as rrtype}
                            <Badge color="info">
                                {nsrrtype(rrtype)}
                            </Badge>
                        {/each}
                    </div>
                {/if}
            </div>
            <CardSubtitle class="mb-2 text-muted fst-italic">
                {#if service}
                    {$servicesSpecs[service._svctype].description}
                {:else}
                    {$t("service.new-description")}
                {/if}
            </CardSubtitle>
            <CardText style="font-size: 90%" class="text-truncate">
                {#if service && service._comment}
                    {service._comment}
                {/if}
            </CardText>
        </CardBody>
    {:else if service && ($userSession.settings.zoneview === ZoneViewList || $userSession.settings.zoneview === ZoneViewRecords)}
        <ListGroupItem
            class="px-2"
            on:click={toggleDetails}
        >
            <strong title={$servicesSpecs[service._svctype].description}>
                {$servicesSpecs[service._svctype].name}
            </strong>
            {#if $servicesSpecs[service._svctype].description}
                <small class="text-muted">
                    {$servicesSpecs[service._svctype].description}
                </small>
            {/if}
            {#if $servicesSpecs[service._svctype].categories}
                {#each $servicesSpecs[service._svctype].categories as category}
                    <Badge color="info" class="mx-1">
                        {category}
                    </Badge>
                {/each}
            {/if}
            {#if service._comment}
                <span class="fst-italic float-end text-muted">
                    {service._comment}
                </span>
            {/if}
        </ListGroupItem>
        {#if $userSession.settings.zoneview === ZoneViewList && showDetails}
            <ListGroupItem>
                <ResourceInput
                    editToolbar
                    specs={$servicesSpecs[service._svctype]}
                    type={service._svctype}
                    bind:value={service.Service}
                    on:delete-this-service={deleteService}
                    on:update-this-service={saveService}
                    on:update-zone-services={(event) => thisZone.set(event.detail)}
                />
            </ListGroupItem>
        {:else if $userSession.settings.zoneview === ZoneViewRecords}
            <ListGroupItem class="p-0">
                <TableRecords
                    service={service.Service}
                    on:show-record={(e) => ctrlRecord.Open({record: e.detail, service})}
                />
            </ListGroupItem>
        {/if}
    {/if}
</svelte:component>
