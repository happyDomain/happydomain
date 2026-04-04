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
    import { Alert, Badge, Button, Card, Icon, Table } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";
    import type { HappydnsExecution } from "$lib/api-base/types.gen";
    import type { CheckerScope, CheckMetric } from "$lib/api/checkers";
    import { listScopedExecutions, getCheckStatus, deleteScopedExecution, deleteAllScopedExecutions, getScopedCheckerMetrics } from "$lib/api/checkers";
    import {
        getExecutionStatusColor,
        getExecutionStatusI18nKey,
        getStatusColor,
        getStatusI18nKey,
        formatCheckDate,
    } from "$lib/utils";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import RunCheckModal from "$lib/components/modals/RunCheckModal.svelte";
    import CheckMetricsChart from "$lib/components/checkers/CheckMetricsChart.svelte";

    interface Props {
        scope: CheckerScope;
        checksBase: string;
        checkerId: string;
        domainName: string;
    }

    let { scope, checksBase, checkerId, domainName }: Props = $props();

    let resolvedName = $state<string>("");
    let executions = $state<HappydnsExecution[]>([]);
    let executionsPromise: Promise<HappydnsExecution[]> = $derived(loadExecutions());
    let runCheckModal = $state<RunCheckModal>();
    let metricsData = $state<CheckMetric[] | null>(null);

    $effect(() => {
        metricsData = null;
        getCheckStatus(checkerId).then((s) => {
            resolvedName = s.name ?? checkerId;
            if (s.has_metrics) {
                getScopedCheckerMetrics(scope, checkerId)
                    .then((m) => (metricsData = m))
                    .catch((e) => console.warn("Failed to load checker metrics", e));
            }
        });
    });

    async function loadExecutions() {
        try {
            executions = await listScopedExecutions(scope, checkerId, { includePlanned: true });
            return executions;
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.executions.error-loading", { error: String(error) }),
                timeout: 10000,
            });
            throw error;
        }
    }

    async function deleteExecution(executionId: string) {
        try {
            await deleteScopedExecution(scope, checkerId, executionId);
            executions = executions.filter((e) => e.id !== executionId);
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.executions.error-deleting", { error: String(error) }),
                timeout: 10000,
            });
        }
    }

    async function deleteAllExecutions() {
        if (!confirm($t("checkers.executions.delete-all-confirm"))) {
            return;
        }
        try {
            await deleteAllScopedExecutions(scope, checkerId);
            // Keep only planned executions (those without an id); completed ones were deleted server-side.
            executions = executions.filter((e) => !e.id);
            toasts.addToast({
                message: $t("checkers.executions.deleted-all"),
                type: "success",
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.executions.error-deleting", { error: String(error) }),
                timeout: 10000,
            });
        }
    }

    let pollTimer: ReturnType<typeof setInterval> | undefined;

    function pollForNewExecution() {
        if (pollTimer) clearInterval(pollTimer);
        const previousCount = executions.length;
        let attempts = 0;
        const maxAttempts = 10;
        const intervalMs = 3000;

        pollTimer = setInterval(async () => {
            attempts++;
            try {
                await loadExecutions();
                if (executions.length > previousCount || attempts >= maxAttempts) {
                    clearInterval(pollTimer);
                    pollTimer = undefined;
                }
            } catch {
                clearInterval(pollTimer);
                pollTimer = undefined;
            }
        }, intervalMs);
    }

    onDestroy(() => {
        if (pollTimer) clearInterval(pollTimer);
    });
</script>

<svelte:head>
    <title>
        {$t("checkers.executions.title", { count: executions.length })} - {resolvedName ||
            checkerId} - happyDomain
    </title>
</svelte:head>

