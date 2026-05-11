import { describe, it, expect, vi } from "vitest";

// errors.ts re-exports from $lib/hey-api, which transitively loads
// $lib/stores/usersession and the generated API client. Mock the deeper deps
// so this test stays focused on the unwrap logic.
vi.mock("$lib/stores/usersession", () => ({
    refreshUserSession: vi.fn(),
}));
vi.mock("$lib/stores/config", () => ({ base: "" }));

import { unwrapSdkResponse, unwrapEmptyResponse } from "./errors";

function fakeResponse(status: number): Response {
    return new Response(null, { status });
}

describe("unwrapSdkResponse", () => {
    it("returns data on a normal 200 response", () => {
        const result = unwrapSdkResponse({
            data: { id: 42, name: "demo" },
            response: fakeResponse(200),
        });
        expect(result).toEqual({ id: 42, name: "demo" });
    });

    it("returns the data field even when status is 204 No Content", () => {
        const result = unwrapSdkResponse({
            data: undefined,
            response: fakeResponse(204),
        });
        expect(result).toBeUndefined();
    });

    it("rethrows when error is an Error instance", () => {
        const e = new Error("boom");
        expect(() => unwrapSdkResponse({ error: e })).toThrow(e);
    });

    it("converts an { errmsg } payload into a thrown Error with that message", () => {
        expect(() => unwrapSdkResponse({ error: { errmsg: "bad thing" } })).toThrow("bad thing");
    });

    it("falls back to String(error) for non-Error / non-errmsg payloads", () => {
        expect(() => unwrapSdkResponse({ error: "raw string" })).toThrow("raw string");
        expect(() => unwrapSdkResponse({ error: 42 })).toThrow("42");
    });

    it("throws when there is neither data nor error", () => {
        expect(() => unwrapSdkResponse({})).toThrow(/neither data nor error/i);
    });

    it("checks error before data (an error wins even when data is set)", () => {
        expect(() => unwrapSdkResponse({ data: { ok: true }, error: { errmsg: "fail" } })).toThrow(
            "fail",
        );
    });
});

describe("unwrapEmptyResponse", () => {
    it("returns true for an ok response with no data", () => {
        expect(unwrapEmptyResponse({ response: fakeResponse(204) })).toBe(true);
    });

    it("returns true when data is present and response is ok", () => {
        expect(unwrapEmptyResponse({ data: { foo: 1 }, response: fakeResponse(200) })).toBe(true);
    });

    it("returns true when data is present even without a response object", () => {
        expect(unwrapEmptyResponse({ data: { foo: 1 } })).toBe(true);
    });

    it("rethrows when error is an Error instance", () => {
        const e = new Error("boom");
        expect(() => unwrapEmptyResponse({ error: e })).toThrow(e);
    });

    it("converts an { errmsg } payload into a thrown Error", () => {
        expect(() => unwrapEmptyResponse({ error: { errmsg: "rejected" } })).toThrow("rejected");
    });

    it("falls back to String(error) for unstructured errors", () => {
        expect(() => unwrapEmptyResponse({ error: "nope" })).toThrow("nope");
    });

    it("throws when there is no data, no error, and no ok response", () => {
        expect(() => unwrapEmptyResponse({})).toThrow(/neither data nor error/i);
    });
});
