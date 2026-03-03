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
    import { fqdn } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import { servicesSpecs, servicesSpecsLoaded } from "$lib/stores/services";
    import { thisZone } from "$lib/stores/thiszone";

    interface Props {
        origin: Domain;
        subdomain: string;
        serviceid: string;
        historyId: string;
    }

    let { origin, subdomain, serviceid, historyId }: Props = $props();

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

    let services = $derived($thisZone?.services[subdomain] ?? []);

    function subdomainPadding(dn: string): string {
        return (dn === "" ? 0 : dn.split(".").length * 10) + "px";
    }

    let servicesPadding = $derived(
        (subdomain === "" ? 0 : subdomain.split(".").length * 10) + 20 + "px",
    );

    function subdomainLink(dn: string): string {
        return `/domains/${origin.domain}/${historyId}#${dn ? dn : "@"}`;
    }

    function serviceLink(svc: { _id?: string }): string {
        return `/domains/${origin.domain}/${historyId}/${subdomain === "" ? "@" : subdomain}/${svc._id}`;
    }
</script>

<div class="mt-4"></div>

<!-- Root domain: bold if editing root subdomain, muted link otherwise -->
<a
    href={subdomainLink("")}
    title={fqdn("", origin.domain)}
    class="d-block text-truncate text-body font-monospace text-decoration-none"
    class:fw-bold={subdomain === ""}
    style="max-width: none; padding-left: 0px"
>
    {fqdn("", origin.domain)}
</a>

<!-- Ancestor subdomains -->
{#each ancestors as ancestor}
    <a
        href={subdomainLink(ancestor)}
        title={fqdn(ancestor, origin.domain)}
        class="d-block text-truncate font-monospace text-muted text-decoration-none"
        style={"max-width: none; padding-left: " + subdomainPadding(ancestor)}
    >
        {ancestor}<span style="opacity: 0.6;">.{origin.domain}</span>
    </a>
{/each}

<!-- Current subdomain in bold (only when not root, root is handled above) -->
{#if subdomain !== ""}
    <a
        href={subdomainLink(subdomain)}
        title={fqdn(subdomain, origin.domain)}
        class="d-block text-truncate font-monospace text-body text-decoration-none fw-bold"
        style={"max-width: none; padding-left: " + subdomainPadding(subdomain)}
    >
        {subdomain}<span class="text-muted" style="opacity: 0.6;">.{origin.domain}</span>
    </a>
{/if}

<!-- Sibling services at current subdomain -->
<ul class="list-unstyled mb-0 overflow-y-auto" style:padding-left={servicesPadding}>
    {#each services as service}
        {@const isActive = service._id === serviceid}
        <li class="mb-1">
            <a
                href={serviceLink(service)}
                class="service-item d-flex align-items-center gap-2 py-2 px-2 rounded text-truncate text-decoration-none {isActive
                    ? 'fw-bold text-primary active'
                    : 'text-muted'}"
                style="max-width: none;"
            >
                {#if $servicesSpecsLoaded && $servicesSpecs[service._svctype]}
                    {$servicesSpecs[service._svctype].name}
                {:else}
                    {service._svctype}
                {/if}
                {#if service._comment}
                    <span class="fst-italic text-muted" style="opacity: 0.6;">
                        {service._comment}
                    </span>
                {/if}
            </a>
        </li>
    {/each}
</ul>

<style>
    .service-item {
        transition: background-color 0.15s;
    }

    .service-item:hover {
        background-color: rgba(0, 0, 0, 0.06);
    }

    .service-item.active {
        background-color: rgba(var(--bs-primary-rgb), 0.1);
    }
</style>
