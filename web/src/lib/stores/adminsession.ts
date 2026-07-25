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

import { get, writable } from "svelte/store";

const STORAGE_KEY = "happydomain_admin_session";

interface AdminSession {
    token: string;
    // expiresAt is the token expiry as an epoch timestamp in milliseconds.
    expiresAt: number;
}

function loadSession(): AdminSession | null {
    if (typeof sessionStorage === "undefined") return null;

    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;

    try {
        const s = JSON.parse(raw) as AdminSession;
        if (!s.token || !s.expiresAt || s.expiresAt <= Date.now()) {
            sessionStorage.removeItem(STORAGE_KEY);
            return null;
        }
        return s;
    } catch {
        sessionStorage.removeItem(STORAGE_KEY);
        return null;
    }
}

export const adminSession = writable<AdminSession | null>(loadSession());

export function setAdminToken(token: string, expiresAt: string | number) {
    const exp =
        typeof expiresAt === "number" ? expiresAt : new Date(expiresAt).getTime();
    const session: AdminSession = { token, expiresAt: exp };

    if (typeof sessionStorage !== "undefined") {
        sessionStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    }
    adminSession.set(session);
}

export function clearAdminToken() {
    if (typeof sessionStorage !== "undefined") {
        sessionStorage.removeItem(STORAGE_KEY);
    }
    adminSession.set(null);
}

// getAdminToken returns the current bearer token, or null when there is none or
// it has expired (clearing it as a side effect in the latter case).
export function getAdminToken(): string | null {
    const s = get(adminSession);
    if (!s) return null;
    if (s.expiresAt <= Date.now()) {
        clearAdminToken();
        return null;
    }
    return s.token;
}

export function isAdminTokenValid(): boolean {
    return getAdminToken() !== null;
}
