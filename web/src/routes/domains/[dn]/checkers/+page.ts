import type { Load } from "@sveltejs/kit";

import { listScopedCheckers } from "$lib/api/checkers";

export const load: Load = async ({ parent, fetch }) => {
    const data = await parent();

    // Left pending on purpose: the page shows a spinner meanwhile.
    const checkerStatuses = listScopedCheckers(
        { domainId: data.domain.id },
        { withAvailables: true },
        fetch,
    );

    return { ...data, checkerStatuses };
};
