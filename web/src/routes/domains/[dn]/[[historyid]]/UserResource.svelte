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
    import { getServiceSpec } from "$lib/api/service_specs";
    import ServiceCard from "./ServiceCard.svelte";
    import ServiceBadges from "./ServiceBadges.svelte";
    import Service from "$lib/components/services/Service.svelte";
    import RecordText from "$lib/components/records/RecordText.svelte";
    import { controls as ctrlRecord } from "$lib/components/modals/Record.svelte";
    import { controls as ctrlService } from "$lib/components/modals/Service.svelte";
    import { printRR } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service";
    import { ZoneViewGrid, ZoneViewList, ZoneViewRecords } from "$lib/model/usersettings";
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

{#if $userSession.settings.zoneview === ZoneViewRecords || $userSession.settings.zoneview === ZoneViewList}
    {#if !services || services.length == 0}
        <div class="d-flex justify-content-center align-items-center">
            {$t("common.no-content")}
        </div>
    {:else}
        <table class="table table-bordered table-hover table-striped" style="table-layout: fixed;">
            <tbody>
                {#each services as service}
                    {#if $userSession.settings.zoneview === ZoneViewRecords}
                        {#await getServiceSpec(service._svctype)}
                        {:then specs}
                            <Service
                                type={service._svctype}
                                {specs}
                                value={service.Service}
                            >
                                {#snippet aservice(type, rr)}
                                    {#if rr}
                                        <tr>
                                            <td
                                                class="d-flex justify-content-between"
                                                style="cursor: pointer"
                                                onclick={() => ctrlRecord.Open(rr, service._domain)}
                                            >
                                                <RecordText
                                                    {dn}
                                                    {origin}
                                                    {rr}
                                                />
                                                <strong class="text-muted" style="white-space: nowrap">{$servicesSpecs[service._svctype].name}</strong>
                                            </td>
                                        </tr>
                                    {/if}
                                {/snippet}
                            </Service>
                        {/await}
                    {:else if $servicesSpecs}
                        <tr>
                            <td
                                class="d-flex justify-content-between gap-2"
                                style="cursor: pointer"
                                onclick={() => ctrlService.Open(service)}
                            >
                                <div style="min-width: 0" class="d-flex align-items-center gap-1">
                                    <strong
                                        title={$servicesSpecs[service._svctype].description ? $servicesSpecs[service._svctype].description : null}
                                        style="white-space: nowrap"
                                    >
                                        {$servicesSpecs[service._svctype].name}
                                    </strong>
                                    {#if service._comment}
                                        <span
                                            class="flex-shrink-1 fst-italic text-muted"
                                            title={service._comment}
                                            style="min-width: 0"
                                        >
                                            {service._comment}
                                        </span>
                                    {/if}
                                </div>
                                <ServiceBadges {service} />
                            </td>
                        </tr>
                    {/if}
                {/each}
            </tbody>
        </table>
    {/if}
{:else}
    <div class="d-flex justify-content-around flex-wrap">
        {#each services as service}
            {#key service}
                <ServiceCard
                    {origin}
                    {service}
                    {zoneId}
                />
            {/key}
        {/each}
        <ServiceCard
            {origin}
            {zoneId}
        />
    </div>
{/if}
