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

    interface Props {
        items?: Array<any>;
        isLoading?: boolean;
        button?: boolean;
        isActive?: (item: any) => boolean;
        links?: boolean;
        loading?: import('svelte').Snippet;
        empty?: import('svelte').Snippet;
        children?: import('svelte').Snippet<[any]>;
        [key: string]: any
    }

    let {
        items = [],
        isLoading = false,
        button = false,
        isActive = () => false,
        links = false,
        loading,
        empty,
        children,
        ...rest
    }: Props = $props();
</script>

<ListGroup {...rest}>
    {#if isLoading}
        <ListGroupItem class="d-flex justify-content-center align-items-center">
            {@render loading?.()}
        </ListGroupItem>
    {:else if items.length == 0}
        <ListGroupItem class="text-center">
            {@render empty?.()}
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
                {@render children?.({ item, })}
            </ListGroupItem>
        {/each}
    {/if}
</ListGroup>
