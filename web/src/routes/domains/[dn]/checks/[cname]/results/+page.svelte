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
        listCheckResults,
        deleteCheckResult,
        deleteAllCheckResults,
        getCheckExecution,
    } from "$lib/api/checkers";
    import type { Domain } from "$lib/model/domain";
    import { CheckScopeType } from "$lib/model/checker";
    import CheckResultsPage from "$lib/components/checkers/CheckResultsPage.svelte";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    const checkName = $derived(page.params.cname || "");
    const dn = $derived(encodeURIComponent(data.domain.domain));
    const cn = $derived(encodeURIComponent(checkName));
</script>

<CheckResultsPage
    domain={data.domain}
    checkerName={checkName}
    targetType={CheckScopeType.CheckScopeDomain}
    targetId={data.domain.id}
    configurePath={`/domains/${dn}/checks/${cn}`}
    resultViewPath={(id) => `/domains/${dn}/checks/${cn}/results/${encodeURIComponent(id)}`}
    loadResults={() => listCheckResults(data.domain.id, checkName)}
    getExecution={(id) => getCheckExecution(data.domain.id, checkName, id)}
    deleteResult={(id) => deleteCheckResult(data.domain.id, checkName, id)}
    deleteAllResults={() => deleteAllCheckResults(data.domain.id, checkName)}
/>
