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

import { get, writable, type Writable } from "svelte/store";

import type { Zone } from "$lib/model/zone";
import { serviceNameOf } from "$lib/services/infos";
import { servicesSpecs } from "$lib/stores/services";
import { toasts } from "$lib/stores/toasts";
import { t } from "$lib/translations";
import { randomUUID } from "$lib/utils/uuid";

/** How long a card stays highlighted after the change that created it. */
export const HIGHLIGHT_DURATION = 2000;

/** Same shape as the `$t` of a component, which `get(t)` types as `unknown`. */
type Translate = (key: string, params?: Record<string, string | number>) => string;

/**
 * Subdomain under which a zone keys its services.
 *
 * The apex is keyed by the empty string, but the zone view hands it around as
 * "@" in places: accept either spelling rather than depend on which one the
 * caller happens to hold.
 */
export function zoneKey(dn: string): string {
    return dn === "@" ? "" : dn;
}

export interface ZoneHighlight {
    /** Subdomain owning the target, as keyed in `Zone.services`. */
    dn: string;
    /** Target service, when the highlight aims at a single card. */
    serviceId?: string;
    /** Bring the target into view, for something the user has not seen yet. */
    scroll: boolean;
    /** Makes two consecutive highlights of the same target distinguishable. */
    token: string;
}

/**
 * Currently highlighted target, if any.
 *
 * Deliberately not derived from, nor reset by, `thisZone`: a zone mutation
 * replaces the whole store (and `getZone` even blanks it in between), so a
 * highlight tied to it would be cleared before it had a chance to show.
 */
export const zoneHighlight: Writable<ZoneHighlight | null> = writable(null);

/**
 * How long a pending highlight waits for its target to appear, before being
 * given up on. Long enough for a slow zone to load, short enough not to fire
 * at whatever the user is looking at by then.
 */
const HIGHLIGHT_EXPIRY = 30000;

/**
 * Points the zone view at a subdomain, or at a single service within it.
 *
 * The highlight stays pending until something displays it: confirming a change
 * usually means navigating back to the zone, and counting down from here would
 * spend the highlight on a page the user cannot see yet.
 */
export function highlightZoneTarget(
    dn: string,
    serviceId?: string,
    { scroll = false }: { scroll?: boolean } = {},
) {
    const token = randomUUID();
    zoneHighlight.set({ dn: zoneKey(dn), serviceId, scroll, token });

    setTimeout(() => forgetHighlight(token), HIGHLIGHT_EXPIRY);
}

function forgetHighlight(token: string) {
    // Leave a newer highlight alone, rather than cutting it short.
    zoneHighlight.update((current) => (current?.token === token ? null : current));
}

/**
 * Starts the countdown of a highlight that is now on screen.
 *
 * Called by whichever element ends up showing it, so that the delay measures
 * how long the user has seen the cue rather than how long ago it was raised.
 */
export function consumeZoneHighlight(token: string) {
    setTimeout(() => forgetHighlight(token), HIGHLIGHT_DURATION);
}

/**
 * Whether the backend answered with a new zone instead of the edited one.
 *
 * Editing a zone that has already been committed or published derivates a new
 * revision (see ActionOnEditableZone, server side), which the user has no way
 * of noticing otherwise.
 */
export function zoneWasForked(before: Zone | null | undefined, after: Zone): boolean {
    return !!before && before.id !== after.id;
}

/**
 * Identifier of the service `after` has under `dn` and `before` had not.
 *
 * Adding a service answers with the whole new zone, not with the created
 * service, so this is the only way to tell which card is the new one.
 */
export function newServiceIdIn(
    before: Zone | null | undefined,
    after: Zone,
    dn: string,
): string | undefined {
    const known = new Set(
        (before?.services?.[dn] ?? []).map((svc) => svc._id).filter((id) => id !== undefined),
    );

    return (after.services?.[dn] ?? []).find((svc) => svc._id && !known.has(svc._id))?._id;
}

export type ServiceChangeKind = "added" | "updated" | "deleted";

interface ServiceChange {
    /** Type of the service, to name it the way the user sees it named. */
    svctype: string;
    dn: string;
    serviceId?: string;
    scroll?: boolean;
    /** A new zone revision was created to hold this change. */
    forked?: boolean;
}

/** Tells the user a service change went through, and points at what changed. */
export function notifyServiceChange(
    kind: ServiceChangeKind,
    { svctype, dn, serviceId, scroll = false, forked = false }: ServiceChange,
) {
    const translate = get(t) as Translate;
    const service = serviceNameOf(translate, get(servicesSpecs)[svctype], svctype);

    toasts.addToast({
        title: translate(`zones.feedback.service-${kind}`, { service }),
        // The change is only staged: say where it stands, and mention the new
        // revision when there is one, as that already implies it is not live.
        message: translate(forked ? "zones.feedback.derived" : "zones.feedback.pending"),
        type: "success",
        timeout: 5000,
    });

    // Nothing left to point at once the service is gone.
    if (kind !== "deleted") highlightZoneTarget(dn, serviceId, { scroll });
}

/** Tells the user the staged changes reached the provider. */
export function notifyZonePublished(nbChanges: number) {
    const translate = get(t) as Translate;

    toasts.addToast({
        title: translate("zones.feedback.published"),
        message: translate("zones.feedback.published-detail", { count: nbChanges }),
        type: "success",
        timeout: 5000,
    });
}
