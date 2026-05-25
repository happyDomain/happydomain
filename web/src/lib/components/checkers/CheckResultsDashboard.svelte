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
    import { Alert, Card, Icon } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { listScopedCheckers, type CheckerScope } from "$lib/api/checkers";
    import type { HappydnsCheckerStatus, HappydnsStatus } from "$lib/api-base/types.gen";
    import { domainLink } from "$lib/stores/domains";
    import { thisZone } from "$lib/stores/thiszone";
    import { fqdn } from "$lib/dns";
    import { StatusCrit, StatusError, StatusWarn } from "$lib/utils/checkers";
    import CheckCard from "./CheckCard.svelte";
    import PageTitle from "$lib/components/PageTitle.svelte";

    interface ServiceTarget {
        subdomain: string;
        serviceId: string;
        serviceLabel: string;
    }

    interface CheckerItem {
        status: HappydnsCheckerStatus;
        checkersBase: string;
        scope: CheckerScope;
        chip?: string;
    }

    interface Section {
        title: string;
        sortKey: string;
        items: CheckerItem[];
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

    // Severity weight: bigger means more important. not-run lands at the bottom.
    function severityWeight(status: HappydnsCheckerStatus): number {
        const code = status.latestExecution?.result?.status;
        if (code === undefined || code === null) return -1;
        return code as number;
    }

    function sortItemsBySeverityThenName(items: CheckerItem[]): CheckerItem[] {
        return [...items].sort((a, b) => {
            const diff = severityWeight(b.status) - severityWeight(a.status);
            if (diff !== 0) return diff;
            const an = a.status.name || a.status.id || "";
            const bn = b.status.name || b.status.id || "";
            return an.localeCompare(bn);
        });
    }

    function countCritical(sections: Section[]): number {
        let n = 0;
        for (const s of sections) {
            for (const c of s.items) {
                const code = c.status.latestExecution?.result?.status;
                if (code === StatusCrit || code === StatusError) n++;
            }
        }
        return n;
    }

    function countWarn(sections: Section[]): number {
        let n = 0;
        for (const s of sections) {
            for (const c of s.items) {
                if (c.status.latestExecution?.result?.status === StatusWarn) n++;
            }
        }
        return n;
    }

    function subdomainTitle(subdomain: string): string {
        return fqdn(subdomain === "@" ? "" : subdomain, domainName);
    }

    // "@" (root) first, then alphabetical by subdomain label.
    function subdomainSortKey(subdomain: string): string {
        return subdomain === "@" ? " " : subdomain;
    }

    async function loadSections(zone: typeof $thisZone): Promise<Section[]> {
        if (serviceTarget) {
            const scope: CheckerScope = {
                domainId,
                zoneId: serviceTarget.zoneId,
                subdomain: serviceTarget.subdomain,
                serviceId: serviceTarget.serviceId,
            };
            const statuses = await listScopedCheckers(scope);
            const base = serviceBase(
                serviceTarget.zoneId,
                serviceTarget.subdomain,
                serviceTarget.serviceId,
            );
            const items: CheckerItem[] = statuses.map((s) => ({
                status: s,
                checkersBase: base,
                scope,
                chip: serviceTarget.serviceLabel,
            }));
            return [
                {
                    title: subdomainTitle(serviceTarget.subdomain),
                    sortKey: subdomainSortKey(serviceTarget.subdomain),
                    items: sortItemsBySeverityThenName(items),
                },
            ];
        }

        // groupKey -> Section; key is the subdomain, "@" for the apex.
        const grouped = new Map<string, Section>();

        const ensureSection = (subdomain: string): Section => {
            let s = grouped.get(subdomain);
            if (!s) {
                s = {
                    title: subdomainTitle(subdomain),
                    sortKey: subdomainSortKey(subdomain),
                    items: [],
                };
                grouped.set(subdomain, s);
            }
            return s;
        };

        const domainStatuses = await listScopedCheckers({ domainId });
        if (domainStatuses.length > 0) {
            const sec = ensureSection("@");
            for (const s of domainStatuses) {
                sec.items.push({
                    status: s,
                    checkersBase: `${domainBase}/checkers`,
                    scope: { domainId },
                });
            }
        }

        if (zone) {
            const targets: Array<{ subdomain: string; serviceId: string; serviceName: string }> =
                [];
            for (const [subdomain, services] of Object.entries(zone.services ?? {})) {
                for (const svc of services ?? []) {
                    if (!svc._id) continue;
                    targets.push({
                        subdomain: subdomain === "" ? "@" : subdomain,
                        serviceId: svc._id,
                        serviceName: serviceLabel(svc),
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
                const sec = ensureSection(tg.subdomain);
                const base = serviceBase(zone.id!, tg.subdomain, tg.serviceId);
                const scope: CheckerScope = {
                    domainId,
                    zoneId: zone.id!,
                    subdomain: tg.subdomain,
                    serviceId: tg.serviceId,
                };
                for (const s of statuses) {
                    sec.items.push({
                        status: s,
                        checkersBase: base,
                        scope,
                        chip: tg.serviceName,
                    });
                }
            });
        }

        const sections = Array.from(grouped.values());
        for (const s of sections) s.items = sortItemsBySeverityThenName(s.items);
        sections.sort((a, b) => a.sortKey.localeCompare(b.sortKey));
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
        {#if sections.every((s) => s.items.length === 0)}
            <Alert color="info">
                <Icon name="info-circle" />
                {$t("checkers.list.no-checks")}
            </Alert>
        {:else}
            {@const critCount = countCritical(sections)}
            {@const warnCount = countWarn(sections)}
            {#if critCount > 0 || warnCount > 0}
                <p class="check-summary mb-3">
                    {#if critCount > 0}
                        <span>{critCount} {$t("checkers.dashboard.critical")}</span>
                    {/if}
                    {#if critCount > 0 && warnCount > 0}
                        <span class="check-summary-sep">·</span>
                    {/if}
                    {#if warnCount > 0}
                        <span>{warnCount} {$t("checkers.dashboard.needs-attention")}</span>
                    {/if}
                </p>
            {/if}
            {#each sections as section}
                {#if section.items.length > 0}
                    <h4 class="mt-4 mb-3">{section.title}</h4>
                    <div class="check-stack">
                        {#each section.items as item}
                            {@const checker = item.status}
                            {@const exec = checker.latestExecution}
                            <CheckCard
                                status={exec?.result?.status}
                                name={checker.name || checker.id || ""}
                                chip={item.chip}
                                configureLink="{item.checkersBase}/{checker.id}"
                                detailsLink={exec?.id ? `${item.checkersBase}/${checker.id}/executions/${exec.id}` : undefined}
                                historyLink="{item.checkersBase}/{checker.id}/executions"
                                message={exec?.result?.message}
                                startedAt={exec?.startedAt}
                                scope={item.scope}
                                checkerId={checker.id}
                                hasMetrics={checker.has_metrics}
                            />
                        {/each}
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

<style>
    .check-summary {
        color: var(--bs-secondary-color, #6c757d);
        font-size: 0.875rem;
    }
    .check-summary-sep {
        margin: 0 0.4rem;
        color: var(--bs-border-color, #dee2e6);
    }

    .check-stack {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
    }
</style>
