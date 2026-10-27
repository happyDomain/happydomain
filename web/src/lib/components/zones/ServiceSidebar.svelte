<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
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
    import { Icon } from "@sveltestrap/sveltestrap";

    import { page } from "$app/state";
    import type { ResolvedPathname } from "$app/types";
    import { controls as ctrlNewService } from "$lib/components/services/NewServicePath.svelte";
    import type { Domain } from "$lib/model/domain";
    import { serviceName } from "$lib/services/infos";
    import { isReadOnlyProvider, providers_idx } from "$lib/stores/providers";
    import { servicesSpecs } from "$lib/stores/services";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";
    import DomainTreeLink from "./DomainTreeLink.svelte";
    import { useZoneEditor } from "./editor";

    interface Props {
        origin: Domain;
        subdomain: string;
        serviceid: string;
        historyId: string;
    }

    let { origin, subdomain, serviceid, historyId }: Props = $props();

    const editor = useZoneEditor(() => origin);

    let readOnly = $derived(
        editor.readOnly || isReadOnlyProvider($providers_idx, origin.id_provider),
    );

    // Ancestors from shallowest to deepest:
    // "" → [], "www" → [], "a.b" → ["b"], "a.b.c" → ["c", "b.c"]
    let ancestors = $derived(
        (() => {
            if (!subdomain) return [];
            const parts = subdomain.split(".");
            const result: string[] = [];
            for (let k = 1; k < parts.length; k++) {
                result.push(parts.slice(parts.length - k).join("."));
            }
            return result;
        })(),
    );

    // Each level of the tree, from the apex down to the current subdomain.
    let levels = $derived(["", ...ancestors, ...(subdomain ? [subdomain] : [])]);

    let services = $derived($thisZone?.services[subdomain] ?? []);

    // The service being created is not in the zone yet, but it is the page
    // the user is on.
    let isNew = $derived(serviceid === "new");
    let newSvcType = $derived(page.url.searchParams.get("type") ?? "");

    function subdomainLink(dn: string): ResolvedPathname {
        // The fragment is what scrolls the zone to that subdomain.
        return `${editor.zoneHref(historyId)}#${dn ? dn : "@"}` as ResolvedPathname;
    }

    function serviceLink(svc: { _id?: string }): ResolvedPathname {
        return editor.serviceHref(historyId, subdomain, svc._id!);
    }
</script>

<div class="mt-4"></div>

<!-- From the apex down to the current subdomain, in bold -->
{#each levels as dn (dn)}
    <DomainTreeLink
        href={subdomainLink(dn)}
        {dn}
        {origin}
        current={dn === subdomain}
        muted={dn !== subdomain && dn !== ""}
    />
{/each}

<!-- Sibling services at current subdomain -->
<div class="tree-services mt-1" style:--depth={levels.length - 1}>
    <div class="d-flex align-items-center gap-2 px-2 pt-1 mb-1">
        <span class="small text-uppercase fw-semibold text-body-secondary">
            {$t("zones.sidebar-services")}
        </span>
        {#if services.length}
            <span class="badge rounded-pill text-bg-light border">{services.length}</span>
        {/if}
    </div>

    {#if services.length || isNew}
        <ul class="list-group list-group-sidebar gap-1 mb-0 overflow-y-auto">
            {#each services as service (service._id)}
                {@const isActive = service._id === serviceid}
                <li class="d-grid">
                    <a
                        href={serviceLink(service)}
                        class="list-group-item list-group-item-action p-2 border-0 rounded"
                        class:active={isActive}
                        aria-current={isActive ? "page" : undefined}
                    >
                        <span class="d-block text-truncate">
                            {$serviceName($servicesSpecs[service._svctype], service._svctype)}
                        </span>
                        {#if service._comment}
                            <span
                                class="d-block text-truncate small fst-italic text-body-secondary"
                                title={service._comment}
                            >
                                {service._comment}
                            </span>
                        {/if}
                    </a>
                </li>
            {/each}
            {#if isNew}
                <li class="d-grid">
                    <span
                        class="service-new list-group-item active p-2 border border-primary rounded"
                        aria-current="page"
                    >
                        <span class="d-block text-truncate">
                            {newSvcType
                                ? $serviceName($servicesSpecs[newSvcType], newSvcType)
                                : $t("service.new")}
                        </span>
                        {#if newSvcType}
                            <span
                                class="d-block text-truncate small fst-italic text-body-secondary"
                            >
                                {$t("service.new")}
                            </span>
                        {/if}
                    </span>
                </li>
            {/if}
        </ul>
    {:else}
        <p class="small fst-italic text-body-secondary px-2 mb-1">
            {$t("zones.sidebar-no-service")}
        </p>
    {/if}

    {#if !readOnly && !isNew}
        <button
            type="button"
            class="service-add btn btn-link btn-sm d-flex align-items-center gap-1 px-2 text-decoration-none"
            onclick={() => ctrlNewService.Open(subdomain)}
        >
            <Icon name="plus" />
            {$t("domains.add-a-service")}
        </button>
    {/if}
</div>

<style>
    /* Hangs from the current subdomain, aligned on DomainTreeLink rails */
    .tree-services {
        --indent: 1.25rem;
        --stem: 0.35rem;
        margin-left: calc(var(--depth) * var(--indent) + var(--stem));
        padding-left: 0.4rem;
        border-left: 1px solid var(--bs-border-color);
    }

    /* Not saved yet: outlined, as a placeholder for the service to come */
    .service-new {
        border-style: dashed !important;
    }
</style>
