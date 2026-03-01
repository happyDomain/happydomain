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
        domains?: Array<ZoneListDomain>;
        no_domain?: import('svelte').Snippet;
        badges?: import('svelte').Snippet<[any]>;
        [key: string]: any
    }

    let {
        flush = false,
        links = false,
        display_by_groups = false,
        domains = [],
        no_domain,
        badges,
        ...rest
    }: Props = $props();

    function genGroups(domains: Array<ZoneListDomain>, display_by_groups: boolean) {
        if (!display_by_groups) {
            return { "": domains };
        }

        const groups: Record<string, Array<ZoneListDomain>> = { };

        for (const domain of domains) {
            const group = domain.group ?? '';
            (groups[group] ??= []).push(domain);
        }

        return groups;
    }

    let groups: Record<string, Array<ZoneListDomain>> = $derived(genGroups(domains, display_by_groups));

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
    {#if badges}{@render badges(domain)}{:else}
        <Badge color="success">OK</Badge>
    {/if}
{/snippet}
<div {...rest}>
    {#if domains.length === 0}
        {@render no_domain?.()}
    {:else}
        {#each Object.keys(groups).sort((a,b) => !a || !b ? (!a ? 1 : -1) : a.toLowerCase().localeCompare(b.toLowerCase())) as gname}
            {@const gdomains = groups[gname]}
            <div
                class:mb-2={Object.keys(groups).length != 1}
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
                        {@const href = getDomainHref(item)}
                        {#if href}
                            <a
                                class="list-group-item list-group-item-action d-flex justify-content-between align-items-center text-dark"
                                {href}
                                onclick={() => dispatch("click", item)}
                            >
                                {@render domainRow(item)}
                            </a>
                        {:else}
                            <button
                                class="list-group-item list-group-item-action d-flex justify-content-between align-items-center text-dark"
                                type="button"
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
