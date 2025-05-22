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

    import SubdomainItem from "./SubdomainItem.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service";
    import { thisZone } from "$lib/stores/thiszone";

    const dispatch = createEventDispatcher();

    interface Props {
        origin: Domain;
        sortedDomainsWithIntermediate: Array<string>;
    }

    let { origin, sortedDomainsWithIntermediate }: Props = $props();
</script>

{#if $thisZone}
    {#each sortedDomainsWithIntermediate as dn}
        <SubdomainItem
            {dn}
            {origin}
            services={$thisZone.services[dn] ? $thisZone.services[dn] : []}
        />
    {/each}
{/if}