<div class="flex-fill mt-1 mb-5">
    <PageTitle
        title={$t("checkers.executions.title", { count: executions.length })}
        subtitle={resolvedName}
        domain={domainName}
    >
        <div class="d-flex gap-2">
            <Button color="dark" href="{checksBase}/{checkerId}">
                <Icon name="gear-fill"></Icon>
                {$t("checkers.executions.configure")}
            </Button>
            <Button
                color="primary"
                onclick={() => runCheckModal?.open(checkerId, resolvedName || checkerId)}
            >
                <Icon name="play-fill"></Icon>
                {$t("checkers.executions.run-check-now")}
            </Button>
            <Button
                color="danger"
                outline
                disabled={executions.filter((e) => e.id).length === 0}
                onclick={deleteAllExecutions}
            >
                <Icon name="trash-fill"></Icon>
                {$t("checkers.executions.delete-all")}
            </Button>
        </div>
    </PageTitle>

    {#await executionsPromise}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.executions.loading")}
            </p>
        </Card>
    {:then _executions}
        {#if metricsData && metricsData.length > 0}
            <Card class="mb-3">
                <div class="card-body">
                    <CheckMetricsChart metrics={metricsData} />
                </div>
            </Card>
        {/if}
        {#if executions.length === 0}
            <Alert color="info">
                <Icon name="info-circle" />
                {$t("checkers.executions.no-results")}
            </Alert>
        {:else}
            <Table hover responsive>
                <thead>
                    <tr>
                        <th>{$t("checkers.executions.table.executed-at")}</th>
                        <th>{$t("checkers.executions.table.status")}</th>
                        <th>{$t("checkers.executions.table.duration")}</th>
                        <th>{$t("checkers.executions.table.actions")}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each executions.toSorted((a, b) => {
                        const aTime = a.startedAt ? new Date(a.startedAt).getTime() : Infinity;
                        const bTime = b.startedAt ? new Date(b.startedAt).getTime() : Infinity;
                        return bTime - aTime;
                    }) as execution}
                        {@const isPending = !execution.id}
                        {@const isRunning =
                            execution.id && execution.startedAt && !execution.endedAt}
                        {@const status = execution.status}
                        {@const duration =
                            execution.startedAt && execution.endedAt
                                ? Math.round(
                                      (new Date(execution.endedAt).getTime() -
                                          new Date(execution.startedAt).getTime()) /
                                          1000,
                                  )
                                : null}
                        <tr>
                            <td>
                                {#if !execution.startedAt}
                                    <span class="text-muted fst-italic">
                                        {$t("checkers.status.planned")}
                                    </span>
                                {:else if isPending}
                                    <span class="text-muted fst-italic">
                                        {formatCheckDate(execution.startedAt)}
                                    </span>
                                {:else}
                                    {formatCheckDate(execution.startedAt)}
                                {/if}
                            </td>
                            <td>
                                {#if isPending}
                                    <Badge color="secondary">{$t("checkers.status.planned")}</Badge>
                                {:else if status == 2 && execution.result}
                                    <Badge color={getStatusColor(execution.result.status)}>
                                        {$t(getStatusI18nKey(execution.result.status))}
                                    </Badge>
                                {:else}
                                    <Badge color={getExecutionStatusColor(status)}>
                                        {$t(getExecutionStatusI18nKey(status))}
                                    </Badge>
                                {/if}
                            </td>
                            <td>
                                {#if isRunning}
                                    <span class="text-muted fst-italic">
                                        {$t("checkers.status.running")}
                                    </span>
                                {:else if duration !== null}
                                    {duration}s
                                {:else}
                                    -
                                {/if}
                            </td>
                            <td>
                                <div class="d-flex gap-1">
                                    <a
                                        href="{checksBase}/{checkerId}/executions/{execution.id}"
                                        class="btn btn-sm btn-outline-primary"
                                        class:disabled={!execution.id && !isRunning}
                                    >
                                        {$t("checkers.executions.view")}
                                    </a>
                                    <Button
                                        color="danger"
                                        size="sm"
                                        outline
                                        disabled={!!isPending || !!isRunning}
                                        onclick={() =>
                                            execution.id && deleteExecution(execution.id)}
                                    >
                                        <Icon name="trash" />
                                    </Button>
                                </div>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </Table>
        {/if}
    {/await}
</div>

<RunCheckModal
    {scope}
    onCheckTriggered={() => pollForNewExecution()}
    bind:this={runCheckModal}
/>
