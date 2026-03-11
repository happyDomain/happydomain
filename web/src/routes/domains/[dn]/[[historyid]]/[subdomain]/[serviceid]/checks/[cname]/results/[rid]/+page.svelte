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
        getCheckStatus,
        getServiceCheckResult,
        getServiceCheckResultHTMLReport,
        getServiceCheckResultMetrics,
    } from "$lib/api/checkers";
    import type { Domain } from "$lib/model/domain";
    import CheckResultView from "$lib/components/checkers/CheckResultView.svelte";

    interface Props {
        data: { domain: Domain; zoneId: string; subdomain: string; serviceid: string };
    }

    let { data }: Props = $props();

    const checkerName = $derived(page.params.cname || "");
    const resultId = $derived(page.params.rid || "");

    const resultPromise = $derived(getServiceCheckResult(data.domain.id, data.zoneId, data.subdomain, data.serviceid, checkerName, resultId));
    const checkPromise = $derived(getCheckStatus(checkerName));
    const htmlReportPromise = $derived(getServiceCheckResultHTMLReport(data.domain.id, data.zoneId, data.subdomain, data.serviceid, checkerName, resultId));
    const getMetrics = $derived(() => getServiceCheckResultMetrics(data.domain.id, data.zoneId, data.subdomain, data.serviceid, checkerName, resultId));
</script>

<svelte:head>
    <title>
        Check Result - {checkerName} - {data.domain.domain} - happyDomain
    </title>
</svelte:head>

<CheckResultView
    {resultPromise}
    {checkPromise}
    {htmlReportPromise}
    {getMetrics}
/>
