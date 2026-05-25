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
    import { Alert, Badge, Card, Icon, Table } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { listScopedCheckers, type CheckerScope } from "$lib/api/checkers";
    import type { HappydnsCheckerStatus } from "$lib/api-base/types.gen";
    import { domainLink } from "$lib/stores/domains";
    import { thisZone } from "$lib/stores/thiszone";
    import { fqdn } from "$lib/dns";
    import { formatCheckDate, getStatusColor, getStatusI18nKey } from "$lib/utils";
    import PageTitle from "$lib/components/PageTitle.svelte";

    interface ServiceTarget {
        subdomain: string;
        serviceId: string;
        serviceLabel: string;
    }

    interface Section {
        title: string;
        checkersBase: string;
        statuses: HappydnsCheckerStatus[];
    }

    interface Props {
        domainId: string;
        domainName: string;
        title: string;
        /** When set, dashboard only lists this single service's checkers. */
        serviceTarget?: ServiceTarget & { zoneId: string };
    }

    let { domainId, domainName, title, serviceTarget }: Props = $props();

    let domainBase = $derived(`/domains/${domainLink(domainId)}`);

    function serviceBase(zoneId: string, subdomain: string, serviceId: string): string {
        return `${domainBase}/${encodeURIComponent(zoneId)}/${encodeURIComponent(subdomain)}/${encodeURIComponent(serviceId)}/checkers`;
    }

    function serviceLabel(svc: { _id?: string; _svctype?: string }): string {
        return svc._svctype || svc._id || "service";
    }

    async function loadSections(zone: typeof $thisZone): Promise<Section[]> {
        const sections: Section[] = [];

        if (serviceTarget) {
            const scope: CheckerScope = {
                domainId,
                zoneId: serviceTarget.zoneId,
                subdomain: serviceTarget.subdomain,
                serviceId: serviceTarget.serviceId,
            };
            const statuses = await listScopedCheckers(scope);
            sections.push({
                title: serviceTarget.serviceLabel,
                checkersBase: serviceBase(serviceTarget.zoneId, serviceTarget.subdomain, serviceTarget.serviceId),
                statuses,
            });
            return sections;
        }

        const domainStatuses = await listScopedCheckers({ domainId });
        sections.push({
            title: domainName,
            checkersBase: `${domainBase}/checkers`,
            statuses: domainStatuses,
        });

        if (!zone) return sections;

        const targets: Array<{ subdomain: string; serviceId: string; label: string }> = [];
        for (const [subdomain, services] of Object.entries(zone.services ?? {})) {
            for (const svc of services ?? []) {
                if (!svc._id) continue;
                targets.push({
                    subdomain: subdomain === "" ? "@" : subdomain,
                    serviceId: svc._id,
                    label: `${serviceLabel(svc)} • ${fqdn(subdomain, domainName)}`,
                });
            }
        }

        const serviceLists = await Promise.all(
            targets.map((tg) =>
                listScopedCheckers({
                    domainId,
                    zoneId: zone.id!,
                    subdomain: tg.subdomain,
                    serviceId: tg.serviceId,
                }).catch(() => [] as HappydnsCheckerStatus[]),
            ),
        );

        targets.forEach((tg, i) => {
            const statuses = serviceLists[i];
            if (statuses.length === 0) return;
            sections.push({
                title: tg.label,
                checkersBase: serviceBase(zone.id!, tg.subdomain, tg.serviceId),
                statuses,
            });
        });

        return sections;
    }

    let sectionsPromise = $derived(loadSections($thisZone));
</script>

<svelte:head>
    <title>{title} - happyDomain</title>
</svelte:head>

<div class="flex-fill mt-1 mb-5">
    <PageTitle {title} domain={domainName} />

    {#await sectionsPromise}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.list.loading")}
            </p>
        </Card>
    {:then sections}
        {#if sections.every((s) => s.statuses.length === 0)}
            <Alert color="info">
                <Icon name="info-circle" />
                {$t("checkers.list.no-checks")}
            </Alert>
        {:else}
            {#each sections as section}
                {#if section.statuses.length > 0}
                    <h4 class="mt-4">{section.title}</h4>
                    <div class="table-responsive">
                        <Table hover class="mb-0">
                            <thead>
                                <tr>
                                    <th>{$t("checkers.list.table.checker")}</th>
                                    <th>{$t("checkers.list.table.status")}</th>
                                    <th>{$t("checkers.list.table.last-run")}</th>
                                    <th class="text-end">{$t("checkers.list.table.actions")}</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each section.statuses as checker}
                                    {@const exec = checker.latestExecution}
                                    {@const status = exec?.result?.status}
                                    <tr>
                                        <td><strong>{checker.name || checker.id}</strong></td>
                                        <td>
                                            {#if exec}
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
                                            {#if exec?.startedAt}
                                                {formatCheckDate(exec.startedAt)}
                                            {:else}
                                                {$t("checkers.never")}
                                            {/if}
                                        </td>
                                        <td class="text-end">
                                            <div class="btn-group btn-group-sm" role="group">
                                                {#if exec?.id}
                                                    <a
                                                        href="{section.checkersBase}/{checker.id}/executions/{exec.id}"
                                                        class="btn btn-outline-primary"
                                                    >
                                                        {$t("checkers.list.view-results")}
                                                    </a>
                                                {/if}
                                                <a
                                                    href="{section.checkersBase}/{checker.id}/executions"
                                                    class="btn btn-outline-secondary"
                                                >
                                                    {$t("checkers.list.history")}
                                                </a>
                                                <a
                                                    href="{section.checkersBase}/{checker.id}"
                                                    class="btn btn-outline-secondary"
                                                >
                                                    {$t("checkers.list.configure")}
                                                </a>
                                            </div>
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </Table>
                    </div>
                {/if}
            {/each}
        {/if}
    {:catch error}
        <Alert color="danger">
            <Icon name="exclamation-triangle-fill" />
            {$t("checkers.list.error-loading", { error: error.message })}
        </Alert>
    {/await}
</div>
