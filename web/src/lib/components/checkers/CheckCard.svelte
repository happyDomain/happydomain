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
    import { Icon } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { getScopedCheckerMetrics, type CheckerScope } from "$lib/api/checkers";
    import type { HappydnsStatus } from "$lib/api-base/types.gen";
    import { formatCheckDate, getStatusColor, getStatusI18nKey } from "$lib/utils";
    import CheckMetricsChart from "./CheckMetricsChart.svelte";

    interface Props {
        class?: string;
        status: HappydnsStatus | undefined;
        name: string;
        chip?: string;
        configureLink?: string;
        detailsLink?: string;
        historyLink?: string;
        messages?: string[];
        startedAt?: Date | string;
        scope?: CheckerScope;
        checkerId?: string;
        hasMetrics?: boolean;
    }

    let {
        class: className = "",
        status,
        name,
        chip,
        configureLink,
        detailsLink,
        historyLink,
        messages,
        startedAt,
        scope,
        checkerId,
        hasMetrics,
    }: Props = $props();

    let color = $derived(getStatusColor(status));
</script>

<article class="card overflow-hidden border-{color}-subtle {className}">
    <header
        class="card-header d-flex align-items-center gap-2 py-2 small fw-semibold bg-{color}-subtle border-{color}-subtle"
    >
        <span class="d-inline-flex align-items-center gap-2 text-uppercase text-{color}-emphasis">
            <span class="d-inline-block p-1 rounded-circle bg-{color}"></span>
            {$t(getStatusI18nKey(status))}
        </span>
        {#if chip}
            <span
                class="badge rounded-pill bg-body-secondary text-body-secondary font-monospace fw-normal"
            >
                {chip}
            </span>
        {/if}
    </header>
    <div class="card-body vstack gap-1">
        <div class="d-flex flex-column flex-sm-row align-items-sm-center gap-3">
            <h5
                class="fs-6 fw-semibold text-body-emphasis mb-0 flex-fill min-w-0 d-flex align-items-center gap-2"
            >
                {name}
                {#if configureLink}
                    <a
                        href={configureLink}
                        class="link-secondary small lh-1 text-decoration-none"
                        title={$t("checkers.list.configure")}
                    >
                        <Icon name="gear-fill" />
                    </a>
                {/if}
            </h5>
            {#if detailsLink}
                <a
                    href={detailsLink}
                    class="btn btn-sm btn-outline-primary flex-shrink-0 align-self-end align-self-sm-auto"
                >
                    {$t("checkers.list.view-details")}
                </a>
            {/if}
        </div>
        {#each messages ?? [] as msg}
            <p class="small text-body-secondary mb-0">{msg}</p>
        {/each}
        {#if startedAt || historyLink}
            <p class="small text-body-secondary mb-0">
                {#if startedAt}
                    {$t("checkers.dashboard.last-run-on", { date: formatCheckDate(startedAt) })}
                {:else}
                    {$t("checkers.dashboard.no-message")}
                {/if}
                {#if historyLink}
                    <span class="mx-1 text-body-tertiary">·</span><a
                        href={historyLink}
                        class="link-secondary link-offset-1 link-underline-opacity-50 link-underline-opacity-100-hover text-nowrap"
                        >{$t("checkers.list.history")}</a
                    >
                {/if}
            </p>
        {/if}
        {#if hasMetrics && checkerId && scope}
            {#await getScopedCheckerMetrics(scope, checkerId)}
                <div class="p-3 text-center text-body-secondary">
                    <span class="spinner-border spinner-border-sm"></span>
                </div>
            {:then metrics}
                {#if metrics && metrics.length > 0}
                    <div class="mt-2">
                        <CheckMetricsChart {metrics} />
                    </div>
                {/if}
            {:catch}
                {""}
            {/await}
        {/if}
    </div>
</article>
