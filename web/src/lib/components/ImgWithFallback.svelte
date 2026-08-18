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
    import ImgMonogram from "$lib/components/ImgMonogram.svelte";

    interface Props {
        /** URL to try loading; leave unset to skip straight to the fallback. */
        src?: string;
        /** Identity the fetch is keyed on, so unrelated errors don't cross-contaminate. Defaults to src. */
        errorKey?: string;
        /** Label for the monogram fallback; leave unset to fall back to an empty placeholder instead. */
        label?: string;
        alt?: string;
        title?: string;
        loading?: "eager" | "lazy";
        style?: string;
        [key: string]: unknown;
    }

    let {
        src,
        errorKey = src,
        label = "",
        alt = label,
        title = label,
        loading,
        style = "max-width: 100%; max-height: 2.5em",
        ...rest
    }: Props = $props();

    // Remember which key failed rather than a bare flag: this component is
    // reused as lists are filtered, so a flag set once would hide the icon of
    // every row rendered afterwards.
    let erroredKey = $state<string | undefined>(undefined);
</script>

{#if src && errorKey !== erroredKey}
    <img {src} {alt} {title} {loading} {style} {...rest} onerror={() => (erroredKey = errorKey)} />
{:else if label}
    <ImgMonogram {label} {style} {...rest} />
{:else}
    <span {style} {...rest}></span>
{/if}

<style>
    /*
      Hide alt text while loading image
    */
    img {
        color: transparent;
    }
</style>
