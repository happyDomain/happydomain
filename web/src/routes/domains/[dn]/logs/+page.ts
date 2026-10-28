import { error, type Load } from "@sveltejs/kit";

import { getDomain, getDomainLogs } from "$lib/api/domains";

// The sidebar lists the days of the logs the page shows: both read them
// from here. The full domain names the zone versions the logs refer to.
export const load: Load = async ({ parent }) => {
    const data = await parent();

    if (!data.domain) {
        error(404, { message: "Domain not found" });
    }

    const [logs, historyDomain] = await Promise.all([
        getDomainLogs(data.domain.id),
        getDomain(data.domain.id),
    ]);

    return { logs: logs ?? [], historyDomain };
};
