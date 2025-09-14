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
    import type { Snippet } from "svelte";

    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service.svelte";
    import { thisZone } from "$lib/stores/thiszone";

    interface Props {
        subdomain: Snippet<[string, Array<ServiceCombined>]>;
        subdomains: Array<string>;
    }

    let { subdomain, subdomains }: Props = $props();
</script>

{#if $thisZone}
    {#each subdomains as dn}
        {@render subdomain(dn, $thisZone.services[dn] ? $thisZone.services[dn] : [])}
    {/each}
{/if}
