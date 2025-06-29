<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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
    import type { Snippet } from "svelte";

    import { getServiceSpec } from "$lib/api/service_specs";
    import Service from "./Service.svelte";
    import SVC from "./SVC.svelte";
    import { controls as ctrlService } from "$lib/components/services/ServiceModal.svelte";
    import { printRR } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service";
    import { ZoneViewGrid, ZoneViewRecords } from "$lib/model/usersettings";
    import { servicesSpecs } from "$lib/stores/services";
    import { userSession } from "$lib/stores/usersession";
    import { t } from "$lib/translations";

    interface Props {
        dn: string;
        origin: Domain;
        services: Array<ServiceCombined>;
        zoneId: string;
    }

    let { dn, origin, services, zoneId }: Props = $props();
</script>

{#if $userSession.settings.zoneview === ZoneViewRecords}
    {#if !services || services.length == 0}
        <div class="d-flex justify-content-center align-items-center">
            {$t("common.no-content")}
        </div>
    {:else}
        <table class="table table-hover table-striped" style="table-layout: fixed;">
            <tbody>
                {#each services as service}
                    {#await getServiceSpec(service._svctype)}
                    {:then specs}
                        <SVC
                            type={service._svctype}
                            {specs}
                            value={service.Service}
                        >
                            {#snippet aservice(type, value)}
                                {#if value}
                                    <tr>
                                        <td
                                            class="d-flex justify-content-between"
                                            style="cursor: pointer"
                                            onclick={() => ctrlService.Open(service)}
                                        >
                                            <div class="text-truncate font-monospace">
                                                {printRR(value, dn, origin.domain)}
                                            </div>
                                            <strong style="white-space: nowrap">{$servicesSpecs[service._svctype].name}</strong>
                                        </td>
                                    </tr>
                                {/if}
                            {/snippet}
                        </SVC>
                    {/await}
                {/each}
            </tbody>
        </table>
    {/if}
{:else}
    <div
        class:d-flex={$userSession &&
                     $userSession.settings.zoneview === ZoneViewGrid}
        class:justify-content-around={$userSession &&
                                     $userSession.settings.zoneview === ZoneViewGrid}
        class:flex-wrap={$userSession &&
                        $userSession.settings.zoneview === ZoneViewGrid}
    >
        {#each services as service}
            {#key service}
                <Service
                    {origin}
                    {service}
                    {zoneId}
                />
            {/key}
        {/each}
        {#if $userSession && $userSession.settings.zoneview === ZoneViewGrid}
            <Service
                {origin}
                {zoneId}
            />
        {/if}
    </div>
{/if}
