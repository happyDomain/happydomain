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
  import { run } from 'svelte/legacy';

    import { createEventDispatcher } from "svelte";

    import { deleteZoneService } from "$lib/api/zone";
    import { controls as ctrlNewService } from "$lib/components/services/NewServicePath.svelte";
    import { controls as ctrlService } from "$lib/components/services/ServiceModal.svelte";
    import Service from "./Service.svelte";
    import SubdomainItemHeader from "./SubdomainItemHeader.svelte";
    import { isReverseZone } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service";
    import { ZoneViewGrid } from "$lib/model/usersettings";
    import { thisZone } from "$lib/stores/thiszone";
    import { userSession } from "$lib/stores/usersession";

    const dispatch = createEventDispatcher();

  interface Props {
    dn: string;
    origin: Domain;
    services: Array<ServiceCombined>;
  }

  let { dn, origin, services }: Props = $props();

    let reverseZone = $state(false);
    run(() => {
    reverseZone = isReverseZone(origin.domain);
  });

    let showResources = $state(true && (services.length > 1 || (services.length === 1 && services[0]._svctype !== "svcs.CNAME" && services[0]._svctype !== "svcs.PTR")));

    function showServiceModal(event: CustomEvent<ServiceCombined>) {
        ctrlService.Open(event.detail);
    }
</script>

{#if $thisZone}
  <div id={dn ? dn : "@"}>
    <SubdomainItemHeader
        {dn}
        {origin}
        {services}
        zoneId={$thisZone.id}
        {reverseZone}
        bind:showResources={showResources}
    />
    {#if showResources}
        <div
            class:d-flex={showResources &&
                $userSession &&
                $userSession.settings.zoneview === ZoneViewGrid}
            class:justify-content-around={showResources &&
                $userSession &&
                $userSession.settings.zoneview === ZoneViewGrid}
            class:flex-wrap={showResources &&
                $userSession &&
                $userSession.settings.zoneview === ZoneViewGrid}
        >
            {#each services as service}
                {#key service}
                    <Service
                        {origin}
                        {service}
                        zoneId={$thisZone.id}
                        on:show-service={showServiceModal}
                        on:update-zone-services={(event) => $thisZone.set(event.detail)}
                    />
                {/key}
            {/each}
            {#if $userSession && $userSession.settings.zoneview === ZoneViewGrid}
                <Service
                    {origin}
                    zoneId={$thisZone.id}
                    on:show-service={() => ctrlNewService.Open(dn)}
                    on:update-zone-services={(event) => $thisZone.set(event.detail)}
                />
            {/if}
        </div>
    {/if}
  </div>
{/if}
