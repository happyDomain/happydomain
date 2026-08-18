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

import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("$lib/stores/usersession", () => ({
    refreshUserSession: vi.fn(),
}));

import { refreshUserSession } from "$lib/stores/usersession";
import {
    NotAuthorizedError,
    ProviderNoDomainListingSupport,
    handleEmptyApiResponse,
    handleAuthApiResponse,
    handleApiResponse,
} from "./errors";

const mockedRefreshUserSession = vi.mocked(refreshUserSession);

beforeEach(() => {
    mockedRefreshUserSession.mockReset();
});

describe("NotAuthorizedError", () => {
    it("is an instance of Error", () => {
        const err = new NotAuthorizedError("nope");
        expect(err).toBeInstanceOf(Error);
    });

    it("sets the message and name", () => {
        const err = new NotAuthorizedError("nope");
        expect(err.message).toBe("nope");
        expect(err.name).toBe("NotAuthorizedError");
    });
});

describe("ProviderNoDomainListingSupport", () => {
    it("is an instance of Error", () => {
        const err = new ProviderNoDomainListingSupport("no listing");
        expect(err).toBeInstanceOf(Error);
    });

    it("sets the message and name", () => {
        const err = new ProviderNoDomainListingSupport("no listing");
        expect(err.message).toBe("no listing");
        expect(err.name).toBe("ProviderNoDomainListingSupport");
    });
});

describe("handleEmptyApiResponse", () => {
    it("returns true for a 204 response without reading the body", async () => {
        const res = new Response(null, { status: 204 });
        const jsonSpy = vi.spyOn(res, "json");

        await expect(handleEmptyApiResponse(res)).resolves.toBe(true);
        expect(jsonSpy).not.toHaveBeenCalled();
    });

    it("delegates to handleApiResponse for non-204 responses", async () => {
        const res = new Response(JSON.stringify(true), { status: 200 });

        await expect(handleEmptyApiResponse(res)).resolves.toBe(true);
    });

    it("delegates a 401 to the session refresh flow", async () => {
        mockedRefreshUserSession.mockRejectedValue(new Error("session expired"));
        const res = new Response(null, { status: 401 });

        await expect(handleEmptyApiResponse(res)).rejects.toThrow(NotAuthorizedError);
        expect(mockedRefreshUserSession).toHaveBeenCalled();
    });
});

describe("handleAuthApiResponse", () => {
    it("returns the parsed JSON body when the response is ok", async () => {
        const res = new Response(JSON.stringify({ foo: "bar" }), { status: 200 });

        await expect(handleAuthApiResponse<{ foo: string }>(res)).resolves.toEqual({
            foo: "bar",
        });
    });

    it("throws ProviderNoDomainListingSupport for the matching errmsg", async () => {
        const res = new Response(
            JSON.stringify({ errmsg: "the provider doesn't support domain listing" }),
            { status: 400 },
        );

        await expect(handleAuthApiResponse(res)).rejects.toThrow(ProviderNoDomainListingSupport);
    });

    it("throws a generic Error with errmsg for other error messages", async () => {
        const res = new Response(JSON.stringify({ errmsg: "something went wrong" }), {
            status: 400,
        });

        await expect(handleAuthApiResponse(res)).rejects.toThrow("something went wrong");
    });

    it("does not throw ProviderNoDomainListingSupport for an unrelated errmsg", async () => {
        const res = new Response(JSON.stringify({ errmsg: "something went wrong" }), {
            status: 400,
        });

        try {
            await handleAuthApiResponse(res);
            throw new Error("expected handleAuthApiResponse to reject");
        } catch (err) {
            expect(err).not.toBeInstanceOf(ProviderNoDomainListingSupport);
        }
    });

    it("throws a status-based Error when the body has no errmsg", async () => {
        const res = new Response(JSON.stringify({}), { status: 503, statusText: "" });

        await expect(handleAuthApiResponse(res)).rejects.toThrow("A 503 error occurs.");
    });
});

describe("handleApiResponse", () => {
    it("returns the parsed JSON body when the response is ok", async () => {
        const res = new Response(JSON.stringify({ foo: "bar" }), { status: 200 });

        await expect(handleApiResponse<{ foo: string }>(res)).resolves.toEqual({ foo: "bar" });
    });

    it("delegates to handleAuthApiResponse for non-401 error responses", async () => {
        const res = new Response(JSON.stringify({ errmsg: "boom" }), { status: 500 });

        await expect(handleApiResponse(res)).rejects.toThrow("boom");
        expect(mockedRefreshUserSession).not.toHaveBeenCalled();
    });

    it("refreshes the session on a 401 and returns the parsed body on success", async () => {
        mockedRefreshUserSession.mockResolvedValue(undefined as never);
        const res = new Response(JSON.stringify({ foo: "bar" }), { status: 401 });

        await expect(handleApiResponse<{ foo: string }>(res)).resolves.toEqual({ foo: "bar" });
        expect(mockedRefreshUserSession).toHaveBeenCalled();
    });

    it("wraps a refresh failure that is an Error in NotAuthorizedError", async () => {
        mockedRefreshUserSession.mockRejectedValue(new Error("refresh failed"));
        const res = new Response(null, { status: 401 });

        await expect(handleApiResponse(res)).rejects.toThrow(NotAuthorizedError);
        await expect(handleApiResponse(res)).rejects.toThrow("refresh failed");
    });

    it("rethrows a non-Error thrown by the session refresh as-is", async () => {
        mockedRefreshUserSession.mockRejectedValue("plain string failure");
        const res = new Response(null, { status: 401 });

        await expect(handleApiResponse(res)).rejects.toBe("plain string failure");
    });
});
