import { redirect, type Load } from "@sveltejs/kit";

import { resolve } from "$app/paths";

// Without a selected version, show the latest one.
export const load: Load = async ({ parent, params }) => {
    const data = await parent();

    const latest = data.historyDomain?.zone_history?.[0];
    if (latest) {
        redirect(
            307,
            resolve("/domains/[dn]/history/[zid]", {
                dn: encodeURIComponent(params.dn ?? ""),
                zid: encodeURIComponent(latest),
            }),
        );
    }

    return {};
};
