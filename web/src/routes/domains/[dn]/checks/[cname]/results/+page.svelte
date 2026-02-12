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
    import {
        Card,
        Alert,
        Icon,
        Table,
        Badge,
        Button,
        Spinner,
        ButtonGroup,
    } from "@sveltestrap/sveltestrap";

    import { onDestroy } from "svelte";

    import { t } from "$lib/translations";
    import { page } from "$app/state";
    import {
        listCheckResults,
        deleteCheckResult,
        deleteAllCheckResults,
        getCheckExecution,
    } from "$lib/api/checks";
    import { getCheckStatus } from "$lib/api/checks";
    import type { Domain } from "$lib/model/domain";
    import type { CheckExecution } from "$lib/model/check";
    import { CheckExecutionStatus, CheckScopeType } from "$lib/model/check";
    import RunCheckModal from "$lib/components/modals/RunCheckModal.svelte";
    import { getStatusColor, getStatusKey, formatDuration, formatCheckDate } from "$lib/utils";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    const checkName = $derived(page.params.cname || "");

    let resultsPromise = $derived(listCheckResults(data.domain.id, checkName));
    let checkPromise = $derived(getCheckStatus(checkName));
    let runCheckModal: RunCheckModal;
    let errorMessage = $state<string | null>(null);
    let pendingExecutions = $state<CheckExecution[]>([]);
    const pollingIntervals = new Map<string, ReturnType<typeof setInterval>>();

    onDestroy(() => {
        for (const id of pollingIntervals.values()) clearInterval(id);
    });

    function handleCheckTriggered(execution_id: string) {
        const placeholder: CheckExecution = {
            id: execution_id,
            checker_name: checkName,
            owner_id: "",
            target_type: CheckScopeType.CheckScopeDomain,
            target_id: data.domain.id,
            status: CheckExecutionStatus.CheckExecutionPending,
            started_at: new Date().toISOString(),
        };
        pendingExecutions = [...pendingExecutions, placeholder];

        const intervalId = setInterval(async () => {
            try {
                const exec = await getCheckExecution(data.domain.id, checkName, execution_id);
                pendingExecutions = pendingExecutions.map((e) =>
                    e.id === execution_id ? exec : e,
                );

                if (
                    exec.status === CheckExecutionStatus.CheckExecutionCompleted ||
                    exec.status === CheckExecutionStatus.CheckExecutionFailed
                ) {
                    clearInterval(intervalId);
                    pollingIntervals.delete(execution_id);
                    pendingExecutions = pendingExecutions.filter((e) => e.id !== execution_id);
                    resultsPromise = listCheckResults(data.domain.id, checkName);
                }
            } catch {
                clearInterval(intervalId);
                pollingIntervals.delete(execution_id);
                pendingExecutions = pendingExecutions.filter((e) => e.id !== execution_id);
            }
        }, 2000);
        pollingIntervals.set(execution_id, intervalId);
    }

    async function handleDeleteResult(resultId: string) {
        if (!confirm($t("checks.results.delete-confirm"))) {
            return;
        }

        try {
            await deleteCheckResult(data.domain.id, checkName, resultId);
            resultsPromise = listCheckResults(data.domain.id, checkName);
        } catch (error: any) {
            errorMessage = error.message || $t("checks.results.delete-failed");
        }
    }

    async function handleDeleteAll() {
        if (!confirm($t("checks.results.delete-all-confirm"))) {
            return;
        }

        try {
            await deleteAllCheckResults(data.domain.id, checkName);
            resultsPromise = listCheckResults(data.domain.id, checkName);
        } catch (error: any) {
            errorMessage = error.message || $t("checks.results.delete-all-failed");
        }
    }
</script>

