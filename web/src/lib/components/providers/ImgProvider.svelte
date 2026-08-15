<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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
    import ImgMonogram from "$lib/components/ImgMonogram.svelte";
    import { providers_idx, providersSpecs } from "$lib/stores/providers";

    interface Props {
        id_provider?: string | undefined;
        ptype?: string | undefined;
        style?: string;
        [key: string]: unknown;
    }

    let {
        id_provider = undefined,
        ptype = undefined,
        style = "max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em",
        ...rest
    }: Props = $props();

    // The provider type this instance stands for, whether it was given
    // directly or has to be read from the provider the user configured.
    let type = $derived(ptype || (id_provider ? $providers_idx?.[id_provider]?._srctype : undefined));

    // Remember which type failed rather than a bare flag: the component is
    // reused as lists are filtered, so a flag set once would hide the icon of
    // every provider rendered afterwards.
    let erroredType = $state<string | undefined>(undefined);

    // The name reads better than the type on a monogram: "OVH" rather than
    // "OVHAPI". It is only there once the specs have been loaded.
    let label = $derived((type && $providersSpecs?.[type]?.name) || type || "");
</script>

{#if type && erroredType !== type}
    <img
        src={"/api/providers/_specs/" + type + "/icon.png"}
        alt={type}
        title={type}
        {style}
        {...rest}
        onerror={() => (erroredType = type)}
    />
{:else if type}
    <ImgMonogram {label} {style} {...rest} />
{:else}
    <span {style} {...rest}></span>
{/if}
