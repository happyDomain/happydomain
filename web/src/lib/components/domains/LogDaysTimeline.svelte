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

    import { activeLogDay, logSeverity, parseLogDay, type LogDay } from "$lib/domain_logs";
    import { t } from "$lib/translations";

    interface Props {
        days: LogDay[];
        onlyProblems: boolean;
        filterHref: (onlyProblems: boolean) => string;
    }

    let { days, onlyProblems, filterHref }: Props = $props();

    const monthFmt = new Intl.DateTimeFormat(undefined, { month: "long", year: "numeric" });
    const dayFmt = new Intl.DateTimeFormat(undefined, {
        weekday: "short",
        day: "numeric",
        month: "short",
    });

    let groups = $derived.by(() => {
        const res: Array<{ month: string; days: LogDay[] }> = [];
        for (const day of days) {
            const month = monthFmt.format(parseLogDay(day.day));
            if (res.length === 0 || res[res.length - 1].month !== month) {
                res.push({ month, days: [] });
            }
            res[res.length - 1].days.push(day);
        }
        return res;
    });

    let containerEl: HTMLElement | null = $state(null);
    $effect(() => {
        if ($activeLogDay && containerEl) {
            containerEl
                .querySelector('[aria-current="location"]')
                ?.scrollIntoView({ block: "nearest" });
        }
    });
</script>

<div bind:this={containerEl}>
    <!-- The parent resolves the filter links, the query string is ours. -->
    <!-- eslint-disable svelte/no-navigation-without-resolve -->
    <div class="btn-group btn-group-sm w-100 mt-3" role="group">
        <a
            href={filterHref(false)}
            class="btn btn-outline-secondary"
            class:active={!onlyProblems}
            aria-current={!onlyProblems ? "page" : undefined}
        >
            {$t("logs.filter.all")}
        </a>
        <a
            href={filterHref(true)}
            class="btn btn-outline-secondary"
            class:active={onlyProblems}
            aria-current={onlyProblems ? "page" : undefined}
        >
            <Icon name="exclamation-triangle" />
            {$t("logs.filter.problems")}
        </a>
    </div>
    <!-- eslint-enable svelte/no-navigation-without-resolve -->

    {#each groups as group (group.month)}
        <div class="timeline-month text-muted small fw-semibold text-uppercase mt-3 mb-1">
            {group.month}
        </div>
        {#each group.days as day (day.day)}
            {@const severity = logSeverity(day.worst)}
            {@const dot = severity === "critical" || severity === "error" ? "danger" : severity}
            <a
                href={"#" + day.day}
                class="timeline-entry d-flex align-items-center gap-1 text-decoration-none text-body timeline-{dot}"
                class:bg-primary-subtle={day.day === $activeLogDay}
                aria-current={day.day === $activeLogDay ? "location" : undefined}
            >
                <span class="text-truncate">{dayFmt.format(parseLogDay(day.day))}</span>
                {#if severity === "critical" || severity === "error"}
                    <Icon name="exclamation-circle-fill" class="text-danger" />
                {:else if severity === "warning"}
                    <Icon name="exclamation-triangle-fill" class="text-warning" />
                {/if}
                <span class="ms-auto small text-muted">{day.logs.length}</span>
            </a>
        {/each}
    {/each}
</div>
