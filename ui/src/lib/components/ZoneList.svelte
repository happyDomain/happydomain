<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import ImgProvider from '$lib/components/providers/ImgProvider.svelte';
 import HListGroup from '$lib/components/ListGroup.svelte';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 interface ZoneListDomain {
     domain: string;
     id_provider: string;
     group?: string;
 }

 export let button = false;
 export let flush = false;
 export let links = false;
 export let display_by_groups = false;
 export let domains: Array<ZoneListDomain> = [];

 let groups: Record<string, Array<ZoneListDomain>> = {};
 $: {
     if (!display_by_groups) {
         groups = { "": domains };
     }

     const tmp: Record<string, Array<ZoneListDomain>> = { };

     for (const domain of domains) {
         if (!domain.group) domain.group = "";
         if (links && !domain.href) domain.href = '/domains/' + encodeURIComponent(domain.domain);

         if (tmp[domain.group] === undefined) {
             tmp[domain.group] = [];
         }

         tmp[domain.group].push(domain);
     }

     groups = tmp;
 }
</script>

<div {...$$restProps}>
    {#if domains.length === 0}
        <slot name="no-domain" />
    {:else}
        {#each Object.keys(groups) as gname}
            {@const gdomains = groups[gname]}
            <div
                class:border-top={Object.keys(groups).length != 1}
                class:mb-4={Object.keys(groups).length != 1}
            >
                {#if Object.keys(groups).length != 1}
                    <div class="text-center" style="height: 1em">
                        <h3 class="d-inline-block px-1" style="background: white; position: relative; top: -.65em">
                            {#if gname === ""}
                                {$t("domaingroups.no-group")}
                            {:else}
                                {gname}
                            {/if}
                        </h3>
                    </div>
                {/if}
                <HListGroup
                    {button}
                    {flush}
                    items={gdomains}
                    {links}
                    on:click={(event) => dispatch("click", event.detail)}
                    let:item={item}
                >
                    <div class="d-flex my-1" style="min-width: 0">
                        <div class="d-inline-block text-center" style="width: 50px;">
                            <ImgProvider id_provider={item.id_provider} />
                        </div>
                        <div class="font-monospace text-truncate flex-shrink-1">
                            {item.domain}
                        </div>
                    </div>
                    <slot name="badges" {item} />
                </HListGroup>
            </div>
        {/each}
    {/if}
</div>
