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
    import { formatCheckDate, getStatusI18nKey } from "$lib/utils";
    import { StatusCrit, StatusError, StatusWarn } from "$lib/utils/checkers";
    import CheckMetricsChart from "./CheckMetricsChart.svelte";

    interface Props {
        status: HappydnsStatus | undefined;
        name: string;
        chip?: string;
        configureLink?: string;
        detailsLink?: string;
        historyLink?: string;
        message?: string;
        startedAt?: Date | string;
        scope?: CheckerScope;
        checkerId?: string;
        hasMetrics?: boolean;
    }

    let {
        status,
        name,
        chip,
        configureLink,
        detailsLink,
        historyLink,
        message,
        startedAt,
        scope,
        checkerId,
        hasMetrics,
    }: Props = $props();

    function severityClass(s: HappydnsStatus | undefined): string {
        switch (s) {
            case StatusCrit:
            case StatusError:
                return "is-critical";
            case StatusWarn:
                return "is-warning";
            case 1:
                return "is-ok";
            case 2:
                return "is-info";
            default:
                return "is-muted";
        }
    }
</script>

<article class="check-card {severityClass(status)}">
    <header class="severity-strip">
        <span class="severity-label">
            {$t(getStatusI18nKey(status))}
        </span>
        {#if chip}
            <span class="check-chip">{chip}</span>
        {/if}
    </header>
    <div class="check-body">
        <div class="check-title-row">
            <h5 class="check-title">
                {name}
                {#if configureLink}
                    <a
                        href={configureLink}
                        class="check-configure-link"
                        title={$t("checkers.list.configure")}
                    >
                        <Icon name="gear-fill" />
                    </a>
                {/if}
            </h5>
            {#if detailsLink}
                <div class="check-actions">
                    <a href={detailsLink} class="btn btn-sm btn-outline-primary">
                        {$t("checkers.list.view-details")}
                    </a>
                </div>
            {/if}
        </div>
        {#if message}
            <p class="check-message">{message}</p>
        {/if}
        {#if startedAt || historyLink}
            <p class="check-footer">
                {#if startedAt}
                    {$t("checkers.dashboard.last-run-on", { date: formatCheckDate(startedAt) })}
                {:else}
                    {$t("checkers.dashboard.no-message")}
                {/if}
                {#if historyLink}
                    <span class="check-message-sep">·</span><a
                        href={historyLink}
                        class="check-history-link">{$t("checkers.list.history")}</a
                    >
                {/if}
            </p>
        {/if}
        {#if hasMetrics && checkerId && scope}
            {#await getScopedCheckerMetrics(scope, checkerId)}
                <div class="check-chart-loading">
                    <span class="spinner-border spinner-border-sm"></span>
                </div>
            {:then metrics}
                {#if metrics && metrics.length > 0}
                    <div class="check-chart">
                        <CheckMetricsChart {metrics} />
                    </div>
                {/if}
            {:catch}
                {""}
            {/await}
        {/if}
    </div>
</article>

<style>
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
    .check-chart {
        margin-top: 0.5rem;
    }
    .check-chart-loading {
        padding: 1rem;
        text-align: center;
        color: var(--bs-secondary-color, #6c757d);
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
