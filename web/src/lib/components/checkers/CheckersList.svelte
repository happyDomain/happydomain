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
    import { Card, Icon, Table, Badge, Button, Spinner } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { updateCheckSchedule, createCheckSchedule } from "$lib/api/checkers";
    import { CheckScopeType, type AvailableChecker } from "$lib/model/checker";
    import { checkers } from "$lib/stores/checkers";
    import { toasts } from "$lib/stores/toasts";
    import RunCheckModal from "$lib/components/modals/RunCheckModal.svelte";
    import { getStatusColor, getStatusKey, formatCheckDate } from "$lib/utils";

    interface Props {
        fetchCheckers: () => Promise<AvailableChecker[]>;
        basePath: string;
        domainId: string;
        zoneId?: string;
        subdomain?: string;
        serviceid?: string;
        targetType: CheckScopeType;
        targetId: string;
        noChecksKey?: string;
    }

    let {
        fetchCheckers,
        basePath,
        domainId,
        zoneId,
        subdomain,
        serviceid,
        targetType,
        targetId,
        noChecksKey = "checkers.list.no-checks",
    }: Props = $props();

    let checksPromise = $state(fetchCheckers());
    let runCheckModal: RunCheckModal;
    let togglingChecks = $state(new Set<string>());

    function handleCheckTriggered(_: string, checkName: string) {
        checksPromise = fetchCheckers();
        navigate(`${basePath}/${encodeURIComponent(checkName)}/results`);
    }

    async function handleToggleEnabled(checker: AvailableChecker) {
        const next = new Set(togglingChecks);
        next.add(checker.checker_name);
        togglingChecks = next;

        try {
            const newEnabled = !checker.enabled;
            if (checker.schedule) {
                await updateCheckSchedule(checker.schedule.id!, {
                    ...checker.schedule,
                    enabled: newEnabled,
                });
            } else {
                await createCheckSchedule({
                    checker_name: checker.checker_name,
                    target_type: targetType,
                    target_id: targetId,
                    interval: 0,
                    enabled: newEnabled,
                });
            }
            checksPromise = fetchCheckers();
        } catch (e: any) {
            toasts.addErrorToast({
                title: $t("checkers.list.error-loading", { error: e.message }),
            });
        } finally {
            const after = new Set(togglingChecks);
            after.delete(checker.checker_name);
            togglingChecks = after;
        }
    }
</script>

{#await checksPromise}
    <div class="mt-5 text-center flex-fill">
        <Spinner />
        <p>{$t("checkers.list.loading")}</p>
    </div>
{:then availableCheckers}
    {#if !$checkers}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("checkers.list.loading-checks")}</p>
        </div>
    {:else if !availableCheckers || availableCheckers.length === 0}
        <Card body class="mt-3">
            <p class="text-center text-muted mb-0">
                <Icon name="info-circle"></Icon>
                {$t(noChecksKey)}
            </p>
        </Card>
    {:else}
        <Table hover striped class="mt-3">
            <thead>
                <tr>
                    <th>{$t("checkers.list.table.checker")}</th>
                    <th>{$t("checkers.list.table.status")}</th>
                    <th>{$t("checkers.list.table.last-run")}</th>
                    <th>{$t("checkers.list.table.schedule")}</th>
                    <th>{$t("checkers.list.table.actions")}</th>
                </tr>
            </thead>
            <tbody>
                {#each availableCheckers as checker}
                    {@const checkInfo = $checkers[checker.checker_name]}
                    <tr>
                        <td class="align-middle">
                            <strong>{checkInfo?.name || checker.checker_name}</strong>
                        </td>
                        <td class="align-middle text-center">
                            {#if checker.last_result !== undefined}
                                <Badge color={getStatusColor(checker.last_result.status)}>
                                    {$t(getStatusKey(checker.last_result.status))}
                                </Badge>
                            {:else}
                                <Badge color="secondary">{$t("checkers.status.not-run")}</Badge>
                            {/if}
                        </td>
                        <td class="align-middle">
                            {formatCheckDate(checker.last_result?.executed_at, "short", $t)}
                        </td>
                        <td class="align-middle">
                            <div class="form-check form-switch mb-0">
                                <input
                                    class="form-check-input"
                                    type="checkbox"
                                    role="switch"
                                    id="toggle-{checker.checker_name}"
                                    checked={checker.enabled}
                                    disabled={togglingChecks.has(checker.checker_name)}
                                    onchange={() => handleToggleEnabled(checker)}
                                />
                                <label
                                    class="form-check-label small"
                                    for="toggle-{checker.checker_name}"
                                >
                                    {checker.enabled
                                        ? $t("checkers.list.schedule.enabled")
                                        : $t("checkers.list.schedule.disabled")}
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
                                            checker.checker_name,
                                            checkInfo?.name || checker.checker_name,
                                        )}
                                >
                                    <Icon name="play-fill"></Icon>
                                    {$t("checkers.list.run-check")}
                                </Button>
                                <Button
                                    size="sm"
                                    color="info"
                                    href={`${basePath}/${encodeURIComponent(checker.checker_name)}/results`}
                                >
                                    <Icon name="bar-chart-fill"></Icon>
                                    {$t("checkers.list.view-results")}
                                </Button>
                                <Button
                                    size="sm"
                                    color="dark"
                                    href={`${basePath}/${encodeURIComponent(checker.checker_name)}`}
                                    title={$t("checkers.list.configure")}
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
            {$t("checkers.list.error-loading", { error: error.message })}
        </p>
    </Card>
{/await}

<RunCheckModal
    {domainId}
    {zoneId}
    {subdomain}
    {serviceid}
    onCheckTriggered={handleCheckTriggered}
    bind:this={runCheckModal}
/>
