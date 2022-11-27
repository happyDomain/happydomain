import { refreshUserSession } from '$lib/stores/usersession';

export async function handleApiResponse(res: Response): Promise<any> {
    const data = await res.json();

    if (res.status == 200) {
        return data;
    } else if (res.status == 401) {
        refreshUserSession();
        throw new Error(data.errmsg);
    } else if (data.errmsg) {
        throw new Error(data.errmsg);
    } else {
        throw new Error("A " + res.status + " error occurs.");
    }
}
