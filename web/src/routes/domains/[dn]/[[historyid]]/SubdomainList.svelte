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
    import { createEventDispatcher } from "svelte";

    import AliasModal, { controls as ctrlAlias } from "./AliasModal.svelte";
    import { controls as ctrlNewService } from "$lib/components/services/NewServicePath.svelte";
    import { controls as ctrlRecord } from '$lib/components/domains/RecordModal.svelte';
    import { controls as ctrlService } from "$lib/components/services/ServiceModal.svelte";
    import SubdomainItem from "./SubdomainItem.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service";
    import type { Zone } from "$lib/model/zone";

    const dispatch = createEventDispatcher();

    export let origin: Domain;
    export let sortedDomains: Array<string>;
    export let sortedDomainsWithIntermediate: Array<string>;
    export let zone: Zone;

    export let newSubdomainModalOpened = false;
    $: if (newSubdomainModalOpened) {
        ctrlNewSubdomain.Open();
    }

    function showRecordModal(event) {
        ctrlRecord.Open(event.detail);
    }

    function showServiceModal(event: CustomEvent<ServiceCombined>) {
        ctrlService.Open(event.detail);
    }
</script>

{#each sortedDomainsWithIntermediate as dn}
    <SubdomainItem
        {dn}
        {origin}
        zoneId={zone.id}
        services={zone.services[dn] ? zone.services[dn] : []}
        on:new-alias={() => ctrlAlias.Open(dn)}
        on:new-service={() => ctrlNewService.Open(dn)}
        on:show-record={showRecordModal}
        on:show-service={showServiceModal}
        on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
    />
{/each}

<AliasModal
    {origin}
    {zone}
    on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
/>
