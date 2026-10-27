import { error, type Load } from "@sveltejs/kit";

import { getDomain } from "$lib/api/domains";

// The domains list carries no zone metadata: the history needs the full
// domain, shared by the sidebar timeline and the page.
export const load: Load = async ({ parent }) => {
    const data = await parent();

    if (!data.domain) {
        error(404, { message: "Domain not found" });
    }

    return { historyDomain: await getDomain(data.domain.id) };
};
