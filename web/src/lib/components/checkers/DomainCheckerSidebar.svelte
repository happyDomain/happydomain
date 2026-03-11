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
    import { page } from "$app/state";
    import type { ClassValue } from "svelte/elements";
    import { Icon, Spinner } from "@sveltestrap/sveltestrap";

    import { checkers } from "$lib/stores/checkers";

    interface Props {
        class?: ClassValue;
        domainName: string;
        currentCheckerName: string;
        checksBase?: string;
        scope?: "domain" | "service";
        serviceType?: string;
    }

    let {
        class: className = "",
        domainName,
        currentCheckerName,
        checksBase: checksBaseProp = undefined,
        scope = "domain",
        serviceType = undefined,
    }: Props = $props();

    let checksBase = $derived(
        checksBaseProp ?? `/domains/${encodeURIComponent(domainName)}/checks`,
    );
    let onResults = $derived(page.route.id?.includes("/results") === true && !page.params.rid);

    function isCheckVisible(checkerInfo: NonNullable<typeof $checks>[string]): boolean {
        const avail = checkerInfo.availability;
        if (!avail) return true;
        if (scope === "domain" && !avail.applyToDomain) return false;
        if (scope === "service") {
            if (!avail.applyToService) return false;
            if (avail.limitToServices && avail.limitToServices.length > 0) {
                if (!serviceType || !avail.limitToServices.includes(serviceType)) return false;
            }
        }
        return true;
    }
</script>

<nav class="checker-sidebar d-flex flex-column h-100 {className}">
    {#if !$checkers}
        <div class="d-flex gap-2 align-items-center justify-content-center my-3 text-muted">
            <Spinner size="sm" color="primary" />
        </div>
    {:else}
        <ul class="list-unstyled mb-0 flex-fill overflow-auto">
            {#each Object.entries($checkers) as [checkerName, checkerInfo]}
                {#if isCheckVisible(checkerInfo)}
                    {@const isActive = checkerName === currentCheckerName}
                    <li>
                        <div
                            class="checker-item d-flex align-items-center gap-1 py-2 px-2 rounded {isActive
                                ? 'fw-bold text-primary active'
                                : 'text-muted'}"
                        >
                            <a
                                href="{checksBase}/{encodeURIComponent(checkerName)}{onResults
                                    ? '/results'
                                    : ''}"
                                class="text-truncate flex-fill text-decoration-none {isActive
                                    ? 'text-primary'
                                    : 'text-muted'}"
                            >
                                {checkerInfo.name || checkerName}
                            </a>
                            {#if onResults}
                                <a
                                    href="{checksBase}/{encodeURIComponent(checkerName)}"
                                    class="checker-action text-muted"
                                    title="Configure"
                                >
                                    <Icon name="gear" />
                                </a>
                            {:else}
                                <a
                                    href="{checksBase}/{encodeURIComponent(checkerName)}/results"
                                    class="checker-action text-muted"
                                    title="Results"
                                >
                                    <Icon name="bar-chart-fill" />
                                </a>
                            {/if}
                        </div>
                    </li>
                {/if}
            {/each}
        </ul>
    {/if}
</nav>

<style>
    .checker-item {
        transition: background-color 0.15s;
    }

    .checker-item:hover {
        background-color: rgba(0, 0, 0, 0.06);
    }

    .checker-action {
        flex-shrink: 0;
        opacity: 0;
        text-decoration: none;
        line-height: 1;
        transition: opacity 0.15s;
    }

    .checker-action.always-visible {
        opacity: 0.6;
    }

    .checker-item:hover .checker-action,
    .checker-item.active .checker-action {
        opacity: 0.6;
    }

    .checker-action:hover {
        opacity: 1 !important;
    }

    .checker-item.active {
        background-color: rgba(var(--bs-primary-rgb), 0.1);
    }
</style>
