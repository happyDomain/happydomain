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
        Card,
        CardBody,
        CardHeader,
        Icon,
        Table,
    } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import type { CheckerScope } from "$lib/api/checkers";
    import { listScopedCheckers } from "$lib/api/checkers";
    import { checkers } from "$lib/stores/checkers";
    import type { CheckerCheckerDefinition, HappydnsCheckerStatus } from "$lib/api-base/types.gen";
    import { getStatusColor, getStatusI18nKey, formatCheckDate } from "$lib/utils";
    import CheckersAvailabilityTable from "./CheckersAvailabilityTable.svelte";
    import PageTitle from "$lib/components/PageTitle.svelte";

    interface Props {
        scope: CheckerScope;
        checksBase: string;
        title: string;
        domainName: string;
        filterAvailability: "applyToDomain" | "applyToService";
    }

    let { scope, checksBase, title, domainName, filterAvailability }: Props = $props();

    let checkersPromise = $derived(listScopedCheckers(scope));

    function getConfiguredCheckerIds(statuses: HappydnsCheckerStatus[]): Set<string> {
        return new Set(statuses.map((s) => s.id).filter((id): id is string => !!id));
    }

    function getUnconfiguredCheckers(configuredIds: Set<string>): [string, CheckerCheckerDefinition][] {
        if (!$checkers) return [];
        return Object.entries($checkers).filter(
            ([id, def]) => !configuredIds.has(id) && def.availability?.[filterAvailability],
        );
    }
</script>

<svelte:head>
    <title>{$t("checkers.list.title")}{domainName} - happyDomain</title>
</svelte:head>

<div class="flex-fill mt-1 mb-5">
    <PageTitle title={title} domain={domainName}></PageTitle>

    {#await checkersPromise}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.list.loading")}
            </p>
        </Card>
    {:then checkerStatuses}
        {#if checkerStatuses.length > 0}
            <div class="table-responsive">
                <Table hover class="mb-0">
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
                        {#each checkerStatuses as checker}
                            {@const status = checker.latestExecution?.result?.status}
                            <tr>
                                <td>
                                    <strong>{checker.name || checker.id}</strong>
                                </td>
                                <td>
                                    {#if checker.latestExecution}
                                        <Badge color={getStatusColor(status)}>
                                            {$t(getStatusI18nKey(status))}
                                        </Badge>
                                    {:else}
                                        <Badge color="secondary">
                                            {$t("checkers.status.not-run")}
                                        </Badge>
                                    {/if}
                                </td>
                                <td>
                                    {#if checker.latestExecution?.startedAt}
                                        {formatCheckDate(checker.latestExecution.startedAt)}
                                    {:else}
                                        {$t("checkers.never")}
                                    {/if}
                                </td>
                                <td>
                                    {#if checker.enabled}
                                        <Badge color="success">
                                            {$t("checkers.list.schedule.enabled")}
                                        </Badge>
                                    {:else}
                                        <Badge color="secondary">
                                            {$t("checkers.list.schedule.disabled")}
                                        </Badge>
                                    {/if}
                                </td>
                                <td>
                                    <div class="d-flex gap-1">
                                        <a
                                            href="{checksBase}/{checker.id}"
                                            class="btn btn-sm btn-outline-primary"
                                        >
                                            {$t("checkers.list.configure")}
                                        </a>
                                        <a
                                            href="{checksBase}/{checker.id}/executions"
                                            class="btn btn-sm btn-outline-secondary"
                                        >
                                            {$t("checkers.list.view-results")}
                                        </a>
                                    </div>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </Table>
            </div>
        {:else}
            <Alert color="info" class="mb-4">
                <Icon name="info-circle" />
                {$t("checkers.list.no-checks")}
            </Alert>
        {/if}

        {@const configuredIds = getConfiguredCheckerIds(checkerStatuses)}
        {@const unconfigured = getUnconfiguredCheckers(configuredIds)}
        {#if unconfigured.length > 0}
            <Card>
                <CardHeader>
                    <strong>{$t("checkers.other-checkers.title")}</strong>
                </CardHeader>
                <CardBody>
                    <p class="text-muted">{$t("checkers.other-checkers.description")}</p>
                    <CheckersAvailabilityTable
                        checkers={unconfigured}
                        basePath={checksBase}
                    />
                </CardBody>
            </Card>
        {/if}
    {:catch error}
        <Alert color="danger">
            <Icon name="exclamation-triangle-fill" />
            {$t("checkers.list.error-loading", { error: error.message })}
        </Alert>
    {/await}
</div>
