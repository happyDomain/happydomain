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
    import { listAvailableChecks, updateCheckSchedule, createCheckSchedule } from "$lib/api/checks";
    import type { Domain } from "$lib/model/domain";
    import { CheckScopeType, type AvailableCheck } from "$lib/model/check";
    import { checks } from "$lib/stores/checks";
    import { toasts } from "$lib/stores/toasts";
    import { formatCheckDate, formatRelative } from "$lib/utils";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    const checkName = $derived(page.params.cname || "");
    const checkDisplayName = $derived($checks?.[checkName]?.name || checkName);

    // Resolved check data
    let check = $state<AvailableCheck | null>(null);
    let loading = $state(true);
    let loadError = $state<string | null>(null);

    // Form state
    let formEnabled = $state(true);
    let formIntervalHours = $state(24);
    let saving = $state(false);

    async function loadCheck() {
        loading = true;
        loadError = null;
        try {
            const checks = await listAvailableChecks(data.domain.id);
            const found = checks?.find((c) => c.checker_name === checkName) ?? null;
            check = found;
            if (found) {
                formEnabled = found.enabled;
                formIntervalHours =
                    found.schedule && found.schedule.interval !== undefined && found.schedule.interval > 0
                        ? found.schedule.interval / (3600 * 1e9)
                        : 24;
            }
        } catch (e: any) {
            loadError = e.message;
        } finally {
            loading = false;
        }
    }

    loadCheck();

    async function handleSave() {
        if (!check) return;
        saving = true;

        try {
            const intervalNs = Math.max(formIntervalHours, 1) * 3600 * 1e9;

            if (check.schedule) {
                await updateCheckSchedule(check.schedule.id!, {
                    ...check.schedule,
                    enabled: formEnabled,
                    interval: intervalNs,
                });
            } else {
                await createCheckSchedule({
                    checker_name: check.checker_name,
                    target_type: CheckScopeType.CheckScopeDomain,
                    target_id: data.domain.id,
                    interval: intervalNs,
                    enabled: formEnabled,
                });
            }

            toasts.addToast({ title: $t("checks.schedule.saved"), type: "success", timeout: 3000 });
            await loadCheck();
        } catch (e: any) {
            toasts.addErrorToast({ title: $t("checks.schedule.save-failed"), message: e.message });
        } finally {
            saving = false;
        }
    }
</script>

<svelte:head>
    <title>
        {checkName} - {$t("checks.schedule.title")} - {data.domain.domain} - happyDomain
    </title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2>
            <span class="font-monospace">{data.domain.domain}</span>
            &ndash;
            {checkDisplayName}
            &ndash; {$t("checks.schedule.title")}
        </h2>
        <div class="d-flex gap-2">
            <Button
                color="info"
                href={`/domains/${encodeURIComponent(data.domain.domain)}/checks/${encodeURIComponent(checkName)}/results`}
            >
                <Icon name="bar-chart-fill"></Icon>
                {$t("checks.list.view-results")}
            </Button>
        </div>
    </div>

    {#if loading}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("checks.list.loading")}</p>
        </div>
    {:else if loadError}
        <Card body color="danger">
            <p class="mb-0">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t("checks.list.error-loading", { error: loadError })}
            </p>
        </Card>
    {:else if !check}
        <Card body>
            <p class="text-center text-muted mb-0">
                <Icon name="info-circle"></Icon>
                {$t("checks.list.no-checks")}
            </p>
        </Card>
    {:else}
        <Card class="mb-4">
            <CardHeader>
                <h4 class="mb-0">
                    <Icon name="clock-history"></Icon>
                    {$t("checks.schedule.card-title")}
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
                                <Badge color="success">{$t("checks.schedule.auto-enabled")}</Badge>
                            {:else}
                                <Badge color="secondary">{$t("checks.schedule.auto-disabled")}</Badge
                                >
                            {/if}
                        </label>
                    </div>
                </div>

                {#if formEnabled}
                    <div class="mb-4">
                        <label for="schedule-interval" class="form-label fw-semibold">
                            {$t("checks.schedule.interval-label")}
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
                                {$t("checks.schedule.hours")}
                            </span>
                        </div>
                        <div class="form-text">
                            {$t("checks.schedule.interval-hint")}
                        </div>
                    </div>
                {/if}

                {#if check.schedule}
                    <div class="mb-4">
                        <div class="row g-3">
                            {#if check.schedule.last_run}
                                <div class="col-auto">
                                    <span class="text-muted fw-semibold">
                                        {$t("checks.schedule.last-run")}:
                                    </span>
                                    <span>
                                        {formatCheckDate(check.schedule.last_run, "medium", $t)}
                                        <small class="text-muted">
                                            ({formatRelative(check.schedule.last_run, $t)})
                                        </small>
                                    </span>
                                </div>
                            {/if}
                            {#if check.enabled && check.schedule.next_run}
                                <div class="col-auto">
                                    <span class="text-muted fw-semibold">
                                        {$t("checks.schedule.next-run")}:
                                    </span>
                                    <span>
                                        {formatCheckDate(check.schedule.next_run, "medium", $t)}
                                        <small class="text-muted">
                                            ({formatRelative(check.schedule.next_run, $t)})
                                        </small>
                                    </span>
                                </div>
                            {/if}
                        </div>
                    </div>
                {:else}
                    <p class="text-muted">
                        <Icon name="info-circle"></Icon>
                        {$t("checks.schedule.no-schedule-yet")}
                    </p>
                {/if}

                <Button color="primary" disabled={saving} onclick={handleSave}>
                    {#if saving}
                        <Spinner size="sm" class="me-1" />
                    {/if}
                    <Icon name="check-lg"></Icon>
                    {$t("checks.schedule.save")}
                </Button>
            </CardBody>
        </Card>
    {/if}
</div>
