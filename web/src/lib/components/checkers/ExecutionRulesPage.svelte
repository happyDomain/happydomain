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
    import { Alert, Icon } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import type { CheckerScope } from "$lib/api/checkers";
    import { getCheckStatus, getScopedExecutionResults } from "$lib/api/checkers";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import ExecutionResultsCard from "./ExecutionResultsCard.svelte";

    interface Props {
        scope: CheckerScope;
        checkerId: string;
        execId: string;
        domainName: string;
    }

    let { scope, checkerId, execId, domainName }: Props = $props();

    let resultsPromise = $derived(getScopedExecutionResults(scope, checkerId, execId));
    let checkerName = $state<string>("");

    $effect(() => {
        getCheckStatus(checkerId).then((s) => {
            checkerName = s.name ?? checkerId;
        });
    });
</script>

<svelte:head>
    <title>{$t("checkers.detail.check-rules")} - {checkerName || checkerId} - happyDomain</title>
</svelte:head>

<div class="flex-fill mt-1 mb-5">
    <PageTitle title={$t("checkers.detail.check-rules")} subtitle={checkerName} domain={domainName} />

    {#await resultsPromise}
        <p class="text-center">
            <span class="spinner-border spinner-border-sm me-2"></span>
            {$t("checkers.result.loading")}
        </p>
    {:then evaluation}
        {#if evaluation}
            <ExecutionResultsCard {evaluation} />
        {:else}
            <Alert color="info">
                <Icon name="info-circle" />
                {$t("checkers.result.no-results")}
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <Icon name="exclamation-triangle-fill" />
            {$t("checkers.result.error-loading", { error: error.message })}
        </Alert>
    {/await}
</div>
