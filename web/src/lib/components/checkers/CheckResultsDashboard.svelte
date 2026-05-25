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
    import { formatCheckDate, getStatusI18nKey } from "$lib/utils";
    import { StatusCrit, StatusError, StatusWarn } from "$lib/utils/checkers";
    import PageTitle from "$lib/components/PageTitle.svelte";

    interface ServiceTarget {
        subdomain: string;
        serviceId: string;
        serviceLabel: string;
    }

    interface CheckerItem {
        status: HappydnsCheckerStatus;
        checkersBase: string;
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

    function severityClass(status: HappydnsStatus | undefined): string {
        switch (status) {
            case StatusCrit:
            case StatusError:
                return "is-critical";
            case StatusWarn:
                return "is-warning";
            case 1: // OK
                return "is-ok";
            case 2: // Info
                return "is-info";
            default:
                return "is-muted";
        }
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
                sec.items.push({ status: s, checkersBase: `${domainBase}/checkers` });
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
                for (const s of statuses) {
                    sec.items.push({ status: s, checkersBase: base, chip: tg.serviceName });
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
                            {@const status = exec?.result?.status}
                            {@const message = exec?.result?.message}
                            <article class="check-card {severityClass(status)}">
                                <header class="severity-strip">
                                    <span class="severity-label">
                                        {$t(getStatusI18nKey(status))}
                                    </span>
                                    {#if item.chip}
                                        <span class="check-chip">{item.chip}</span>
                                    {/if}
                                </header>
                                <div class="check-body">
                                    <div class="check-title-row">
                                        <h5 class="check-title">
                                            {checker.name || checker.id}
                                            <a
                                                href="{item.checkersBase}/{checker.id}"
                                                class="check-configure-link"
                                                title={$t("checkers.list.configure")}
                                            >
                                                <Icon name="gear-fill" />
                                            </a>
                                        </h5>
                                        {#if exec?.id}
                                            <div class="check-actions">
                                                <a
                                                    href="{item.checkersBase}/{checker.id}/executions/{exec.id}"
                                                    class="btn btn-sm btn-outline-primary"
                                                >
                                                    {$t("checkers.list.view-details")}
                                                </a>
                                            </div>
                                        {/if}
                                    </div>
                                    {#if message}
                                        <p class="check-message">{message}</p>
                                    {/if}
                                    <p class="check-footer">
                                        {#if exec?.startedAt}
                                            {$t("checkers.dashboard.last-run-on", { date: formatCheckDate(exec.startedAt) })}
                                        {:else}
                                            {$t("checkers.dashboard.no-message")}
                                        {/if}
                                        <span class="check-message-sep">·</span><a href="{item.checkersBase}/{checker.id}/executions" class="check-history-link">{$t("checkers.list.history")}</a>
                                    </p>
                                </div>
                            </article>
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

    .check-card {
        border: 1px solid var(--strip-border, #dee2e6);
        border-radius: 0.5rem;
        background: #ffffff;
        overflow: hidden;
    }

    .severity-strip {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.5rem 0.875rem;
        background: var(--strip-bg, #f8f9fa);
        border-bottom: 1px solid var(--strip-border, #dee2e6);
        font-size: 0.7rem;
        letter-spacing: 0.06em;
        text-transform: uppercase;
        font-weight: 600;
    }
    .severity-label {
        color: var(--strip-fg, #495057);
        display: inline-flex;
        align-items: center;
    }
    .severity-label::before {
        content: "";
        display: inline-block;
        width: 0.5rem;
        height: 0.5rem;
        border-radius: 50%;
        background: var(--strip-fg, #495057);
        margin-right: 0.5rem;
    }

    .check-chip {
        background: rgba(0, 0, 0, 0.06);
        color: #495057;
        padding: 0.15rem 0.55rem;
        border-radius: 999px;
        font-family: var(--bs-font-monospace, ui-monospace, SFMono-Regular, Menlo, monospace);
        font-size: 0.7rem;
        letter-spacing: 0.02em;
        text-transform: none;
    }

    .check-body {
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
        padding: 1rem 1.125rem;
    }
    .check-title-row {
        display: flex;
        align-items: center;
        gap: 1rem;
    }
    .check-title {
        flex: 1 1 auto;
        min-width: 0;
        font-size: 1.05rem;
        font-weight: 600;
        margin: 0;
        color: #1a1a1a;
        display: flex;
        align-items: center;
        gap: 0.4rem;
    }
    .check-configure-link {
        color: #adb5bd;
        font-size: 0.85rem;
        line-height: 1;
        text-decoration: none;
    }
    .check-configure-link:hover {
        color: #495057;
    }
    .check-message {
        margin: 0;
        color: #6c757d;
        font-size: 0.9rem;
        line-height: 1.4;
    }
    .check-footer {
        margin: 0;
        color: #6c757d;
        font-size: 0.78rem;
        line-height: 1.4;
    }
    .check-message-sep {
        margin: 0 0.4rem;
        color: var(--bs-border-color, #dee2e6);
    }
    .check-history-link {
        color: #6c757d;
        text-decoration: underline;
        text-decoration-style: dotted;
        text-underline-offset: 2px;
        white-space: nowrap;
    }
    .check-history-link:hover {
        color: #495057;
        text-decoration-style: solid;
    }
    .check-actions {
        flex: 0 0 auto;
        display: flex;
        gap: 0.375rem;
    }

    .check-card.is-critical {
        --strip-bg: #fdecec;
        --strip-border: #f5b5b5;
        --strip-fg: #c0392b;
    }
    .check-card.is-warning {
        --strip-bg: #fff7e0;
        --strip-border: #f0d97a;
        --strip-fg: #b7791f;
    }
    .check-card.is-ok {
        --strip-bg: #e8f5ee;
        --strip-border: #b6dcc4;
        --strip-fg: #2f855a;
    }
    .check-card.is-info {
        --strip-bg: #e7f1fb;
        --strip-border: #b6d3f0;
        --strip-fg: #2c5282;
    }
    .check-card.is-muted {
        --strip-bg: #f1f3f5;
        --strip-border: #dee2e6;
        --strip-fg: #6c757d;
    }

    @media (max-width: 575px) {
        .check-title-row {
            flex-direction: column;
            align-items: stretch;
        }
        .check-actions {
            justify-content: flex-end;
        }
    }
</style>
