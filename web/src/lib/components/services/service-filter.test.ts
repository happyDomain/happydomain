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

    it("allows any record type without provider, as in a zone template", () => {
        const svc = makeSvc({ restrictions: { needTypes: [16] } });
        const result = filterServices([svc], null, NO_SERVICES, "", "");
        expect(result.available).toEqual([svc]);
        expect(result.disabled).toEqual([]);
    });

    it("refuses the origin of a zone without provider, as in a zone template", () => {
        const origin = makeSvc({ _svctype: "abstract.Origin", restrictions: { rootOnly: true } });
        const result = filterServices([origin], null, NO_SERVICES, "", "");
        expect(result.available).toEqual([]);
        expect(result.disabled).toEqual([
            { svc: origin, reason: "belongs to the domain the template is applied to." },
        ]);
    });

    it("refuses the NS of a zone without provider, as in a zone template", () => {
        const ns = makeSvc({ _svctype: "abstract.NSOnlyOrigin", restrictions: { rootOnly: true } });
        const result = filterServices([ns], null, NO_SERVICES, "", "");
        expect(result.available).toEqual([]);
        expect(result.disabled).toEqual([
            { svc: ns, reason: "belongs to the domain the template is applied to." },
        ]);
    });

    it("keeps the other root-only services to the root without provider", () => {
        const svc = makeSvc({ _svctype: "svcs.Root", restrictions: { rootOnly: true } });
        expect(filterServices([svc], null, NO_SERVICES, "", "").available).toEqual([svc]);
        expect(filterServices([svc], null, NO_SERVICES, "www", "").disabled).toEqual([
            { svc, reason: "can only be present at the root of your domain." },
        ]);
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

    it("matches on a keyword, case-insensitively", () => {
        const a = makeSvc({ keywords: ["GitLab Pages", "Stripe"] });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "gitlab");
        expect(result.available).toEqual([a]);
    });

    it("tolerates a null keywords list", () => {
        const a = makeSvc({ keywords: null });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "gitlab");
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

describe("filterServices: relevance order", () => {
    it("puts the exact name match before a description match", () => {
        const caldav = makeSvc({ _svctype: "svcs.CalDAV", name: "CalDAV" });
        const server = makeSvc({ _svctype: "abstract.Server", name: "Server" });
        const translate = translator({
            "svcinfo.svcs.CalDAV.description": "Locate your calendar server",
        });
        const result = filterServices(
            [caldav, server],
            makeProvider(),
            NO_SERVICES,
            "",
            "server",
            null,
            translate,
        );
        expect(result.available).toEqual([server, caldav]);
    });

    it("puts the service holding a typed record type first, whatever its case", () => {
        const caldav = makeSvc({ _svctype: "svcs.CalDAV", name: "CalDAV" });
        const server = makeSvc({ _svctype: "abstract.Server", name: "Server", record_types: [1] });
        for (const term of ["A", "a"]) {
            const result = filterServices([caldav, server], makeProvider(), NO_SERVICES, "", term);
            expect(result.available).toEqual([server, caldav]);
        }
    });

    it("puts a name prefix before a match in the middle of the name", () => {
        const middle = makeSvc({ _svctype: "svcs.Middle", name: "Big Widget" });
        const prefix = makeSvc({ _svctype: "svcs.Prefix", name: "Widget Pro" });
        const result = filterServices([middle, prefix], makeProvider(), NO_SERVICES, "", "widget");
        expect(result.available).toEqual([prefix, middle]);
    });

    it("puts a name match before a keyword match", () => {
        const keyword = makeSvc({ _svctype: "svcs.Keyword", name: "Pages", keywords: ["GitLab"] });
        const named = makeSvc({ _svctype: "svcs.Named", name: "GitLab verification" });
        const result = filterServices([keyword, named], makeProvider(), NO_SERVICES, "", "gitlab");
        expect(result.available).toEqual([named, keyword]);
    });

    it("also orders the disabled list by relevance", () => {
        const caldav = makeSvc({
            _svctype: "svcs.CalDAV",
            name: "CalDAV",
            restrictions: { rootOnly: true },
        });
        const server = makeSvc({
            _svctype: "abstract.Server",
            name: "Server",
            record_types: [1],
            restrictions: { rootOnly: true },
        });
        const result = filterServices([caldav, server], makeProvider(), NO_SERVICES, "sub", "a");
        expect(result.disabled.map((d) => d.svc)).toEqual([server, caldav]);
    });

    it("sorts alphabetically on the translated name without search term", () => {
        const b = makeSvc({ _svctype: "svcs.B", name: "beta", weight: 1 });
        const a = makeSvc({ _svctype: "svcs.A", name: "Alpha", record_types: [1], weight: 99999 });
        const c = makeSvc({ _svctype: "svcs.C", name: "zeta" });
        const translate = translator({ "svcinfo.svcs.C.name": "Gamma" });
        const result = filterServices(
            [c, b, a],
            makeProvider(),
            NO_SERVICES,
            "",
            "",
            null,
            translate,
        );
        expect(result.available).toEqual([a, b, c]);
    });

    it("puts the orphan record last without search term", () => {
        const orphan = makeSvc({ _svctype: "svcs.Orphan", name: "Orphan Record" });
        const srv = makeSvc({ _svctype: "svcs.UnknownSRV", name: "Service Record" });
        const result = filterServices([srv, orphan], makeProvider(), NO_SERVICES, "", "");
        expect(result.available).toEqual([srv, orphan]);
    });
});

describe("filterServices: dashes", () => {
    it("finds a dashed name without the dash", () => {
        const a = makeSvc({ name: "E-mail sender" });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "email");
        expect(result.available).toEqual([a]);
    });

    it("finds an undashed name with a dash", () => {
        const a = makeSvc({ name: "Email sender" });
        const result = filterServices([a], makeProvider(), NO_SERVICES, "", "e-mail");
        expect(result.available).toEqual([a]);
    });

    it("drops dashes in keywords and descriptions too", () => {
        const kw = makeSvc({ _svctype: "svcs.Kw", name: "kw", keywords: ["e-mail"] });
        const desc = makeSvc({ _svctype: "svcs.Desc", name: "desc" });
        const translate = translator({ "svcinfo.svcs.Desc.description": "Secures e-mail" });
        const result = filterServices(
            [kw, desc],
            makeProvider(),
            NO_SERVICES,
            "",
            "email",
            null,
            translate,
        );
        expect(result.available).toEqual([kw, desc]);
    });
});

describe("filterServices: weight", () => {
    it("puts the generic service before the specific ones holding the same record type", () => {
        const autoconfig = makeSvc({
            _svctype: "abstract.EMailAutoconfig",
            name: "Email auto-configuration",
            record_types: [33],
            weight: 1,
        });
        const srv = makeSvc({
            _svctype: "svcs.UnknownSRV",
            name: "Service Record",
            record_types: [33],
            weight: 99999,
        });
        const result = filterServices([autoconfig, srv], makeProvider(), NO_SERVICES, "", "SRV");
        expect(result.available).toEqual([srv, autoconfig]);
    });

    it("puts the orphan record last, despite its weight and a better match", () => {
        const orphan = makeSvc({
            _svctype: "svcs.Orphan",
            name: "Orphan Record",
            family: "",
            weight: 99999999,
        });
        const srv = makeSvc({
            _svctype: "svcs.UnknownSRV",
            name: "Service Record",
            family: "",
            weight: 99999,
        });
        const result = filterServices([orphan, srv], makeProvider(), NO_SERVICES, "", "record");
        expect(result.available).toEqual([srv, orphan]);

        // "o" starts the orphan name, but only sits inside the other one.
        const prefix = filterServices([orphan, srv], makeProvider(), NO_SERVICES, "", "o");
        expect(prefix.available).toEqual([srv, orphan]);
    });

    it("does not let the weight beat a better match", () => {
        const heavy = makeSvc({ _svctype: "svcs.Heavy", name: "Big Widget", weight: 99999 });
        const light = makeSvc({ _svctype: "svcs.Light", name: "Widget", weight: 1 });
        const result = filterServices([heavy, light], makeProvider(), NO_SERVICES, "", "widget");
        expect(result.available).toEqual([light, heavy]);
    });
});
