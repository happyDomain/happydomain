<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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
    import { resolve } from "$app/paths";
    import { page } from "$app/state";

    import { Icon } from "@sveltestrap/sveltestrap";

    import PageTitle from "$lib/components/PageTitle.svelte";
    import {
        activeLogDay,
        groupLogsByDay,
        isProblem,
        logSeverity,
        logZoneVersion,
        parseLogDay,
        type LogSeverity,
    } from "$lib/domain_logs";
    import type { Domain, DomainLog } from "$lib/model/domain";
    import { domainLink } from "$lib/stores/domains";
    import { getUser } from "$lib/stores/users";
    import { t } from "$lib/translations";

    interface Props {
        data: { domain: Domain; historyDomain: Domain; logs: DomainLog[] };
    }

    let { data }: Props = $props();

    let onlyProblems = $derived(page.url.searchParams.get("only") === "problems");
    let days = $derived(groupLogsByDay(onlyProblems ? data.logs.filter(isProblem) : data.logs));

    // Naming the author only helps when several people act on the domain.
    let severalAuthors = $derived(new Set(data.logs.map((l) => l.id_user)).size > 1);

    const ICONS: Record<LogSeverity, { name: string; color: string; label: string }> = {
        critical: { name: "x-octagon-fill", color: "text-danger", label: "CRIT" },
        error: { name: "x-circle-fill", color: "text-danger", label: "ERR" },
        warning: { name: "exclamation-triangle-fill", color: "text-warning", label: "WARN" },
        success: { name: "check-circle-fill", color: "text-success", label: "INFO" },
        info: { name: "info-circle", color: "text-info", label: "INFO" },
        debug: { name: "bug", color: "text-secondary", label: "DEBUG" },
    };

    const dayFmt = new Intl.DateTimeFormat(undefined, { dateStyle: "full" });
    const timeFmt = new Intl.DateTimeFormat(undefined, { timeStyle: "short" });
    const versionFmt = new Intl.DateTimeFormat(undefined, {
        dateStyle: "medium",
        timeStyle: "short",
    });

    // Scroll-spy: the active day is the last one whose header went past the
    // top of the viewport, the first one before any did.
    let feedEl: HTMLElement | null = $state(null);
    let ticking = false;

    function updateActiveDay() {
        ticking = false;
        if (!feedEl) return;
        let active: string | undefined;
        for (const section of feedEl.querySelectorAll<HTMLElement>("section[id]")) {
            if (active !== undefined && section.getBoundingClientRect().top > 80) break;
            active = section.id;
        }
        activeLogDay.set(active);
    }

    function onscroll() {
        if (!ticking) {
            ticking = true;
            requestAnimationFrame(updateActiveDay);
        }
    }

    $effect(() => {
        // Recompute when the filter changes the list of days.
        void days;
        updateActiveDay();
        return () => activeLogDay.set(undefined);
    });
</script>

<svelte:window {onscroll} />

<div class="flex-fill pb-4 pt-2">
    <PageTitle
        title={$t("logs.title")}
        domain={data.domain.domain}
        subtitle={$t("logs.subtitle")}
    />

    {#if days.length === 0}
        <p class="text-center text-muted mt-5">
            {onlyProblems ? $t("logs.no-problem") : $t("logs.no-entry")}
        </p>
    {:else}
        <div bind:this={feedEl}>
            {#each days as day (day.day)}
                <section id={day.day} class="log-day mb-3">
                    <h2
                        class="h6 fw-semibold position-sticky top-0 z-1 py-2 mb-0 border-bottom bg-body"
                    >
                        {dayFmt.format(parseLogDay(day.day))}
                    </h2>
                    <ul class="list-unstyled mb-0">
                        {#each day.logs as log (log.id)}
                            {@const severity = logSeverity(log.level)}
                            {@const icon = ICONS[severity]}
                            {@const zid = logZoneVersion(log, data.historyDomain.zone_history)}
                            {@const zmeta = zid ? data.historyDomain.zone_meta?.[zid] : undefined}
                            <li
                                class="d-flex align-items-start gap-2 px-2 py-2 border-bottom"
                                class:bg-danger-subtle={severity === "critical" ||
                                    severity === "error"}
                            >
                                <time
                                    class="log-time flex-shrink-0 font-monospace small text-body-secondary"
                                    datetime={new Date(log.date).toISOString()}
                                    title={new Date(log.date).toLocaleString()}
                                >
                                    {timeFmt.format(new Date(log.date))}
                                </time>
                                <Icon
                                    name={icon.name}
                                    class={icon.color}
                                    title={icon.label}
                                    aria-label={icon.label}
                                />
                                <div class="flex-fill min-w-0">
                                    <div class="text-break">{log.content}</div>
                                    {#if zid}
                                        <a
                                            href={resolve("/domains/[dn]/history/[zid]", {
                                                dn: encodeURIComponent(
                                                    domainLink(data.domain.domain),
                                                ),
                                                zid: encodeURIComponent(zid),
                                            })}
                                            class="small"
                                        >
                                            {$t("logs.see-version", {
                                                date: zmeta
                                                    ? versionFmt.format(
                                                          new Date(zmeta.last_modified),
                                                      )
                                                    : zid,
                                            })}
                                            <Icon name="arrow-right-short" />
                                        </a>
                                    {/if}
                                </div>
                                <div class="d-flex align-items-center gap-1 small text-muted">
                                    {#await getUser(log.id_user)}
                                        <img
                                            src={"/api/users/" +
                                                encodeURIComponent(log.id_user) +
                                                "/avatar.png"}
                                            alt={log.id_user}
                                            title={log.id_user}
                                            class="log-avatar rounded-1"
                                        />
                                    {:then user}
                                        <img
                                            src={"/api/users/" +
                                                encodeURIComponent(log.id_user) +
                                                "/avatar.png"}
                                            alt={user.email}
                                            title={user.email}
                                            class="log-avatar rounded-1"
                                        />
                                        {#if severalAuthors}
                                            <span class="text-truncate" style="max-width: 12em">
                                                {user.email}
                                            </span>
                                        {/if}
                                    {/await}
                                </div>
                            </li>
                        {/each}
                    </ul>
                </section>
            {/each}
        </div>
    {/if}
</div>

<style>
    .log-day {
        /* Leave room for the sticky header when jumping to a day. */
        scroll-margin-top: 0.5rem;
    }

    .log-time {
        width: 3.5em;
        line-height: 1.5rem;
    }

    /* As tall as a line of the message, whatever the size of its wrapper. */
    .log-avatar {
        height: 1.5rem;
    }
</style>
