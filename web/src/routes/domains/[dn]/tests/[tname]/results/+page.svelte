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
        listTestResults,
        deleteTestResult,
        deleteAllTestResults,
        getTestExecution,
    } from "$lib/api/tests";
    import { getPluginStatus } from "$lib/api/plugins";
    import type { Domain } from "$lib/model/domain";
    import type { TestExecution } from "$lib/model/test";
    import { TestExecutionStatus } from "$lib/model/test";
    import RunTestModal from "$lib/components/modals/RunTestModal.svelte";
    import { getStatusColor, getStatusKey, formatDuration, formatTestDate } from "$lib/utils";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    const testName = $derived(page.params.tname || "");

    let resultsPromise = $derived(listTestResults(data.domain.id, testName));
    let pluginPromise = $derived(getPluginStatus(testName));
    let runTestModal: RunTestModal;
    let errorMessage = $state<string | null>(null);
    let pendingExecutions = $state<TestExecution[]>([]);
    const pollingIntervals = new Map<string, ReturnType<typeof setInterval>>();

    onDestroy(() => {
        for (const id of pollingIntervals.values()) clearInterval(id);
    });

    function handleTestTriggered(execution_id: string) {
        const placeholder: TestExecution = {
            id: execution_id,
            plugin_name: testName,
            user_id: "",
            target_id: data.domain.id,
            status: TestExecutionStatus.TestExecutionPending,
            started_at: new Date().toISOString(),
        };
        pendingExecutions = [...pendingExecutions, placeholder];

        const intervalId = setInterval(async () => {
            try {
                const exec = await getTestExecution(data.domain.id, testName, execution_id);
                pendingExecutions = pendingExecutions.map((e) =>
                    e.id === execution_id ? exec : e,
                );

                if (
                    exec.status === TestExecutionStatus.TestExecutionCompleted ||
                    exec.status === TestExecutionStatus.TestExecutionFailed
                ) {
                    clearInterval(intervalId);
                    pollingIntervals.delete(execution_id);
                    pendingExecutions = pendingExecutions.filter((e) => e.id !== execution_id);
                    resultsPromise = listTestResults(data.domain.id, testName);
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
        if (!confirm($t("tests.results.delete-confirm"))) {
            return;
        }

        try {
            await deleteTestResult(data.domain.id, testName, resultId);
            resultsPromise = listTestResults(data.domain.id, testName);
        } catch (error: any) {
            errorMessage = error.message || $t("tests.results.delete-failed");
        }
    }

    async function handleDeleteAll() {
        if (!confirm($t("tests.results.delete-all-confirm"))) {
            return;
        }

        try {
            await deleteAllTestResults(data.domain.id, testName);
            resultsPromise = listTestResults(data.domain.id, testName);
        } catch (error: any) {
            errorMessage = error.message || $t("tests.results.delete-all-failed");
        }
    }
</script>

<svelte:head>
    <title>{testName} Results - {data.domain.domain} - happyDomain</title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2>
            <span class="font-monospace">{data.domain.domain}</span>
            &ndash;
            {#await pluginPromise then plugin}
                {plugin.name || testName}
            {:catch}
                {testName}
            {/await}
        </h2>
        <div class="d-flex gap-2">
            <Button
                color="dark"
                href={`/domains/${encodeURIComponent(data.domain.domain)}/tests/${encodeURIComponent(testName)}`}
            >
                <Icon name="gear-fill"></Icon>
                {$t("tests.results.configure")}
            </Button>
            {#await pluginPromise then plugin}
                <Button
                    color="primary"
                    onclick={() => runTestModal.open(testName, plugin.name || testName)}
                >
                    <Icon name="play-fill"></Icon>
                    {$t("tests.results.run-test-now")}
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
            <p>{$t("tests.results.loading")}</p>
        </div>
    {:then results}
        {#if !results || results.length === 0}
            <Card body>
                <p class="text-center text-muted mb-0">
                    <Icon name="info-circle"></Icon>
                    {$t("tests.results.no-results")}
                </p>
            </Card>
        {:else}
            <div class="d-flex justify-content-between align-items-center mb-2">
                <h4>{$t("tests.results.title", { count: results.length })}</h4>
                <Button size="sm" color="danger" outline onclick={handleDeleteAll}>
                    <Icon name="trash"></Icon>
                    {$t("tests.results.delete-all")}
                </Button>
            </div>

            <Table hover striped>
                <thead>
                    <tr>
                        <th>{$t("tests.results.table.executed-at")}</th>
                        <th class="text-center">{$t("tests.results.table.status")}</th>
                        <th>{$t("tests.results.table.message")}</th>
                        <th>{$t("tests.results.table.duration")}</th>
                        <th class="text-center">{$t("tests.results.table.type")}</th>
                        <th>{$t("tests.results.table.actions")}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each pendingExecutions as exec (exec.id)}
                        <tr class="table-warning">
                            <td class="align-middle">
                                {formatTestDate(exec.started_at, "short", $t)}
                            </td>
                            <td class="align-middle text-center">
                                <Badge color="secondary">
                                    {$t("tests.status.pending")}
                                </Badge>
                            </td>
                            <td class="align-middle text-muted">
                                {exec.status === TestExecutionStatus.TestExecutionRunning
                                    ? $t("tests.results.pending.running")
                                    : $t("tests.results.pending.queued")}
                            </td>
                            <td class="align-middle">—</td>
                            <td class="align-middle text-center">
                                <Badge color="secondary">
                                    {#if exec.schedule_id}
                                        <Icon name="clock"></Icon>
                                        {$t("tests.results.type.scheduled")}
                                    {:else}
                                        <Icon name="hand-index"></Icon>
                                        {$t("tests.results.type.manual")}
                                    {/if}
                                </Badge>
                            </td>
                            <td class="align-middle"></td>
                        </tr>
                    {/each}
                    {#each results as result}
                        <tr>
                            <td class="align-middle">
                                {formatTestDate(result.executed_at, "short", $t)}
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
                                    {#if result.scheduled_test}
                                        <Icon name="clock"></Icon>
                                        {$t("tests.results.type.scheduled")}
                                    {:else}
                                        <Icon name="hand-index"></Icon>
                                        {$t("tests.results.type.manual")}
                                    {/if}
                                </Badge>
                            </td>
                            <td class="align-middle">
                                <ButtonGroup size="sm">
                                    <Button
                                        color="primary"
                                        href={`/domains/${encodeURIComponent(data.domain.domain)}/tests/${encodeURIComponent(testName)}/results/${encodeURIComponent(result.id)}`}
                                    >
                                        <Icon name="eye-fill"></Icon>
                                        {$t("tests.results.view")}
                                    </Button>
                                    <Button
                                        color="danger"
                                        outline
                                        onclick={() => handleDeleteResult(result.id)}
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
                {$t("tests.results.error-loading", { error: error.message })}
            </p>
        </Card>
    {/await}
</div>

<RunTestModal
    domainId={data.domain.id}
    onTestTriggered={handleTestTriggered}
    bind:this={runTestModal}
/>
