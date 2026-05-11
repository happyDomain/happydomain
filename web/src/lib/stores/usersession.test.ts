import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";

vi.mock("$lib/api-base/sdk.gen", () => ({
    getAuth: vi.fn(),
}));

const setRefreshingMock = vi.hoisted(() => vi.fn());
vi.mock("$lib/hey-api", () => ({
    setRefreshingSession: setRefreshingMock,
}));

import { userSession, refreshUserSession } from "./usersession";
import { getAuth } from "$lib/api-base/sdk.gen";

const getAuthMock = vi.mocked(getAuth);

describe("userSession store", () => {
    beforeEach(() => {
        userSession.set({} as never);
        getAuthMock.mockReset();
        setRefreshingMock.mockReset();
    });

    it("starts as an empty object", () => {
        expect(get(userSession)).toEqual({});
    });

    it("refreshUserSession populates the store on a successful auth response", async () => {
        const user = { id: "u1", email: "a@b.test" };
        getAuthMock.mockResolvedValueOnce({
            data: user,
            response: new Response(null, { status: 200 }),
        } as never);

        const result = await refreshUserSession();

        expect(result).toEqual(user);
        expect(get(userSession)).toEqual(user);
    });

    it("refreshUserSession resets the store and rethrows on auth failure", async () => {
        userSession.set({ id: "stale" } as never);
        getAuthMock.mockResolvedValueOnce({
            error: { errmsg: "session expired" },
        } as never);

        await expect(refreshUserSession()).rejects.toThrow("session expired");
        expect(get(userSession)).toEqual({});
    });

    it("refreshUserSession toggles the refreshing flag on entry and exit", async () => {
        getAuthMock.mockResolvedValueOnce({
            data: { id: "u1" },
            response: new Response(null, { status: 200 }),
        } as never);

        await refreshUserSession();

        expect(setRefreshingMock).toHaveBeenNthCalledWith(1, true);
        expect(setRefreshingMock).toHaveBeenLastCalledWith(false);
    });

    it("refreshUserSession releases the refreshing flag even on failure", async () => {
        getAuthMock.mockResolvedValueOnce({ error: { errmsg: "no" } } as never);

        await expect(refreshUserSession()).rejects.toThrow();
        expect(setRefreshingMock).toHaveBeenCalledWith(true);
        expect(setRefreshingMock).toHaveBeenLastCalledWith(false);
    });
});
