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
    import type { CheckerLinks } from "$lib/checker_links";
    import { Alert, Card, Icon } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { listScopedCheckers, type CheckerScope } from "$lib/api/checkers";
    import type { HappydnsCheckerStatus, HappydnsStatus } from "$lib/api-base/types.gen";
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
        links: CheckerLinks;
        scope: CheckerScope;
        chip?: string;
    }

    interface Section {
        title: string;
        domain: string;
        sortKey: string;
        items: CheckerItem[];
    }

    interface Props {
        domainId: string;
        domainName: string;
        title: string;
        /** When set, dashboard only lists this single service's checkers. */
        serviceTarget?: ServiceTarget & { zoneId: string };
        /**
         * Where the checkers of the domain, or of one of its services, live.
         * Handed in because only the route knows: this component is shared.
         */
        linksFor: (target?: { zoneId: string; subdomain: string; serviceId: string }) => CheckerLinks;
    }

    let { domainId, domainName, title, serviceTarget, linksFor }: Props = $props();

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
            const links = linksFor(serviceTarget);
            const items: CheckerItem[] = statuses.map((s) => ({
                status: s,
                links,
                scope,
                chip: serviceTarget.serviceLabel,
            }));
            return [
                {
                    title: subdomainTitle(serviceTarget.subdomain),
                    domain: serviceTarget.subdomain,
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
                    domain: subdomain,
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
            const links = linksFor();
            for (const s of domainStatuses) {
                sec.items.push({ status: s, links, scope: { domainId } });
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
                const links = linksFor({
                    zoneId: zone.id!,
                    subdomain: tg.subdomain,
                    serviceId: tg.serviceId,
                });
                const scope: CheckerScope = {
                    domainId,
                    zoneId: zone.id!,
                    subdomain: tg.subdomain,
                    serviceId: tg.serviceId,
                };
                for (const s of statuses) {
                    sec.items.push({ status: s, links, scope, chip: tg.serviceName });
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
            {@const needsLook = critCount + warnCount}
            {@const totalItems = sections.reduce((n, s) => n + s.items.length, 0)}
            {@const okCount = totalItems - needsLook}
            {@const critPct = totalItems > 0 ? (critCount / totalItems) * 100 : 0}
            {@const warnPct = totalItems > 0 ? (warnCount / totalItems) * 100 : 0}
            {@const okPct = totalItems > 0 ? (okCount / totalItems) * 100 : 0}

            <div
                class="health-panel mb-5"
                class:health-all-ok={needsLook === 0}
                class:health-has-warn={needsLook > 0 && critCount === 0}
                class:health-has-crit={critCount > 0}
            >
                <div class="health-main">
                    <span class="health-count">{needsLook > 0 ? needsLook : okCount}</span>
                    <span class="health-label">
                        {needsLook > 0
                            ? $t("checkers.dashboard.need-a-look")
                            : $t("checkers.dashboard.all-passing")}
                    </span>
                </div>
                <div class="health-details">
                    <div
                        class="health-bar"
                        title="{critPct.toFixed(0)}% critical / {warnPct.toFixed(
                            0,
                        )}% warning / {okPct.toFixed(0)}% passing"
                    >
                        {#if critPct > 0}
                            <div class="health-seg seg-crit" style="width:{critPct}%"></div>
                        {/if}
                        {#if warnPct > 0}
                            <div class="health-seg seg-warn" style="width:{warnPct}%"></div>
                        {/if}
                        {#if okPct > 0}
                            <div class="health-seg seg-ok" style="width:{okPct}%"></div>
                        {/if}
                    </div>
                    <div class="health-stats">
                        {#if critCount > 0}
                            <span class="hstat hstat-crit">
                                {critCount}
                                {$t("checkers.dashboard.critical")}
                            </span>
                        {/if}
                        {#if warnCount > 0}
                            <span class="hstat hstat-warn">
                                {warnCount}
                                {$t("checkers.dashboard.needs-attention")}
                            </span>
                        {/if}
                        {#if okCount > 0}
                            <span class="hstat hstat-ok">
                                {okCount}
                                {$t("checkers.dashboard.passing")}
                            </span>
                        {/if}
                    </div>
                </div>
            </div>

            {#each sections as section}
                {#if section.items.length > 0}
                    <div class="d-flex align-items-end mt-5 mb-4 gap-3">
                        <h2 class="mb-0 font-monospace" id={section.domain}>{section.title}</h2>
                        <hr class="mt-0 mb-2 border-3 border-dark flex-fill" />
                    </div>
                    <div class="check-stack">
                        {#each section.items as item}
                            {@const checker = item.status}
                            {@const exec = checker.latestExecution}
                            <CheckCard
                                status={exec?.result?.status}
                                name={checker.name || checker.id || ""}
                                chip={item.chip}
                                configureLink={item.links.checker(checker.id!)}
                                detailsLink={exec?.id
                                    ? item.links.execution!(checker.id!, exec.id!)
                                    : undefined}
                                historyLink={item.links.executions!(checker.id!)}
                                messages={exec?.messages}
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
    /* ── health panel ── */
    .health-panel {
        --hd-crit: #ef4444;
        --hd-warn: #f59e0b;
        --hd-ok: #22c55e;

        display: flex;
        align-items: center;
        gap: 1.5rem;
        padding: 1rem 1.25rem;
        border-radius: 0.5rem;
        border: 1px solid var(--bs-border-color);
        border-left: 4px solid var(--bs-border-color);
        background: var(--bs-body-bg);
        transition: border-left-color 0.2s;
    }

    :global([data-bs-theme="dark"]) .health-panel {
        --hd-crit: #f87171;
        --hd-warn: #fbbf24;
        --hd-ok: #4ade80;
    }

    .health-panel.health-has-crit {
        border-left-color: var(--hd-crit);
    }
    .health-panel.health-has-warn {
        border-left-color: var(--hd-warn);
    }
    .health-panel.health-all-ok {
        border-left-color: var(--hd-ok);
    }

    .health-main {
        display: flex;
        flex-direction: column;
        align-items: flex-end;
        min-width: 4ch;
        flex-shrink: 0;
    }

    .health-count {
        font-family: var(--bs-font-monospace);
        font-size: 2.25rem;
        font-weight: 700;
        line-height: 1;
    }

    .health-has-crit .health-count {
        color: var(--hd-crit);
    }
    .health-has-warn .health-count {
        color: var(--hd-warn);
    }
    .health-all-ok .health-count {
        color: var(--hd-ok);
    }

    .health-label {
        font-size: 0.68rem;
        letter-spacing: 0.07em;
        text-transform: uppercase;
        color: var(--bs-secondary-color);
        white-space: nowrap;
    }

    .health-details {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    .health-bar {
        display: flex;
        height: 5px;
        border-radius: 3px;
        overflow: hidden;
        background: var(--bs-border-color);
        gap: 2px;
    }

    .health-seg {
        border-radius: 2px;
        transition: width 0.5s ease;
    }

    .seg-crit {
        background: var(--hd-crit);
    }
    .seg-warn {
        background: var(--hd-warn);
    }
    .seg-ok {
        background: var(--hd-ok);
    }

    .health-stats {
        display: flex;
        flex-wrap: wrap;
        gap: 0.9rem;
    }

    .hstat {
        font-size: 0.78rem;
        font-family: var(--bs-font-monospace);
        display: flex;
        align-items: center;
        gap: 0.35rem;
        color: var(--bs-secondary-color);
    }

    .hstat::before {
        content: "";
        display: inline-block;
        width: 7px;
        height: 7px;
        border-radius: 50%;
        flex-shrink: 0;
    }

    .hstat-crit {
        color: var(--hd-crit);
    }
    .hstat-warn {
        color: var(--hd-warn);
    }

    .hstat-crit::before {
        background: var(--hd-crit);
    }
    .hstat-warn::before {
        background: var(--hd-warn);
    }
    .hstat-ok::before {
        background: var(--hd-ok);
    }

    /* ── check stack ── */
    .check-stack {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
    }
</style>
