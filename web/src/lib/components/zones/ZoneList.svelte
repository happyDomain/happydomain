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

<script lang="ts" generics="T extends HappydnsDomain = HappydnsDomain">
    import { createEventDispatcher, type Snippet } from "svelte";
    import { fly, fade } from "svelte/transition";
    import { flip } from "svelte/animate";

    import { Badge, Icon } from "@sveltestrap/sveltestrap";
    import { ListGroup } from "@sveltestrap/sveltestrap";
    import DomainWithProvider from "$lib/components/domains/DomainWithProvider.svelte";
    import { updateDomain } from "$lib/api/domains";
    import type { HappydnsDomain } from "$lib/api-base/types.gen";
    import { domains_idx, newlyGroups, refreshDomains } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        flush?: boolean;
        links?: boolean;
        display_by_groups?: boolean;
        show_empty_groups?: boolean;
        domains?: Array<T>;
        no_domain?: Snippet;
        badges?: Snippet<[{ domain: T }]>;
        [key: string]: unknown;
    }

    let {
        flush = false,
        links = false,
        display_by_groups = false,
        show_empty_groups = false,
        domains = [],
        no_domain,
        badges,
        ...rest
    }: Props = $props();

    function genGroups(
        domains: Array<T>,
        display_by_groups: boolean,
        show_empty_groups: boolean,
        extraGroups: Array<string>,
    ) {
        if (!display_by_groups) {
            return { "": domains };
        }

        const groups: Record<string, Array<T>> = {};

        for (const domain of domains) {
            const group = domain.group ?? "";
            (groups[group] ??= []).push(domain);
        }

        if (show_empty_groups) {
            for (const g of extraGroups) {
                if (!(g in groups)) {
                    groups[g] = [];
                }
            }
        }

        return groups;
    }

    let localDomains: Array<T> = $derived([...domains]);

    let groups: Record<string, Array<T>> = $derived(
        genGroups(localDomains, display_by_groups, show_empty_groups, $newlyGroups),
    );

    let collapsedGroups: Set<string> = $state(
        new Set(JSON.parse(localStorage.getItem("collapsedGroups") || "[]")),
    );

    let nonEmptyGroupCount: number = $derived(
        Object.values(groups).filter((g) => g.length > 0).length,
    );
    let draggedDomain: T | null = $state(null);
    let dragOverGroup: string | null = $state(null);

    function toggleGroup(gname: string) {
        const next = new Set(collapsedGroups);
        if (next.has(gname)) {
            next.delete(gname);
        } else {
            next.add(gname);
        }
        collapsedGroups = next;
        localStorage.setItem("collapsedGroups", JSON.stringify([...next]));
    }

    function soloGroup(gname: string) {
        const allGroups = Object.keys(groups);
        const next = new Set(allGroups.filter((g) => g !== gname));
        collapsedGroups = next;
        localStorage.setItem("collapsedGroups", JSON.stringify([...next]));
    }

    async function handleDrop(targetGroup: string) {
        if (!draggedDomain || (draggedDomain.group ?? "") === targetGroup) return;

        const fullDomain = $domains_idx[draggedDomain.domain] ?? $domains_idx[draggedDomain.id];
        if (!fullDomain) return;

        const prevGroup = draggedDomain.group;
        const domainId = draggedDomain.domain || draggedDomain.id;

        // Optimistic update
        localDomains = localDomains.map((d) =>
            d.domain === domainId || d.id === domainId ? { ...d, group: targetGroup } : d,
        );
        draggedDomain = null;
        dragOverGroup = null;

        fullDomain.group = targetGroup;
        try {
            await updateDomain(fullDomain.id, { group: targetGroup });
            refreshDomains();
        } catch {
            // Revert on error
            fullDomain.group = prevGroup ?? "";
            localDomains = localDomains.map((d) =>
                d.domain === domainId || d.id === domainId ? { ...d, group: prevGroup } : d,
            );
        }
    }

    function getDomainHref(domain: T): string | undefined {
        if (links) {
            if ($domains_idx[domain.domain]) {
                return "/domains/" + encodeURIComponent(domain.domain);
            } else {
                return "/domains/" + encodeURIComponent(domain.id);
            }
        }
        return undefined;
    }
</script>

