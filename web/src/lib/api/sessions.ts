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

import {
    getSessions,
    getSession,
    postSessions,
    putSessionsBySessionId,
    deleteSessionsBySessionId,
    deleteSessions as deleteSdkSessions,
} from "$lib/api-base/sdk.gen";
import type { Session } from "$lib/model/session";
import { unwrapSdkResponse, unwrapEmptyResponse } from "./errors";

export async function listSessions(): Promise<Array<Session>> {
    return unwrapSdkResponse(await getSessions()) as Array<Session>;
}

/**
 * Get the current session.
 * Uses the /session endpoint (singular).
 */
export async function getCurrentSession(): Promise<Session> {
    return unwrapSdkResponse(await getSession()) as unknown as Session;
}

export async function addSession(description: string): Promise<Session> {
    return unwrapSdkResponse(
        await postSessions({
            body: { description },
        }),
    ) as unknown as Session;
}

export async function updateSession(session: Session): Promise<Session> {
    if (!session.id) {
        throw new Error(
            "updateSession requires an existing session id; use addSession to create a new one",
        );
    }
    return unwrapSdkResponse(
        await putSessionsBySessionId({
            path: { sessionId: session.id },
            body: { description: session.description, exp: session.exp },
        }),
    ) as unknown as Session;
}

export async function deleteSession(id: string): Promise<boolean> {
    return unwrapEmptyResponse(
        await deleteSessionsBySessionId({
            path: { sessionId: id },
        }),
    );
}

export async function deleteSessions(): Promise<boolean> {
    return unwrapEmptyResponse(await deleteSdkSessions());
}