<svelte:head>
    <title>{checkName} Results - {data.domain.domain} - happyDomain</title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2>
            <span class="font-monospace">{data.domain.domain}</span>
            &ndash;
            {#await checkPromise then check}
                {check.name || checkName}
            {:catch}
                {checkName}
            {/await}
        </h2>
        <div class="d-flex gap-2">
            <Button
                color="dark"
                href={`/domains/${encodeURIComponent(data.domain.domain)}/checks/${encodeURIComponent(checkName)}`}
            >
                <Icon name="gear-fill"></Icon>
                {$t("checks.results.configure")}
            </Button>
            {#await checkPromise then check}
                <Button
                    color="primary"
                    onclick={() => runCheckModal.open(checkName, check.name || checkName)}
                >
                    <Icon name="play-fill"></Icon>
                    {$t("checks.results.run-check-now")}
                </Button>
            {/await}
        </div>
    </div>

    {#if errorMessage}
        {#key errorMessage}
            <Alert color="danger" dismissible>
                <Icon name="exclamation-triangle-fill"></Icon>
                {errorMessage}
            </Alert>
        {/key}
    {/if}

    {#await resultsPromise}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("checks.results.loading")}</p>
        </div>
    {:then results}
        {#if (!results || results.length === 0) && pendingExecutions.length === 0}
            <Card body>
                <p class="text-center text-muted mb-0">
                    <Icon name="info-circle"></Icon>
                    {$t("checks.results.no-results")}
                </p>
            </Card>
        {:else}
            <div class="d-flex justify-content-between align-items-center mb-2">
                <h4>{$t("checks.results.title", { count: results?.length ?? 0 })}</h4>
                {#if results?.length}
                    <Button size="sm" color="danger" outline onclick={handleDeleteAll}>
                        <Icon name="trash"></Icon>
                        {$t("checks.results.delete-all")}
                    </Button>
                {/if}
            </div>

            <Table hover striped>
                <thead>
                    <tr>
                        <th>{$t("checks.results.table.executed-at")}</th>
                        <th class="text-center">{$t("checks.results.table.status")}</th>
                        <th>{$t("checks.results.table.message")}</th>
                        <th>{$t("checks.results.table.duration")}</th>
                        <th class="text-center">{$t("checks.results.table.type")}</th>
                        <th>{$t("checks.results.table.actions")}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each pendingExecutions as exec (exec.id)}
                        <tr class="table-warning">
                            <td class="align-middle">
                                {formatCheckDate(exec.started_at, "short", $t)}
                            </td>
                            <td class="align-middle text-center">
                                <Badge
                                    color={exec.status === CheckExecutionStatus.CheckExecutionRunning
                                        ? "info"
                                        : "secondary"}
                                >
                                    {exec.status === CheckExecutionStatus.CheckExecutionRunning
                                        ? $t("checks.results.pending.running")
                                        : $t("checks.results.pending.queued")}
                                </Badge>
                            </td>
                            <td class="align-middle text-muted">
                                {exec.status === CheckExecutionStatus.CheckExecutionRunning
                                    ? $t("checks.results.pending.running-description")
                                    : $t("checks.results.pending.queued-description")}
                            </td>
                            <td class="align-middle">—</td>
                            <td class="align-middle text-center">
                                <Badge color="secondary">
                                    {#if exec.schedule_id}
                                        <Icon name="clock"></Icon>
                                        {$t("checks.results.type.scheduled")}
                                    {:else}
                                        <Icon name="hand-index"></Icon>
                                        {$t("checks.results.type.manual")}
                                    {/if}
                                </Badge>
                            </td>
                            <td class="align-middle"></td>
                        </tr>
                    {/each}
                    {#each results ?? [] as result}
                        <tr>
                            <td class="align-middle">
                                {formatCheckDate(result.executed_at, "short", $t)}
                            </td>
                            <td class="align-middle text-center">
                                <Badge color={getStatusColor(result.status)}>
                                    {$t(getStatusKey(result.status))}
                                </Badge>
                            </td>
                            <td class="align-middle">
                                {result.status_line}
                                {#if result.error}
                                    <br />
                                    <small class="text-danger">{result.error}</small>
                                {/if}
                            </td>
                            <td class="align-middle">
                                {formatDuration(result.duration, $t)}
                            </td>
                            <td class="align-middle text-center">
                                <Badge color="secondary">
                                    {#if result.scheduled_check}
                                        <Icon name="clock"></Icon>
                                        {$t("checks.results.type.scheduled")}
                                    {:else}
                                        <Icon name="hand-index"></Icon>
                                        {$t("checks.results.type.manual")}
                                    {/if}
                                </Badge>
                            </td>
                            <td class="align-middle">
                                <ButtonGroup size="sm">
                                    <Button
                                        color="primary"
                                        href={`/domains/${encodeURIComponent(data.domain.domain)}/checks/${encodeURIComponent(checkName)}/results/${encodeURIComponent(result.id!)}`}
                                    >
                                        <Icon name="eye-fill"></Icon>
                                        {$t("checks.results.view")}
                                    </Button>
                                    <Button
                                        color="danger"
                                        outline
                                        onclick={() => handleDeleteResult(result.id!)}
                                    >
                                        <Icon name="trash"></Icon>
                                    </Button>
                                </ButtonGroup>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </Table>
        {/if}
    {:catch error}
        <Card body color="danger">
            <p class="mb-0">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t("checks.results.error-loading", { error: error.message })}
            </p>
        </Card>
    {/await}
</div>

<RunCheckModal
    domainId={data.domain.id}
    onCheckTriggered={handleCheckTriggered}
    bind:this={runCheckModal}
/>
