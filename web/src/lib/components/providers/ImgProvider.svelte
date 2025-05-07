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
    import { providers_idx } from "$lib/stores/providers";

    export let id_provider: string | undefined = undefined;
    export let ptype: string | undefined = undefined;

    export let style = "max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em";
</script>

{#if ptype}
    <img
        src={"/api/providers/_specs/" + ptype + "/icon.png"}
        alt={ptype}
        title={ptype}
        {style}
        {...$$restProps}
    />
{:else if id_provider && $providers_idx && $providers_idx[id_provider]}
    <img
        src={"/api/providers/_specs/" + $providers_idx[id_provider]._srctype + "/icon.png"}
        alt={$providers_idx[id_provider]._srctype}
        title={$providers_idx[id_provider]._srctype}
        {style}
        {...$$restProps}
    />
{:else}
    <span {style} {...$$restProps}></span>
{/if}
