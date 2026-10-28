<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
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

<script module lang="ts">
    // Depth of a subdomain in the tree: "" → 0, "www" → 1, "a.b" → 2
    export function domainDepth(dn: string): number {
        return dn ? dn.split(".").length : 0;
    }

    // For each subdomain of a sorted list, tells which columns of the tree
    // keep a vertical rail below that row: rails[i][c] is true when a later
    // row at depth c + 1 hangs from the same branch.
    export function treeRails(dns: string[]): boolean[][] {
        const open: boolean[] = [];
        const rails: boolean[][] = new Array(dns.length);

        for (let i = dns.length - 1; i >= 0; i--) {
            const depth = domainDepth(dns[i]);
            rails[i] = Array.from({ length: depth }, (_, c) => !!open[c]);
            // A shallower row closes every deeper branch.
            open.length = depth;
            if (depth > 0) open[depth - 1] = true;
        }

        return rails;
    }
</script>

<script lang="ts">
    import type { Snippet } from "svelte";
    import type { HTMLAnchorAttributes } from "svelte/elements";

    import { fqdn } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";

    interface Props extends HTMLAnchorAttributes {
        dn: string;
        origin: Domain;
        rails?: boolean[];
        current?: boolean;
        muted?: boolean;
        // Shown at the end of the row of dn, as a status or a count.
        end?: Snippet<[string]>;
    }

    let { dn, origin, rails = [], current = false, muted = false, end, ...rest }: Props = $props();

    let depth = $derived(domainDepth(dn));
    let name = $derived(fqdn(dn, origin.domain));
    // The leftmost label stands out, the parent part is faded.
    let head = $derived(dn ? dn.split(".")[0] : name);
</script>

<a
    {...rest}
    title={name}
    class="tree-level position-relative d-flex align-items-center gap-2 font-monospace text-decoration-none"
    class:tree-child={depth > 0}
    class:tree-tee={rails[depth - 1]}
    class:fw-bold={current}
    class:text-body={!muted}
    class:text-muted={muted}
    style:--depth={depth}
    aria-current={current ? "location" : undefined}
>
    {#each rails.slice(0, -1) as rail, col}
        {#if rail}<span class="tree-rail" style:--col={col}></span>{/if}
    {/each}
    <span class="flex-fill text-truncate"
        >{head}<span class="fw-normal opacity-50">{name.slice(head.length)}</span></span
    >
    {@render end?.(dn)}
</a>

<style>
    /* Each level is shifted by --indent; its square connector starts below
       the parent's first character and bends into its own name. */
    .tree-level {
        --indent: 1.25rem;
        --stem: 0.35rem;
        max-width: none;
        padding-left: calc(var(--depth) * var(--indent));
    }

    .tree-level:hover {
        color: var(--bs-primary) !important;
    }

    .tree-child::before,
    .tree-tee::after,
    .tree-rail {
        content: "";
        position: absolute;
        border-left: 1px solid var(--bs-border-color);
    }

    .tree-child::before {
        top: 0;
        left: calc((var(--depth) - 1) * var(--indent) + var(--stem));
        width: calc(var(--indent) - var(--stem) - 0.25rem);
        height: 50%;
        border-bottom: 1px solid var(--bs-border-color);
    }

    /* A later sibling follows: the connector goes on down */
    .tree-tee::after {
        top: 50%;
        bottom: 0;
        left: calc((var(--depth) - 1) * var(--indent) + var(--stem));
    }

    .tree-rail {
        top: 0;
        bottom: 0;
        left: calc(var(--col) * var(--indent) + var(--stem));
    }
</style>
