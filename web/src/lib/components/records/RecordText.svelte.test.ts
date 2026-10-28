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

import { cleanup, render } from "@testing-library/svelte";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

import type { dnsRR } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import { loadTranslations } from "$lib/translations";
import RecordText from "./RecordText.svelte";

const origin = { id: "d1", domain: "example.com." } as unknown as Domain;

function a(name: string, ttl = 3600): dnsRR {
    return { Hdr: { Name: name, Rrtype: 1, Class: 1, Ttl: ttl }, A: "192.0.2.1" } as dnsRR;
}

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => cleanup());

describe("RecordText", () => {
    it("shows the type, the name in the zone, the value and the TTL", () => {
        const { container } = render(RecordText, { dn: "www", origin, rr: a("") });
        expect(container.querySelector(".record-type")).toHaveTextContent("A");
        expect(container.querySelector(".record-name")).toHaveTextContent(/^www$/);
        expect(container.querySelector(".record-value")).toHaveTextContent("192.0.2.1");
        expect(container.querySelector(".record-ttl")).toHaveTextContent("TTL 1h");
    });

    it("names the apex @, and a record under the subdomain by its full path", () => {
        const apex = render(RecordText, { dn: "@", origin, rr: a("") });
        expect(apex.container.querySelector(".record-name")).toHaveTextContent(/^@$/);
        cleanup();

        const sub = render(RecordText, { dn: "www", origin, rr: a("_acme") });
        expect(sub.container.querySelector(".record-name")).toHaveTextContent(/^_acme\.www$/);
    });

    it("tells the whole record on hover, and no TTL when it has none", () => {
        const { container } = render(RecordText, { dn: "www", origin, rr: a("", 0) });
        expect(container.firstElementChild).toHaveAttribute(
            "title",
            expect.stringMatching(/^www\.example\.com\.\s+IN\s+A\s+192\.0\.2\.1$/),
        );
        expect(container.querySelector(".record-ttl")).toBeNull();
    });
});
