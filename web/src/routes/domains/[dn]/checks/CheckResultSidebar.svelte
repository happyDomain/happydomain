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
    // SvelteKit imports
    import { navigate } from "$lib/stores/config";

    // Component imports
    import {
        Badge,
        Button,
        ButtonGroup,
        Card,
        CardBody,
        CardHeader,
        CardTitle,
        Icon,
        Spinner,
        Table,
    } from "@sveltestrap/sveltestrap";

    // Store imports
    import { currentCheckResult, currentCheckInfo, showHTMLReport } from "$lib/stores/checks";
    import { toasts } from "$lib/stores/toasts";

    // API imports
    import { deleteCheckResult, getCheckResultHTMLReport, triggerCheck } from "$lib/api/checks";

    // Utility imports
    import { getStatusColor, getStatusKey, formatDuration, formatCheckDate } from "$lib/utils";
    import { t } from "$lib/translations";

    // Model imports
    import type { Domain } from "$lib/model/domain";

    // Props
    interface Props {
        domain: Domain;
        cname: string;
        rid: string;
    }

    let { domain, cname, rid }: Props = $props();

    // Local state
    let isRelaunching = $state(false);

    // Functions
    async function handleRelaunch() {
        if (!$currentCheckResult) return;
        isRelaunching = true;
        try {
            await triggerCheck(domain.id, cname, $currentCheckResult.options);
            navigate(
                `/domains/${encodeURIComponent(domain.domain)}/checks/${encodeURIComponent(cname)}`,
            );
        } catch (error: any) {
            toasts.addErrorToast({ message: error.message || $t("checks.result.relaunch-failed") });
        } finally {
            isRelaunching = false;
        }
    }

    async function handleDelete() {
        if (!confirm($t("checks.result.delete-confirm"))) return;
        try {
            await deleteCheckResult(domain.id, cname, rid);
            navigate(
                `/domains/${encodeURIComponent(domain.domain)}/checks/${encodeURIComponent(cname)}`,
            );
        } catch (error: any) {
            toasts.addErrorToast({ message: error.message || $t("checks.result.delete-failed") });
        }
    }

    function downloadBlob(content: string, filename: string, mime: string) {
        const blob = new Blob([content], { type: mime });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        a.click();
        URL.revokeObjectURL(url);
    }

    async function downloadHTML() {
        const html = await getCheckResultHTMLReport(domain.id, cname, rid);
        downloadBlob(html, `${cname}-${rid}.html`, "text/html");
    }

    function downloadJSON() {
        if (!$currentCheckResult) return;
        downloadBlob(
            JSON.stringify($currentCheckResult.report, null, 2),
            `${cname}-${rid}.json`,
            "application/json",
        );
    }
</script>

