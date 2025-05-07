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

    import { ListGroup, ListGroupItem } from "@sveltestrap/sveltestrap";

    const dispatch = createEventDispatcher();

    export let items: Array<any> = [];
    export let isLoading = false;
    export let button = false;
    export let isActive: (item: any) => boolean = () => false;
    export let links = false;
</script>

<ListGroup {...$$restProps}>
    {#if isLoading}
        <ListGroupItem class="d-flex justify-content-center align-items-center">
            <slot name="loading" />
        </ListGroupItem>
    {:else if items.length == 0}
        <ListGroupItem class="text-center">
            <slot name="empty" />
        </ListGroupItem>
    {:else}
        {#each items as item}
            <ListGroupItem
                active={isActive(item)}
                tag={button ? "button" : undefined}
                class="d-flex justify-content-between align-items-center"
                href={links ? item.href : undefined}
                on:click={() => dispatch("click", item)}
            >
                <slot {item} />
            </ListGroupItem>
        {/each}
    {/if}
</ListGroup>
