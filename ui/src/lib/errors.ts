import { refreshUserSession } from '$lib/stores/usersession';

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
    } else if (!res.ok) {
        const data = await res.json();

        if (data.errmsg) {
            switch (data.errmsg) {
                case "Provider doesn't support domain listing.":
                    throw new ProviderNoDomainListingSupport(data.errmsg);
                default:
                    throw new Error(data.errmsg);
            }
        } else {
            throw new Error("A " + res.status + " error occurs.");
        }
    }

    return await res.json() as T;
}
