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
    import {
        Badge,
    } from "@sveltestrap/sveltestrap";

    import { nsrrtype } from '$lib/dns';
    import type { ServiceCombined } from '$lib/model/service.svelte';
    import { servicesSpecs } from '$lib/stores/services';
    import { userSession } from '$lib/stores/usersession';

    interface Props {
        service: ServiceCombined | null;
    }

    let { service }: Props = $props();

</script>

{#if service && $userSession.settings && $servicesSpecs}
    {#if $servicesSpecs[service._svctype].categories && $servicesSpecs[service._svctype].categories.length && !$userSession.settings.showrrtypes}
        <div class="d-flex align-items-center gap-1">
            {#each $servicesSpecs[service._svctype].categories as category}
                <Badge color="secondary">
                    {category}
                </Badge>
            {/each}
        </div>
    {:else if $servicesSpecs[service._svctype].record_types && $servicesSpecs[service._svctype].record_types.length && $userSession.settings.showrrtypes}
        <div class="d-flex align-items-center gap-1">
            {#each $servicesSpecs[service._svctype].record_types as rrtype}
                <Badge color="info">
                    {nsrrtype(rrtype)}
                </Badge>
            {/each}
        </div>
    {/if}
{/if}
