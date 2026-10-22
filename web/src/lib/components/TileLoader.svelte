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
     The branded loader: the logo mark itself, its five tiles lighting up one
     after the other, tracing a loop around the mark.

     It is the same drawing as the splash screen in app.html, which has to
     live there because it shows before any script loads: keep both in sync.
-->

<script lang="ts">
    import { t } from "$lib/translations";

    interface Props {
        /** Also draw the wordmark next to the tiles, as on the splash screen. */
        wordmark?: boolean;
        /** Accessible name; defaults to the generic waiting message. */
        label?: string;
        /** CSS width of the drawing; its height follows. */
        width?: string;
        class?: string;
        [key: string]: unknown;
    }

    let { wordmark = false, label, width, class: className = "", ...rest }: Props = $props();

    // Coordinates are those of the full logo (882 x 221); without the wordmark
    // the viewBox just crops to the mark. Tiles are ordered along the tour, not
    // by position: bottom-left, top-right, top-left, bottom-right, centre, then
    // round again.
    const TILES: [number, number][] = [
        [0, 116.73],
        [232.91, 0],
        [0, 0],
        [232.91, 116.73],
        [116.478, 58.3945],
    ];

    // Rotate the tour so it does not always start on the same tile.
    const start = Math.floor(Math.random() * TILES.length);

    function rank(i: number): number {
        return (i - start + TILES.length) % TILES.length;
    }
</script>

<svg
    class="tile-loader {className}"
    viewBox={wordmark ? "0 0 882 221" : "0 0 335.675 219.718"}
    style:width={width ?? (wordmark ? "300px" : "4rem")}
    role="status"
    aria-label={label ?? $t("wait.wait")}
    {...rest}
>
    {#if wordmark}
        <image href="/img/happyDomain-text-stacked.svg" x="409" y="0" width="473" height="221" />
    {/if}
    {#each TILES as [x, y], i}
        <rect
            style:--i={rank(i)}
            {x}
            {y}
            width="102.765"
            height="102.988"
            rx="38.2391"
            ry="38.3217"
        />
    {/each}
</svg>

<style>
    .tile-loader {
        /* The mark barely breathes: it rests at its true size and peaks just
           over it, the colour carrying most of the signal. Tiles sit 13.7
           units apart, and a peak of 1.1 eats 5.1 units on each side, so even
           two neighbours at full swell stay clear of one another. */
        --rest: 1;
        --peak: 1.1;
        --cycle: 1.5s;
        /* Tiles rest on a pale green and flash to the brand green. */
        --tile-rest: #b5e8d3;
        --tile-peak: #00aa7d;
        display: inline-block;
        max-width: 80vw;
        height: auto;
        vertical-align: middle;
        /* Tiles at full swell overhang the viewBox on its left and top. */
        overflow: visible;
    }

    /* On the dark canvas the pale green glares: rest on a deep green instead
       (green-800, the dark theme's accent border), so only the lit tile
       stands out. */
    :global([data-bs-theme="dark"]) .tile-loader {
        --tile-rest: #00543a;
    }

    rect {
        fill: var(--tile-rest);
        transform-box: fill-box;
        transform-origin: center;
        transform: scale(var(--rest));
        animation: pulse var(--cycle) linear infinite;
        animation-delay: calc(var(--i, 0) * var(--cycle) / 5);
    }

    /* Grows and saturates fast, then eases back while the next tile is
       already on its way up, the two overlap, so the tour never stalls. */
    @keyframes pulse {
        0% {
            transform: scale(var(--rest));
            fill: var(--tile-rest);
            animation-timing-function: cubic-bezier(0.34, 1.56, 0.64, 1);
        }
        10% {
            transform: scale(var(--peak));
            fill: var(--tile-peak);
            animation-timing-function: cubic-bezier(0.4, 0, 0.6, 1);
        }
        34%,
        100% {
            transform: scale(var(--rest));
            fill: var(--tile-rest);
        }
    }

    @media (prefers-reduced-motion: reduce) {
        /* Same relay, colour only: nothing moves or resizes. */
        rect {
            --cycle: 3s;
            transform: none;
            animation-name: relay;
        }
    }

    @keyframes relay {
        0%,
        100% {
            fill: var(--tile-rest);
        }
        10% {
            fill: var(--tile-peak);
        }
        34% {
            fill: var(--tile-rest);
        }
    }
</style>
