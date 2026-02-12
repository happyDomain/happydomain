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
    import { navigate } from "$lib/stores/config";
    import { page } from "$app/state";
    import { Card, Icon, Table, Badge, Button, Spinner } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { listAvailableChecks, updateCheckSchedule, createCheckSchedule } from "$lib/api/checks";
    import type { Domain } from "$lib/model/domain";
    import { CheckScopeType, type AvailableCheck } from "$lib/model/check";
    import { checks } from "$lib/stores/checks";
    import { toasts } from "$lib/stores/toasts";
    import RunCheckModal from "$lib/components/modals/RunCheckModal.svelte";
    import { getStatusColor, getStatusKey, formatCheckDate } from "$lib/utils";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    let checksPromise = $derived(listAvailableChecks(data.domain.id));
    let runCheckModal: RunCheckModal;
    let togglingChecks = $state(new Set<string>());

    function handleCheckTriggered(_: string, checkName: string) {
        // Refresh the check list to show updated status
        checksPromise = listAvailableChecks(data.domain.id);
        navigate(`/domains/${page.params.dn!}/checks/${checkName}/results`);
    }

    async function handleToggleEnabled(check: AvailableCheck) {
        const next = new Set(togglingChecks);
        next.add(check.checker_name);
        togglingChecks = next;

        try {
            const newEnabled = !check.enabled;
            if (check.schedule) {
                await updateCheckSchedule(check.schedule.id!, {
                    ...check.schedule,
                    enabled: newEnabled,
                });
            } else {
                // No schedule record yet — create one to persist the disabled state.
                // (Enabled → Enabled needs no action since that's the implicit default.)
                await createCheckSchedule({
                    checker_name: check.checker_name,
                    target_type: CheckScopeType.CheckScopeDomain,
                    target_id: data.domain.id,
                    interval: 0,
                    enabled: newEnabled,
                });
            }
            checksPromise = listAvailableChecks(data.domain.id);
        } catch (e: any) {
            toasts.addErrorToast({ title: $t("checks.list.error-loading", { error: e.message }) });
        } finally {
            const after = new Set(togglingChecks);
            after.delete(check.checker_name);
            togglingChecks = after;
        }
    }
</script>

<svelte:head>
    <title>Checks - {data.domain.domain} - happyDomain</title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <h2>
        {$t("checks.list.title")}<span class="font-monospace">{data.domain.domain}</span>
    </h2>

    {#await checksPromise}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("checks.list.loading")}</p>
        </div>
    {:then availableChecks}
        {#if !$checks}
            <div class="mt-5 text-center flex-fill">
                <Spinner />
                <p>{$t("checks.list.loading-checks")}</p>
            </div>
        {:else if !availableChecks || availableChecks.length === 0}
            <Card body class="mt-3">
                <p class="text-center text-muted mb-0">
                    <Icon name="info-circle"></Icon>
                    {$t("checks.list.no-checks")}
                </p>
            </Card>
        {:else}
            <Table hover striped class="mt-3">
                <thead>
                    <tr>
                        <th>{$t("checks.list.table.checker")}</th>
                        <th>{$t("checks.list.table.status")}</th>
                        <th>{$t("checks.list.table.last-run")}</th>
                        <th>{$t("checks.list.table.schedule")}</th>
                        <th>{$t("checks.list.table.actions")}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each availableChecks as check}
                        {@const checkInfo = $checks[check.checker_name]}
                        <tr>
                            <td class="align-middle">
                                <strong>{checkInfo?.name || check.checker_name}</strong>
                            </td>
                            <td class="align-middle text-center">
                                {#if check.last_result !== undefined}
                                    <Badge color={getStatusColor(check.last_result.status)}>
                                        {$t(getStatusKey(check.last_result.status))}
                                    </Badge>
                                {:else}
                                    <Badge color="secondary">{$t("checks.status.not-run")}</Badge>
                                {/if}
                            </td>
                            <td class="align-middle">
                                {formatCheckDate(check.last_result?.executed_at, "short", $t)}
                            </td>
                            <td class="align-middle">
                                <div class="form-check form-switch mb-0">
                                    <input
                                        class="form-check-input"
                                        type="checkbox"
                                        role="switch"
                                        id="toggle-{check.checker_name}"
                                        checked={check.enabled}
                                        disabled={togglingChecks.has(check.checker_name)}
                                        onchange={() => handleToggleEnabled(check)}
                                    />
                                    <label
                                        class="form-check-label small"
                                        for="toggle-{check.checker_name}"
                                    >
                                        {check.enabled
                                            ? $t("checks.list.schedule.enabled")
                                            : $t("checks.list.schedule.disabled")}
                                    </label>
                                </div>
                            </td>
                            <td class="align-middle">
                                <div class="d-flex gap-2">
                                    <Button
                                        size="sm"
                                        color="primary"
                                        onclick={() =>
                                            runCheckModal.open(
                                                check.checker_name,
                                                checkInfo?.name || check.checker_name,
                                            )}
                                    >
                                        <Icon name="play-fill"></Icon>
                                        {$t("checks.list.run-check")}
                                    </Button>
                                    <Button
                                        size="sm"
                                        color="info"
                                        href={`/domains/${encodeURIComponent(data.domain.domain)}/checks/${encodeURIComponent(check.checker_name)}/results`}
                                    >
                                        <Icon name="bar-chart-fill"></Icon>
                                        {$t("checks.list.view-results")}
                                    </Button>
                                    <Button
                                        size="sm"
                                        color="dark"
                                        href={`/domains/${encodeURIComponent(data.domain.domain)}/checks/${encodeURIComponent(check.checker_name)}`}
                                        title={$t("checks.list.configure")}
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
                {$t("checks.list.error-loading", { error: error.message })}
            </p>
        </Card>
    {/await}
</div>

<RunCheckModal
    domainId={data.domain.id}
    onCheckTriggered={handleCheckTriggered}
    bind:this={runCheckModal}
/>
