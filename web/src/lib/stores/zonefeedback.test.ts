import { describe, it, expect, beforeEach, afterEach, vitest } from "vitest";
import { get } from "svelte/store";

import {
    HIGHLIGHT_DURATION,
    consumeZoneHighlight,
    highlightZoneTarget,
    newServiceIdIn,
    zoneHighlight,
    zoneKey,
    zoneWasForked,
} from "./zonefeedback";
import { makeService, makeZone } from "$lib/test-utils/fixtures";

describe("zoneWasForked", () => {
    it("reports a fork when the server answered with another zone", () => {
        const before = makeZone({ id: "zone-1" });
        const after = makeZone({ id: "zone-2" });
        expect(zoneWasForked(before, after)).toBe(true);
    });

    it("stays quiet when the same zone was edited in place", () => {
        const before = makeZone({ id: "zone-1" });
        const after = makeZone({ id: "zone-1" });
        expect(zoneWasForked(before, after)).toBe(false);
    });

    it("claims nothing when there is no zone to compare against", () => {
        expect(zoneWasForked(null, makeZone({ id: "zone-1" }))).toBe(false);
    });
});

describe("newServiceIdIn", () => {
    const known = makeService("svcs.SPF", {}, { _id: "svc-1", _domain: "www" });
    const added = makeService("svcs.CAA", {}, { _id: "svc-2", _domain: "www" });

    it("finds the service the zone gained", () => {
        const before = makeZone({ services: { www: [known] } });
        const after = makeZone({ services: { www: [known, added] } });
        expect(newServiceIdIn(before, after, "www")).toBe("svc-2");
    });

    it("finds the first service of a brand new subdomain", () => {
        const before = makeZone({ services: {} });
        const after = makeZone({ services: { www: [added] } });
        expect(newServiceIdIn(before, after, "www")).toBe("svc-2");
    });

    it("returns nothing when the subdomain gained no service", () => {
        const before = makeZone({ services: { www: [known] } });
        const after = makeZone({ services: { www: [known] } });
        expect(newServiceIdIn(before, after, "www")).toBeUndefined();
    });

    it("ignores services added to another subdomain", () => {
        const before = makeZone({ services: { www: [known] } });
        const after = makeZone({ services: { www: [known], blog: [added] } });
        expect(newServiceIdIn(before, after, "www")).toBeUndefined();
    });
});

describe("highlightZoneTarget", () => {
    beforeEach(() => {
        vitest.useFakeTimers();
        zoneHighlight.set(null);
    });

    afterEach(() => {
        vitest.useRealTimers();
    });

    it("points at the target", () => {
        highlightZoneTarget("www", "svc-1", { scroll: true });

        expect(get(zoneHighlight)).toMatchObject({
            dn: "www",
            serviceId: "svc-1",
            scroll: true,
        });
    });

    it("keeps waiting while the zone is on its way back", () => {
        highlightZoneTarget("www", "svc-1");

        // Navigating back can outlast the time the cue is meant to show for:
        // nothing has displayed it yet, so it must still be pending.
        vitest.advanceTimersByTime(HIGHLIGHT_DURATION * 3);
        expect(get(zoneHighlight)).toMatchObject({ dn: "www", serviceId: "svc-1" });
    });

    it("clears once its target has shown it", () => {
        highlightZoneTarget("www", "svc-1");
        const { token } = get(zoneHighlight)!;

        consumeZoneHighlight(token);
        vitest.advanceTimersByTime(HIGHLIGHT_DURATION);
        expect(get(zoneHighlight)).toBeNull();
    });

    it("gives up on a target that never appears", () => {
        highlightZoneTarget("www", "svc-1");

        vitest.advanceTimersByTime(60000);
        expect(get(zoneHighlight)).toBeNull();
    });

    it("does not cut short a highlight raised in the meantime", () => {
        highlightZoneTarget("www", "svc-1");
        const first = get(zoneHighlight)!.token;

        // A second change lands before the first one was ever displayed.
        highlightZoneTarget("blog", "svc-2");

        consumeZoneHighlight(first);
        vitest.advanceTimersByTime(HIGHLIGHT_DURATION);
        expect(get(zoneHighlight)).toMatchObject({ dn: "blog", serviceId: "svc-2" });
    });

    it("keys the apex the way the zone does, whichever spelling it was given", () => {
        highlightZoneTarget("@", "svc-1");
        expect(get(zoneHighlight)).toMatchObject({ dn: "" });
    });
});

describe("zoneKey", () => {
    it("reads both spellings of the apex as the zone's own key", () => {
        expect(zoneKey("@")).toBe("");
        expect(zoneKey("")).toBe("");
    });

    it("leaves a regular subdomain alone", () => {
        expect(zoneKey("www")).toBe("www");
    });
});
