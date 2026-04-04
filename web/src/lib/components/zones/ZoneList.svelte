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
    import { createEventDispatcher, type Snippet } from "svelte";

    import { Badge } from "@sveltestrap/sveltestrap";
    import { ListGroup } from "@sveltestrap/sveltestrap";
    import DomainWithProvider from "$lib/components/domains/DomainWithProvider.svelte";
    import { updateDomain } from "$lib/api/domains";
    import { domains_idx, newlyGroups, refreshDomains } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface ZoneListDomain {
        id: string;
        domain: string;
        id_provider: string;
        group?: string;
        href?: string;
    }

    interface Props {
        flush?: boolean;
        links?: boolean;
        display_by_groups?: boolean;
        show_empty_groups?: boolean;
        domains?: Array<ZoneListDomain>;
        no_domain?: Snippet;
        badges?: Snippet<[{ domain: ZoneListDomain }]>;
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
        domains: Array<ZoneListDomain>,
        display_by_groups: boolean,
        show_empty_groups: boolean,
        extraGroups: Array<string>,
    ) {
        if (!display_by_groups) {
            return { "": domains };
        }

        const groups: Record<string, Array<ZoneListDomain>> = {};

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

    let localDomains: Array<ZoneListDomain> = $derived([...domains]);

    let groups: Record<string, Array<ZoneListDomain>> = $derived(
        genGroups(localDomains, display_by_groups, show_empty_groups, $newlyGroups),
    );

    let draggedDomain: ZoneListDomain | null = $state(null);
    let dragOverGroup: string | null = $state(null);

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

    function getDomainHref(domain: ZoneListDomain): string | undefined {
        if (links && !domain.href) {
            if ($domains_idx[domain.domain]) {
                return "/domains/" + encodeURIComponent(domain.domain);
            } else {
                return "/domains/" + encodeURIComponent(domain.id);
            }
        }
        return domain.href;
    }
</script>

{#snippet domainRow(domain: ZoneListDomain)}
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
                    <div class="d-flex align-items-center">
                        <hr class="flex-fill" />
                        <h3 class="px-2">
                            {#if gname === ""}
                                {$t("domaingroups.no-group")}
                            {:else}
                                {gname}
                            {/if}
                        </h3>
                        <hr class="flex-fill" />
                    </div>
                {/if}
                <ListGroup {flush}>
                    {#if display_by_groups && gdomains.length === 0}
                        <div
                            class="list-group-item text-center text-muted py-3 empty-group-placeholder"
                        >
                            {$t("domaingroups.drop-here")}
                        </div>
                    {/if}
                    {#each gdomains as item}
                        {@const href = getDomainHref(item)}
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
                    {/each}
                </ListGroup>
            </div>
        {/each}
    {/if}
</div>

<style>
    .drag-over {
        background-color: var(--bs-primary-bg-subtle, #cfe2ff);
        border-radius: 0.25rem;
        outline: 2px dashed var(--bs-primary, #0d6efd);
    }
    .empty-group-placeholder {
        min-height: 3rem;
        border-style: dashed;
    }
</style>
