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

    let aliases: Record<string, Array<string>>;
    $: {
        const tmp: Record<string, Array<string>> = {};

        for (const dn of sortedDomains) {
            if (!zone.services[dn]) continue;

            zone.services[dn].forEach(function (svc) {
                if (svc._svctype === "svcs.CNAME") {
                    if (!tmp[svc.Service.Target]) {
                        tmp[svc.Service.Target] = [];
                    }
                    tmp[svc.Service.Target].push(dn);
                }
            });
        }
        if (tmp["@"]) tmp[""] = tmp["@"];

        aliases = tmp;
    }

    function showServiceModal(event: CustomEvent<ServiceCombined>) {
        ctrlService.Open(event.detail);
    }
</script>

{#each sortedDomainsWithIntermediate as dn}
    <SubdomainItem
        aliases={aliases[dn] ? aliases[dn] : []}
        {dn}
        {origin}
        zoneId={zone.id}
        services={zone.services[dn] ? zone.services[dn] : []}
        on:new-alias={() => ctrlAlias.Open(dn)}
        on:new-service={() => ctrlNewService.Open(dn)}
        on:show-service={showServiceModal}
        on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
    />
{/each}

<AliasModal
    {origin}
    {zone}
    on:update-zone-services={(event) => dispatch("update-zone-services", event.detail)}
/>
