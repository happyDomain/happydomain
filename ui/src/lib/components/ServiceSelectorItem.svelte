<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Badge,
     ListGroupItem,
 } from 'sveltestrap';

 import type { ServiceInfos } from '$lib/model/service_specs';

 const dispatch = createEventDispatcher();

 export let active = false;
 export let disabled = false;
 export let reason: string = "";

 export let svc: ServiceInfos;
</script>

<ListGroupItem
    {active}
    class="d-flex"
    {disabled}
    tag="button"
    on:click={() => dispatch("click")}
>
    {#if svc._svcicon}
        <div class="d-inline-block align-self-center text-center" style="width: 75px;">
            <img src={svc._svcicon} alt={svc.name} style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
        </div>
    {/if}
    <div class="flex-fill">
        {svc.name}
        {#if reason}
            <small class="font-italic text-danger">{reason}</small>
        {:else}
            <small class="text-muted">{svc.description}</small>
        {/if}
        {#if svc.categories}
            {#each svc.categories as category}
                <Badge
                    color="secondary"
                    class="float-end ms-1"
                >
                    {category}
                </Badge>
            {/each}
        {/if}
    </div>
</ListGroupItem>
