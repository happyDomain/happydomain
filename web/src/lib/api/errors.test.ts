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

import { describe, expect, it } from "vitest";

import { unwrapEmptyResponse, unwrapSdkResponse } from "./errors";

describe("unwrapSdkResponse", () => {
    it("returns the data on success", () => {
        expect(unwrapSdkResponse({ data: { hello: "world" } })).toEqual({ hello: "world" });
    });

    it("throws the error object as-is when it is already an Error", () => {
        const err = new Error("boom");
        expect(() => unwrapSdkResponse({ error: err })).toThrow(err);
    });

    it("throws with the errmsg field when the error carries one", () => {
        expect(() => unwrapSdkResponse({ error: { errmsg: "domain not found" } })).toThrow(
            "domain not found",
        );
    });

    it("throws a generic error for any other error shape", () => {
        expect(() => unwrapSdkResponse({ error: "nope" })).toThrow("nope");
    });

    it("prefers the error over any data present alongside it", () => {
        expect(() =>
            unwrapSdkResponse({ error: { errmsg: "conflict" }, data: { hello: "world" } }),
        ).toThrow("conflict");
    });

    it("returns the data on a 204 No Content response", () => {
        const result = unwrapSdkResponse({
            data: undefined,
            response: { status: 204 } as Response,
        });
        expect(result).toBeUndefined();
    });

    it("throws when there is neither data nor error", () => {
        expect(() => unwrapSdkResponse({})).toThrow("SDK response contains neither data nor error");
    });
});

describe("unwrapEmptyResponse", () => {
    it("returns true when the response is ok", () => {
        expect(unwrapEmptyResponse({ response: { ok: true } as Response })).toBe(true);
    });

    it("returns true when data is present, regardless of response.ok", () => {
        expect(unwrapEmptyResponse({ data: {}, response: { ok: false } as Response })).toBe(true);
    });

    it("throws the error object as-is when it is already an Error", () => {
        const err = new Error("boom");
        expect(() => unwrapEmptyResponse({ error: err })).toThrow(err);
    });

    it("throws with the errmsg field when the error carries one", () => {
        expect(() => unwrapEmptyResponse({ error: { errmsg: "forbidden" } })).toThrow("forbidden");
    });

    it("throws a generic error for any other error shape", () => {
        expect(() => unwrapEmptyResponse({ error: 42 })).toThrow("42");
    });

    it("throws when there is neither data, a successful response, nor an error", () => {
        expect(() => unwrapEmptyResponse({ response: { ok: false } as Response })).toThrow(
            "SDK response contains neither data nor error",
        );
    });

    it("throws when the response object is entirely absent", () => {
        expect(() => unwrapEmptyResponse({})).toThrow(
            "SDK response contains neither data nor error",
        );
    });
});
