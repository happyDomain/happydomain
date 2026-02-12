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
    import { goto } from "$app/navigation";
    import { getTestResult, deleteTestResult, triggerTest } from "$lib/api/tests";
    import { getPluginStatus } from "$lib/api/plugins";
    import type { Domain } from "$lib/model/domain";
    import type { TestResult } from "$lib/model/test";
    import { getStatusColor, getStatusKey, formatDuration, formatTestDate } from "$lib/utils";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    const testName = $derived(page.params.tname || "");
    const resultId = $derived(page.params.rid || "");

    let resultPromise = $derived(getTestResult(data.domain.id, testName, resultId));
    let pluginPromise = $derived(getPluginStatus(testName));
    let errorMessage = $state<string | null>(null);
    let resolvedResult = $state<TestResult | null>(null);
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
            await triggerTest(data.domain.id, testName, resolvedResult.options);
            goto(
                `/domains/${encodeURIComponent(data.domain.domain)}/tests/${encodeURIComponent(testName)}`,
            );
        } catch (error: any) {
            errorMessage = error.message || $t("tests.result.relaunch-failed");
        } finally {
            isRelaunching = false;
        }
    }

    async function handleDelete() {
        if (!confirm($t("tests.result.delete-confirm"))) {
            return;
        }

        try {
            await deleteTestResult(data.domain.id, testName, resultId);
            goto(
                `/domains/${encodeURIComponent(data.domain.domain)}/tests/${encodeURIComponent(testName)}`,
            );
        } catch (error: any) {
            errorMessage = error.message || $t("tests.result.delete-failed");
        }
    }
</script>

<svelte:head>
    <title>
        Test Result - {testName} - {data.domain.domain} - happyDomain
    </title>
</svelte:head>

<div class="flex-fill pb-4 pt-2 mw-100">
    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2 class="text-truncate">
            <span class="font-monospace">{data.domain.domain}</span>
            &ndash;
            {$t("tests.result.title")}
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
                    {$t("tests.result.relaunch")}
                </span>
            </Button>
            <Button color="danger" outline onclick={handleDelete} disabled={!resolvedResult}>
                <Icon name="trash"></Icon>
                <span class="d-none d-lg-inline">
                    {$t("tests.result.delete")}
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

    {#await Promise.all([resultPromise, pluginPromise])}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("tests.result.loading")}</p>
        </div>
    {:then [result, plugin]}
        <Row>
            <Col lg>
                <Card class="mb-3">
                    <CardHeader>
                        <div class="d-flex justify-content-between align-items-center">
                            <div class="d-flex align-items-end gap-2">
                                <h4 class="mb-0">
                                    {plugin.name || testName}
                                </h4>
                                {#if plugin.version}
                                    <small
                                        class="text-muted"
                                        title={$t("tests.result.field.plugin-version")}
                                    >
                                        {plugin.version}
                                    </small>
                                {/if}
                            </div>
                            {#if result.scheduled_test}
                                <Badge color="info">
                                    <Icon name="clock"></Icon>
                                    {$t("tests.result.type.scheduled")}
                                </Badge>
                            {:else}
                                <Badge color="secondary">
                                    <Icon name="hand-index"></Icon>
                                    {$t("tests.result.type.manual")}
                                </Badge>
                            {/if}
                        </div>
                    </CardHeader>
                    <CardBody class="p-2">
                        <Table borderless size="sm" class="mb-0">
                            <tbody>
                                <tr>
                                    <th style="width: 200px">{$t("tests.result.field.domain")}</th>
                                    <td class="font-monospace">{data.domain.domain}</td>
                                </tr>
                                <tr>
                                    <th>{$t("tests.result.field.executed-at")}</th>
                                    <td>{formatTestDate(result.executed_at, "long", $t)}</td>
                                </tr>
                                <tr>
                                    <th>{$t("tests.result.field.duration")}</th>
                                    <td>{formatDuration(result.duration, $t)}</td>
                                </tr>
                                <tr>
                                    <th>{$t("tests.result.field.status")}</th>
                                    <td>
                                        <Badge color={getStatusColor(result.status)}>
                                            {$t(getStatusKey(result.status))}
                                        </Badge>
                                    </td>
                                </tr>
                                <tr>
                                    <th>{$t("tests.result.field.status-message")}</th>
                                    <td>{result.status_line}</td>
                                </tr>
                                {#if result.error}
                                    <tr>
                                        <th>{$t("tests.result.field.error")}</th>
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
                                {$t("tests.result.test-options")}
                            </h5>
                        </CardHeader>
                        <CardBody class="p-2">
                            <Table borderless size="sm" class="mb-0">
                                <tbody>
                                    {#each Object.entries(plugin.options ?? {}) as [optKey, optVals]}
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
                        {$t("tests.result.full-report")}
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
                {$t("tests.result.error-loading", { error: error.message })}
            </p>
        </Card>
    {/await}
</div>

<style>
    pre {
        overflow-x: scroll;
    }
</style>
