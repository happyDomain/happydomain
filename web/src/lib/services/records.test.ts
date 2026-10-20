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

import { describe, expect, it } from "vitest";

import { ensureTrailingDot, isStubRecord } from "./records";

describe("isStubRecord", () => {
    it("recognizes the empty records the backend pre-allocates", () => {
        expect(isStubRecord({})).toBe(true);
        expect(isStubRecord({ Hdr: {} })).toBe(true);
        expect(isStubRecord({ Hdr: { Name: "" } })).toBe(true);
    });

    it("keeps records that carry an owner name", () => {
        expect(isStubRecord({ Hdr: { Name: "autoconfig" } })).toBe(false);
    });

    it("is not a stub when there is no record at all", () => {
        expect(isStubRecord(undefined)).toBe(false);
        expect(isStubRecord(null)).toBe(false);
    });
});

describe("ensureTrailingDot", () => {
    it("appends the dot of a fully qualified name", () => {
        expect(ensureTrailingDot("happydomain.example.com")).toBe("happydomain.example.com.");
    });

    it("does not double an existing dot", () => {
        expect(ensureTrailingDot("happydomain.example.com.")).toBe("happydomain.example.com.");
    });

    it("leaves an empty host empty", () => {
        expect(ensureTrailingDot("")).toBe("");
    });
});
