// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { describe, it, expect } from "vitest";

import { filterServices } from "./service-filter";
import { ServiceInfos } from "$lib/model/service_specs.svelte";
import type { HappydnsService } from "$lib/api-base/types.gen";
import type { ProviderInfos } from "$lib/model/provider";

function makeProvider(overrides: Partial<ProviderInfos> = {}): ProviderInfos {
    return {
        capabilities: [],
        description: "",
        helplink: "",
        name: "",
        website: "",
        ...overrides,
    };
}

function makeSvc(
    overrides: Partial<ConstructorParameters<typeof ServiceInfos>[0]> = {},
): ServiceInfos {
    return new ServiceInfos({
        _svctype: "svcs.Test",
        name: "Test service",
        family: "provider",
        ...overrides,
    });
}

function translator(translations: Record<string, string>) {
    return (key: string) => translations[key] ?? key;
}

const NO_SERVICES: Record<string, Array<HappydnsService>> = {};

describe("filterServices: hidden family", () => {
    it("drops hidden services from both available and disabled", () => {
        const hidden = makeSvc({ _svctype: "svcs.Hidden", family: "hidden" });
        const result = filterServices([hidden], makeProvider(), NO_SERVICES, "", "");
        expect(result.available).toEqual([]);
        expect(result.disabled).toEqual([]);
    });

    it("keeps hidden services out even when they would otherwise match every filter", () => {
        const hidden = makeSvc({ _svctype: "svcs.Hidden", family: "hidden", name: "widget" });
        const result = filterServices(
            [hidden],
            makeProvider(),
            NO_SERVICES,
            "",
            "widget",
            "hidden",
        );
        expect(result.available).toEqual([]);
        expect(result.disabled).toEqual([]);
    });
});

describe("filterServices: available vs disabled split", () => {
    it("puts a service with no restrictions in available", () => {
        const svc = makeSvc();
        const result = filterServices([svc], makeProvider(), NO_SERVICES, "", "");
        expect(result.available).toEqual([svc]);
        expect(result.disabled).toEqual([]);
    });

    it("puts a rootOnly service under a subdomain into disabled with a reason", () => {
        const svc = makeSvc({ restrictions: { rootOnly: true } });
        const result = filterServices([svc], makeProvider(), NO_SERVICES, "www", "");
        expect(result.available).toEqual([]);
        expect(result.disabled).toEqual([
            { svc, reason: "can only be present at the root of your domain." },
        ]);
    });

    it("allows a rootOnly service at the root of the domain", () => {
        const svc = makeSvc({ restrictions: { rootOnly: true } });
        const result = filterServices([svc], makeProvider(), NO_SERVICES, "", "");
        expect(result.available).toEqual([svc]);
        expect(result.disabled).toEqual([]);
    });

    it("puts a service needing an unsupported record type into disabled", () => {
        const svc = makeSvc({ restrictions: { needTypes: [16] } });
        const provider = makeProvider({ capabilities: ["rr-1-a"] });
        const result = filterServices([svc], provider, NO_SERVICES, "", "");
        expect(result.available).toEqual([]);
        expect(result.disabled).toEqual([
            { svc, reason: "is not available on this domain name hosting provider." },
        ]);
    });

    it("allows a service whose needed record type is supported by the provider", () => {
        const svc = makeSvc({ restrictions: { needTypes: [16] } });
        const provider = makeProvider({ capabilities: ["rr-16-txt"] });
        const result = filterServices([svc], provider, NO_SERVICES, "", "");
        expect(result.available).toEqual([svc]);
        expect(result.disabled).toEqual([]);
    });

    it("splits a mixed list into available and disabled", () => {
        const ok = makeSvc({ _svctype: "svcs.Ok", name: "ok" });
        const restricted = makeSvc({
            _svctype: "svcs.Restricted",
            name: "restricted",
            restrictions: { rootOnly: true },
        });
        const result = filterServices([ok, restricted], makeProvider(), NO_SERVICES, "sub", "");
        expect(result.available).toEqual([ok]);
        expect(result.disabled).toEqual([
            { svc: restricted, reason: "can only be present at the root of your domain." },
        ]);
    });
});

describe("filterServices: family filter", () => {
    it("matches every family when filteredFamily is null", () => {
        const a = makeSvc({ _svctype: "svcs.A", family: "provider" });
        const b = makeSvc({ _svctype: "svcs.B", family: "abstract" });
        const result = filterServices([a, b], makeProvider(), NO_SERVICES, "", "", null);
        expect(result.available).toEqual([a, b]);
    });

    it("keeps only services of the requested family", () => {
        const a = makeSvc({ _svctype: "svcs.A", family: "provider" });
        const b = makeSvc({ _svctype: "svcs.B", family: "abstract" });
        const result = filterServices([a, b], makeProvider(), NO_SERVICES, "", "", "abstract");
        expect(result.available).toEqual([b]);
    });

    it("also narrows the disabled list by family", () => {
        const restrictedProvider = makeSvc({
            _svctype: "svcs.A",
            family: "provider",
            restrictions: { rootOnly: true },
        });
        const restrictedAbstract = makeSvc({
            _svctype: "svcs.B",
            family: "abstract",
            restrictions: { rootOnly: true },
        });
        const result = filterServices(
            [restrictedProvider, restrictedAbstract],
            makeProvider(),
            NO_SERVICES,
            "sub",
            "",
            "abstract",
        );
        expect(result.disabled.map((d) => d.svc)).toEqual([restrictedAbstract]);
    });

    it("returns nothing for a family that has no matching services", () => {
        const a = makeSvc({ family: "provider" });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "", "nonexistent");
        expect(result.available).toEqual([]);
        expect(result.disabled).toEqual([]);
    });
});

