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
    import { onDestroy } from "svelte";
    import { Alert, Card, Container, Icon } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import type { CheckerScope, CheckMetric } from "$lib/api/checkers";
    import type { HappydnsCheckEvaluation } from "$lib/api-base/types.gen";
    import {
        getScopedExecution,
        getScopedExecutionObservations,
        getScopedExecutionMetrics,
        getScopedExecutionResults,
        getCheckStatus,
    } from "$lib/api/checkers";
    import {
        currentExecution,
        currentCheckInfo,
        currentObservations,
        reportViewMode,
        cachedHTMLReport,
    } from "$lib/stores/checkers";
    import ExecutionResultsCard from "./ExecutionResultsCard.svelte";
    import ObservationReportCard from "./ObservationReportCard.svelte";

    interface Props {
        scope: CheckerScope;
        checkerId: string;
        execId: string;
    }

    let { scope, checkerId, execId }: Props = $props();

    let checkerName = $state<string>("");
    let loading = $state(true);
    let error = $state<string | undefined>(undefined);
    let metricsData = $state<CheckMetric[] | null>(null);
    let evaluationData = $state<HappydnsCheckEvaluation | null>(null);

    $effect(() => {
        loading = true;
        error = undefined;
        metricsData = null;
        evaluationData = null;
        cachedHTMLReport.set(null);

        Promise.all([
            getScopedExecution(scope, checkerId, execId),
            getCheckStatus(checkerId),
            getScopedExecutionObservations(scope, checkerId, execId),
        ]).then(
            ([execution, checkerInfo, observations]) => {
                currentExecution.set(execution);
                currentCheckInfo.set(checkerInfo);
                currentObservations.set(observations);
                checkerName = checkerInfo.name ?? checkerId;
                // Load rules data
                getScopedExecutionResults(scope, checkerId, execId)
                    .then((e) => (evaluationData = e))
                    .catch((e) => console.warn("Failed to load execution results", e));
                // Default to metrics view if supported, then HTML, then rules, then JSON
                if (checkerInfo.has_metrics) {
                    reportViewMode.set("metrics");
                    getScopedExecutionMetrics(scope, checkerId, execId)
                        .then((m) => (metricsData = m))
                        .catch((e) => console.warn("Failed to load execution metrics", e));
                } else if (checkerInfo.has_html_report) {
                    reportViewMode.set("html");
                } else {
                    reportViewMode.set("rules");
                }
                loading = false;
            },
            (err) => {
                error = err.message;
                loading = false;
            },
        );
    });

    onDestroy(() => {
        currentExecution.set(undefined);
        currentCheckInfo.set(undefined);
        currentObservations.set(undefined);
        reportViewMode.set("json");
        cachedHTMLReport.set(null);
    });
</script>

<svelte:head>
    <title>{$t("checkers.execution.title")} - {checkerName || checkerId} - happyDomain</title>
</svelte:head>

{#if loading}
    <Container class="flex-fill d-flex align-items-start mt-5">
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.result.loading")}
            </p>
        </Card>
    </Container>
{:else if error}
    <Container class="flex-fill d-flex align-items-start mt-5">
        <Alert class="flex-fill" color="danger">
            <Icon name="exclamation-triangle-fill" />
            {$t("checkers.result.error-loading", { error })}
        </Alert>
    </Container>
{:else if $reportViewMode === "rules" && evaluationData}
    <Container class="flex-fill d-flex flex-column mt-3">
        <ExecutionResultsCard evaluation={evaluationData} />
    </Container>
{:else if $currentObservations}
    <ObservationReportCard
        observations={$currentObservations}
        metrics={metricsData}
        {scope}
        {checkerId}
        {execId}
    />
{/if}
