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
        Badge,
        Button,
        ButtonGroup,
        Card,
        CardHeader,
        Icon,
        Spinner,
        Table,
    } from "@sveltestrap/sveltestrap";

    import { navigate } from "$lib/stores/config";
    import { currentExecution, currentCheckInfo, currentObservations, reportViewMode, cachedHTMLReport } from "$lib/stores/checkers";
    import { toasts } from "$lib/stores/toasts";
    import type { CheckerScope } from "$lib/api/checkers";
    import {
        triggerScopedCheck,
        deleteScopedExecution,
        getScopedExecutionHTMLReport,
    } from "$lib/api/checkers";
    import {
        getExecutionStatusColor,
        getExecutionStatusI18nKey,
        getStatusColor,
        getStatusI18nKey,
        formatCheckDate,
        downloadBlob,
    } from "$lib/utils";
    import { t } from "$lib/translations";
    import type { Domain } from "$lib/model/domain";

    interface Props {
        domain: Domain;
        checkerId: string;
        execId: string;
        checksBase: string;
        scope: CheckerScope;
    }

    let { domain, checkerId, execId, checksBase, scope }: Props = $props();

    let isRelaunching = $state(false);

    async function handleRelaunch() {
        isRelaunching = true;
        try {
            const execution = await triggerScopedCheck(scope, checkerId);
            toasts.addToast({
                message: $t("checkers.run-check.triggered-success", { id: execution.id ?? "" }),
                type: "success",
                timeout: 5000,
            });
            if (execution.id) {
                navigate(
                    `${checksBase}/${encodeURIComponent(checkerId)}/executions/${execution.id}`,
                );
            }
        } catch (error: any) {
            toasts.addErrorToast({
                message: error.message || $t("checkers.result.relaunch-failed"),
            });
        } finally {
            isRelaunching = false;
        }
    }

    let isDeleting = $state(false);

    async function handleDelete() {
        if (!$currentExecution?.id) return;
        isDeleting = true;
        try {
            await deleteScopedExecution(scope, checkerId, $currentExecution.id);
            navigate(`${checksBase}/${encodeURIComponent(checkerId)}/executions`);
        } catch (error: any) {
            toasts.addErrorToast({
                message:
                    error.message ||
                    $t("checkers.executions.error-deleting", { error: String(error) }),
            });
        } finally {
            isDeleting = false;
        }
    }

    function downloadJSON() {
        if (!$currentObservations?.data) return;
        downloadBlob(
            JSON.stringify($currentObservations.data, null, 2),
            `${checkerId}-${execId}.json`,
            "application/json",
        );
    }

    async function downloadHTML() {
        if (!$currentObservations?.data) return;
        const keys = Object.keys($currentObservations.data);
        if (keys.length === 0) return;
        try {
            const html = $cachedHTMLReport ?? await getScopedExecutionHTMLReport(scope, checkerId, execId, keys[0]);
            downloadBlob(html, `${checkerId}-${execId}.html`, "text/html");
        } catch (error: any) {
            toasts.addErrorToast({
                message: error.message || "Failed to download HTML report",
            });
        }
    }
</script>

