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

<!--
     Stands in for an icon we could not get: a domain nobody has crawled and
     whose site declares no favicon, a provider that refuses our fetch, or an
     instance configured to fetch no icon at all.

     It is drawn rather than fetched, so it costs no request and works offline.
     The colour derives from the name, which is what makes a list of domains
     without favicons remain readable: every row looks different, and the same
     name always gets the same colour.
-->

<script lang="ts">
    interface Props {
        /** The domain or provider name the monogram stands for. */
        label: string;
        style?: string;
        /** Drawing size in pixels, which behaves like an image's intrinsic size. */
        size?: number;
        [key: string]: unknown;
    }

    let {
        label,
        style = "max-width: 100%; max-height: 2.5em",
        size = 32,
        ...rest
    }: Props = $props();

    // FNV-1a. Any stable hash would do; this one is short, has no dependency
    // and spreads close names (example.com/example.net) across the wheel.
    function hueOf(value: string): number {
        let hash = 0x811c9dc5;

        for (let i = 0; i < value.length; i++) {
            hash ^= value.charCodeAt(i);
            hash = Math.imul(hash, 0x01000193);
        }

        return Math.abs(hash) % 360;
    }

    // The first letter or digit of the label, ignoring any leading
    // punctuation.
    function initialOf(value: string): string {
        return value.match(/\p{L}|\p{N}/u)?.[0].toUpperCase() ?? "";
    }

    let initial = $derived(initialOf(label));
    let hue = $derived(hueOf(label));
</script>

<svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 32 32"
    width={size}
    height={size}
    role="img"
    aria-label={label}
    {style}
    {...rest}
>
    <title>{label}</title>
    <rect width="32" height="32" rx="7" fill="hsl({hue} 45% 42%)" />
    {#if initial}
        <text
            x="16"
            y="17"
            text-anchor="middle"
            dominant-baseline="central"
            fill="#fff"
            font-family="system-ui, -apple-system, 'Segoe UI', sans-serif"
            font-size="17"
            font-weight="600">{initial}</text
        >
    {/if}
</svg>
