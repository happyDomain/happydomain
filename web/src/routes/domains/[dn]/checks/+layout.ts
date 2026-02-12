import { type Load } from "@sveltejs/kit";

import { checkers, refreshCheckers } from "$lib/stores/checkers";
import { get } from "svelte/store";

export const load: Load = async ({ parent }) => {
    const data = await parent();

    if (get(checkers) === undefined) {
        refreshCheckers();
    }

    return {
        ...data,
        isTestsPage: true,
    };
};
