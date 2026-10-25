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
    import type { DomainWatchPhase } from "$lib/model/watchlist";
    import { t } from "$lib/translations";
    import { LIFECYCLE_STEPS, PHASE_STYLE, lifecycleStep } from "$lib/utils/watchlist";

    interface Props {
        phase?: DomainWatchPhase;
    }

    let { phase }: Props = $props();

    let current = $derived(lifecycleStep(phase));

    // An available domain may never have been registered: do not pretend it
    // went through the previous steps.
    let walked = $derived(phase === "available" ? 0 : current);

    // A registered domain is no alarm: show it in the primary color.
    let currentColor = $derived(
        !phase || phase === "registered" ? "primary" : PHASE_STYLE[phase].color,
    );
</script>

<ol class="lifecycle list-unstyled d-flex mb-0" aria-label={$t("watchlist.lifecycle")}>
    {#each LIFECYCLE_STEPS as step, i (step)}
        <li
            class="step"
            class:current={i === current}
            title={$t(`watchlist.phase-descriptions.${step}`)}
            aria-current={i === current ? "step" : undefined}
        >
            <div class="progress">
                {#if i === current}
                    <div
                        class="progress-bar w-100 bg-{currentColor}"
                        class:progress-bar-striped={phase !== "available"}
                    ></div>
                {:else if i < walked}
                    <div class="progress-bar w-100 bg-primary bg-opacity-50"></div>
                {/if}
            </div>
            <span class="label text-truncate" class:text-body={i <= current}>
                {$t(`watchlist.steps.${step}`)}
            </span>
        </li>
    {/each}
</ol>

<style>
    .lifecycle {
        gap: 3px;
    }

    .step {
        flex: 1;
        display: flex;
        flex-direction: column;
        min-width: 0;
        cursor: help;
    }

    .progress {
        --bs-progress-height: 0.4rem;
    }

    /* The current step stands out: taller, in the color of its phase. */
    .step.current .progress {
        --bs-progress-height: 0.6rem;
        margin-top: -0.1rem;
    }

    .label {
        margin-top: 0.3rem;
        font-size: 0.7rem;
        line-height: 1.1;
        color: var(--bs-secondary-color);
    }

    .step.current .label {
        margin-top: 0.2rem;
        font-weight: 600;
    }

    /* On narrow cards, only the current step keeps its label. */
    @media (max-width: 480px) {
        .step:not(.current) .label {
            visibility: hidden;
        }
    }
</style>
