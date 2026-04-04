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
    import { page } from "$app/state";
    import { Icon } from "@sveltestrap/sveltestrap";

    import type { CheckerScope } from "$lib/api/checkers";
    import { t } from "$lib/translations";
    import type { Domain } from "$lib/model/domain";
    import { thisZone } from "$lib/stores/thiszone";
    import DomainCheckerSidebar from "./DomainCheckerSidebar.svelte";
    import ExecutionSidebarContent from "./ExecutionSidebarContent.svelte";

    interface Props {
        domain: Domain;
        checksBase: string;
        backHref: string;
        serviceContext?: {
            zoneId: string;
            subdomain: string;
            serviceid: string;
        };
    }

    let { domain, checksBase, backHref, serviceContext }: Props = $props();

    let scope: CheckerScope = $derived(
        serviceContext
            ? { domainId: domain.id, zoneId: serviceContext.zoneId, subdomain: serviceContext.subdomain, serviceId: serviceContext.serviceid }
            : { domainId: domain.id },
    );

    let serviceType = $derived.by(() => {
        if (!serviceContext) return undefined;
        const svcs =
            $thisZone?.services[serviceContext.subdomain == "@" ? "" : serviceContext.subdomain];
        const svc = svcs?.find((s) => s._id === serviceContext.serviceid);
        return svc?._svctype;
    });
</script>

{#if page.params.execId}
    <a
        href={`${checksBase}/${encodeURIComponent(page.params.checkerId!)}/executions`}
        class="sidebar-back d-flex align-items-center gap-1 mt-3 text-muted text-decoration-none fw-semibold"
    >
        <Icon name="chevron-left" />
        {$t("zones.return-to-results")}
    </a>
    <ExecutionSidebarContent
        {domain}
        checkerId={page.params.checkerId!}
        execId={page.params.execId}
        {checksBase}
        {scope}
    />
{:else if page.params.checkerId}
    <a
        href={checksBase}
        class="sidebar-back d-flex align-items-center gap-1 mt-3 text-muted text-decoration-none fw-semibold"
    >
        <Icon name="chevron-left" />
        {$t("checkers.title")}
    </a>
    <DomainCheckerSidebar
        class="mt-3"
        domainName={domain.domain}
        currentCheckerName={page.params.checkerId}
        {checksBase}
        scope={serviceContext ? "service" : "domain"}
        {serviceType}
    />
    <div class="flex-fill"></div>
{:else}
    <a
        href={backHref}
        class="sidebar-back d-flex align-items-center gap-1 mt-3 text-muted text-decoration-none fw-semibold"
    >
        <Icon name="chevron-left" />
        {$t("zones.return-to")}
    </a>
{/if}
