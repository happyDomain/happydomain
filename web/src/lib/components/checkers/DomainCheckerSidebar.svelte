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
    import { page } from "$app/state";
    import type { ClassValue } from "svelte/elements";
    import { Icon, Spinner } from "@sveltestrap/sveltestrap";

    import { checkers } from "$lib/stores/checkers";
    import { t } from "$lib/translations";

    interface Props {
        class?: ClassValue;
        domainName: string;
        currentCheckerName: string;
        links: CheckerLinks;
        scope?: "domain" | "service";
        serviceType?: string;
    }

    let {
        class: className,
        domainName,
        currentCheckerName,
        links,
        scope = "domain",
        serviceType = undefined,
    }: Props = $props();

    let visibleCheckers = $derived(
        Object.entries($checkers ?? {})
            .filter(([, checkerInfo]) => isCheckVisible(checkerInfo))
            .sort(([, a], [, b]) => (a.name || "").localeCompare(b.name || "")),
    );

    let onResults = $derived(
        page.route.id?.includes("/executions") === true && !page.params.execId,
    );

    function isCheckVisible(checkerInfo: NonNullable<typeof $checkers>[string]): boolean {
        const avail = checkerInfo.availability;
        if (!avail) return true;
        if (scope === "domain" && !avail.applyToDomain && !avail.applyToZone) return false;
        if (scope === "service") {
            if (!avail.applyToService) return false;
            if (avail.limitToServices && avail.limitToServices.length > 0) {
                if (!serviceType || !avail.limitToServices.includes(serviceType)) return false;
            }
        }
        return true;
    }
</script>

<nav class={["checker-sidebar d-flex flex-column h-100", className]}>
    {#if !$checkers}
        <div class="d-flex gap-2 align-items-center justify-content-center my-3 text-muted">
            <Spinner size="sm" color="primary" />
        </div>
    {:else if visibleCheckers.length === 0}
        <p class="small fst-italic text-body-secondary px-2 mb-0">
            {$t("checkers.sidebar-no-checker")}
        </p>
    {:else}
        <ul class="list-group list-group-sidebar gap-1 mb-0 flex-fill overflow-auto">
            {#each visibleCheckers as [checkerName, checkerInfo] (checkerName)}
                {@const isActive = checkerName === currentCheckerName}
                <li class="d-grid">
                    <div
                        class="checker-item list-group-item list-group-item-action d-flex align-items-center gap-1 p-2 border-0 rounded"
                        class:active={isActive}
                    >
                        <a
                            href={onResults && links.executions
                                ? links.executions(checkerName)
                                : links.checker(checkerName)}
                            class="text-truncate flex-fill text-body text-decoration-none"
                            aria-current={isActive ? "page" : undefined}
                        >
                            <span>{checkerInfo.name || checkerName}</span>
                        </a>
                        {#if onResults}
                            <a
                                href={links.checker(checkerName)}
                                class="checker-action flex-shrink-0 lh-1 text-muted text-decoration-none"
                                title="Configure"
                            >
                                <Icon name="gear" />
                            </a>
                        {:else}
                            <a
                                href={links.executions!(checkerName)}
                                class="checker-action flex-shrink-0 lh-1 text-muted text-decoration-none"
                                title="Results"
                            >
                                <Icon name="bar-chart-fill" />
                            </a>
                        {/if}
                    </div>
                </li>
            {/each}
        </ul>
    {/if}
</nav>

<style>
    /* The other page of the checker shows up on the hovered or active row. */
    .checker-action {
        opacity: 0;
        transition: opacity 0.15s;
    }

    .checker-item:hover .checker-action,
    .checker-item.active .checker-action {
        opacity: 0.6;
    }

    .checker-action:hover {
        opacity: 1 !important;
    }
</style>
