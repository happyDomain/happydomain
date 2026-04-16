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
    import { Alert, Badge, Button, Card, Icon, Table } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { base } from "$lib/stores/config";
    import { toasts } from "$lib/stores/toasts";
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

    let metricsApiUrl = $derived(
        scope.zoneId && scope.subdomain !== undefined && scope.serviceId
            ? `${base}/api/domains/${scope.domainId}/zone/${scope.zoneId}/${scope.subdomain}/services/${scope.serviceId}/checkers/metrics`
            : `${base}/api/domains/${scope.domainId}/checkers/metrics`
    );

    async function copyMetricsUrl() {
        try {
            await navigator.clipboard.writeText(metricsApiUrl);
            toasts.addToast({
                message: $t("checkers.list.prometheus-metrics-copied"),
                type: "success",
                timeout: 3000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.list.prometheus-metrics-copy-failed", { error: String(error) }),
                timeout: 5000,
            });
        }
    }

    function getConfiguredCheckerIds(statuses: HappydnsCheckerStatus[]): Set<string> {
        return new Set(statuses.map((s) => s.id).filter((id): id is string => !!id));
    }

    function getUnconfiguredCheckers(configuredIds: Set<string>): [string, CheckerCheckerDefinition][] {
        if (!$checkers) return [];
        return Object.entries($checkers).filter(
            ([id, def]) => !configuredIds.has(id) && def.availability?.[filterAvailability],
        );
    }

    function getChildrenCheckers(configuredIds: Set<string>): [string, CheckerCheckerDefinition][] {
        if (!$checkers) return [];
        return Object.entries($checkers).filter(
            ([id, def]) =>
                !configuredIds.has(id) &&
                !def.availability?.[filterAvailability] &&
                (def.availability?.applyToZone || def.availability?.applyToService),
        );
    }
</script>

<svelte:head>
    <title>{$t("checkers.list.title")}{domainName} - happyDomain</title>
</svelte:head>

<div class="flex-fill mt-1 mb-5">
    <PageTitle {title} domain={domainName}>
        <Button
            color="secondary"
            outline
            onclick={copyMetricsUrl}
            title={metricsApiUrl}
        >
            <Icon name="graph-up-arrow"></Icon>
            {$t("checkers.list.prometheus-metrics")}
        </Button>
    </PageTitle>

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
            <h4 class="mt-4">{$t("checkers.other-checkers.title")}</h4>
            <p class="text-muted">{$t("checkers.other-checkers.description")}</p>
            <CheckersAvailabilityTable checkers={unconfigured} basePath={checksBase} />
        {/if}

        {@const children = getChildrenCheckers(configuredIds)}
        {#if children.length > 0}
            <h4 class="mt-4">{$t("checkers.children-checkers.title")}</h4>
            <p class="text-muted">{$t("checkers.children-checkers.description")}</p>
            <CheckersAvailabilityTable checkers={children} basePath={checksBase} />
        {/if}
    {:catch error}
        <Alert color="danger">
            <Icon name="exclamation-triangle-fill" />
            {$t("checkers.list.error-loading", { error: error.message })}
        </Alert>
    {/await}
</div>