{#if $currentCheckResult}
    <Card class="mt-3">
        <CardHeader class="px-2">
            <div class="d-flex justify-content-between align-items-center">
                <strong class="text-truncate">{$currentCheckInfo?.name || cname}</strong>
                {#if $currentCheckResult.scheduled_check}
                    <Badge color="info" class="flex-shrink-0">
                        <Icon name="clock"></Icon>
                        {$t("checks.result.type.scheduled")}
                    </Badge>
                {:else}
                    <Badge color="secondary" class="flex-shrink-0">
                        <Icon name="hand-index"></Icon>
                        {$t("checks.result.type.manual")}
                    </Badge>
                {/if}
            </div>
        </CardHeader>
        <Table borderless size="sm" class="mb-0">
            <tbody>
                <tr>
                    <th style="width: 80px; white-space: nowrap"
                        >{$t("checks.result.field.executed-at")}</th
                    >
                    <td>{formatCheckDate($currentCheckResult.executed_at, "short", $t)}</td>
                </tr>
                <tr>
                    <th>{$t("checks.result.field.duration")}</th>
                    <td>{formatDuration($currentCheckResult.duration, $t)}</td>
                </tr>
                <tr>
                    <th>{$t("checks.result.field.status")}</th>
                    <td>
                        <Badge color={getStatusColor($currentCheckResult.status)}>
                            {$t(getStatusKey($currentCheckResult.status))}
                        </Badge>
                    </td>
                </tr>
                <tr>
                    <th>{$t("checks.result.field.status-message")}</th>
                    <td class="text-truncate" style="max-width: 0"
                        >{$currentCheckResult.status_line}</td
                    >
                </tr>
                {#if $currentCheckResult.error}
                    <tr>
                        <th>{$t("checks.result.field.error")}</th>
                        <td class="text-danger text-truncate" style="max-width: 0"
                            >{$currentCheckResult.error}</td
                        >
                    </tr>
                {/if}
            </tbody>
        </Table>
    </Card>
    {#if $currentCheckInfo?.options && $currentCheckResult.options && Object.keys($currentCheckResult.options).length > 0}
        <Card class="mt-3">
            <CardHeader>
                <CardTitle class="h6 mb-0">
                    <Icon name="sliders"></Icon>
                    {$t("checks.result.check-options")}
                </CardTitle>
            </CardHeader>
            <Table borderless size="sm" class="mb-0">
                <tbody>
                    {#each Object.entries($currentCheckInfo.options) as [optKey, optVals]}
                        {#each optVals as option}
                            {@const value =
                                (option.id ? $currentCheckResult.options[option.id] : undefined) ||
                                option.default ||
                                option.placeholder ||
                                ""}
                            <tr>
                                <th
                                    class="text-truncate"
                                    style="max-width: 90px"
                                    title={option.label}
                                >
                                    {option.label}:
                                </th>
                                <td class:text-truncate={typeof value !== "object"}>
                                    {#if typeof value === "object"}
                                        <pre class="mb-0" style="font-size: 0.75em"><code
                                                >{JSON.stringify(value, null, 2)}</code
                                            ></pre>
                                    {:else}
                                        {value}
                                    {/if}
                                </td>
                            </tr>
                        {/each}
                    {/each}
                </tbody>
            </Table>
        </Card>
    {/if}

    <div class="my-3 flex-fill"></div>

    {#if $currentCheckInfo?.has_html_report || $currentCheckResult.report != null}
        {#if $currentCheckInfo?.has_html_report}
            <ButtonGroup class="w-100 mb-2">
                <Button
                    size="sm"
                    color="secondary"
                    outline
                    active={$showHTMLReport}
                    onclick={() => showHTMLReport.set(true)}
                >
                    <Icon name="file-earmark-richtext"></Icon>
                    {$t("checks.result.view-html")}
                </Button>
                <Button
                    size="sm"
                    color="secondary"
                    outline
                    active={!$showHTMLReport}
                    onclick={() => showHTMLReport.set(false)}
                >
                    <Icon name="braces"></Icon>
                    {$t("checks.result.view-json")}
                </Button>
            </ButtonGroup>
        {/if}
        <ButtonGroup class="w-100">
            {#if $currentCheckInfo?.has_html_report}
                <Button size="sm" color="outline-secondary" onclick={downloadHTML}>
                    <Icon name="download"></Icon>
                    {$t("checks.result.download-html")}
                </Button>
            {/if}
            {#if $currentCheckResult.report != null}
                <Button size="sm" color="outline-secondary" onclick={downloadJSON}>
                    <Icon name="download"></Icon>
                    {$t("checks.result.download-json")}
                </Button>
            {/if}
        </ButtonGroup>
    {/if}
{:else}
    <div class="flex-fill"></div>
{/if}

<div class="mt-2 d-flex gap-2">
    <Button
        class="flex-fill"
        color="primary"
        outline
        onclick={handleRelaunch}
        disabled={!$currentCheckResult || isRelaunching}
    >
        {#if isRelaunching}
            <Spinner size="sm" />
        {:else}
            <Icon name="arrow-repeat"></Icon>
        {/if}
        {$t("checks.result.relaunch")}
    </Button>
    <Button color="danger" outline onclick={handleDelete} disabled={!$currentCheckResult}>
        <Icon name="trash"></Icon>
    </Button>
</div>
