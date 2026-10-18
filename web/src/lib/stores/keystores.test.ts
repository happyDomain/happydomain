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
import { effectiveKeystore, eligibleKeystores, keystoreKindLabel, keystoreOf } from "./keystores";

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
        ref: "file",
        scope: "personal",
        name: "My file",
        available: true,
        unattended: true,
        references: true,
        referenceHint: "the key of the secret in your file",
        objects: [{ objectType: "provider", objectId: "p2" }],
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
        expect(eligibleKeystores(list, false).map((ks) => ks.ref)).toEqual([
            "instance",
            "mine",
            "file",
        ]);
    });

    it("offers only those opening without the owner to a background object", () => {
        expect(eligibleKeystores(list, true).map((ks) => ks.ref)).toEqual(["instance", "file"]);
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

describe("effectiveKeystore", () => {
    it("is the keystore chosen", () => {
        expect(effectiveKeystore(list, "provider", "p1", "file", false)?.ref).toBe("file");
    });

    it("is the keystore the object uses, without a choice", () => {
        expect(effectiveKeystore(list, "provider", "p2", "", false)?.references).toBe(true);
    });

    it("is the default keystore for a new object", () => {
        expect(effectiveKeystore(list, "provider", undefined, "", false)?.ref).toBe("instance");
    });

    it("is no keystore when the one chosen is unknown", () => {
        expect(effectiveKeystore(list, "provider", "p1", "nope", false)).toBeUndefined();
    });

    it("is the default keystore for a stored object no keystore claims", () => {
        expect(effectiveKeystore(list, "provider", "p9", "", false)?.ref).toBe("instance");
    });

    it("skips a default keystore needing its owner for a background object", () => {
        const attended: Array<Keystore> = [
            {
                ref: "mine",
                scope: "personal",
                available: true,
                unattended: false,
                default: true,
                objects: [],
            },
        ];
        expect(effectiveKeystore(attended, "channel", undefined, "", true)).toBeUndefined();
    });
});

// Stands for sveltekit-i18n, which echoes the key back when it is unknown.
function translator(translations: Record<string, string>) {
    return (key: string) => translations[key] ?? key;
}

describe("keystoreKindLabel", () => {
    it("names a kind by its translation", () => {
        const t = translator({ "keystores.kinds.instance": "Encrypted by the instance" });
        expect(keystoreKindLabel(t, "instance")).toBe("Encrypted by the instance");
    });

    it("falls back on the identifier of a kind not translated", () => {
        expect(keystoreKindLabel(translator({}), "hsm")).toBe("hsm");
    });
});
