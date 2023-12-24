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
     ListGroup,
     NavItem,
     NavLink,
     Spinner,
 } from 'sveltestrap';

 import { getProviderSpec } from '$lib/api/provider_specs';
 import ServiceSelectorItem from '$lib/components/ServiceSelectorItem.svelte';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ProviderInfos } from '$lib/model/provider';
 import type { ServiceCombined } from '$lib/model/service';
 import { passRestrictions, type ServiceInfos } from '$lib/model/service_specs';
 import { providers_idx } from '$lib/stores/providers';
 import { servicesSpecs } from '$lib/stores/services';
 import { t } from '$lib/translations';

 export let dn: string;
 export let origin: Domain | DomainInList;
 export let value: string | null = null;
 export let zservices: Record<string, Array<ServiceCombined>>;

 let families = [
     {
         label: 'Services',
         family: 'abstract'
     },
     {
         label: 'Providers',
         family: 'provider'
     },
     {
         label: 'Raw DNS resources',
         family: ''
     }
 ];

 let provider_specs: ProviderInfos | null = null;
 $: getProviderSpec($providers_idx[origin.id_provider]._srctype).then(
     (prvdspecs) => {
         provider_specs = prvdspecs;
     }
 );

 let filtered_family: string | null = null;

 let availableNewServices: Array<ServiceInfos> = [];
 let disabledNewServices: Array<{svc: ServiceInfos; reason: string;}> = [];

 $: {
     if (provider_specs && $servicesSpecs) {
         const ans: Array<ServiceInfos> = [];
         const dns: Array<{svc: ServiceInfos; reason: string;}> = [];

         for (const idx in $servicesSpecs) {
             const svc = $servicesSpecs[idx];

             if (svc.family === "hidden") {
                 continue;
             }

             const reason = passRestrictions(svc, provider_specs, zservices, dn);
             if (reason == null) {
                 ans.push(svc);
             } else {
                 dns.push({svc, reason});
             }
         }

         availableNewServices = ans;
         disabledNewServices = dns;
     }
 }
</script>

{#if !provider_specs || !$servicesSpecs}
    <div class="d-flex justify-content-center">
        <Spinner color="primary" />
    </div>
{:else}
    <ul class="nav nav-tabs sticky-top pt-3 mb-2" style="background: white">
        <NavItem>
            <NavLink
                active={filtered_family == null}
                on:click={() => filtered_family = null}
            >
                {$t('service.all')}
            </NavLink>
        </NavItem>
        {#each families as family}
            <NavItem>
                <NavLink
                    active={filtered_family == family.family}
                    on:click={() => filtered_family = family.family}
                >
                    {family.label}
                </NavLink>
            </NavItem>
        {/each}
    </ul>
    <ListGroup>
        {#each availableNewServices as svc}
            {#if (filtered_family == null || svc.family == filtered_family)}
                <ServiceSelectorItem
                    active={value === svc._svctype}
                    {svc}
                    on:click={() => value = svc._svctype}
                />
            {/if}
        {/each}
        {#each disabledNewServices as {svc, reason}}
            {#if (filtered_family == null || svc.family == filtered_family)}
                <ServiceSelectorItem
                    active={value === svc._svctype}
                    disabled
                    {reason}
                    {svc}
                    on:click={() => value = svc._svctype}
                />
            {/if}
        {/each}
    </ListGroup>
{/if}