{#if $currentExecution}
    <Card class="mt-3">
        <CardHeader class="px-2">
            <div class="d-flex justify-content-between align-items-center">
                <strong class="text-truncate">{$currentCheckInfo?.name || checkerId}</strong>
                <Badge
                    color={getExecutionStatusColor($currentExecution.status)}
                    class="flex-shrink-0"
                >
                    {$t(getExecutionStatusI18nKey($currentExecution.status))}
                </Badge>
            </div>
        </CardHeader>
        <div class="overflow-x-auto rounded-2">
            <Table borderless size="sm" class="mb-0">
                <tbody>
                    <tr>
                        <th style="width: 80px; white-space: nowrap">
                            {$t("checkers.result.field.executed-at")}
                        </th>
                        <td>{formatCheckDate($currentExecution.startedAt)}</td>
                    </tr>
                    {#if $currentExecution.endedAt}
                        <tr>
                            <th>{$t("checkers.execution.field.ended-at")}</th>
                            <td>{formatCheckDate($currentExecution.endedAt)}</td>
                        </tr>
                    {/if}
                    <tr>
                        <th>{$t("checkers.result.field.status")}</th>
                        <td>
                            <Badge color={getStatusColor($currentExecution.result?.status)}>
                                {$t(getStatusI18nKey($currentExecution.result?.status))}
                            </Badge>
                        </td>
                    </tr>
                    {#if $currentExecution.result?.message}
                        <tr>
                            <th>{$t("checkers.result.field.status-message")}</th>
                            <td class="text-truncate" style="max-width: 0">
                                {$currentExecution.result.message}
                            </td>
                        </tr>
                    {/if}
                    {#if $currentExecution.error}
                        <tr>
                            <th>{$t("checkers.result.field.error")}</th>
                            <td class="text-danger text-truncate" style="max-width: 0">
                                {$currentExecution.error}
                            </td>
                        </tr>
                    {/if}
                    {#if $currentExecution.trigger}
                        <tr>
                            <th>{$t("checkers.execution.field.trigger")}</th>
                            <td><code>{JSON.stringify($currentExecution.trigger)}</code></td>
                        </tr>
                    {/if}
                </tbody>
            </Table>
        </div>
    </Card>

    <div class="my-3 flex-fill"></div>

    <ButtonGroup class="w-100 mb-2">
        {#if $currentCheckInfo?.has_metrics}
            <Button
                size="sm"
                color="secondary"
                outline
                active={$reportViewMode === "metrics"}
                onclick={() => {
                    reportViewMode.set("metrics");
                }}
            >
                <Icon name="graph-up"></Icon>
                {$t("checkers.result.view-metrics")}
            </Button>
        {/if}
        {#if $currentCheckInfo?.has_html_report}
            <Button
                size="sm"
                color="secondary"
                outline
                active={$reportViewMode === "html"}
                onclick={() => {
                    reportViewMode.set("html");
                }}
            >
                <Icon name="file-earmark-richtext"></Icon>
                {$t("checkers.result.view-html")}
            </Button>
        {/if}
        <Button
            size="sm"
            color="secondary"
            outline
            active={$reportViewMode === "rules"}
            onclick={() => {
                reportViewMode.set("rules");
            }}
        >
            <Icon name="list-check"></Icon>
            {$t("checkers.detail.check-rules")}
        </Button>
        <Button
            size="sm"
            color="secondary"
            outline
            active={$reportViewMode === "json"}
            onclick={() => {
                reportViewMode.set("json");
            }}
        >
            <Icon name="braces"></Icon>
            {$t("checkers.result.view-json")}
        </Button>
    </ButtonGroup>
    <ButtonGroup class="w-100">
        {#if $currentCheckInfo?.has_html_report}
            <Button size="sm" color="outline-secondary" onclick={downloadHTML}>
                <Icon name="download"></Icon>
                {$t("checkers.result.download-html")}
            </Button>
        {/if}
        <Button
            size="sm"
            color="outline-secondary"
            onclick={downloadJSON}
            disabled={!$currentObservations?.data}
        >
            <Icon name="download"></Icon>
            {$t("checkers.result.download-json")}
        </Button>
    </ButtonGroup>
{:else}
    <div class="flex-fill"></div>
{/if}

<div class="mt-2 d-flex gap-2">
    <Button
        class="flex-fill"
        color="primary"
        outline
        onclick={handleRelaunch}
        disabled={!$currentExecution || isRelaunching}
    >
        {#if isRelaunching}
            <Spinner size="sm" />
        {:else}
            <Icon name="arrow-repeat"></Icon>
        {/if}
        {$t("checkers.result.relaunch")}
    </Button>
    <Button
        color="danger"
        outline
        onclick={handleDelete}
        disabled={!$currentExecution?.id || isDeleting}
    >
        {#if isDeleting}
            <Spinner size="sm" />
        {:else}
            <Icon name="trash"></Icon>
        {/if}
    </Button>
</div>
