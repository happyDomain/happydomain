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
    import { nsrrtype } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { ProviderInfos } from "$lib/model/provider";
    import type { ServiceCombined } from "$lib/model/service.svelte";
    import { passRestrictions, type ServiceInfos } from "$lib/model/service_specs.svelte";
    import { providers_idx } from "$lib/stores/providers";
    import { servicesSpecs } from "$lib/stores/services";
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

    let availableNewServices: Array<ServiceInfos> = $state([]);
    let disabledNewServices: Array<{ svc: ServiceInfos; reason: string }> = $state([]);

    $effect(() => {
        if (provider_specs && $servicesSpecs) {
            const ans: Array<ServiceInfos> = [];
            const dns: Array<{ svc: ServiceInfos; reason: string }> = [];

            for (const idx in $servicesSpecs) {
                const svc = $servicesSpecs[idx];

                if (svc.family === "hidden") {
                    continue;
                }

                const reason = passRestrictions(svc, provider_specs, zservices, dn);
                if (reason == null) {
                    ans.push(svc);
                } else {
                    dns.push({ svc, reason });
                }
            }

            availableNewServices = ans;
            disabledNewServices = dns;
        }
    });

    function svc_match(svc: ServiceInfos, arg1: string | null, arg2: string) {
        return (
            filtered_family == null || svc.family == filtered_family
        ) && (
            !$filteredName ||
            svc.name.toLowerCase().indexOf($filteredName.toLowerCase()) >= 0 ||
            svc.description.toLowerCase().indexOf($filteredName.toLowerCase()) >= 0 ||
            (svc.record_types && svc.record_types.filter((rtype) => nsrrtype(rtype).toLowerCase().indexOf($filteredName.toLowerCase()) >= 0).length) ||
            (svc.categories && svc.categories.filter((category) => category.toLowerCase().indexOf($filteredName.toLowerCase()) >= 0).length)
        )
    }
</script>

{#if !provider_specs || !$servicesSpecs}
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
        {#each availableNewServices as svc}
            {#if svc_match(svc, filtered_family, $filteredName)}
                <ServiceSelectorItem
                    active={value === svc._svctype}
                    {svc}
                    on:click={() => (value = svc._svctype)}
                />
            {/if}
        {/each}
        {#each disabledNewServices as { svc, reason }}
            {#if svc_match(svc, filtered_family, $filteredName)}
                <ServiceSelectorItem
                    active={value === svc._svctype}
                    disabled
                    {reason}
                    {svc}
                    on:click={() => (value = svc._svctype)}
                />
            {/if}
        {/each}
    </ListGroup>
{/if}
