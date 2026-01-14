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
    import { ListGroup, NavItem, NavLink, Spinner } from "@sveltestrap/sveltestrap";

    import { getProviderSpec } from "$lib/api/provider_specs";
    import ServiceSelectorItem from "./ServiceSelectorItem.svelte";
    import { filterServices } from "./service-filter";
    import type { Domain } from "$lib/model/domain";
    import type { ProviderInfos } from "$lib/model/provider";
    import type { ServiceCombined } from "$lib/model/service.svelte";
    import type { ServiceInfos } from "$lib/model/service_specs.svelte";
    import { providers_idx } from "$lib/stores/providers";
    import { servicesSpecsList, servicesSpecsLoaded } from "$lib/stores/services";
    import { filteredName } from "$lib/stores/serviceSelector";
    import { t } from "$lib/translations";

    interface Props {
        dn: string;
        origin: Domain;
        value?: string | null;
        zservices: Record<string, Array<ServiceCombined>>;
    }

    let {
        dn,
        origin,
        value = $bindable(null),
        zservices
    }: Props = $props();

    let families = [
        {
            label: "Services",
            family: "abstract",
        },
        {
            label: "Providers",
            family: "provider",
        },
        {
            label: "Raw DNS resources",
            family: "",
        },
    ];

    let provider_specs: ProviderInfos | null = $state(null);
    $effect(() => {
        getProviderSpec($providers_idx[origin.id_provider]._srctype).then((prvdspecs) => {
            provider_specs = prvdspecs;
        });
    });

    let filtered_family: string | null = $state(null);

    let filteredServicesResult = $derived(
        provider_specs !== null
            ? filterServices($servicesSpecsList, provider_specs, zservices, dn, $filteredName, filtered_family)
            : { available: [], disabled: [] }
    );

    function handleKeyDown(event: KeyboardEvent) {
        if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') {
            return;
        }

        // Only handle if we have a filtered result
        if (!provider_specs || !$servicesSpecsLoaded) return;

        event.preventDefault();

        // Combine available and disabled services into a single navigable list
        const allServices = [
            ...filteredServicesResult.available,
            ...filteredServicesResult.disabled.map(d => d.svc)
        ];

        if (allServices.length === 0) return;

        // Find current index based on selected value
        const currentIndex = allServices.findIndex(svc => svc._svctype === value);

        let newIndex: number;
        if (event.key === 'ArrowDown') {
            // Move down, wrap to top if at end
            newIndex = currentIndex < allServices.length - 1 ? currentIndex + 1 : 0;
        } else {
            // Move up, wrap to bottom if at top
            newIndex = currentIndex > 0 ? currentIndex - 1 : allServices.length - 1;
        }

        value = allServices[newIndex]._svctype;
    }
</script>

<svelte:document on:keydown={handleKeyDown} />

{#if !provider_specs || !$servicesSpecsLoaded}
    <div class="d-flex justify-content-center">
        <Spinner color="primary" />
    </div>
{:else}
    {#if !$filteredName || filtered_family}
        <ul class="nav nav-tabs sticky-top" style="background: white">
            <NavItem>
                <NavLink active={filtered_family == null} on:click={() => (filtered_family = null)}>
                    {$t("service.all")}
                </NavLink>
            </NavItem>
            {#each families as family}
                <NavItem>
                    <NavLink
                        active={filtered_family == family.family}
                        on:click={() => (filtered_family = family.family)}
                    >
                        {family.label}
                    </NavLink>
                </NavItem>
            {/each}
        </ul>
    {:else}
        <div class="mb-3"></div>
    {/if}
    <ListGroup>
        {#each filteredServicesResult.available as svc}
            <ServiceSelectorItem
                active={value === svc._svctype}
                {svc}
                on:click={() => (value = svc._svctype)}
            />
        {/each}
        {#each filteredServicesResult.disabled as { svc, reason }}
            <ServiceSelectorItem
                active={value === svc._svctype}
                disabled
                {reason}
                {svc}
                on:click={() => (value = svc._svctype)}
            />
        {/each}
    </ListGroup>
{/if}
