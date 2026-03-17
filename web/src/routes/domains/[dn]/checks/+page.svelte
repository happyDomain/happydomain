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
    import { Spinner } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import CheckersAvailabilityTable from "$lib/components/checkers/CheckersAvailabilityTable.svelte";
    import { listAvailableCheckers } from "$lib/api/checkers";
    import type { Domain } from "$lib/model/domain";
    import type { AvailableChecker } from "$lib/model/checker";
    import { CheckScopeType } from "$lib/model/checker";
    import CheckersList from "$lib/components/checkers/CheckersList.svelte";
    import { checkers } from "$lib/stores/checkers";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    let domainCheckers = $state<AvailableChecker[] | null>(null);

    async function fetchAndStoreDomainCheckers(): Promise<AvailableChecker[]> {
        const result = await listAvailableCheckers(data.domain.id);
        domainCheckers = result;
        return result;
    }

    let domainCheckerNames = $derived(
        domainCheckers
            ? new Set(domainCheckers.map((c: AvailableChecker) => c.checker_name))
            : null,
    );

    let otherCheckers = $derived.by(() => {
        if (!$checkers || !domainCheckerNames) return null;
        return Object.entries($checkers).filter(
            ([name]) => !domainCheckerNames!.has(name) && $checkers[name].options?.domainOpts,
        );
    });

    const basePath = $derived(`/domains/${encodeURIComponent(data.domain.domain)}/checks`);
</script>

<svelte:head>
    <title>Checks - {data.domain.domain} - happyDomain</title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <PageTitle title={$t("checkers.list.title")} domain={data.domain.domain} />

    <CheckersList
        fetchCheckers={fetchAndStoreDomainCheckers}
        {basePath}
        domainId={data.domain.id}
        targetType={CheckScopeType.CheckScopeDomain}
        targetId={data.domain.id}
        noChecksKey="checkers.list.no-checks"
    />

    {#if otherCheckers === null}
        <div class="mt-5 text-center">
            <Spinner size="sm" />
            <span class="ms-2">{$t("checkers.list.loading-checks")}</span>
        </div>
    {:else if otherCheckers.length > 0}
        <h4 class="mt-5 mb-3">{$t("checkers.other-checkers.title")}</h4>
        <p class="text-muted">{$t("checkers.other-checkers.description")}</p>

        <CheckersAvailabilityTable
            checkers={otherCheckers}
            {basePath}
            configureKey="checkers.other-checkers.configure"
        />
    {/if}
</div>
