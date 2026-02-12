import { type Load } from "@sveltejs/kit";

import { checks, refreshChecks } from "$lib/stores/checks";
import { get } from "svelte/store";

export const load: Load = async ({ parent }) => {
    const data = await parent();

    if (get(checks) === undefined) {
        refreshChecks();
    }

    return {
        ...data,
        isTestsPage: true,
    };
};
