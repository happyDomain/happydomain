<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import hList from '$lib/components/hList.svelte';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let button = false;
 export let display_by_groups = false;
 export let domains = [];
</script>

<div>
    {#if domains.length === 0}
        <slot name="no-domain" />
    {:else}
        {#each Objects.keys(groups) as gname}
            {@const gdomains = groups[gname]}
            <div
                class:border-top={Object.keys(groups).length != 1}
                style="margin-top: 1.4em"
            >
                {#if Object.keys(groups).length != 1}
                    <div class="text-center" style="height: 1em">
                        <h3 class="d-inline-block px-1" style="background: white; position: relative; top: -.65em">
                            {#if group === undefined}
                                {$t("domaingroups.no-group")}
                            {:else}
                                {gname}
                            {/if}
                        </h3>
                    </div>
                {/if}
                <hList
                    items={gdomains}
                    {button}
                    on:click={(event) => dispatch('click', event.details)}
                    let:item={item}
                >
                    <div class="text-monospace">
                        <div class="d-inline-block text-center" style="width: 50px;">
                            <ImgProvider id_provider={item.id_provider} />
                        </div>
                        {item.domain}
                    </div>
                    <slot name="badges" domain={item} />
                </hList>
            </div>
        {/each}
    {/if}
</div>
