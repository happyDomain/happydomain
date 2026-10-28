import type { Load } from "@sveltejs/kit";
import { get } from "svelte/store";

import { checkers, refreshCheckers } from "$lib/stores/checkers";

export const load: Load = async ({ parent, fetch }) => {
    const data = await parent();

    if (!get(checkers)) await refreshCheckers(fetch);

    return {
        ...data,
    };
};
