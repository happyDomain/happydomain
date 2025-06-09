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

import { refreshUserSession } from "$lib/stores/usersession";

export class NotAuthorizedError extends Error {
    constructor(message: string) {
        super(message);
        this.name = "NotAuthorizedError";
    }
}

export class ProviderNoDomainListingSupport extends Error {
    constructor(message: string) {
        super(message);
        this.name = "ProviderNoDomainListingSupport";
    }
}

export async function handleEmptyApiResponse(res: Response): Promise<boolean> {
    if (res.status == 204) {
        return true;
    }

    return handleApiResponse<boolean>(res);
}

export async function handleAuthApiResponse<T>(res: Response): Promise<T> {
    if (!res.ok) {
        const data = await res.json();

        if (data.errmsg) {
            switch (data.errmsg) {
                case "the provider doesn't support domain listing":
                    throw new ProviderNoDomainListingSupport(data.errmsg);
                default:
                    throw new Error(data.errmsg);
            }
        } else {
            throw new Error("A " + res.status + " error occurs.");
        }
    }

    return (await res.json()) as T;
}

export async function handleApiResponse<T>(res: Response): Promise<T> {
    if (res.status == 401) {
        try {
            await refreshUserSession();
        } catch (err) {
            if (err instanceof Error) {
                throw new NotAuthorizedError(err.message);
            } else {
                throw err;
            }
        }
    } else {
        return handleAuthApiResponse<T>(res);
    }

    return (await res.json()) as T;
}
