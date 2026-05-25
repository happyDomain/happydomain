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
    import type {
        CheckerCheckerDefinition,
        HappydnsCheckEvaluation,
        HappydnsExecution,
    } from "$lib/api-base/types.gen";
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
        disableMetrics,
    } from "$lib/stores/checkers";
    import CheckCard from "./CheckCard.svelte";
    import CheckerLoader from "./CheckerLoader.svelte";
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
    let running = $state(false);
    let error = $state<string | undefined>(undefined);
    let metricsData = $state<CheckMetric[] | null>(null);
    let evaluationData = $state<HappydnsCheckEvaluation | null>(null);

    let pollTimer: ReturnType<typeof setInterval> | undefined;

    function isInProgress(status: number | undefined): boolean {
        return status === 0 || status === 1;
    }

    function loadTerminalData(
        execution: HappydnsExecution,
        checkerInfo: CheckerCheckerDefinition,
        scopeArg: CheckerScope,
        checkerIdArg: string,
        execIdArg: string,
    ) {
        getScopedExecutionObservations(scopeArg, checkerIdArg, execIdArg)
            .then((observations) => {
                currentObservations.set(observations);
                if (
                    (!observations || Object.keys(observations.data).length == 0) &&
                    ($reportViewMode == "html" || $reportViewMode == "metrics")
                ) {
                    reportViewMode.set("rules");
                }
            })
            .catch((e) => console.warn("Failed to load execution observations", e));
        getScopedExecutionResults(scopeArg, checkerIdArg, execIdArg)
            .then((e) => (evaluationData = e))
            .catch((e) => console.warn("Failed to load execution results", e));
        if (execution.status === 3) {
            reportViewMode.set("rules");
        } else if (checkerInfo.has_html_report) {
            reportViewMode.set("html");
        } else if (checkerInfo.has_metrics) {
            reportViewMode.set("metrics");
        } else {
            reportViewMode.set("rules");
        }
        if (checkerInfo.has_metrics) {
            getScopedExecutionMetrics(scopeArg, checkerIdArg, execIdArg)
                .then((m) => {
                    disableMetrics.set(false);
                    metricsData = m;
                })
                .catch((e) => {
                    console.warn("Failed to load execution metrics", e);
                    if ($reportViewMode == "metrics") reportViewMode.set("rules");
                    disableMetrics.set(true);
                });
        }
    }

    function startPolling(
        checkerInfo: CheckerCheckerDefinition,
        scopeArg: CheckerScope,
        checkerIdArg: string,
        execIdArg: string,
    ) {
        if (pollTimer) clearInterval(pollTimer);
        pollTimer = setInterval(async () => {
            try {
                const execution = await getScopedExecution(scopeArg, checkerIdArg, execIdArg);
                currentExecution.set(execution);
                if (!isInProgress(execution.status)) {
                    clearInterval(pollTimer);
                    pollTimer = undefined;
                    running = false;
                    loadTerminalData(execution, checkerInfo, scopeArg, checkerIdArg, execIdArg);
                    loading = false;
                }
            } catch (e) {
                console.warn("Failed to poll execution status", e);
            }
        }, 3000);
    }

    $effect(() => {
        loading = true;
        running = false;
        error = undefined;
        metricsData = null;
        evaluationData = null;
        cachedHTMLReport.set(null);
        if (pollTimer) {
            clearInterval(pollTimer);
            pollTimer = undefined;
        }

        const scopeArg = scope;
        const checkerIdArg = checkerId;
        const execIdArg = execId;

        Promise.all([
            getScopedExecution(scopeArg, checkerIdArg, execIdArg),
            getCheckStatus(checkerIdArg),
        ]).then(
            ([execution, checkerInfo]) => {
                currentExecution.set(execution);
                currentCheckInfo.set(checkerInfo);
                checkerName = checkerInfo.name ?? checkerIdArg;
                if (isInProgress(execution.status)) {
                    running = true;
                    startPolling(checkerInfo, scopeArg, checkerIdArg, execIdArg);
                    return;
                }
                loadTerminalData(execution, checkerInfo, scopeArg, checkerIdArg, execIdArg);
                loading = false;
            },
            (err) => {
                error = err.message;
                loading = false;
            },
        );
    });

    onDestroy(() => {
        if (pollTimer) {
            clearInterval(pollTimer);
            pollTimer = undefined;
        }
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
    <Container class="flex-fill d-flex flex-column align-items-center justify-content-center mt-5">
        <CheckerLoader
            icon={running ? "broadcast" : "search"}
            label={running
                ? $t("checkers.execution.status.running")
                : $t("checkers.result.loading")}
        />
    </Container>
{:else if error}
    <Container class="flex-fill d-flex align-items-start mt-5">
        <Alert class="flex-fill" color="danger">
            <Icon name="exclamation-triangle-fill" />
            {$t("checkers.result.error-loading", { error })}
        </Alert>
    </Container>
{:else}
    <Container class="mt-3 mb-3">
        <CheckCard
            status={$currentExecution?.result?.status}
            name={checkerName || checkerId}
            message={$currentExecution?.result?.message}
            startedAt={$currentExecution?.startedAt}
        />
    </Container>
    {#if $reportViewMode === "rules" && evaluationData}
        <Container class="flex-fill d-flex flex-column">
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
{/if}
