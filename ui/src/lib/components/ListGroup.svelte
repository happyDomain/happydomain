<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     ListGroup,
     ListGroupItem,
 } from 'sveltestrap';

 const dispatch = createEventDispatcher();

 export let items: Array<any> = [];
 export let isLoading = false;
 export let button = false;
 export let isActive: (item: any) => boolean = () => false;
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
                tag={button?"button":undefined}
                class="d-flex justify-content-between align-items-center"
                on:click={() => dispatch("click", item)}
            >
                <slot {item} />
            </ListGroupItem>
        {/each}
    {/if}
</ListGroup>
