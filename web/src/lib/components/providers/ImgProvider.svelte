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
    import ImgWithFallback from "$lib/components/ImgWithFallback.svelte";
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
        style = "width: 2.5em; height: 2.5em; object-fit: scale-down;",
        ...rest
    }: Props = $props();

    // The provider type this instance stands for, whether it was given
    // directly or has to be read from the provider the user configured.
    let type = $derived(
        ptype || (id_provider ? $providers_idx?.[id_provider]?._srctype : undefined),
    );

    // The name reads better than the type on a monogram: "OVH" rather than
    // "OVHAPI". It is only there once the specs have been loaded.
    let label = $derived((type && $providersSpecs?.[type]?.name) || type || "");
</script>

<ImgWithFallback
    src={type ? "/api/providers/_specs/" + type + "/icon" : undefined}
    errorKey={type}
    alt={type}
    title={type}
    {label}
    {style}
    {...rest}
/>
