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
    import { t } from "$lib/translations";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import { listAvailableCheckers } from "$lib/api/checkers";
    import type { Domain } from "$lib/model/domain";
    import { CheckScopeType } from "$lib/model/checker";
    import CheckersList from "$lib/components/checkers/CheckersList.svelte";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();
</script>

<svelte:head>
    <title>Checks - {data.domain.domain} - happyDomain</title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <PageTitle title={$t("checkers.list.title")} domain={data.domain.domain} />

    <CheckersList
        fetchCheckers={() => listAvailableCheckers(data.domain.id)}
        basePath={`/domains/${encodeURIComponent(data.domain.domain)}/checks`}
        domainId={data.domain.id}
        targetType={CheckScopeType.CheckScopeDomain}
        targetId={data.domain.id}
        noChecksKey="checkers.list.no-checks"
    />
</div>
