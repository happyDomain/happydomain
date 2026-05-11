import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Mock the dependent modules BEFORE importing hey-api so the import resolves
// against the mocked versions. vi.hoisted lets us safely create state shared
// between mock factories and tests.
const { refreshMock } = vi.hoisted(() => ({ refreshMock: vi.fn() }));

vi.mock("$lib/stores/usersession", () => ({
    refreshUserSession: refreshMock,
}));

vi.mock("$lib/stores/config", () => ({
    base: "",
}));

import {
    createClientConfig,
    setRefreshingSession,
    NotAuthorizedError,
    CaptchaRequiredError,
    RateLimitedError,
    ProviderNoDomainListingSupport,
} from "./hey-api";

type Fetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

function jsonResponse(body: unknown, status = 200): Response {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

function emptyResponse(status = 200): Response {
    return new Response(null, { status });
}

function getCustomFetch(): Fetch {
    const config = createClientConfig({});
    return config.fetch as Fetch;
}

describe("customFetch (hey-api)", () => {
    let fetchMock: ReturnType<typeof vi.fn>;
    const realFetch = globalThis.fetch;

    beforeEach(() => {
        fetchMock = vi.fn();
        globalThis.fetch = fetchMock as unknown as typeof fetch;
        refreshMock.mockReset();
        setRefreshingSession(false);
    });

    afterEach(() => {
        globalThis.fetch = realFetch;
    });

    it("passes 2xx responses through untouched", async () => {
        const ok = jsonResponse({ hello: "world" }, 200);
        fetchMock.mockResolvedValueOnce(ok);

        const fn = getCustomFetch();
        const res = await fn("https://example.test/x");

        expect(res.status).toBe(200);
        await expect(res.json()).resolves.toEqual({ hello: "world" });
        expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    it("does not call refresh on a 2xx response", async () => {
        fetchMock.mockResolvedValueOnce(emptyResponse(204));
        await getCustomFetch()("https://example.test/x");
        expect(refreshMock).not.toHaveBeenCalled();
    });

    it("throws RateLimitedError on a 429 with rate_limited payload", async () => {
        fetchMock.mockResolvedValueOnce(
            jsonResponse({ rate_limited: true, errmsg: "Slow down." }, 429),
        );

        await expect(getCustomFetch()("https://example.test/login")).rejects.toBeInstanceOf(
            RateLimitedError,
        );
    });

    it("uses the default RateLimitedError message when errmsg is missing", async () => {
        fetchMock.mockResolvedValueOnce(jsonResponse({ rate_limited: true }, 429));

        await expect(getCustomFetch()("https://example.test/login")).rejects.toMatchObject({
            name: "RateLimitedError",
            message: expect.stringMatching(/too many/i),
        });
    });

    it("throws CaptchaRequiredError on 401 with captcha_required payload", async () => {
        fetchMock.mockResolvedValueOnce(
            jsonResponse({ captcha_required: true, errmsg: "Solve the captcha." }, 401),
        );

        await expect(getCustomFetch()("https://example.test/login")).rejects.toBeInstanceOf(
            CaptchaRequiredError,
        );
        expect(refreshMock).not.toHaveBeenCalled();
    });

    it("on 401 without captcha, refreshes the session and retries the original request", async () => {
        const retryBody = jsonResponse({ ok: 1 }, 200);
        fetchMock
            .mockResolvedValueOnce(jsonResponse({ errmsg: "Unauthorized" }, 401))
            .mockResolvedValueOnce(retryBody);
        refreshMock.mockResolvedValueOnce(undefined);

        const res = await getCustomFetch()("https://example.test/x");

        expect(refreshMock).toHaveBeenCalledTimes(1);
        expect(fetchMock).toHaveBeenCalledTimes(2);
        expect(res.status).toBe(200);
        await expect(res.json()).resolves.toEqual({ ok: 1 });
    });

    it("on 401 → refresh failure, throws NotAuthorizedError (no retry)", async () => {
        fetchMock.mockResolvedValueOnce(jsonResponse({ errmsg: "Unauthorized" }, 401));
        refreshMock.mockRejectedValueOnce(new Error("refresh denied"));

        await expect(getCustomFetch()("https://example.test/x")).rejects.toBeInstanceOf(
            NotAuthorizedError,
        );
        expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    it("on 401 while a refresh is already in progress, fails immediately without recursing", async () => {
        setRefreshingSession(true);
        fetchMock.mockResolvedValueOnce(jsonResponse({ errmsg: "Unauthorized" }, 401));

        await expect(getCustomFetch()("https://example.test/x")).rejects.toBeInstanceOf(
            NotAuthorizedError,
        );
        expect(refreshMock).not.toHaveBeenCalled();
        expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    it("does not loop: if the post-refresh retry returns 401 it is returned as-is, not re-refreshed", async () => {
        // The retry uses raw fetch (not customFetch) so a second 401 surfaces
        // to the caller; this prevents an infinite refresh loop on a persistently-bad session.
        fetchMock
            .mockResolvedValueOnce(jsonResponse({ errmsg: "Unauthorized" }, 401))
            .mockResolvedValueOnce(jsonResponse({ errmsg: "Still nope" }, 401));
        refreshMock.mockResolvedValueOnce(undefined);

        const res = await getCustomFetch()("https://example.test/x");
        expect(res.status).toBe(401);
        expect(refreshMock).toHaveBeenCalledTimes(1);
        expect(fetchMock).toHaveBeenCalledTimes(2);
    });

    it("translates the openapi3filter 'invalid session' 400 into NotAuthorizedError", async () => {
        const body = {
            error: "error in openapi3filter.SecurityRequirementsError: security requirements failed: invalid session",
        };
        fetchMock.mockResolvedValueOnce(jsonResponse(body, 400));

        await expect(getCustomFetch()("https://example.test/x")).rejects.toBeInstanceOf(
            NotAuthorizedError,
        );
    });

    it("translates 'doesn't support domain listing' errmsg into ProviderNoDomainListingSupport", async () => {
        fetchMock.mockResolvedValueOnce(
            jsonResponse({ errmsg: "the provider doesn't support domain listing" }, 502),
        );

        await expect(
            getCustomFetch()("https://example.test/providers/foo/list"),
        ).rejects.toBeInstanceOf(ProviderNoDomainListingSupport);
    });

    it("propagates other 4xx/5xx error responses unchanged when no special case matches", async () => {
        const body = jsonResponse({ errmsg: "boom" }, 500);
        fetchMock.mockResolvedValueOnce(body);

        const res = await getCustomFetch()("https://example.test/x");
        expect(res.status).toBe(500);
    });

    it("propagates a network error (fetch rejection) to the caller", async () => {
        fetchMock.mockRejectedValueOnce(new TypeError("network failure"));

        await expect(getCustomFetch()("https://example.test/x")).rejects.toThrow("network failure");
    });
});

describe("createClientConfig", () => {
    it("merges a baseUrl and the customFetch into the supplied config", () => {
        const config = createClientConfig({ headers: { "X-Test": "y" } } as never);
        expect(typeof config.fetch).toBe("function");
        expect(typeof config.baseUrl).toBe("string");
        expect(config.baseUrl).toContain("/api/");
    });
});
