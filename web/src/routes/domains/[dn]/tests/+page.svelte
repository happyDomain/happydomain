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
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { Card, Icon, Table, Badge, Button, Spinner } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { listAvailableTests, updateTestSchedule, createTestSchedule } from "$lib/api/tests";
    import type { Domain } from "$lib/model/domain";
    import { TestScopeType, type AvailableTest } from "$lib/model/test";
    import { plugins } from "$lib/stores/plugins";
    import { toasts } from "$lib/stores/toasts";
    import RunTestModal from "$lib/components/modals/RunTestModal.svelte";
    import { getStatusColor, getStatusKey, formatTestDate } from "$lib/utils";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    let testsPromise = $derived(listAvailableTests(data.domain.id));
    let runTestModal: RunTestModal;
    let togglingTests = $state(new Set<string>());

    function handleTestTriggered(_: string, pluginName: string) {
        // Refresh the test list to show updated status
        testsPromise = listAvailableTests(data.domain.id);
        goto(`/domains/${page.params.dn!}/tests/${pluginName}/results`);
    }

    async function handleToggleEnabled(test: AvailableTest) {
        const next = new Set(togglingTests);
        next.add(test.plugin_name);
        togglingTests = next;

        try {
            const newEnabled = !test.enabled;
            if (test.schedule) {
                await updateTestSchedule(test.schedule.id, {
                    ...test.schedule,
                    enabled: newEnabled,
                });
            } else {
                // No schedule record yet — create one to persist the disabled state.
                // (Enabled → Enabled needs no action since that's the implicit default.)
                await createTestSchedule({
                    plugin_name: test.plugin_name,
                    target_type: TestScopeType.TestScopeDomain,
                    target_id: data.domain.id,
                    interval: 0,
                    enabled: newEnabled,
                });
            }
            testsPromise = listAvailableTests(data.domain.id);
        } catch (e: any) {
            toasts.addErrorToast({ title: $t("tests.list.error-loading", { error: e.message }) });
        } finally {
            const after = new Set(togglingTests);
            after.delete(test.plugin_name);
            togglingTests = after;
        }
    }

</script>

<svelte:head>
    <title>Tests - {data.domain.domain} - happyDomain</title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <h2>
        {$t("tests.list.title")}<span class="font-monospace">{data.domain.domain}</span>
    </h2>

    {#await testsPromise}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("tests.list.loading")}</p>
        </div>
    {:then tests}
        {#if !$plugins}
            <div class="mt-5 text-center flex-fill">
                <Spinner />
                <p>{$t("tests.list.loading-plugins")}</p>
            </div>
        {:else if !tests || tests.length === 0}
            <Card body class="mt-3">
                <p class="text-center text-muted mb-0">
                    <Icon name="info-circle"></Icon>
                    {$t("tests.list.no-tests")}
                </p>
            </Card>
        {:else}
            <Table hover striped class="mt-3">
                <thead>
                    <tr>
                        <th>{$t("tests.list.table.plugin")}</th>
                        <th>{$t("tests.list.table.status")}</th>
                        <th>{$t("tests.list.table.last-run")}</th>
                        <th>{$t("tests.list.table.schedule")}</th>
                        <th>{$t("tests.list.table.actions")}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each tests as test}
                        {@const pluginInfo = $plugins[test.plugin_name]}
                        <tr>
                            <td class="align-middle">
                                <strong>{pluginInfo?.name || test.plugin_name}</strong>
                                <small class="ms-1 text-muted">
                                    {pluginInfo?.version || $t("tests.list.unknown-version")}
                                </small>
                            </td>
                            <td class="align-middle text-center">
                                {#if test.last_result !== undefined}
                                    <Badge color={getStatusColor(test.last_result.status)}>
                                        {$t(getStatusKey(test.last_result.status))}
                                    </Badge>
                                {:else}
                                    <Badge color="secondary">{$t("tests.status.not-run")}</Badge>
                                {/if}
                            </td>
                            <td class="align-middle">
                                {formatTestDate(test.last_result?.executed_at, "short", $t)}
                            </td>
                            <td class="align-middle">
                                <div class="form-check form-switch mb-0">
                                    <input
                                        class="form-check-input"
                                        type="checkbox"
                                        role="switch"
                                        id="toggle-{test.plugin_name}"
                                        checked={test.enabled}
                                        disabled={togglingTests.has(test.plugin_name)}
                                        onchange={() => handleToggleEnabled(test)}
                                    />
                                    <label
                                        class="form-check-label small"
                                        for="toggle-{test.plugin_name}"
                                    >
                                        {test.enabled
                                            ? $t("tests.list.schedule.enabled")
                                            : $t("tests.list.schedule.disabled")}
                                    </label>
                                </div>
                            </td>
                            <td class="align-middle">
                                <div class="d-flex gap-2">
                                    <Button
                                        size="sm"
                                        color="primary"
                                        onclick={() =>
                                            runTestModal.open(
                                                test.plugin_name,
                                                pluginInfo?.name || test.plugin_name,
                                            )}
                                    >
                                        <Icon name="play-fill"></Icon>
                                        {$t("tests.list.run-test")}
                                    </Button>
                                    <Button
                                        size="sm"
                                        color="info"
                                        href={`/domains/${encodeURIComponent(data.domain.domain)}/tests/${encodeURIComponent(test.plugin_name)}/results`}
                                    >
                                        <Icon name="bar-chart-fill"></Icon>
                                        {$t("tests.list.view-results")}
                                    </Button>
                                    <Button
                                        size="sm"
                                        color="dark"
                                        href={`/domains/${encodeURIComponent(data.domain.domain)}/tests/${encodeURIComponent(test.plugin_name)}`}
                                        title={$t("tests.list.configure")}
                                    >
                                        <Icon name="gear"></Icon>
                                    </Button>
                                </div>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </Table>
        {/if}
    {:catch error}
        <Card body color="danger" class="mt-3">
            <p class="mb-0">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t("tests.list.error-loading", { error: error.message })}
            </p>
        </Card>
    {/await}
</div>

<RunTestModal
    domainId={data.domain.id}
    onTestTriggered={handleTestTriggered}
    bind:this={runTestModal}
/>
