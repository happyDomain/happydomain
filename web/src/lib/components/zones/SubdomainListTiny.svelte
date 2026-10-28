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
    import type { Snippet } from "svelte";

    import type { Domain } from "$lib/model/domain";
    import DomainTreeLink, { treeRails } from "./DomainTreeLink.svelte";

    interface Props {
        domains: Array<string>;
        origin: Domain;
        // The page of each subdomain, "" being the apex.
        href: (dn: string) => string;
        // The subdomain of the page, when they have one each.
        current?: string;
        // Shown at the end of the row of each subdomain.
        end?: Snippet<[string]>;
    }

    let { domains, origin, href, current, end }: Props = $props();

    let rails = $derived(treeRails(domains));

    let containerEl: HTMLElement | null = $state(null);

    $effect(() => {
        if (current !== undefined && containerEl) {
            const el = containerEl.querySelector(`a[href="${href(current)}"]`);
            el?.scrollIntoView({ block: "nearest" });
        }
    });
</script>

<div bind:this={containerEl}>
    {#each domains as dn, i (dn)}
        <DomainTreeLink
            href={href(dn)}
            {dn}
            {origin}
            rails={rails[i]}
            current={current === dn}
            {end}
            style="scroll-margin-block: 2em;"
        />
    {/each}
</div>
