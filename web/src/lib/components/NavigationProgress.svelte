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
    import { navigating } from "$app/state";

    import { t } from "$lib/translations";

    // Most navigations resolve immediately, showing the bar right away would
    // only make it flash.
    const APPEAR_DELAY = 150;

    let visible = $state(false);

    $effect(() => {
        // Track navigating.to itself so a new navigation started before the
        // previous one finished is treated as a fresh start, restarting the
        // bar's animation instead of reusing a stalled one.
        navigating.to;

        visible = false;

        if (!navigating.to) {
            return;
        }

        const timer = setTimeout(() => (visible = true), APPEAR_DELAY);
        return () => clearTimeout(timer);
    });
</script>

{#key navigating.to}
    {#if visible}
        <div class="nav-progress" role="progressbar" aria-label={$t("wait.wait")}>
            <div class="nav-progress-bar"></div>
        </div>
    {/if}
{/key}

<style>
    .nav-progress {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        z-index: 1090;
        height: 3px;
        background-color: rgba(var(--bs-primary-rgb), 0.15);
    }

    .nav-progress-bar {
        width: 0;
        height: 100%;
        background-color: var(--bs-primary);
        /* The end of the navigation is unpredictable: creep towards, but never
           reach, the end of the track. */
        animation: nav-progress-creep 10s cubic-bezier(0.15, 0.85, 0.25, 1) forwards;
    }

    @keyframes nav-progress-creep {
        from {
            width: 0;
        }
        to {
            width: 90%;
        }
    }

    @media (prefers-reduced-motion: reduce) {
        .nav-progress-bar {
            width: 100%;
            animation: none;
            opacity: 0.5;
        }
    }
</style>
