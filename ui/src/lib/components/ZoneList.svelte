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
 import { createEventDispatcher } from 'svelte';

 import ImgProvider from '$lib/components/providers/ImgProvider.svelte';
 import HListGroup from '$lib/components/ListGroup.svelte';
 import { t } from '$lib/translations';

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
                        <h3 class="d-inline-block px-2 bg-light" style="position: relative; top: -.65em">
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
