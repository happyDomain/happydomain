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

    import { t } from "$lib/translations";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import { listServiceAvailableCheckers } from "$lib/api/checkers";
    import type { Domain } from "$lib/model/domain";
    import { CheckScopeType } from "$lib/model/checker";
    import { servicesSpecs, servicesSpecsLoaded } from "$lib/stores/services";
    import { thisZone } from "$lib/stores/thiszone";
    import CheckersList from "$lib/components/checkers/CheckersList.svelte";

    interface Props {
        data: { domain: Domain; zoneId: string; subdomain: string; serviceid: string };
    }

    let { data }: Props = $props();

    let serviceName = $derived.by(() => {
        const svcs = $thisZone?.services[data.subdomain];
        const svc = svcs?.find((s) => s._id === data.serviceid);
        if (!svc) return data.serviceid;
        return ($servicesSpecsLoaded && $servicesSpecs[svc._svctype]?.name) || svc._svctype;
    });

    let basePath = $derived.by(() => {
        const dn = encodeURIComponent(data.domain.domain);
        const historyid = page.params.historyid ? encodeURIComponent(page.params.historyid) : "";
        const sub = encodeURIComponent(page.params.subdomain!);
        const svc = encodeURIComponent(data.serviceid);
        return `/domains/${dn}/${historyid}/${sub}/${svc}/checks`;
    });
</script>

<svelte:head>
    <title>
        {$t("checkers.list.title-service", { service: serviceName } as any)} - {data.domain.domain} -
        happyDomain
    </title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <PageTitle
        title={$t("checkers.list.title-service", { service: serviceName } as any)}
        domain={data.domain.domain}
    />

    <CheckersList
        fetchCheckers={() =>
            listServiceAvailableCheckers(data.domain.id, data.zoneId, data.subdomain, data.serviceid)}
        {basePath}
        domainId={data.domain.id}
        zoneId={data.zoneId}
        subdomain={data.subdomain}
        serviceid={data.serviceid}
        targetType={CheckScopeType.CheckScopeService}
        targetId={data.serviceid}
        noChecksKey="checkers.list.no-checks-service"
    />
</div>
