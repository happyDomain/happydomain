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
    import { tick } from "svelte";

    import { Spinner } from "@sveltestrap/sveltestrap";

    import { getProviderSpec } from "$lib/api/provider_specs";
    import ServiceSelectorItem from "./ServiceSelectorItem.svelte";
    import { filterServices } from "./service-filter";
    import type { Domain } from "$lib/model/domain";
    import type { ProviderInfos } from "$lib/model/provider";
    import type { HappydnsService } from "$lib/api-base/types.gen";
    import { providers_idx } from "$lib/stores/providers";
    import { servicesSpecsList, servicesSpecsLoaded } from "$lib/stores/services";
    import { filteredName } from "$lib/stores/serviceSelector";
    import { t } from "$lib/translations";

    interface Props {
        dn: string;
        origin: Domain;
        value?: string | null;
        zservices: Record<string, Array<HappydnsService>>;
    }

    let { dn, origin, value = $bindable(null), zservices }: Props = $props();

    const families: Array<{ label: string; family: string | null }> = [
        { label: "service.all", family: null },
        { label: "service.family.abstract", family: "abstract" },
        { label: "service.family.provider", family: "provider" },
        { label: "service.family.raw", family: "" },
    ];

    // undefined while loading; null for an origin without provider, as a
    // zone template, whose services are not limited to what one supports.
    let provider_specs: ProviderInfos | null | undefined = $state(undefined);
    $effect(() => {
        if (!origin.id_provider) {
            provider_specs = null;
            return;
        }
        const provider = $providers_idx[origin.id_provider];
        if (!provider) return;
        getProviderSpec(provider._srctype).then((prvdspecs) => {
            provider_specs = prvdspecs;
        });
    });

    let filtered_family: string | null = $state(null);

    function filterFamily(family: string | null) {
        return provider_specs !== undefined
            ? filterServices(
                  $servicesSpecsList,
                  provider_specs,
                  zservices,
                  dn,
                  $filteredName,
                  family,
                  $t,
              )
            : { available: [], disabled: [] };
    }

    let filteredServicesResult = $derived(filterFamily(filtered_family));

    // How many services each family holds for the current search.
    let familyCounts = $derived(
        families.map(({ family }) => {
            const { available, disabled } = filterFamily(family);
            return available.length + disabled.length;
        }),
    );

    function handleKeyDown(event: KeyboardEvent) {
        if (event.key !== "ArrowUp" && event.key !== "ArrowDown") {
            return;
        }

        // Only handle if we have a filtered result
        if (provider_specs === undefined || !$servicesSpecsLoaded) return;

        event.preventDefault();

        // Unavailable services cannot be chosen, so they are skipped.
        const allServices = filteredServicesResult.available;

        if (allServices.length === 0) return;

        // Find current index based on selected value
        const currentIndex = allServices.findIndex((svc) => svc._svctype === value);

        let newIndex: number;
        if (event.key === "ArrowDown") {
            // Move down, wrap to top if at end
            newIndex = currentIndex < allServices.length - 1 ? currentIndex + 1 : 0;
        } else {
            // Move up, wrap to bottom if at top
            newIndex = currentIndex > 0 ? currentIndex - 1 : allServices.length - 1;
        }

        value = allServices[newIndex]._svctype;
        tick().then(() =>
            document.querySelector(".service-list .active")?.scrollIntoView({ block: "nearest" }),
        );
    }
</script>

<svelte:document on:keydown={handleKeyDown} />

{#if provider_specs === undefined || !$servicesSpecsLoaded}
    <div class="d-flex justify-content-center py-4">
        <Spinner color="primary" />
    </div>
{:else}
    <div class="d-flex flex-wrap gap-2 sticky-top py-3 border-bottom bg-body">
        {#each families as family, i (family.label)}
            <button
                type="button"
                class="btn btn-sm btn-outline-secondary rounded-pill px-3"
                class:active={filtered_family === family.family}
                aria-pressed={filtered_family === family.family}
                onclick={() => (filtered_family = family.family)}
            >
                {$t(family.label)}
                <span class="opacity-75 ms-1">{familyCounts[i]}</span>
            </button>
        {/each}
    </div>
    <div class="service-list list-group gap-1 pt-2" role="listbox">
        {#each filteredServicesResult.available as svc (svc._svctype)}
            <ServiceSelectorItem
                active={value === svc._svctype}
                {svc}
                on:click={() => (value = svc._svctype)}
            />
        {/each}
        {#if filteredServicesResult.disabled.length}
            <div class="small fw-semibold text-uppercase text-body-secondary px-2 pt-3 pb-1">
                {$t("service.unavailable-here", { count: filteredServicesResult.disabled.length })}
            </div>
            {#each filteredServicesResult.disabled as { svc, reason } (svc._svctype)}
                <ServiceSelectorItem disabled {reason} {svc} />
            {/each}
        {/if}
        {#if !filteredServicesResult.available.length && !filteredServicesResult.disabled.length}
            <p class="text-center text-muted py-4 mb-0">
                {$t("service.no-match", { query: $filteredName })}
            </p>
        {/if}
    </div>
{/if}