{#snippet domainRow(domain: T)}
    <DomainWithProvider {domain} />
    {#if badges}{@render badges({ domain })}{:else}
        <Badge color="success">OK</Badge>
    {/if}
{/snippet}
<div {...rest}>
    {#if domains.length === 0}
        {@render no_domain?.()}
    {:else}
        {#each Object.keys(groups).sort((a, b) => {
            const aEmpty = groups[a].length === 0;
            const bEmpty = groups[b].length === 0;
            if (aEmpty !== bEmpty) return aEmpty ? 1 : -1;
            if (!a || !b) return !a ? 1 : -1;
            return a.toLowerCase().localeCompare(b.toLowerCase());
        }) as gname}
            {@const gdomains = groups[gname]}
            <div
                role="list"
                class:mb-2={Object.keys(groups).length != 1}
                class:drag-over={display_by_groups && dragOverGroup === gname}
                ondragover={display_by_groups
                    ? (e) => {
                          e.preventDefault();
                          dragOverGroup = gname;
                      }
                    : undefined}
                ondragleave={display_by_groups
                    ? (e) => {
                          if (!e.currentTarget.contains(e.relatedTarget as Node))
                              dragOverGroup = null;
                      }
                    : undefined}
                ondrop={display_by_groups
                    ? (e) => {
                          e.preventDefault();
                          handleDrop(gname);
                      }
                    : undefined}
            >
                {#if Object.keys(groups).length != 1}
                    <button
                        type="button"
                        class="group-header"
                        onclick={() => toggleGroup(gname)}
                        ondblclick={(e) => {
                            e.preventDefault();
                            soloGroup(gname);
                        }}
                    >
                        <span
                            class="group-chevron"
                            class:collapsed={collapsedGroups.has(gname) && nonEmptyGroupCount > 1}
                            ><Icon name="chevron-down" /></span
                        >
                        <span class="group-label">
                            {#if gname === ""}
                                {$t("domaingroups.no-group")}
                            {:else}
                                {gname}
                            {/if}
                        </span>
                    </button>
                {/if}
                {#if !collapsedGroups.has(gname) || nonEmptyGroupCount <= 1}
                    <ListGroup {flush}>
                        {#if display_by_groups && gdomains.length === 0}
                            <div
                                class="list-group-item text-center text-muted py-3 empty-group-placeholder"
                            >
                                {$t("domaingroups.drop-here")}
                            </div>
                        {/if}
                        {#each gdomains as item (item.id || item.domain)}
                            {@const href = getDomainHref(item)}
                            <div
                                in:fly={{ y: 12, duration: 250, delay: 30 }}
                                out:fade={{ duration: 150 }}
                                animate:flip={{ duration: 250 }}
                            >
                                {#if href}
                                    <a
                                        class="list-group-item list-group-item-action d-flex justify-content-between align-items-center text-dark"
                                        class:draggable-item={display_by_groups}
                                        {href}
                                        draggable={display_by_groups || undefined}
                                        ondragstart={display_by_groups
                                            ? () => {
                                                  draggedDomain = item;
                                              }
                                            : undefined}
                                        ondragend={display_by_groups
                                            ? () => {
                                                  draggedDomain = null;
                                                  dragOverGroup = null;
                                              }
                                            : undefined}
                                        onclick={() => dispatch("click", item)}
                                    >
                                        {@render domainRow(item)}
                                    </a>
                                {:else}
                                    <button
                                        class="list-group-item list-group-item-action d-flex justify-content-between align-items-center text-dark"
                                        class:draggable-item={display_by_groups}
                                        type="button"
                                        draggable={display_by_groups || undefined}
                                        ondragstart={display_by_groups
                                            ? () => {
                                                  draggedDomain = item;
                                              }
                                            : undefined}
                                        ondragend={display_by_groups
                                            ? () => {
                                                  draggedDomain = null;
                                                  dragOverGroup = null;
                                              }
                                            : undefined}
                                        onclick={() => dispatch("click", item)}
                                    >
                                        {@render domainRow(item)}
                                    </button>
                                {/if}
                            </div>
                        {/each}
                    </ListGroup>
                {/if}
            </div>
        {/each}
    {/if}
</div>

<style>
    .group-header {
        display: flex;
        align-items: center;
        gap: 0.25rem;
        margin-bottom: 1rem;
        margin-top: 1.5rem;
        width: 100%;
        border: none;
        background: none;
        padding: 0;
        cursor: pointer;
    }

    .group-header::before,
    .group-header::after {
        content: "";
        flex: 1;
        height: 1px;
        background: var(--bs-border-color);
    }

    .group-chevron {
        color: var(--bs-secondary-color);
        transition: transform 0.2s ease;
    }

    .group-chevron.collapsed {
        transform: rotate(-90deg);
    }

    .group-label {
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--bs-secondary-color);
        white-space: nowrap;
    }

    .drag-over {
        background-color: var(--bs-primary-bg-subtle, #cfe2ff);
        border-radius: 0.5rem;
        outline: 2px dashed var(--bs-primary, #0d6efd);
        transition:
            background-color 0.2s ease,
            outline-color 0.2s ease;
    }

    .empty-group-placeholder {
        min-height: 3rem;
        border-style: dashed;
        transition: background-color 0.2s ease;
    }

    :global(.list-group-item) {
        transition:
            background-color 0.15s ease,
            box-shadow 0.15s ease;
    }

    .draggable-item {
        cursor: grab;
    }

    .draggable-item:active {
        cursor: grabbing;
        opacity: 0.7;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    }
</style>
