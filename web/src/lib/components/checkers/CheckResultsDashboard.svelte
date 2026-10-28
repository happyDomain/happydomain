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
    import type { CheckSection, CheckTarget } from "$lib/domain_checks";
    import { fqdn } from "$lib/dns";
    import { StatusCrit, StatusError, StatusWarn } from "$lib/utils/checkers";
    import CheckCard from "./CheckCard.svelte";
    import PageTitle from "$lib/components/PageTitle.svelte";

    interface Props {
        domainName: string;
        title: string;
        // Pending while the checks load, or while the zone does.
        sections?: Promise<CheckSection[]>;
        /**
         * Where the checkers of the domain, or of one of its services, live.
         * Handed in because only the route knows: this component is shared.
         */
        linksFor: (target?: CheckTarget) => CheckerLinks;
    }

    let { domainName, title, sections: sectionsPromise, linksFor }: Props = $props();

    function countCritical(sections: CheckSection[]): number {
        let n = 0;
        for (const s of sections) {
            for (const c of s.items) {
                const code = c.status.latestExecution?.result?.status;
                if (code === StatusCrit || code === StatusError) n++;
            }
        }
        return n;
    }

    function countWarn(sections: CheckSection[]): number {
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
</script>

<svelte:head>
    <title>{title} - HappyDomain</title>
</svelte:head>

<div class="flex-fill mt-1 mb-5">
    <PageTitle {title} domain={domainName}></PageTitle>

    {#snippet loading()}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.list.loading")}
            </p>
        </Card>
    {/snippet}

    {#if !sectionsPromise}
        {@render loading()}
    {:else}
        {#await sectionsPromise}
            {@render loading()}
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
                            <h2 class="mb-0 font-monospace" id={section.subdomain}>
                                {subdomainTitle(section.subdomain)}
                            </h2>
                            <hr class="mt-0 mb-2 border-3 border-dark flex-fill" />
                        </div>
                        <div class="check-stack">
                            {#each section.items as item}
                                {@const checker = item.status}
                                {@const links = linksFor(item.target)}
                                {@const exec = checker.latestExecution}
                                <CheckCard
                                    status={exec?.result?.status}
                                    name={checker.name || checker.id || ""}
                                    chip={item.chip}
                                    configureLink={links.checker(checker.id!)}
                                    detailsLink={exec?.id
                                        ? links.execution!(checker.id!, exec.id!)
                                        : undefined}
                                    historyLink={links.executions!(checker.id!)}
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
    {/if}
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
