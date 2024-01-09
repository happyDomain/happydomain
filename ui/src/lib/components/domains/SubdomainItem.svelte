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
 import { createEventDispatcher } from 'svelte';

 import {
     Badge,
     Button,
     Icon,
     Popover,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import { deleteZoneService } from '$lib/api/zone';
 import Service from '$lib/components/domains/Service.svelte';
 import { fqdn } from '$lib/dns';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ServiceCombined } from '$lib/model/service';
 import { ZoneViewGrid } from '$lib/model/usersettings';
 import { userSession } from '$lib/stores/usersession';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let aliases: Array<string> = [];
 export let dn: string;
 export let origin: Domain | DomainInList;
 export let services: Array<ServiceCombined>;
 export let zoneId: string;

 let showResources = true;

 function isCNAME(services: Array<ServiceCombined>) {
     return services.length === 1 && services[0]._svctype === 'svcs.CNAME';
 }

 let deleteServiceInProgress = false;
 function deleteCNAME() {
     deleteServiceInProgress = true;
     deleteZoneService(origin, zoneId, services[0]).then(
         (z) => {
             dispatch("update-zone-services", z);
             deleteServiceInProgress = false;
         },
         (err) => {
             deleteServiceInProgress = false;
             throw err;
         }
     );
 }

 function showServiceModal(service: ServiceCombined) {
     dispatch("show-service", service);
 }
</script>

{#if isCNAME(services)}
    <div id={dn}>
        <h2
            class="sticky-top"
            style="background: white; z-index: 1"
        >
            <span style="white-space: nowrap">
                <Icon name="link" />
                <span
                    class="font-monospace"
                    title={fqdn(dn, origin.domain)}
                >
                    {fqdn(dn, origin.domain)}
                </span>
            </span>
            <span style="white-space: nowrap">
                <Icon name="arrow-right" />
                <span class="font-monospace">
                    {services[0].Service.Target}
                </span>
            </span>
            <Button
                type="button"
                color="primary"
                size="sm"
                class="ml-2"
                on:click={() => dispatch("new-service")}
            >
                <Icon name="plus" />
                {$t('service.add')}
            </Button>
            <Button
                type="button"
                color="info"
                outline
                size="sm"
                class="ml-2"
                on:click={() => showServiceModal(services[0])}
            >
                <Icon name="pencil" />
                {$t('domains.edit-target')}
            </Button>
            <Button
                type="button"
                color="danger"
                disabled={deleteServiceInProgress}
                outline
                size="sm"
                class="ml-2"
                on:click={deleteCNAME}
            >
                {#if deleteServiceInProgress}
                    <Spinner size="sm" />
                {:else}
                    <Icon name="x-circle" />
                {/if}
                {$t('domains.drop-alias')}
            </Button>
        </h2>
    </div>
{:else}
    <div id={dn?dn:'@'}>
        <div
            class="d-flex align-items-center sticky-top mb-2 gap-2"
            style="background: white; z-index: 1"
        >
            <h2
                style="white-space: nowrap; cursor: pointer;"
                class="mb-0"
                on:click={() => showResources = !showResources}
                on:keypress={() => showResources = !showResources}
            >
                {#if showResources}
                    <Icon name="chevron-down" />
                {:else}
                    <Icon name="chevron-right" />
                {/if}
                <span
                    class="font-monospace"
                    title={fqdn(dn, origin.domain)}
                >
                    {fqdn(dn, origin.domain)}
                </span>
            </h2>
            {#if aliases.length != 0}
                <Badge
                    id={"popoverbadge-" + dn.replace('.', '__')}
                    style="cursor: pointer;"
                >
                    + {$t('domains.n-aliases', {n: aliases.length})}
                </Badge>
                <Popover
                    dismissible
                    placement="bottom"
                    target={"popoverbadge-" + dn.replace('.', '__')}
                    class="font-monospace"
                >
                    {#each aliases as alias}
                        <a href={"#" + alias}>
                            {alias}
                        </a>
                        <br>
                    {/each}
                </Popover>
            {/if}
            {#if $userSession && $userSession.settings.zoneview !== ZoneViewGrid}
                <Button
                    type="button"
                    color="primary"
                    size="sm"
                    on:click={() => dispatch("new-service")}
                >
                    <Icon name="plus" />
                    {$t('domains.add-a-service')}
                </Button>
            {/if}
            <Button
                type="button"
                color="primary"
                outline
                size="sm"
                on:click={() => dispatch("new-alias")}
            >
                <Icon name="link" />
                {$t('domains.add-an-alias')}
            </Button>
        </div>
        {#if showResources}
            <div
                class:d-flex={showResources && $userSession && $userSession.settings.zoneview === ZoneViewGrid}
                class:justify-content-around={showResources && $userSession && $userSession.settings.zoneview === ZoneViewGrid}
                class:flex-wrap={showResources && $userSession && $userSession.settings.zoneview === ZoneViewGrid}
            >
                {#each services as service}
                    {#key service}
                        <Service
                            {origin}
                            {service}
                            {zoneId}
                            on:show-service={(event) => showServiceModal(event.detail)}
                            on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
                        />
                    {/key}
                {/each}
                {#if $userSession && $userSession.settings.zoneview === ZoneViewGrid}
                    <Service
                        {origin}
                        {zoneId}
                        on:show-service={() => dispatch("new-service")}
                        on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
                    />
                {/if}
            </div>
        {/if}
    </div>
{/if}
