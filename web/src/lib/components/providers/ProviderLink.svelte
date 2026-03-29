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

<script lang="ts">
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import { providers_idx, providersSpecs } from "$lib/stores/providers";

    interface Props {
        id_provider: string;
        onclick?: (e: MouseEvent) => void;
    }

    let { id_provider, onclick }: Props = $props();
</script>

{#if $providers_idx && $providers_idx[id_provider]}
    {@const provider = $providers_idx[id_provider]}
    <a
        href="/providers/{encodeURIComponent(id_provider)}"
        class="d-flex align-items-center gap-2 text-decoration-none"
        {onclick}
    >
        <ImgProvider
            {id_provider}
            style="max-width: 1.5em; max-height: 1.5em; object-fit: contain;"
        />
        {#if provider._comment}
            {provider._comment}
        {:else if $providersSpecs && $providersSpecs[provider._srctype]}
            {$providersSpecs[provider._srctype].name}
        {:else}
            {provider._srctype}
        {/if}
    </a>
{:else}
    <em class="text-muted">{id_provider}</em>
{/if}
