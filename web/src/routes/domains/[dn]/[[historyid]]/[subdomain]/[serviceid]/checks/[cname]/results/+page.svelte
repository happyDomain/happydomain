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
    import {
        listServiceCheckResults,
        deleteServiceCheckResult,
        deleteAllServiceCheckResults,
        getServiceCheckExecution,
        getServiceCheckMetrics,
    } from "$lib/api/checkers";
    import type { Domain } from "$lib/model/domain";
    import { CheckScopeType } from "$lib/model/checker";
    import CheckResultsPage from "$lib/components/checkers/CheckResultsPage.svelte";

    interface Props {
        data: { domain: Domain; zoneId: string; subdomain: string; serviceid: string };
    }

    let { data }: Props = $props();

    const checkerName = $derived(page.params.cname || "");

    const basePath = $derived(() => {
        const dn = encodeURIComponent(data.domain.domain);
        const historyid = page.params.historyid ? encodeURIComponent(page.params.historyid) : "";
        const sub = encodeURIComponent(page.params.subdomain!);
        const svc = encodeURIComponent(data.serviceid);
        return `/domains/${dn}/${historyid}/${sub}/${svc}/checks`;
    });
    const cn = $derived(encodeURIComponent(checkerName));
</script>

<CheckResultsPage
    domain={data.domain}
    checkerName={checkerName}
    targetType={CheckScopeType.CheckScopeService}
    targetId={data.serviceid}
    configurePath={`${basePath()}/${cn}`}
    resultViewPath={(id) => `${basePath()}/${cn}/results/${encodeURIComponent(id)}`}
    loadResults={() =>
        listServiceCheckResults(data.domain.id, data.zoneId, data.subdomain, data.serviceid, checkerName)}
    getExecution={(id) =>
        getServiceCheckExecution(data.domain.id, data.zoneId, data.subdomain, data.serviceid, checkerName, id)}
    deleteResult={(id) =>
        deleteServiceCheckResult(data.domain.id, data.zoneId, data.subdomain, data.serviceid, checkerName, id)}
    deleteAllResults={() =>
        deleteAllServiceCheckResults(data.domain.id, data.zoneId, data.subdomain, data.serviceid, checkerName)}
    loadMetrics={() =>
        getServiceCheckMetrics(data.domain.id, data.zoneId, data.subdomain, data.serviceid, checkerName, 50)}
    zoneId={data.zoneId}
    subdomain={data.subdomain}
    serviceid={data.serviceid}
/>
