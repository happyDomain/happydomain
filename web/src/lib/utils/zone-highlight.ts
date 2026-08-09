// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import type { Action } from "svelte/action";

import {
    consumeZoneHighlight,
    zoneHighlight,
    zoneKey,
    type ZoneHighlight,
} from "$lib/stores/zonefeedback";

export interface HighlightTargetParams {
    dn: string;
    serviceId?: string;
}

const HIGHLIGHT_CLASS = "zone-highlight";

function prefersReducedMotion(): boolean {
    return (
        typeof window !== "undefined" &&
        window.matchMedia("(prefers-reduced-motion: reduce)").matches
    );
}

function matches(highlight: ZoneHighlight | null, params: HighlightTargetParams): boolean {
    if (!highlight || highlight.dn !== zoneKey(params.dn)) return false;

    // A highlight without a service aims at the subdomain as a whole.
    return highlight.serviceId === undefined || highlight.serviceId === params.serviceId;
}

/**
 * Marks its element as the target of the current zone change, if it is one.
 *
 * The zone view rebuilds every card on each mutation, so the mark has to be
 * read back from the store on mount rather than applied to the element by
 * whoever raised it.
 */
export const highlightTarget: Action<HTMLElement, HighlightTargetParams> = (node, params) => {
    let current = params;
    let latest: ZoneHighlight | null = null;
    let shownFor: string | undefined;

    function apply() {
        const active = matches(latest, current);
        node.classList.toggle(HIGHLIGHT_CLASS, active);

        // Everything below belongs to showing a highlight for the first time,
        // however often the store or the parameters change while it lasts.
        if (!active || !latest || shownFor === latest.token) return;
        shownFor = latest.token;

        // The countdown starts here rather than where the highlight was
        // raised: getting back to the zone can take a while, and the cue has
        // to outlive the trip.
        consumeZoneHighlight(latest.token);

        if (latest.scroll) {
            // Landing here from another page, the router still has its own
            // scroll reset to apply: wait it out rather than fight it.
            requestAnimationFrame(() =>
                node.scrollIntoView({
                    block: "center",
                    behavior: prefersReducedMotion() ? "auto" : "smooth",
                }),
            );
        }
    }

    const unsubscribe = zoneHighlight.subscribe((highlight) => {
        latest = highlight;
        apply();
    });

    return {
        // The cards are rebuilt in place on every mutation, so an element can
        // be handed a different service than the one it was created for.
        update(next: HighlightTargetParams) {
            current = next;
            apply();
        },
        destroy() {
            unsubscribe();
            node.classList.remove(HIGHLIGHT_CLASS);
        },
    };
};
