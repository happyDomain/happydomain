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
    import { page } from "$app/state";
    import {
        Badge,
        Button,
        Card,
        CardBody,
        CardHeader,
        Icon,
        Input,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { listAvailableTests, updateTestSchedule, createTestSchedule } from "$lib/api/tests";
    import type { Domain } from "$lib/model/domain";
    import { TestScopeType, type AvailableTest } from "$lib/model/test";
    import { plugins } from "$lib/stores/plugins";
    import { toasts } from "$lib/stores/toasts";
    import { formatTestDate, formatRelative } from "$lib/utils";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    const testName = $derived(page.params.tname || "");
    const pluginName = $derived($plugins?.[testName]?.name || testName);

    // Resolved test data
    let test = $state<AvailableTest | null>(null);
    let loading = $state(true);
    let loadError = $state<string | null>(null);

    // Form state
    let formEnabled = $state(true);
    let formIntervalHours = $state(24);
    let saving = $state(false);

    async function loadTest() {
        loading = true;
        loadError = null;
        try {
            const tests = await listAvailableTests(data.domain.id);
            const found = tests?.find((t) => t.plugin_name === testName) ?? null;
            test = found;
            if (found) {
                formEnabled = found.enabled;
                formIntervalHours =
                    found.schedule && found.schedule.interval > 0
                        ? found.schedule.interval / (3600 * 1e9)
                        : 24;
            }
        } catch (e: any) {
            loadError = e.message;
        } finally {
            loading = false;
        }
    }

    loadTest();

    async function handleSave() {
        if (!test) return;
        saving = true;

        try {
            const intervalNs = Math.max(formIntervalHours, 1) * 3600 * 1e9;

            if (test.schedule) {
                await updateTestSchedule(test.schedule.id, {
                    ...test.schedule,
                    enabled: formEnabled,
                    interval: intervalNs,
                });
            } else {
                await createTestSchedule({
                    plugin_name: test.plugin_name,
                    target_type: TestScopeType.TestScopeDomain,
                    target_id: data.domain.id,
                    interval: intervalNs,
                    enabled: formEnabled,
                });
            }

            toasts.addToast({ title: $t("tests.schedule.saved"), type: "success", timeout: 3000 });
            await loadTest();
        } catch (e: any) {
            toasts.addErrorToast({ title: $t("tests.schedule.save-failed"), message: e.message });
        } finally {
            saving = false;
        }
    }
</script>

<svelte:head>
    <title>
        {testName} - {$t("tests.schedule.title")} - {data.domain.domain} - happyDomain
    </title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2>
            <span class="font-monospace">{data.domain.domain}</span>
            &ndash;
            {pluginName}
            &ndash; {$t("tests.schedule.title")}
        </h2>
        <div class="d-flex gap-2">
            <Button
                color="info"
                href={`/domains/${encodeURIComponent(data.domain.domain)}/tests/${encodeURIComponent(testName)}/results`}
            >
                <Icon name="bar-chart-fill"></Icon>
                {$t("tests.list.view-results")}
            </Button>
        </div>
    </div>

    {#if loading}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("tests.list.loading")}</p>
        </div>
    {:else if loadError}
        <Card body color="danger">
            <p class="mb-0">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t("tests.list.error-loading", { error: loadError })}
            </p>
        </Card>
    {:else if !test}
        <Card body>
            <p class="text-center text-muted mb-0">
                <Icon name="info-circle"></Icon>
                {$t("tests.list.no-tests")}
            </p>
        </Card>
    {:else}
        <Card class="mb-4">
            <CardHeader>
                <h4 class="mb-0">
                    <Icon name="clock-history"></Icon>
                    {$t("tests.schedule.card-title")}
                </h4>
            </CardHeader>
            <CardBody>
                <div class="mb-4">
                    <div class="form-check form-switch">
                        <input
                            class="form-check-input"
                            type="checkbox"
                            role="switch"
                            id="schedule-enabled"
                            bind:checked={formEnabled}
                            disabled={saving}
                        />
                        <label class="form-check-label" for="schedule-enabled">
                            {#if formEnabled}
                                <Badge color="success">{$t("tests.schedule.auto-enabled")}</Badge>
                            {:else}
                                <Badge color="secondary">{$t("tests.schedule.auto-disabled")}</Badge
                                >
                            {/if}
                        </label>
                    </div>
                </div>

                {#if formEnabled}
                    <div class="mb-4">
                        <label for="schedule-interval" class="form-label fw-semibold">
                            {$t("tests.schedule.interval-label")}
                        </label>
                        <div class="input-group" style="max-width: 300px;">
                            <Input
                                type="number"
                                id="schedule-interval"
                                min={1}
                                step={1}
                                bind:value={formIntervalHours}
                                disabled={saving}
                            />
                            <span class="input-group-text">
                                {$t("tests.schedule.hours")}
                            </span>
                        </div>
                        <div class="form-text">
                            {$t("tests.schedule.interval-hint")}
                        </div>
                    </div>
                {/if}

                {#if test.schedule}
                    <div class="mb-4">
                        <div class="row g-3">
                            {#if test.schedule.last_run}
                                <div class="col-auto">
                                    <span class="text-muted fw-semibold">
                                        {$t("tests.schedule.last-run")}:
                                    </span>
                                    <span>
                                        {formatTestDate(test.schedule.last_run, "medium", $t)}
                                        <small class="text-muted">
                                            ({formatRelative(test.schedule.last_run, $t)})
                                        </small>
                                    </span>
                                </div>
                            {/if}
                            {#if test.enabled && test.schedule.next_run}
                                <div class="col-auto">
                                    <span class="text-muted fw-semibold">
                                        {$t("tests.schedule.next-run")}:
                                    </span>
                                    <span>
                                        {formatTestDate(test.schedule.next_run, "medium", $t)}
                                        <small class="text-muted">
                                            ({formatRelative(test.schedule.next_run, $t)})
                                        </small>
                                    </span>
                                </div>
                            {/if}
                        </div>
                    </div>
                {:else}
                    <p class="text-muted">
                        <Icon name="info-circle"></Icon>
                        {$t("tests.schedule.no-schedule-yet")}
                    </p>
                {/if}

                <Button color="primary" disabled={saving} onclick={handleSave}>
                    {#if saving}
                        <Spinner size="sm" class="me-1" />
                    {/if}
                    <Icon name="check-lg"></Icon>
                    {$t("tests.schedule.save")}
                </Button>
            </CardBody>
        </Card>
    {/if}
</div>
