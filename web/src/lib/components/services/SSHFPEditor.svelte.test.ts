// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { loadTranslations } from "$lib/translations";

const fetchSSHHostKeys = vi.fn();

vi.mock("$lib/api/resolver", () => ({
    fetchSSHHostKeys: (...args: unknown[]) => fetchSSHHostKeys(...args),
}));

import Editor from "./SSHFPEditor.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

beforeEach(() => {
    fetchSSHHostKeys.mockReset();
});

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mount(value: any, props: Record<string, unknown> = {}) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const all: any = { dn: "www", origin: { domain: "example.com." }, value, ...props };
    return render(Editor, all);
}

const HDR = { Name: "", Rrtype: 44, Class: 1, Ttl: 3600 };

const retrieveButton = () => screen.getByRole("button", { name: /retrieve/i });

describe("SSHFP editor: retrieving host keys", () => {
    it("seeds the host from the service and asks on port 22", async () => {
        const user = userEvent.setup();
        fetchSSHHostKeys.mockResolvedValueOnce({ status: "ok", keys: [] });
        mount({ SSHFP: [] });

        expect((screen.getByDisplayValue("www.example.com") as HTMLInputElement).value).toBe(
            "www.example.com",
        );
        await user.click(retrieveButton());

        expect(fetchSSHHostKeys).toHaveBeenCalledWith(
            { host: "www.example.com", port: 22 },
            expect.any(AbortSignal),
        );
    });

    it("imports both fingerprints, skipping the published ones", async () => {
        const user = userEvent.setup();
        const value = { SSHFP: [{ Hdr: HDR, Algorithm: 4, Type: 2, FingerPrint: "AB" }] };
        fetchSSHHostKeys.mockResolvedValueOnce({
            status: "ok",
            keys: [{ algorithm: 4, sha256: "ab", sha1: "CD" }],
        });
        mount(value);

        await user.click(retrieveButton());

        expect(await screen.findByText("1 fingerprint(s) added.")).toBeTruthy();
        expect(value.SSHFP).toEqual([
            { Hdr: HDR, Algorithm: 4, Type: 2, FingerPrint: "AB" },
            { Hdr: HDR, Algorithm: 4, Type: 1, FingerPrint: "cd" },
        ]);
    });

    it("says so when the server offers nothing new", async () => {
        const user = userEvent.setup();
        fetchSSHHostKeys.mockResolvedValueOnce({ status: "ok" });
        mount({ SSHFP: [] });

        await user.click(retrieveButton());

        expect(await screen.findByText(/already published/)).toBeTruthy();
    });

    it("shows a translated sentence for a known status", async () => {
        const user = userEvent.setup();
        fetchSSHHostKeys.mockResolvedValueOnce({ status: "timeout", errorMsg: "raw message" });
        mount({ SSHFP: [] });

        await user.click(retrieveButton());

        expect(await screen.findByText(/did not answer in time/)).toBeTruthy();
        expect(screen.queryByText("raw message")).toBeNull();
    });

    it("falls back on the server message for an unknown status", async () => {
        const user = userEvent.setup();
        fetchSSHHostKeys.mockResolvedValueOnce({ status: "weird", errorMsg: "raw message" });
        mount({ SSHFP: [] });

        await user.click(retrieveButton());

        expect(await screen.findByText("raw message")).toBeTruthy();
    });

    it("shows the message of a failed request", async () => {
        const user = userEvent.setup();
        fetchSSHHostKeys.mockRejectedValueOnce(new Error("network down"));
        mount({ SSHFP: [] });

        await user.click(retrieveButton());

        expect(await screen.findByText("network down")).toBeTruthy();
    });

    it("clears the error of the previous run", async () => {
        const user = userEvent.setup();
        fetchSSHHostKeys.mockRejectedValueOnce(new Error("network down"));
        fetchSSHHostKeys.mockResolvedValueOnce({ status: "ok" });
        mount({ SSHFP: [] });

        await user.click(retrieveButton());
        await screen.findByText("network down");
        await user.click(retrieveButton());

        expect(await screen.findByText(/already published/)).toBeTruthy();
        expect(screen.queryByText("network down")).toBeNull();
    });

    it("disables the button while the host is empty", async () => {
        const user = userEvent.setup();
        mount({ SSHFP: [] });

        await user.clear(screen.getByDisplayValue("www.example.com"));

        expect((retrieveButton() as HTMLButtonElement).disabled).toBe(true);
    });

    it("disables the form while the request is pending", async () => {
        const user = userEvent.setup();
        let resolve: (v: unknown) => void = () => {};
        fetchSSHHostKeys.mockReturnValueOnce(new Promise((r) => (resolve = r)));
        mount({ SSHFP: [] });

        await user.click(retrieveButton());
        expect((retrieveButton() as HTMLButtonElement).disabled).toBe(true);

        resolve({ status: "ok" });
        await screen.findByText(/already published/);
        expect((retrieveButton() as HTMLButtonElement).disabled).toBe(false);
    });

    it("aborts the pending request when unmounted", async () => {
        const user = userEvent.setup();
        let signal: AbortSignal | undefined;
        fetchSSHHostKeys.mockImplementationOnce((_body: unknown, s: AbortSignal) => {
            signal = s;
            return new Promise(() => {});
        });
        const { unmount } = mount({ SSHFP: [] });

        await user.click(retrieveButton());
        unmount();

        expect(signal?.aborted).toBe(true);
    });

    it("offers no retrieval on a readonly service", () => {
        mount({ SSHFP: [] }, { readonly: true });

        expect(screen.queryByRole("button", { name: /retrieve/i })).toBeNull();
    });
});

describe("SSHFP editor: rows", () => {
    it("adds a SHA-256 Ed25519 row", async () => {
        const user = userEvent.setup();
        const value = { SSHFP: [] as unknown[] };
        mount(value);

        await user.click(screen.getByRole("button", { name: /new row|add/i }));

        expect(value.SSHFP).toEqual([{ Hdr: HDR, Algorithm: 4, Type: 2, FingerPrint: "" }]);
    });
});

describe("SSHFP editor: records", () => {
    it("binds the inputs to the record fields", async () => {
        const user = userEvent.setup();
        const value = {
            SSHFP: [{ Hdr: { Rrtype: 44 }, Algorithm: 4, Type: 2, FingerPrint: "ab" }],
        };
        mount(value);

        await user.type(screen.getByDisplayValue("ab"), "cd");

        expect(value.SSHFP[0].FingerPrint).toBe("abcd");
    });
});
