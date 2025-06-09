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
    import { createEventDispatcher } from "svelte";

    import { Badge } from "@sveltestrap/sveltestrap";
    import { ListGroup } from "@sveltestrap/sveltestrap";
    import DomainWithProvider from "$lib/components/domains/DomainWithProvider.svelte";
    import { domains_idx } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface ZoneListDomain {
        domain: string;
        id_provider: string;
        group?: string;
        href?: string;
    }

    export let button = false;
    export let flush = false;
    export let links = false;
    export let display_by_groups = false;
    export let domains: Array<ZoneListDomain> = [];

    let groups: Record<string, Array<ZoneListDomain>> = {};
    let draggedItem: ZoneListDomain | null = null;

    $: {
        if (!display_by_groups) {
            groups = { "": domains };
        }

        const tmp: Record<string, Array<ZoneListDomain>> = { };

        for (const domain of domains) {
            if (!domain.group) domain.group = "";
            if (links && !domain.href) {
                if ($domains_idx[domain.domain])
                    domain.href = "/domains/" + encodeURIComponent(domain.domain);
                else domain.href = "/domains/" + encodeURIComponent(domain.id);
            }

            if (tmp[domain.group] === undefined) {
                tmp[domain.group] = [];
            }

            tmp[domain.group].push(domain);
        }

        groups = tmp;
    }

    function handleDragStart(event: DragEvent, item: ZoneListDomain) {
        console.log("test drag start")
        draggedItem = item;
        event.dataTransfer?.setData('text/plain', item.domain);
    }

    function handleDrop(event: DragEvent, groupName: string) {
        console.log("drop")
        event.preventDefault();
        if (draggedItem) {
            draggedItem.group = groupName;
            // Update the groups
            groups = { ...groups };
        }
    }

    function handleDragOver(event: DragEvent) {
        event.preventDefault();
    }
</script>

<div {...$$restProps}>
    {#if domains.length === 0}
        <slot name="no-domain" />
    {:else}
        {#each Object.keys(groups).sort((a,b) => !a || !b ? (!a ? 1 : -1) : a.toLowerCase().localeCompare(b.toLowerCase())) as gname}
            {@const gdomains = groups[gname]}
            <div
                class:mb-2={Object.keys(groups).length != 1}
                on:drop|preventDefault={(event) => handleDrop(event, gname)}
                on:dragover|preventDefault={handleDragOver}
            >
                {#if Object.keys(groups).length != 1}
                    <div class="d-flex align-items-center">
                        <hr class="flex-fill">
                        <h3
                            class="px-2"
                        >
                            {#if gname === ""}
                                {$t("domaingroups.no-group")}
                            {:else}
                                {gname}
                            {/if}
                        </h3>
                        <hr class="flex-fill">
                    </div>
                {/if}
                <ListGroup
                    {flush}
                >
                    {#each gdomains as item}
                        <svelte:element
                            this={item.href ? "a" : "button"}
                            class="list-group-item list-group-item-action d-flex justify-content-between align-items-center text-dark"
                            href={item.href}
                            on:click={() => dispatch("click", item)}
                            draggable={true}
                            on:dragstart={(event) => handleDragStart(event, item)}
                        >
                            <DomainWithProvider domain={item} />
                            <Badge color="success">OK</Badge>
                        </svelte:element>
                    {/each}
                </ListGroup>
            </div>
        {/each}
    {/if}
</div>
