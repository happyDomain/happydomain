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

import { describe, expect, it } from "vitest";

import type { Keystore } from "$lib/api/keystores";
import { eligibleKeystores, keystoreOf } from "./keystores";

const list: Array<Keystore> = [
    {
        ref: "instance",
        scope: "instance",
        available: true,
        unattended: true,
        default: true,
        objects: [],
    },
    {
        ref: "mine",
        scope: "personal",
        name: "Mine",
        available: true,
        unattended: false,
        objects: [{ objectType: "provider", objectId: "p1" }],
    },
    {
        ref: "gone",
        scope: "personal",
        name: "Gone",
        available: false,
        unattended: true,
        objects: [],
    },
];

describe("eligibleKeystores", () => {
    it("offers every available keystore to an object used with its owner", () => {
        expect(eligibleKeystores(list, false).map((ks) => ks.ref)).toEqual(["instance", "mine"]);
    });

    it("offers only those opening without the owner to a background object", () => {
        expect(eligibleKeystores(list, true).map((ks) => ks.ref)).toEqual(["instance"]);
    });
});

describe("keystoreOf", () => {
    it("finds the keystore holding an object", () => {
        expect(keystoreOf(list, "provider", "p1")?.ref).toBe("mine");
    });

    it("tells apart objects of another type", () => {
        expect(keystoreOf(list, "notification-channel", "p1")).toBeUndefined();
    });

    it("finds none for an object not stored yet", () => {
        expect(keystoreOf(list, "provider", undefined)).toBeUndefined();
    });
});
