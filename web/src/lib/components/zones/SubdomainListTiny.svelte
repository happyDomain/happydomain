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
    import type { Domain } from "$lib/model/domain";
    import DomainTreeLink, { treeRails } from "./DomainTreeLink.svelte";

    interface Props {
        domains: Array<string>;
        origin: Domain;
    }

    let { domains, origin }: Props = $props();

    let rails = $derived(treeRails(domains));

    let activeDn = $state("");
    let containerEl: HTMLElement | null = $state(null);

    $effect(() => {
        activeDn = "";
        // Built and read out within this scope, never kept as reactive state.
        // eslint-disable-next-line svelte/prefer-svelte-reactivity
        const visibleElements = new Map<string, number>();

        const observer = new IntersectionObserver(
            (entries) => {
                for (const entry of entries) {
                    if (entry.isIntersecting) {
                        visibleElements.set(entry.target.id, entry.boundingClientRect.top);
                    } else {
                        visibleElements.delete(entry.target.id);
                    }
                }

                let bestId = "";
                let bestTop = Infinity;
                for (const [id, top] of visibleElements) {
                    if (top < bestTop) {
                        bestTop = top;
                        bestId = id;
                    }
                }
                if (bestId) activeDn = bestId;
            },
            { rootMargin: "0px 0px -70% 0px", threshold: 0 },
        );

        for (const dn of domains) {
            const id = dn ? dn : "@";
            const el = document.getElementById(id);
            if (el) observer.observe(el);
        }

        return () => observer.disconnect();
    });

    $effect(() => {
        if (activeDn && containerEl) {
            const el = containerEl.querySelector(`a[href="#${activeDn}"]`);
            el?.scrollIntoView({ block: "nearest", behavior: "smooth" });
        }
    });
</script>

<div bind:this={containerEl}>
    {#each domains as dn, i}
        {@const id = dn ? dn : "@"}
        <DomainTreeLink
            href={"#" + id}
            {dn}
            {origin}
            rails={rails[i]}
            current={activeDn === id}
            style="scroll-margin-block: 2em;"
        />
    {/each}
</div>
