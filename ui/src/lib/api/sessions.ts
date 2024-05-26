// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
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

import { handleEmptyApiResponse, handleApiResponse } from '$lib/errors';
import type { Session } from '$lib/model/session';

export async function listSessions(): Promise<Array<Session>> {
    const res = await fetch('/api/sessions', {headers: {'Accept': 'application/json'}});
    return (await handleApiResponse<Array<Session>>(res));
}

export async function getSession(id: string): Promise<Session> {
    id = encodeURIComponent(id);
    const res = await fetch(`/api/sessions/${id}`, {headers: {'Accept': 'application/json'}});
    return await handleApiResponse<Session>(res);
}

export async function getCurrentSession(): Promise<Session> {
    const res = await fetch('/api/session', {headers: {'Accept': 'application/json'}});
    return await handleApiResponse<Session>(res);
}

export async function addSession(description: string): Promise<Session> {
    const res = await fetch('/api/sessions', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify({
            description,
        }),
    });
    return await handleApiResponse<Session>(res);
}

export async function updateSession(session: Session): Promise<Session> {
    const res = await fetch('/api/sessions' + (session.id ? `/${session.id}` : ''), {
        method: session.id?'PUT':'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(session),
    });
    return await handleApiResponse<Session>(res);
}

export async function deleteSession(id: string): Promise<boolean> {
    const res = await fetch(`/api/sessions/${id}`, {
        method: 'DELETE',
        headers: {'Accept': 'application/json'},
    });
    return await handleEmptyApiResponse(res);
}

export async function deleteSessions(): Promise<boolean> {
    const res = await fetch('/api/sessions', {
        method: 'DELETE',
        headers: {'Accept': 'application/json'},
    });
    return await handleEmptyApiResponse(res);
}
