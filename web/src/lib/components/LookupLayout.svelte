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
    /** Duration of the morph between the search and the results layouts. */
    export const MORPH_DURATION = 550;
</script>

<script lang="ts">
    import { untrack, type Snippet } from "svelte";
    import { fade } from "svelte/transition";

    import PageTitle from "$lib/components/PageTitle.svelte";

    interface Props {
        /** Show the results: the form moves into a sidebar */
        expanded: boolean;
        title: string;
        /** Shown above the title in the results layout */
        domain?: string;
        /** Shown below the title in the search layout */
        subtitle?: string;
        search: Snippet;
        /** Uses of the tool, explained below the form in the search layout */
        guide?: {
            title: string;
            intro: string;
            icon: string;
            uses: Array<{ icon: string; title: string; text: string }>;
        };
        /** Results, shown next to the sidebar */
        children?: Snippet;
    }

    let { expanded, title, domain, subtitle, search, guide, children }: Props = $props();

    // Morph between the search layout (form centered) and the results
    // layout (form in a sidebar): the elements keep their DOM node, we
    // measure them before the layout switch and animate them from their
    // old box to their new one (FLIP).
    const EASING = "cubic-bezier(0.65, 0, 0.35, 1)";

    let asideEl: HTMLElement | undefined = $state();
    let headerEl: HTMLElement | undefined = $state();
    let cardEl: HTMLElement | undefined = $state();
    let asideRect: DOMRect | undefined;
    let snapshots: Array<{ el: HTMLElement; rect: DOMRect }> = [];

    $effect.pre(() => {
        void expanded;
        untrack(() => {
            asideRect = asideEl?.getBoundingClientRect();
            snapshots = [headerEl, cardEl]
                .filter((el): el is HTMLElement => el !== undefined)
                .map((el) => ({ el, rect: el.getBoundingClientRect() }));
        });
    });

    $effect(() => {
        void expanded;
        untrack(() => {
            if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

            // Measure every final box before starting any animation: an
            // animated parent would otherwise distort its children's boxes.
            const targets = snapshots.map(({ el, rect }) => ({
                el,
                from: rect,
                to: el.getBoundingClientRect(),
            }));
            // A centered element moves with its own width: find where the
            // layout puts it while it still has its starting width.
            const starts = targets.map(({ el, from }) => {
                const width = el.style.width;
                el.style.width = `${from.width}px`;
                const start = el.getBoundingClientRect();
                el.style.width = width;
                return start;
            });
            targets.forEach(({ el, from, to }, i) => morph(el, from, to, starts[i]));
            snapshots = [];

            // The aside itself takes its final size at once (its children
            // are positioned against it), only its background sweeps.
            if (asideEl && asideRect) sweepBackground(asideEl, asideRect);
            asideRect = undefined;
        });
    });

    function sweepBackground(el: HTMLElement, from: DOMRect): void {
        const to = el.getBoundingClientRect();
        if (Math.abs(from.width - to.width) < 1 || to.width === 0) return;

        el.animate([{ transform: `scaleX(${from.width / to.width})` }, { transform: "none" }], {
            duration: MORPH_DURATION,
            easing: EASING,
            pseudoElement: "::before",
        });
    }

    function morph(el: HTMLElement, from: DOMRect, to: DOMRect, start: DOMRect): void {
        if (
            Math.abs(from.left - to.left) < 1 &&
            Math.abs(from.top - to.top) < 1 &&
            Math.abs(from.width - to.width) < 1
        )
            return;

        const dx = from.left - start.left;
        const dy = from.top - start.top;
        el.animate(
            [
                { transform: `translate(${dx}px, ${dy}px)`, width: `${from.width}px` },
                { transform: "none", width: `${to.width}px` },
            ],
            { duration: MORPH_DURATION, easing: EASING },
        );
    }
</script>

<div class="lookup-layout" class:expanded>
    <aside class="lookup-aside" bind:this={asideEl}>
        <div class="lookup-panel" class:container={!expanded}>
            <div bind:this={headerEl}>
                <PageTitle
                    class="border-0"
                    {title}
                    domain={expanded ? domain : undefined}
                    subtitle={expanded ? undefined : subtitle}
                />
            </div>
            <div class="lookup-stage">
                <div class="card rounded-4 lookup-card" bind:this={cardEl}>
                    <div class="card-body">
                        {@render search()}
                    </div>
                </div>
            </div>
            {#if guide && !expanded}
                <div class="card border-0 bg-body-tertiary" in:fade={{ delay: MORPH_DURATION }}>
                    <div class="card-body p-4 p-md-5">
                        <h2 class="h4 mb-3">
                            <i class="bi bi-{guide.icon} text-primary me-2"></i>{guide.title}
                        </h2>
                        <p class="text-muted">{guide.intro}</p>
                        <div class="row g-4 mt-1">
                            {#each guide.uses as use (use.title)}
                                <div class="col-md-4">
                                    <div class="d-flex gap-3">
                                        <span class="use-icon">
                                            <i class="bi bi-{use.icon}"></i>
                                        </span>
                                        <div>
                                            <h3 class="h6 fw-bold mb-1">{use.title}</h3>
                                            <p class="small text-muted mb-0">{use.text}</p>
                                        </div>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    </div>
                </div>
            {/if}
        </div>
    </aside>
    {#if expanded}
        <section
            class="lookup-results"
            in:fade={{ delay: MORPH_DURATION * 0.5, duration: MORPH_DURATION * 0.5 }}
        >
            {@render children?.()}
        </section>
    {/if}
</div>

<style>
    .lookup-layout {
        flex: 1 1 auto;
        display: flex;
        flex-direction: column;
    }

    .lookup-aside {
        flex: 1 1 auto;
        display: flex;
        flex-direction: column;
        padding-block: 1.25rem;
        position: relative;
        isolation: isolate;
    }

    .lookup-aside::before {
        content: "";
        position: absolute;
        inset: 0;
        z-index: -1;
        transform-origin: left top;
        background-color: var(--bs-light);
        opacity: 0;
        transition: opacity 550ms;
    }

    .expanded .lookup-aside::before {
        opacity: 1;
    }

    /* Search layout: title at the top like every other page, form
       centered in the remaining space. */
    .lookup-panel {
        flex: 1 1 auto;
        display: flex;
        flex-direction: column;
    }

    .lookup-stage {
        flex: 1 1 auto;
        display: flex;
        align-items: center;
        justify-content: center;
        padding-bottom: 3rem;
    }

    .lookup-card {
        width: 100%;
        max-width: 40rem;
        padding: 0.5rem;
        transition: padding 550ms;
    }

    .expanded .lookup-aside {
        flex: 0 0 auto;
        padding: 1rem 0.75rem 3rem;
    }

    .expanded .lookup-panel {
        flex: 0 0 auto;
        display: block;
        position: sticky;
        top: 1rem;
    }

    .expanded .lookup-stage {
        display: block;
        padding-bottom: 0;
    }

    .expanded .lookup-card {
        max-width: none;
        padding: 0;
    }

    .lookup-results {
        flex: 1 1 auto;
        min-width: 0;
        padding: 1rem 0.75rem 3rem;
    }

    .use-icon {
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 2rem;
        height: 2rem;
        border-radius: 50%;
        background: var(--bs-primary);
        color: #fff;
    }

    @media (min-width: 768px) {
        .lookup-layout.expanded {
            flex-direction: row;
        }

        .expanded .lookup-aside {
            width: 33.333%;
        }

        .lookup-results {
            padding-inline: 1.5rem;
        }
    }
</style>