describe("filterServices: name filter", () => {
    it("matches nothing filter as everything (empty string)", () => {
        const a = makeSvc({ name: "widget" });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "");
        expect(result.available).toEqual([a]);
    });

    it("matches on the service name, case-insensitively", () => {
        const a = makeSvc({ name: "Wonderful Widget" });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "WIDGET");
        expect(result.available).toEqual([a]);
    });

    it("excludes services whose name does not match", () => {
        const a = makeSvc({ name: "Wonderful Widget" });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "gizmo");
        expect(result.available).toEqual([]);
    });

    it("matches on the translated description", () => {
        const a = makeSvc({ _svctype: "svcs.Widget", name: "widget" });
        const translate = translator({
            "svcinfo.svcs.Widget.description": "Handles gizmos and gadgets",
        });
        const result = filterServices(
            [a],
            makeProvider(),
            NO_SERVICES,
            "",
            "gizmo",
            null,
            translate,
        );
        expect(result.available).toEqual([a]);
    });

    it("matches on a record type via its DNS mnemonic", () => {
        const a = makeSvc({ record_types: [16] }); // 16 == TXT
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "txt");
        expect(result.available).toEqual([a]);
    });

    it("does not match an unrelated record type", () => {
        const a = makeSvc({ record_types: [1] }); // A
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "txt");
        expect(result.available).toEqual([]);
    });

    it("tolerates a null record_types list", () => {
        const a = makeSvc({ record_types: null });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "txt");
        expect(result.available).toEqual([]);
    });

    it("matches on a category", () => {
        const a = makeSvc({ categories: ["Email", "Security"] });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "security");
        expect(result.available).toEqual([a]);
    });

    it("tolerates a null categories list", () => {
        const a = makeSvc({ categories: null });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "security");
        expect(result.available).toEqual([]);
    });

    it("also filters the disabled list by name", () => {
        const restricted = makeSvc({ name: "Wonderful Widget", restrictions: { rootOnly: true } });
        const match = filterServices([restricted], makeProvider(), NO_SERVICES, "sub", "widget");
        expect(match.disabled).toHaveLength(1);

        const noMatch = filterServices([restricted], makeProvider(), NO_SERVICES, "sub", "gizmo");
        expect(noMatch.disabled).toEqual([]);
    });
});

describe("filterServices: combined family and name filters (AND semantics)", () => {
    it("requires both the family and the name to match", () => {
        const a = makeSvc({ _svctype: "svcs.A", family: "provider", name: "widget" });
        const b = makeSvc({ _svctype: "svcs.B", family: "abstract", name: "widget" });
        const c = makeSvc({ _svctype: "svcs.C", family: "provider", name: "gizmo" });

        const result = filterServices(
            [a, b, c],
            makeProvider(),
            NO_SERVICES,
            "",
            "widget",
            "provider",
        );
        expect(result.available).toEqual([a]);
    });
});

describe("filterServices: translate callback", () => {
    it("defaults to identity, which falls back on the raw service name (translation misses)", () => {
        const a = makeSvc({ _svctype: "svcs.Widget", name: "Le Gadget" });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "gadget");
        expect(result.available).toEqual([a]);
    });

    it("prefers the translated name over the raw one when a translation is provided", () => {
        const a = makeSvc({ _svctype: "svcs.Widget", name: "Widget" });
        const translate = translator({ "svcinfo.svcs.Widget.name": "Gadget" });
        const result = filterServices(
            [a],
            makeProvider(),
            NO_SERVICES,
            "",
            "gadget",
            null,
            translate,
        );
        expect(result.available).toEqual([a]);
    });

    it("does not match the raw name once a (different) translation exists", () => {
        const a = makeSvc({ _svctype: "svcs.Widget", name: "Widget" });
        const translate = translator({ "svcinfo.svcs.Widget.name": "Gadget" });
        const result = filterServices(
            [a],
            makeProvider(),
            NO_SERVICES,
            "",
            "widget",
            null,
            translate,
        );
        expect(result.available).toEqual([]);
    });
});

describe("filterServices: full scenario", () => {
    it("filters a heterogeneous list of hidden, available, and disabled services", () => {
        const hidden = makeSvc({
            _svctype: "svcs.Hidden",
            family: "hidden",
            name: "widget hidden",
        });
        const availableMatch = makeSvc({
            _svctype: "svcs.Avail",
            family: "provider",
            name: "widget available",
        });
        const availableNoMatch = makeSvc({
            _svctype: "svcs.AvailOther",
            family: "provider",
            name: "gizmo",
        });
        const disabledMatch = makeSvc({
            _svctype: "svcs.Disabled",
            family: "provider",
            name: "widget disabled",
            restrictions: { rootOnly: true },
        });

        const result = filterServices(
            [hidden, availableMatch, availableNoMatch, disabledMatch],
            makeProvider(),
            NO_SERVICES,
            "sub",
            "widget",
            "provider",
        );

        expect(result.available).toEqual([availableMatch]);
        expect(result.disabled).toEqual([
            { svc: disabledMatch, reason: "can only be present at the root of your domain." },
        ]);
    });
});
