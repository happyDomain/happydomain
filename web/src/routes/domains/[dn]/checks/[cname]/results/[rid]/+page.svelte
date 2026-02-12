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
        Alert,
        Badge,
        Button,
        Card,
        CardBody,
        CardHeader,
        Col,
        Icon,
        Row,
        Spinner,
        Table,
    } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { page } from "$app/state";
    import { navigate } from "$lib/stores/config";
    import {
        getCheckStatus,
        getCheckResult,
        deleteCheckResult,
        triggerCheck,
    } from "$lib/api/checks";
    import type { Domain } from "$lib/model/domain";
    import type { CheckResult } from "$lib/model/check";
    import { getStatusColor, getStatusKey, formatDuration, formatCheckDate } from "$lib/utils";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    const checkName = $derived(page.params.cname || "");
    const resultId = $derived(page.params.rid || "");

    let resultPromise = $derived(getCheckResult(data.domain.id, checkName, resultId));
    let checkPromise = $derived(getCheckStatus(checkName));
    let errorMessage = $state<string | null>(null);
    let resolvedResult = $state<CheckResult | null>(null);
    let isRelaunching = $state(false);

    $effect(() => {
        resultPromise.then((r) => {
            resolvedResult = r;
        });
    });

    async function handleRelaunch() {
        if (!resolvedResult) return;

        isRelaunching = true;
        try {
            await triggerCheck(data.domain.id, checkName, resolvedResult.options);
            navigate(
                `/domains/${encodeURIComponent(data.domain.domain)}/checks/${encodeURIComponent(checkName)}`,
            );
        } catch (error: any) {
            errorMessage = error.message || $t("checks.result.relaunch-failed");
        } finally {
            isRelaunching = false;
        }
    }

    async function handleDelete() {
        if (!confirm($t("checks.result.delete-confirm"))) {
            return;
        }

        try {
            await deleteCheckResult(data.domain.id, checkName, resultId);
            navigate(
                `/domains/${encodeURIComponent(data.domain.domain)}/checks/${encodeURIComponent(checkName)}`,
            );
        } catch (error: any) {
            errorMessage = error.message || $t("checks.result.delete-failed");
        }
    }
</script>

<svelte:head>
    <title>
        Check Result - {checkName} - {data.domain.domain} - happyDomain
    </title>
</svelte:head>

<div class="flex-fill pb-4 pt-2 mw-100">
    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2 class="text-truncate">
            <span class="font-monospace">{data.domain.domain}</span>
            &ndash;
            {$t("checks.result.title")}
        </h2>
        <div class="d-flex gap-2">
            <Button
                color="primary"
                outline
                onclick={handleRelaunch}
                disabled={!resolvedResult || isRelaunching}
            >
                {#if isRelaunching}
                    <Spinner size="sm" />
                {:else}
                    <Icon name="arrow-repeat"></Icon>
                {/if}
                <span class="d-none d-lg-inline">
                    {$t("checks.result.relaunch")}
                </span>
            </Button>
            <Button color="danger" outline onclick={handleDelete} disabled={!resolvedResult}>
                <Icon name="trash"></Icon>
                <span class="d-none d-lg-inline">
                    {$t("checks.result.delete")}
                </span>
            </Button>
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

    {#await Promise.all([resultPromise, checkPromise])}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("checks.result.loading")}</p>
        </div>
    {:then [result, check]}
        <Row>
            <Col lg>
                <Card class="mb-3">
                    <CardHeader>
                        <div class="d-flex justify-content-between align-items-center">
                            <div class="d-flex align-items-end gap-2">
                                <h4 class="mb-0">
                                    {check.name || checkName}
                                </h4>
                            </div>
                            {#if result.scheduled_check}
                                <Badge color="info">
                                    <Icon name="clock"></Icon>
                                    {$t("checks.result.type.scheduled")}
                                </Badge>
                            {:else}
                                <Badge color="secondary">
                                    <Icon name="hand-index"></Icon>
                                    {$t("checks.result.type.manual")}
                                </Badge>
                            {/if}
                        </div>
                    </CardHeader>
                    <CardBody class="p-2">
                        <Table borderless size="sm" class="mb-0">
                            <tbody>
                                <tr>
                                    <th style="width: 200px">{$t("checks.result.field.domain")}</th>
                                    <td class="font-monospace">{data.domain.domain}</td>
                                </tr>
                                <tr>
                                    <th>{$t("checks.result.field.executed-at")}</th>
                                    <td>{formatCheckDate(result.executed_at, "long", $t)}</td>
                                </tr>
                                <tr>
                                    <th>{$t("checks.result.field.duration")}</th>
                                    <td>{formatDuration(result.duration, $t)}</td>
                                </tr>
                                <tr>
                                    <th>{$t("checks.result.field.status")}</th>
                                    <td>
                                        <Badge color={getStatusColor(result.status)}>
                                            {$t(getStatusKey(result.status))}
                                        </Badge>
                                    </td>
                                </tr>
                                <tr>
                                    <th>{$t("checks.result.field.status-message")}</th>
                                    <td>{result.status_line}</td>
                                </tr>
                                {#if result.error}
                                    <tr>
                                        <th>{$t("checks.result.field.error")}</th>
                                        <td class="text-danger">{result.error}</td>
                                    </tr>
                                {/if}
                            </tbody>
                        </Table>
                    </CardBody>
                </Card>
            </Col>
            {#if result.options && Object.keys(result.options).length > 0}
                <Col lg>
                    <Card class="mb-3">
                        <CardHeader>
                            <h5 class="mb-0">
                                <Icon name="sliders"></Icon>
                                {$t("checks.result.check-options")}
                            </h5>
                        </CardHeader>
                        <CardBody class="p-2">
                            <Table borderless size="sm" class="mb-0">
                                <tbody>
                                    {#each Object.entries(check.options ?? {}) as [optKey, optVals]}
                                        {#each optVals as option}
                                            {@const value =
                                                (option.id
                                                    ? result.options[option.id]
                                                    : undefined) ||
                                                option.default ||
                                                option.placeholder ||
                                                ""}
                                            <tr>
                                                <th
                                                    class="text-truncate"
                                                    style="max-width: min(200px, 40vw)"
                                                    title={option.label}
                                                >
                                                    {option.label}:
                                                </th>
                                                <td class:text-truncate={typeof value !== "object"}>
                                                    {#if typeof value === "object"}
                                                        <pre class="mb-0"><code
                                                                >{JSON.stringify(
                                                                    value,
                                                                    null,
                                                                    2,
                                                                )}</code
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
                        </CardBody>
                    </Card>
                </Col>
            {/if}
        </Row>

        {#if result.report}
            <Card>
                <CardHeader>
                    <h5 class="mb-0">
                        <Icon name="file-earmark-text"></Icon>
                        {$t("checks.result.full-report")}
                    </h5>
                </CardHeader>
                <CardBody class="text-truncate p-0">
                    {#if typeof result.report === "string"}
                        <pre class="bg-light p-3 rounded mb-0"><code>{result.report}</code></pre>
                    {:else}
                        <pre class="bg-light p-3 rounded mb-0"><code
                                >{JSON.stringify(result.report, null, 2)}</code
                            ></pre>
                    {/if}
                </CardBody>
            </Card>
        {/if}
    {:catch error}
        <Card body color="danger">
            <p class="mb-0">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t("checks.result.error-loading", { error: error.message })}
            </p>
        </Card>
    {/await}
</div>

<style>
    pre {
        overflow-x: scroll;
    }
</style>
